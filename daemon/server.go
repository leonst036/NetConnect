package daemon

import (
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

type DaemonState struct {
	IsConnected     bool   `json:"isConnected"`
	ServerAddress   string `json:"serverAddress"`
	DeviceName      string `json:"deviceName"`
	OverlayIP       string `json:"overlayIP"`
	Username        string `json:"username,omitempty"`
	IsAuthenticated bool   `json:"isAuthenticated"`
	Error           string `json:"error,omitempty"`
}

type DaemonServer struct {
	mu          sync.Mutex
	relayURL    string
	targetID    string
	client      *auth.Client
	dev         *network.Device
	routeMgr    *network.DeviceRouteManager
	tunRouter   *network.TUNRouter
	isConnected bool
	server      *http.Server
}

func NewDaemonServer(relayURL, targetID string) *DaemonServer {
	if relayURL == "" {
		relayURL = utils.GetEnv("NETLINK_RELAY_URL", "http://localhost:4535")
	}
	client := auth.GetOrCreateClient(relayURL)
	return &DaemonServer{
		relayURL: relayURL,
		targetID: targetID,
		client:   client,
	}
}

func (ds *DaemonServer) Start(port int) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/status", ds.handleStatus)
	mux.HandleFunc("/api/connect", ds.handleConnect)
	mux.HandleFunc("/api/disconnect", ds.handleDisconnect)
	mux.HandleFunc("/api/settings", ds.handleSettings)
	mux.HandleFunc("/api/auth/device-flow/start", ds.handleDeviceFlowStart)
	mux.HandleFunc("/api/auth/device-flow/poll", ds.handleDeviceFlowPoll)
	mux.HandleFunc("/api/auth/logout", ds.handleLogout)

	ds.server = &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", port),
		Handler: mux,
	}

	fmt.Printf("[NetConnect Daemon] Control server listening on http://127.0.0.1:%d\n", port)
	return ds.server.ListenAndServe()
}

func (ds *DaemonServer) Stop() {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	_ = ds.disconnectLocked()
	if ds.server != nil {
		_ = ds.server.Close()
	}
}

func (ds *DaemonServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	overlayIP := ""
	if ds.dev != nil {
		overlayIP = ds.dev.OverlayIP.String()
	}

	username := ""
	isAuthenticated := false
	if ds.client != nil {
		username = ds.client.Username()
		isAuthenticated = ds.client.Token() != ""
	}

	state := DaemonState{
		IsConnected:     ds.isConnected,
		ServerAddress:   ds.relayURL,
		DeviceName:      ds.targetID,
		OverlayIP:       overlayIP,
		Username:        username,
		IsAuthenticated: isAuthenticated,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(state)
}

func (ds *DaemonServer) handleDeviceFlowStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		ServerAddress string `json:"serverAddress"`
		DeviceName    string `json:"deviceName"`
	}
	_ = json.NewDecoder(r.Body).Decode(&payload)

	ds.mu.Lock()
	if payload.ServerAddress != "" {
		ds.relayURL = payload.ServerAddress
	}
	if payload.DeviceName != "" {
		ds.targetID = payload.DeviceName
	}
	if ds.client == nil || ds.client.RelayURL() != ds.relayURL {
		ds.client = auth.GetOrCreateClient(ds.relayURL)
	}
	client := ds.client
	deviceName := ds.targetID
	ds.mu.Unlock()

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	res, err := client.StartDeviceAuth(ctx, deviceName)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

func (ds *DaemonServer) handleDeviceFlowPoll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		DeviceCode string `json:"device_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.DeviceCode == "" {
		http.Error(w, "Invalid device code", http.StatusBadRequest)
		return
	}

	ds.mu.Lock()
	client := ds.client
	ds.mu.Unlock()

	if client == nil {
		http.Error(w, "Auth client not initialized", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	res, err := client.PollDeviceAuth(ctx, payload.DeviceCode)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if res.Status == "approved" || (res.Token != "" && res.Error == "") {
		ds.mu.Lock()
		if res.TargetID != "" {
			ds.targetID = res.TargetID
		}
		ds.mu.Unlock()
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

func (ds *DaemonServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	ds.mu.Lock()
	if ds.client != nil {
		ds.client.Logout()
	}
	ds.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (ds *DaemonServer) handleConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	fmt.Println("[NetConnect Daemon] Received Connect request from GUI...")
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if err := ds.connectLocked(); err != nil {
		fmt.Printf("[NetConnect Daemon] Connect failed: %v\n", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	fmt.Println("[NetConnect Daemon] Connected successfully.")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (ds *DaemonServer) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	fmt.Println("[NetConnect Daemon] Received Disconnect request from GUI...")
	ds.mu.Lock()
	defer ds.mu.Unlock()

	_ = ds.disconnectLocked()

	fmt.Println("[NetConnect Daemon] Disconnected successfully.")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (ds *DaemonServer) handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		ServerAddress string `json:"serverAddress"`
		DeviceName    string `json:"deviceName"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ds.mu.Lock()
	defer ds.mu.Unlock()

	if payload.ServerAddress != "" {
		ds.relayURL = payload.ServerAddress
	}
	if payload.DeviceName != "" {
		ds.targetID = payload.DeviceName
	}

	if ds.client != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		_ = ds.client.UpdateConfig(ctx, ds.relayURL, ds.targetID)
	}

	fmt.Printf("[NetConnect Daemon] Settings updated: server=%s, device=%s\n", ds.relayURL, ds.targetID)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (ds *DaemonServer) connectLocked() error {
	if ds.isConnected {
		return nil
	}

	if _, err := netlink.Ping(ds.relayURL, 5*time.Second); err != nil {
		return fmt.Errorf("relay ping failed: %w", err)
	}

	overlayCIDR, err := network.DetectCIDR()
	if err != nil {
		return fmt.Errorf("detect CIDR failed: %w", err)
	}
	targetSubnets, err := network.DetectSubnets()
	if err != nil {
		return fmt.Errorf("detect subnets failed: %w", err)
	}

	dev, err := network.CreateVirtualDevice(overlayCIDR, targetSubnets)
	if err != nil {
		return fmt.Errorf("TUN device creation failed (%w). If kernel was updated, please reboot to load the 'tun' module.", err)
	}
	ds.dev = dev

	routeMgr := network.NewDeviceRouteManager(dev.Name(), dev.OverlayIP, ds.relayURL)
	routeMgr.StartSyncLoop(5 * time.Minute)
	ds.routeMgr = routeMgr

	tunRouter := network.NewTUNRouter(dev.Interface, routeMgr, ds.relayURL, ds.targetID)
	go tunRouter.Start()
	ds.tunRouter = tunRouter

	ds.isConnected = true
	fmt.Printf("[NetConnect Daemon] Connected on %s (Overlay IP: %s)\n", dev.Name(), dev.OverlayIP.String())
	return nil
}


func (ds *DaemonServer) disconnectLocked() error {
	if !ds.isConnected {
		return nil
	}

	if ds.routeMgr != nil {
		ds.routeMgr.Stop()
		ds.routeMgr = nil
	}
	if ds.tunRouter != nil {
		ds.tunRouter.Stop()
		ds.tunRouter = nil
	}
	if ds.dev != nil {
		_ = ds.dev.Close()
		ds.dev = nil
	}

	ds.isConnected = false
	return nil
}
