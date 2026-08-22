package netlink

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Ping sends an HTTP GET request to the NetLink relay ping endpoint.
func Ping(relayURL string, timeout time.Duration) (string, error) {
	if !strings.HasPrefix(relayURL, "http://") && !strings.HasPrefix(relayURL, "https://") {
		relayURL = "http://" + relayURL
	}
	endpoint := strings.TrimRight(relayURL, "/") + "/api/netconnect/ping"

	client := &http.Client{
		Timeout: timeout,
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ping failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// StartPingLoop sends periodic ping requests to the NetLink relay in a background goroutine.
func StartPingLoop(relayURL string, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Initial ping
		resp, err := Ping(relayURL, 5*time.Second)
		if err != nil {
			fmt.Printf("[NetLink Ping] Failed to ping relay (%s): %v\n", relayURL, err)
		} else {
			fmt.Printf("[NetLink Ping] Success: %s\n", strings.TrimSpace(resp))
		}

		for range ticker.C {
			resp, err := Ping(relayURL, 5*time.Second)
			if err != nil {
				fmt.Printf("[NetLink Ping] Failed to ping relay (%s): %v\n", relayURL, err)
			} else {
				fmt.Printf("[NetLink Ping] Success: %s\n", strings.TrimSpace(resp))
			}
		}
	}()
}