package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DeviceCodeResponse holds the response from initiating device authorization.
type DeviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
	Error                   string `json:"error,omitempty"`
}

// DeviceTokenResponse holds the result of polling for device authorization completion.
type DeviceTokenResponse struct {
	Status   string `json:"status,omitempty"` // "pending", "approved", "expired", "denied"
	Token    string `json:"token,omitempty"`
	TargetID string `json:"target_id,omitempty"`
	Username string `json:"username,omitempty"`
	Error    string `json:"error,omitempty"`
}

// StartDeviceAuth requests a new device authorization session from the NetLink relay server.
func (c *Client) StartDeviceAuth(ctx context.Context, deviceName string) (*DeviceCodeResponse, error) {
	c.mu.RLock()
	relayURL := c.relayURL
	if deviceName == "" {
		deviceName = c.targetID
	}
	httpClient := c.httpClient
	c.mu.RUnlock()

	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	payload := map[string]string{
		"device_name": deviceName,
		"client_type": "netconnect-desktop",
	}
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/api/auth/device/code", relayURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate device auth on %s: %w", relayURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("device auth request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var res DeviceCodeResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("invalid device code response: %w", err)
	}

	if res.Error != "" {
		return nil, fmt.Errorf("device auth error: %s", res.Error)
	}

	if res.Interval <= 0 {
		res.Interval = 2
	}
	if res.ExpiresIn <= 0 {
		res.ExpiresIn = 300
	}
	if res.VerificationURIComplete == "" && res.VerificationURI != "" && res.UserCode != "" {
		sep := "?"
		if strings.Contains(res.VerificationURI, "?") {
			sep = "&"
		}
		res.VerificationURIComplete = fmt.Sprintf("%s%scode=%s", res.VerificationURI, sep, res.UserCode)
	}

	return &res, nil
}

// PollDeviceAuth polls the NetLink relay server for device authorization approval.
func (c *Client) PollDeviceAuth(ctx context.Context, deviceCode string) (*DeviceTokenResponse, error) {
	c.mu.RLock()
	relayURL := c.relayURL
	httpClient := c.httpClient
	c.mu.RUnlock()

	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	payload := map[string]string{
		"device_code": deviceCode,
	}
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/api/auth/device/token", relayURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("device token poll failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var res DeviceTokenResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("invalid device token poll response: %w", err)
	}

	if res.Status == "approved" || (res.Token != "" && res.Error == "") {
		c.mu.Lock()
		c.token = res.Token
		c.ticket = ""
		if res.TargetID != "" {
			c.targetID = res.TargetID
		}
		c.username = res.Username
		c.mu.Unlock()
	}

	return &res, nil
}
