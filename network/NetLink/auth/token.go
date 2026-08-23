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

	"github.com/leonst036/NetConnect/utils"
)

// InvalidateToken clears both the cached token and ticket.
func (c *Client) InvalidateToken() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = ""
	c.ticket = ""
}

// GetToken returns an existing token or requests a new one from the relay.
func (c *Client) GetToken(ctx context.Context) (string, error) {
	c.mu.RLock()
	if c.token != "" {
		token := c.token
		c.mu.RUnlock()
		return token, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	return c.getTokenLocked(ctx)
}

func (c *Client) getTokenLocked(ctx context.Context) (string, error) {
	if c.token != "" {
		return c.token, nil
	}

	if token := utils.GetEnv("NETLINK_TOKEN", ""); token != "" {
		c.token = token
		return c.token, nil
	}
	if token := utils.GetEnv("RELAY_TOKEN", ""); token != "" {
		c.token = token
		return c.token, nil
	}

	username := utils.GetEnv("NETLINK_USERNAME", "")
	password := utils.GetEnv("NETLINK_PASSWORD", "")
	if username != "" && password != "" {
		token, err := c.login(ctx, username, password)
		if err == nil && token != "" {
			c.token = token
			return c.token, nil
		}
	}

	token, err := c.requestDeviceToken(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to obtain device token: %w", err)
	}

	c.token = token
	return c.token, nil
}

// requestDeviceToken requests a device JWT token via /api/validate-target.
func (c *Client) requestDeviceToken(ctx context.Context) (string, error) {
	return c.requestDeviceTokenFor(ctx, c.relayURL, c.targetID)
}

// requestDeviceTokenFor requests a device JWT token for a specified relay URL and target ID.
func (c *Client) requestDeviceTokenFor(ctx context.Context, relayURL, targetID string) (string, error) {
	if !strings.HasPrefix(relayURL, "http://") && !strings.HasPrefix(relayURL, "https://") {
		relayURL = "http://" + relayURL
	}
	relayURL = strings.TrimRight(relayURL, "/")

	endpoint := fmt.Sprintf("%s/api/validate-target?target=%s", relayURL, targetID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return "", err
	}

	httpClient := c.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("validate-target request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("validate-target returned status %d: %s", resp.StatusCode, string(body))
	}

	var res ValidateTargetResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return "", fmt.Errorf("invalid json response: %w", err)
	}

	if !res.Valid || res.Token == "" {
		if res.Error != "" {
			return "", fmt.Errorf("device validation failed: %s", res.Error)
		}
		return "", fmt.Errorf("device target %s is not valid or no token returned", targetID)
	}

	return res.Token, nil
}

func (c *Client) UpdateConfig(ctx context.Context, newRelayURL, newTargetID string) error {
	c.mu.RLock()
	curRelayURL := c.relayURL
	curTargetID := c.targetID
	c.mu.RUnlock()

	targetRelayURL := strings.TrimSpace(newRelayURL)
	if targetRelayURL == "" {
		targetRelayURL = curRelayURL
	}
	if !strings.HasPrefix(targetRelayURL, "http://") && !strings.HasPrefix(targetRelayURL, "https://") {
		targetRelayURL = "http://" + targetRelayURL
	}
	targetRelayURL = strings.TrimRight(targetRelayURL, "/")

	targetDeviceName := strings.TrimSpace(newTargetID)
	if targetDeviceName == "" {
		targetDeviceName = curTargetID
	}

	newToken, err := c.requestDeviceTokenFor(ctx, targetRelayURL, targetDeviceName)
	if err != nil {
		return fmt.Errorf("failed to validate device '%s' on %s: %w", targetDeviceName, targetRelayURL, err)
	}

	c.mu.Lock()
	c.relayURL = targetRelayURL
	c.targetID = targetDeviceName
	c.token = newToken
	c.ticket = ""
	c.mu.Unlock()

	return nil
}

// login logs in using username and password to obtain a token.
func (c *Client) login(ctx context.Context, username, password string) (string, error) {
	endpoint := fmt.Sprintf("%s/api/login", c.relayURL)
	payload := map[string]string{
		"username": username,
		"password": password,
		"target":   c.targetID,
	}
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}

	var loginRes LoginResponse
	if err := json.Unmarshal(body, &loginRes); err != nil {
		return "", fmt.Errorf("invalid login response: %w", err)
	}

	if loginRes.Token == "" {
		return "", fmt.Errorf("no token returned from login: %s", loginRes.Error)
	}

	return loginRes.Token, nil
}
