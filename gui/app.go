package main

import (
	"context"
	"fmt"
	"time"

	"github.com/leonst036/NetConnect/network/NetLink/auth"
	netlink "github.com/leonst036/NetConnect/network/NetLink"
	"github.com/leonst036/NetConnect/utils"
)

// SettingsData represents the settings state exchanged with the frontend.
type SettingsData struct {
	ServerAddress string `json:"serverAddress"`
	DeviceName    string `json:"deviceName"`
}

// App struct
type App struct {
	ctx    context.Context
	client *auth.Client
}

func NewApp() *App {
	relayURL := utils.GetEnv("NETLINK_RELAY_URL", "http://localhost:5173")
	client := auth.GetOrCreateClient(relayURL)
	return &App{
		client: client,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// GetSettings returns current server address and device name.
func (a *App) GetSettings() SettingsData {
	if a.client == nil {
		relayURL := utils.GetEnv("NETLINK_RELAY_URL", "http://localhost:5173")
		a.client = auth.GetOrCreateClient(relayURL)
	}
	return SettingsData{
		ServerAddress: a.client.RelayURL(),
		DeviceName:    a.client.TargetID(),
	}
}

// SaveSettings validates and applies settings by requesting a new token from the relay.
// If the token request fails, the old token and settings are preserved, and an error is returned.
func (a *App) SaveSettings(serverAddress string, deviceName string) error {
	if a.client == nil {
		relayURL := utils.GetEnv("NETLINK_RELAY_URL", "http://localhost:5173")
		a.client = auth.GetOrCreateClient(relayURL)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.client.UpdateConfig(ctx, serverAddress, deviceName); err != nil {
		return err
	}

	fmt.Printf("[NetConnect] Settings updated: server=%s, device=%s\n", a.client.RelayURL(), a.client.TargetID())
	return nil
}

// Connect pings the NetLink relay.
func (a *App) Connect() error {
	if a.client == nil {
		relayURL := utils.GetEnv("NETLINK_RELAY_URL", "http://localhost:5173")
		a.client = auth.GetOrCreateClient(relayURL)
	}

	_, err := netlink.Ping(a.client.RelayURL(), 5*time.Second)
	if err != nil {
		fmt.Printf("[NetConnect] Connect ping failed: %v\n", err)
		return err
	}
	fmt.Println("[NetConnect] Connected successfully!")
	return nil
}

