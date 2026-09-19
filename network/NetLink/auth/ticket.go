package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// InvalidateTicket clears the cached ticket.
func (c *Client) InvalidateTicket() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ticket = ""
}

// GetTicket returns a valid ticket, requesting a new one if necessary.
func (c *Client) GetTicket(ctx context.Context) (string, error) {
	c.mu.RLock()
	if c.ticket != "" {
		t := c.ticket
		c.mu.RUnlock()
		return t, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ticket != "" {
		return c.ticket, nil
	}

	token, err := c.getTokenLocked(ctx)
	if err != nil {
		return "", err
	}

	ticket, err := c.requestTicket(ctx, token)
	if err != nil {
		return "", fmt.Errorf("failed to obtain ticket: %w", err)
	}

	c.ticket = ticket
	return c.ticket, nil
}

// requests a new ticket from /api/auth/ticket.
func (c *Client) requestTicket(ctx context.Context, token string) (string, error) {
	endpoint := fmt.Sprintf("%s/api/auth/ticket", c.relayURL)
	payload := map[string]string{}
	if c.targetID != "" {
		payload["target"] = c.targetID
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
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ticket request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ticket request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var res TicketResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return "", fmt.Errorf("invalid ticket response: %w", err)
	}

	if !res.Success || res.Ticket == "" {
		errMsg := res.Error
		if errMsg == "" {
			errMsg = res.Details
		}
		if errMsg == "" {
			errMsg = "no ticket returned"
		}
		return "", fmt.Errorf("failed to get ticket: %s", errMsg)
	}

	return res.Ticket, nil
}
