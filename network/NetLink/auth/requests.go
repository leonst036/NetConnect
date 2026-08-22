package auth

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Do executes an HTTP request with automatic ticket authentication and retry on 401.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	ticket, err := c.GetTicket(req.Context())
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket for request: %w", err)
	}

	var bodyBytes []byte
	if req.Body != nil && req.Body != http.NoBody {
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	req.Header.Set("Authorization", "Ticket "+ticket)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// Retry once on 401 Unauthorized with a refreshed ticket
	if resp.StatusCode == http.StatusUnauthorized {
		_ = resp.Body.Close()
		c.InvalidateTicket()

		newTicket, err := c.GetTicket(req.Context())
		if err != nil {
			return nil, fmt.Errorf("failed to refresh ticket after 401: %w", err)
		}

		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}
		req.Header.Set("Authorization", "Ticket "+newTicket)

		return c.httpClient.Do(req)
	}

	return resp, nil
}

// Request creates and sends an authenticated HTTP request using the ticket system.
func (c *Client) Request(ctx context.Context, method, endpoint string, body io.Reader) (*http.Response, error) {
	url := endpoint
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = fmt.Sprintf("%s/%s", c.relayURL, strings.TrimPrefix(endpoint, "/"))
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	return c.Do(req)
}

// Get performs an authenticated GET request using the ticket system.
func (c *Client) Get(ctx context.Context, endpoint string) (*http.Response, error) {
	return c.Request(ctx, http.MethodGet, endpoint, nil)
}

// Post performs an authenticated POST request using the ticket system.
func (c *Client) Post(ctx context.Context, endpoint string, contentType string, body io.Reader) (*http.Response, error) {
	url := endpoint
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = fmt.Sprintf("%s/%s", c.relayURL, strings.TrimPrefix(endpoint, "/"))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	return c.Do(req)
}
