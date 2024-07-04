package vms

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/bestserversio/spy/internal/config"
)

const STEAM_API_URL = "https://api.steampowered.com/IGameServersService/GetServerList/v1/"

func RetrieveServers(cfg *config.Config, appId int) ([]Server, error) {
	var servers []Server
	var err error = nil

	if !cfg.Vms.Enabled {
		return servers, err
	}

	// Create HTTP client with timeout.
	client := http.Client{
		Timeout: time.Duration(cfg.Vms.Timeout) * time.Second,
	}

	// Compile query parameters.
	params := url.Values{}

	// Add limit parameter.
	params.Add("limit", strconv.Itoa(cfg.Vms.Limit))

	// Add key parameter.
	params.Add("key", cfg.Vms.ApiToken)

	// Start building filters string.
	filters := fmt.Sprintf("\\appid\\%d", appId)

	// Add no players if exclude empty is set.
	if cfg.Vms.ExcludeEmpty {
		filters = fmt.Sprintf("%s\\noplayers\\1", filters)
	}

	// Add filters parameter
	params.Add("filters", filters)

	// Compile URL.
	url := fmt.Sprintf("%s?%s", STEAM_API_URL, params.Encode())

	// Create response and check.
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return servers, err
	}

	// Only accept JSON.
	req.Header.Add("Content-Type", "application/json")

	// Send response and check.
	res, err := client.Do(req)

	if err != nil {
		return servers, nil
	}

	defer res.Body.Close()

	// Read response.
	b, err := io.ReadAll(res.Body)

	if err != nil {
		return servers, err
	}

	retrieveResp := Response{}

	err = json.Unmarshal(b, &retrieveResp)

	servers = retrieveResp.Response.Servers

	return servers, err
}
