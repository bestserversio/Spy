package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bestserversio/spy/internal/config"
	"github.com/bestserversio/spy/internal/scanners"
	"github.com/bestserversio/spy/internal/servers"
	"github.com/bestserversio/spy/internal/utils"
	"github.com/bestserversio/spy/internal/vms"
)

const VERSION = "1.0.0"
const HELPMENU = `
./spy [OPTIONS]\n\n
-l --list => Prints config and exits.\n
-v --version => Prints version and exits.\n
-h --help => Prints help menu and exits.\n
-c --cfg => Path to config file.\n
`

func main() {
	// Command line options and parse command line.
	var list bool
	var version bool
	var help bool

	flag.BoolVar(&list, "l", false, "Prints config settings and exits.")
	flag.BoolVar(&list, "list", false, "Prints config settings and exits.")

	flag.BoolVar(&version, "v", false, "Prints version and exits.")
	flag.BoolVar(&version, "version", false, "Prints number and exits.")

	flag.BoolVar(&help, "h", false, "Prints help menu and exits.")
	flag.BoolVar(&help, "help", false, "Prints help menu and exits.")

	cfgPath := flag.String("cfg", "/etc/bestservers/spy.json", "Path to config file.")

	flag.Parse()

	// Check for version.
	if version {
		fmt.Println(VERSION)

		os.Exit(0)
	}

	// Check for help menu.
	if version {
		fmt.Print(HELPMENU)

		os.Exit(0)
	}

	// Initialize config.
	cfg := config.Config{}

	// Load defaults.
	cfg.LoadDefaults()

	// Attempt to load config.
	err := cfg.LoadFromFs(*cfgPath)

	if err != nil {
		fmt.Println("Error loading config file. Resorting to defaults...")
		fmt.Println(err)
	}

	utils.DebugMsg(2, cfg.Verbose, "[CFG] Initial config loaded...")

	// Check if we want to print our config settings.
	if list {
		cfg.PrintConfig()

		os.Exit(0)
	}

	// Check for web API updating.
	if cfg.WebApi.Enabled {
		go func() {
			for {
				// Make sure web config is still enabled.
				if !cfg.WebApi.Enabled {
					return
				}

				// Get web API interval.
				interval := time.Duration(cfg.WebApi.Interval) * time.Second

				utils.DebugMsg(3, cfg.Verbose, "[WEB_API] Retrieving web API from '%s%s'.", cfg.WebApi.Host, cfg.WebApi.Endpoint)

				data, err := cfg.LoadFromWeb()

				if err != nil {
					utils.DebugMsg(1, cfg.Verbose, "[WEB_API] Failed to retrieve web API from '%s%s'.", cfg.WebApi.Host, cfg.WebApi.Endpoint)

					time.Sleep(interval)

					continue
				}

				utils.DebugMsg(6, cfg.Verbose, "[WEB_API] Loading JSON => %s", data)

				// Check if we need to save new config to the file system.
				if cfg.WebApi.SaveToFs {
					err := os.WriteFile(*cfgPath, []byte(data), 0644)

					if err != nil {
						utils.DebugMsg(0, cfg.Verbose, "[WEB_API] Failed to write web config to file system (%s) due to error :: %s", *cfgPath, err.Error())
					} else {
						utils.DebugMsg(2, cfg.Verbose, "[WEB_API] Successfully wrote new data to file system (%s)!", *cfgPath)
					}
				}

				// Resetup scanners.
				scanners.SetupScanners(&cfg)

				// If we have no interval, close this function now.
				if cfg.WebApi.Interval < 1 {
					return
				}

				time.Sleep(interval)
			}
		}()
	}

	// Setup remove inactive.
	go func() {
		for {
			if !cfg.RemoveInactive.Enabled {
				time.Sleep(time.Second * 1)

				continue
			}

			var inactive_time *int = nil

			if cfg.RemoveInactive.InactiveTime > 0 {
				inactive_time = new(int)
				*inactive_time = cfg.RemoveInactive.InactiveTime
			}

			cnt, err := servers.RemoveInactive(&cfg, inactive_time)

			if err != nil {
				utils.DebugMsg(1, cfg.Verbose, "[INACTIVE] Failed to remove inactive servers due to error :: %s", err.Error())
			} else {
				utils.DebugMsg(1, cfg.Verbose, "[INACTIVE] Removed %d inactive servers!", cnt)
			}

			time.Sleep(time.Second * time.Duration(cfg.RemoveInactive.Interval))
		}
	}()

	// Create VMS.
	go vms.DoVms(&cfg)

	// Setup scanners.
	scanners.SetupScanners(&cfg)

	// Make a signal.
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)

	<-sigc

	utils.DebugMsg(0, cfg.Verbose, "[MAIN] Exiting...")
	os.Exit(0)
}
