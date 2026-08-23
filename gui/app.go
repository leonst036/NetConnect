package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/leonst036/NetConnect/network"
	netlink "github.com/leonst036/NetConnect/network/NetLink"
	"github.com/leonst036/NetConnect/network/NetLink/auth"
	"github.com/leonst036/NetConnect/utils"
)

const daemonBaseURL = "http://127.0.0.1:4545"

// SettingsData represents the settings state exchanged with the frontend.
type SettingsData struct {
	ServerAddress string `json:"serverAddress"`
	DeviceName    string `json:"deviceName"`
}

// App struct
type App struct {
	ctx         context.Context
	client      *auth.Client
	mu          sync.Mutex
	dev         *network.Device
	routeMgr    *network.DeviceRouteManager
	tunRouter   *network.TUNRouter
	isConnected bool
	isDevMock   bool
	httpClient  *http.Client
}

func NewApp() *App {
	relayURL := utils.GetEnv("NETLINK_RELAY_URL", "http://localhost:4535")
	client := auth.GetOrCreateClient(relayURL)
	return &App{
		client: client,
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	fmt.Println("[NetConnect GUI] Startup completed.")
}

func (a *App) shutdown(ctx context.Context) {
	_ = a.Disconnect()
}

func (a *App) isDaemonAvailable() bool {
	resp, err := a.httpClient.Get(daemonBaseURL + "/api/status")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// GetSettings returns current server address and device name.
func (a *App) GetSettings() SettingsData {
	if a.isDaemonAvailable() {
		resp, err := a.httpClient.Get(daemonBaseURL + "/api/status")
		if err == nil {
			defer resp.Body.Close()
			var data struct {
				ServerAddress string `json:"serverAddress"`
				DeviceName    string `json:"deviceName"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&data); err == nil && data.ServerAddress != "" {
				return SettingsData{
					ServerAddress: data.ServerAddress,
					DeviceName:    data.DeviceName,
				}
			}
		}
	}

	if a.client == nil {
		relayURL := utils.GetEnv("NETLINK_RELAY_URL", "http://localhost:4535")
		a.client = auth.GetOrCreateClient(relayURL)
	}
	return SettingsData{
		ServerAddress: a.client.RelayURL(),
		DeviceName:    a.client.TargetID(),
	}
}

// SaveSettings validates and applies settings.
func (a *App) SaveSettings(serverAddress string, deviceName string) error {
	if a.isDaemonAvailable() {
		body, _ := json.Marshal(map[string]string{
			"serverAddress": serverAddress,
			"deviceName":    deviceName,
		})
		resp, err := a.httpClient.Post(daemonBaseURL+"/api/settings", "application/json", bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("daemon settings update failed: %w", err)
		}
		defer resp.Body.Close()
	}

	if a.client == nil {
		relayURL := utils.GetEnv("NETLINK_RELAY_URL", "http://localhost:4535")
		a.client = auth.GetOrCreateClient(relayURL)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.client.UpdateConfig(ctx, serverAddress, deviceName); err != nil {
		return err
	}

	fmt.Printf("[NetConnect GUI] Settings updated: server=%s, device=%s\n", a.client.RelayURL(), a.client.TargetID())
	return nil
}

// IsConnected returns whether the connection is active.
func (a *App) IsConnected() bool {
	if a.isDaemonAvailable() {
		resp, err := a.httpClient.Get(daemonBaseURL + "/api/status")
		if err == nil {
			defer resp.Body.Close()
			var data struct {
				IsConnected bool `json:"isConnected"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&data); err == nil {
				return data.IsConnected
			}
		}
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	return a.isConnected
}

// Connect delegates to the root Daemon if active, or creates the local TUN device.
func (a *App) Connect() error {
	fmt.Println("[NetConnect GUI] Connect button clicked!")
	if a.isDaemonAvailable() {
		fmt.Println("[NetConnect GUI] Forwarding Connect request to background Daemon (127.0.0.1:4545)...")
		resp, err := a.httpClient.Post(daemonBaseURL+"/api/connect", "application/json", nil)
		if err != nil {
			fmt.Printf("[NetConnect GUI] Daemon connection error: %v\n", err)
			return fmt.Errorf("failed to contact NetConnect daemon: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			var errData struct {
				Error string `json:"error"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&errData)
			fmt.Printf("[NetConnect GUI] Daemon returned error: %s\n", errData.Error)
			return fmt.Errorf("daemon error: %s", errData.Error)
		}
		fmt.Println("[NetConnect GUI] Successfully connected via background root daemon.")
		return nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.isConnected {
		return nil
	}

	if a.client == nil {
		relayURL := utils.GetEnv("NETLINK_RELAY_URL", "http://localhost:4535")
		a.client = auth.GetOrCreateClient(relayURL)
	}

	relayURL := a.client.RelayURL()

	// Verify connection to relay
	_, err := netlink.Ping(relayURL, 5*time.Second)
	if err != nil {
		fmt.Printf("[NetConnect GUI] Relay ping failed: %v\n", err)
		return fmt.Errorf("relay ping failed: %w", err)
	}

	// Try creating local TUN device
	overlayCIDR, err := network.DetectCIDR()
	if err == nil {
		targetSubnets, errSubnets := network.DetectSubnets()
		if errSubnets == nil {
			dev, errDev := network.CreateVirtualDevice(overlayCIDR, targetSubnets)
			if errDev == nil {
				a.dev = dev
				routeMgr := network.NewDeviceRouteManager(dev.Name(), dev.OverlayIP, relayURL)
				routeMgr.StartSyncLoop(5 * time.Minute)
				a.routeMgr = routeMgr

				tunRouter := network.NewTUNRouter(dev.Interface, routeMgr, relayURL, a.client.TargetID())
				go tunRouter.Start()
				a.tunRouter = tunRouter

				a.isConnected = true
				fmt.Printf("[NetConnect GUI] Connected directly on %s\n", dev.Name())
				return nil
			} else {
				fmt.Printf("[NetConnect GUI] Local TUN creation failed: %v\n", errDev)
			}
		}
	}

	a.isDevMock = true
	a.isConnected = true
	fmt.Println("[NetConnect GUI] Notice: Running in Dev Mock Mode. Start `sudo go run main.go` for full TUN device & kernel routing.")
	return nil
}

// Disconnect delegates to root Daemon or cleans up local state.
func (a *App) Disconnect() error {
	fmt.Println("[NetConnect GUI] Disconnect button clicked!")
	if a.isDaemonAvailable() {
		resp, err := a.httpClient.Post(daemonBaseURL+"/api/disconnect", "application/json", nil)
		if err != nil {
			fmt.Printf("[NetConnect GUI] Daemon disconnect error: %v\n", err)
			return fmt.Errorf("failed to contact daemon: %w", err)
		}
		defer resp.Body.Close()
		fmt.Println("[NetConnect GUI] Disconnected via background daemon.")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.routeMgr != nil {
		a.routeMgr.Stop()
		a.routeMgr = nil
	}
	if a.tunRouter != nil {
		a.tunRouter.Stop()
		a.tunRouter = nil
	}
	if a.dev != nil {
		_ = a.dev.Close()
		a.dev = nil
	}

	a.isConnected = false
	a.isDevMock = false
	fmt.Println("[NetConnect GUI] Disconnected successfully.")
	return nil
}
