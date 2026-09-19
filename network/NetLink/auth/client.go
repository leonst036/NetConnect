package auth

import (
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/leonst036/NetConnect/utils"
)

// Client manages authentication tokens, tickets, and authorized HTTP requests to NetLink.
type Client struct {
	relayURL   string
	targetID   string
	deviceID   string
	token      string
	ticket     string
	username   string
	mu         sync.RWMutex
	httpClient *http.Client
}

var (
	defaultClient *Client
	defaultMu     sync.RWMutex
)

// NewClient initializes a new NetLink Client with auto-discovered target ID.
func NewClient(relayURL string) *Client {
	return NewClientWithTarget(relayURL, "")
}

// NewClientWithTarget initializes a new NetLink Client for a specific target ID.
func NewClientWithTarget(relayURL, targetID string) *Client {
	if !strings.HasPrefix(relayURL, "http://") && !strings.HasPrefix(relayURL, "https://") {
		relayURL = "http://" + relayURL
	}
	relayURL = strings.TrimRight(relayURL, "/")

	deviceID := utils.GetEnv("NETLINK_DEVICE_ID", "")
	if deviceID == "" {
		hostname, err := os.Hostname()
		if err == nil && hostname != "" {
			deviceID = "netconnect-" + hostname
		} else {
			deviceID = "netconnect-device"
		}
	}

	if targetID == "" {
		targetID = utils.GetEnv("NETLINK_TARGET_ID", "")
	}

	token := utils.GetEnv("NETLINK_TOKEN", "")
	if token == "" {
		token = utils.GetEnv("RELAY_TOKEN", "")
	}

	return &Client{
		relayURL: relayURL,
		targetID: targetID,
		deviceID: deviceID,
		token:    token,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetOrCreateClient returns or initializes the default Client.
func GetOrCreateClient(relayURL string) *Client {
	defaultMu.Lock()
	defer defaultMu.Unlock()

	normalized := relayURL
	if !strings.HasPrefix(normalized, "http://") && !strings.HasPrefix(normalized, "https://") {
		normalized = "http://" + normalized
	}
	normalized = strings.TrimRight(normalized, "/")

	if defaultClient == nil || defaultClient.relayURL != normalized {
		defaultClient = NewClient(relayURL)
	}
	return defaultClient
}

// SetDefaultClient sets the global default client.
func SetDefaultClient(client *Client) {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	defaultClient = client
}

// GetDefaultClient returns the global default client.
func GetDefaultClient() *Client {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultClient
}

// SetHTTPClient sets the underlying HTTP client.
func (c *Client) SetHTTPClient(client *http.Client) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.httpClient = client
}

// RelayURL returns the configured base relay URL.
func (c *Client) RelayURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.relayURL
}

// TargetID returns the device target ID.
func (c *Client) TargetID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.targetID
}

// DeviceID returns the client device ID.
func (c *Client) DeviceID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.deviceID
}

// Token returns the current device token.
func (c *Client) Token() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token
}

// Username returns the authenticated username, if logged in.
func (c *Client) Username() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.username
}

// Logout clears the stored tokens, tickets, and user session.
func (c *Client) Logout() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = ""
	c.ticket = ""
	c.username = ""
}

