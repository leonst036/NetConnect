package netlink

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/leonst036/NetConnect/network/NetLink/auth"
	"github.com/leonst036/NetConnect/utils"
)

const prefix = "[NetLink Ping]"

// Ping sends an HTTP GET request to the NetLink relay ping endpoint using the ticket system.
func Ping(relayURL string, timeout time.Duration) (string, error) {
	client := auth.GetOrCreateClient(relayURL)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resp, err := client.Get(ctx, "/api/netconnect/ping")
	if err != nil {
		return "", fmt.Errorf(prefix+" ping failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf(prefix+" relay returned status %d: %s", resp.StatusCode, string(body))
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
			fmt.Printf(prefix+" Failed to ping relay (%s): %v\n", relayURL, err)
		} else {
			fmt.Printf(prefix+" Success: %s\n", strings.TrimSpace(resp))
		}
		err_counter := 0
		for range ticker.C {
			resp, err := Ping(relayURL, 5*time.Second)
			if err != nil {
				fmt.Printf(prefix+" Failed to ping relay (%s): %v\n", relayURL, err)
				err_counter++
			} else {
				fmt.Printf(prefix+" Success: %s\n", strings.TrimSpace(resp))
				err_counter = 0
			}
			if err_counter >= 3 {
				fmt.Println(prefix + " Failed to ping relay, are you offline or is relay down?")
				fmt.Println(prefix+" Unresponded pings: ", err_counter)
				if err_counter == 3 {
					err := utils.NotifyUser("NetLink", "Failed to ping relay, are you offline or is relay down?")
					if err != nil {
						fmt.Println(prefix+" Failed to send notification: ", err)
					}
				}
			}
		}
	}()
}
