package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/leonst036/NetConnect/network"
	netlink "github.com/leonst036/NetConnect/network/NetLink"
	"github.com/leonst036/NetConnect/network/NetLink/auth"
	"github.com/leonst036/NetConnect/utils"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const daemonBaseURL = "http://127.0.0.1:4545"

type SettingsData struct {
	ServerAddress   string `json:"serverAddress"`
	DeviceName      string `json:"deviceName"`
	Username        string `json:"username,omitempty"`
	IsAuthenticated bool   `json:"isAuthenticated"`
	AutoStart       bool   `json:"autoStart"`
}

type App struct {
	ctx         context.Context
	client      *auth.Client
	mu          sync.Mutex
	dev         *network.Device
	routeMgr    *network.DeviceRouteManager
	tunRouter   *network.TUNRouter
	isConnected bool
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

func (a *App) GetSettings() SettingsData {
	if a.isDaemonAvailable() {
		resp, err := a.httpClient.Get(daemonBaseURL + "/api/status")
		if err == nil {
			defer resp.Body.Close()
			var data struct {
				ServerAddress   string `json:"serverAddress"`
				DeviceName      string `json:"deviceName"`
				Username        string `json:"username"`
				IsAuthenticated bool   `json:"isAuthenticated"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&data); err == nil && data.ServerAddress != "" {
				return SettingsData{
					ServerAddress:   data.ServerAddress,
					DeviceName:      data.DeviceName,
					Username:        data.Username,
					IsAuthenticated: data.IsAuthenticated,
					AutoStart:       a.GetAutoStart(),
				}
			}
		}
	}

	if a.client == nil {
		relayURL := utils.GetEnv("NETLINK_RELAY_URL", "http://localhost:4535")
		a.client = auth.GetOrCreateClient(relayURL)
	}
	return SettingsData{
		ServerAddress:   a.client.RelayURL(),
		DeviceName:      a.client.TargetID(),
		Username:        a.client.Username(),
		IsAuthenticated: a.client.Token() != "",
		AutoStart:       a.GetAutoStart(),
	}
}

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

		if resp.StatusCode != http.StatusOK {
			var errData struct {
				Error string `json:"error"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&errData)
			return fmt.Errorf("daemon settings update failed: %s", errData.Error)
		}
		return nil
	}

	if a.client == nil {
		relayURL := utils.GetEnv("NETLINK_RELAY_URL", "http://localhost:4535")
		a.client = auth.GetOrCreateClient(relayURL)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return a.client.UpdateConfig(ctx, serverAddress, deviceName)
}

func (a *App) StartDeviceLogin(serverAddress string, deviceName string) (*auth.DeviceCodeResponse, error) {
	if a.isDaemonAvailable() {
		body, _ := json.Marshal(map[string]string{
			"serverAddress": serverAddress,
			"deviceName":    deviceName,
		})
		resp, err := a.httpClient.Post(daemonBaseURL+"/api/auth/device-flow/start", "application/json", bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("daemon device auth start failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			var errData struct {
				Error string `json:"error"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&errData)
			return nil, fmt.Errorf("device auth start failed: %s", errData.Error)
		}

		var res auth.DeviceCodeResponse
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			return nil, fmt.Errorf("invalid response from daemon: %w", err)
		}
		return &res, nil
	}

	if a.client == nil || (serverAddress != "" && a.client.RelayURL() != serverAddress) {
		if serverAddress == "" {
			serverAddress = utils.GetEnv("NETLINK_RELAY_URL", "http://localhost:4535")
		}
		a.client = auth.GetOrCreateClient(serverAddress)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return a.client.StartDeviceAuth(ctx, deviceName)
}

func (a *App) PollDeviceLogin(deviceCode string) (*auth.DeviceTokenResponse, error) {
	if a.isDaemonAvailable() {
		body, _ := json.Marshal(map[string]string{
			"device_code": deviceCode,
		})
		resp, err := a.httpClient.Post(daemonBaseURL+"/api/auth/device-flow/poll", "application/json", bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("daemon device auth poll failed: %w", err)
		}
		defer resp.Body.Close()

		var res auth.DeviceTokenResponse
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			return nil, fmt.Errorf("invalid response from daemon: %w", err)
		}
		return &res, nil
	}

	if a.client == nil {
		relayURL := utils.GetEnv("NETLINK_RELAY_URL", "http://localhost:4535")
		a.client = auth.GetOrCreateClient(relayURL)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return a.client.PollDeviceAuth(ctx, deviceCode)
}

func (a *App) Logout() error {
	if a.isDaemonAvailable() {
		resp, err := a.httpClient.Post(daemonBaseURL+"/api/auth/logout", "application/json", nil)
		if err != nil {
			return fmt.Errorf("daemon logout failed: %w", err)
		}
		defer resp.Body.Close()
	}

	if a.client != nil {
		a.client.Logout()
	}

	return nil
}

func (a *App) OpenVerificationURL(url string) {
	if a.ctx != nil {
		runtime.BrowserOpenURL(a.ctx, url)
	}
}

func (a *App) HideWindow() {
	if a.ctx != nil {
		runtime.WindowHide(a.ctx)
	}
}

func (a *App) ShowWindow() {
	if a.ctx != nil {
		runtime.WindowShow(a.ctx)
	}
}

func (a *App) QuitApp() {
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}

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
		return false
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if a.dev != nil {
		return a.isConnected
	}
	return false
}

func (a *App) Connect() error {
	if a.isDaemonAvailable() {
		resp, err := a.httpClient.Post(daemonBaseURL+"/api/connect", "application/json", nil)
		if err != nil {
			return fmt.Errorf("failed to contact NetConnect daemon: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			var errData struct {
				Error string `json:"error"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&errData)
			return fmt.Errorf("daemon error: %s", errData.Error)
		}
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

	if _, err := netlink.Ping(relayURL, 5*time.Second); err != nil {
		return fmt.Errorf("relay ping failed: %w", err)
	}

	overlayCIDR, err := network.DetectCIDR()
	if err != nil {
		return fmt.Errorf("failed to detect CIDR: %w", err)
	}

	targetSubnets, errSubnets := network.DetectSubnets()
	if errSubnets != nil {
		return fmt.Errorf("failed to detect subnets: %w", errSubnets)
	}

	dev, errDev := network.CreateVirtualDevice(overlayCIDR, targetSubnets)
	if errDev != nil {
		return fmt.Errorf("local TUN creation failed: %w", errDev)
	}

	a.dev = dev
	routeMgr := network.NewDeviceRouteManager(dev.Name(), dev.OverlayIP, relayURL)
	routeMgr.StartSyncLoop(5 * time.Minute)
	a.routeMgr = routeMgr

	tunRouter := network.NewTUNRouter(dev.Interface, routeMgr, relayURL, a.client.TargetID())
	go tunRouter.Start()
	a.tunRouter = tunRouter

	a.isConnected = true
	return nil
}

func (a *App) Disconnect() error {
	if a.isDaemonAvailable() {
		resp, err := a.httpClient.Post(daemonBaseURL+"/api/disconnect", "application/json", nil)
		if err != nil {
			return fmt.Errorf("failed to contact daemon: %w", err)
		}
		defer resp.Body.Close()
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
	return nil
}

func (a *App) GetAutoStart() bool {
	cmd := exec.Command("systemctl", "--user", "is-enabled", "netconnect.service")
	if err := cmd.Run(); err == nil {
		return true
	}

	cmdSys := exec.Command("systemctl", "is-enabled", "netconnect.service")
	if err := cmdSys.Run(); err == nil {
		return true
	}

	homeDir, err := os.UserHomeDir()
	if err == nil {
		desktopPath := filepath.Join(homeDir, ".config", "autostart", "netconnect.desktop")
		if _, err := os.Stat(desktopPath); err == nil {
			return true
		}
	}

	return false
}

func (a *App) SetAutoStart(enabled bool) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	userServiceDir := filepath.Join(homeDir, ".config", "systemd", "user")
	userServicePath := filepath.Join(userServiceDir, "netconnect.service")
	autostartDir := filepath.Join(homeDir, ".config", "autostart")
	autostartPath := filepath.Join(autostartDir, "netconnect.desktop")

	if enabled {
		exe, err := os.Executable()
		if err != nil {
			exe = "/usr/local/bin/netconnect"
		}

		if err := os.MkdirAll(userServiceDir, 0755); err != nil {
			return fmt.Errorf("failed to create systemd user dir: %w", err)
		}

		serviceContent := fmt.Sprintf(`[Unit]
Description=NetConnect Overlay VPN Service
After=network.target default.target

[Service]
Type=simple
ExecStart=%s
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=default.target
`, exe)

		if err := os.WriteFile(userServicePath, []byte(serviceContent), 0644); err != nil {
			return fmt.Errorf("failed to write systemd user service: %w", err)
		}

		_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
		_ = exec.Command("systemctl", "--user", "enable", "netconnect.service").Run()

		_ = os.MkdirAll(autostartDir, 0755)
		autostartContent := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=NetConnect
Comment=NetConnect VPN Auto-Start
Exec=%s
Icon=netconnect
Terminal=false
Categories=Network;RemoteAccess;
X-GNOME-Autostart-enabled=true
`, exe)
		_ = os.WriteFile(autostartPath, []byte(autostartContent), 0644)

		return nil
	}

	_ = exec.Command("systemctl", "--user", "disable", "netconnect.service").Run()
	_ = os.Remove(userServicePath)
	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
	_ = os.Remove(autostartPath)

	return nil
}
