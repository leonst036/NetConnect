package devices

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/leonst036/NetConnect/network/NetLink/auth"
)

const prefix = "[NetLink Devices]"

func FetchDevices(relayURL string, timeout time.Duration) ([]Device, error) {
	client := auth.GetOrCreateClient(relayURL)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resp, err := client.Get(ctx, "/api/net-graph/scan")
	if err != nil {
		return nil, fmt.Errorf(prefix+" failed to fetch devices: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(prefix+" relay returned status %d: %s", resp.StatusCode, string(body))
	}

	return parseDevices(body)
}

func parseDevices(body []byte) ([]Device, error) {
	var devList []Device
	if err := json.Unmarshal(body, &devList); err != nil {
		return nil, fmt.Errorf(prefix+" failed to parse device list: %w", err)
	}
	return devList, nil
}
