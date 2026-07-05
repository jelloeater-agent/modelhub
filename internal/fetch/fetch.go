// Package fetch handles fetching and parsing data from all sources.
package fetch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DefaultClient is reused across fetches for connection pooling.
var DefaultClient = &http.Client{
	Timeout: 45 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:    10,
		IdleConnTimeout: 30 * time.Second,
	},
}

// FetchJSON fetches a URL and unmarshals the JSON response.
// If apiKey is non-empty, it's set as the x-api-key header.
func FetchJSON(url string, apiKey string, target any) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("fetch %s: create req: %w", url, err)
	}
	if apiKey != "" {
		req.Header.Set("x-api-key", apiKey)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("fetch %s: status %d: %s", url, resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("fetch %s: decode: %w", url, err)
	}
	return nil
}
