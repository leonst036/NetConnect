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
	IsConnected   bool   `json:"isConnected"`
	ServerAddress string `json:"serverAddress"`
	DeviceName    string `json:"deviceName"`
	OverlayIP     string `json:"overlayIP"`
	Error         string `json:"error,omitempty"`
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

	state := DaemonState{
		IsConnected:   ds.isConnected,
		ServerAddress: ds.relayURL,
		DeviceName:    ds.targetID,
		OverlayIP:     overlayIP,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(state)
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

	fmt.Printf("[NetConnect Daemon] Pinging relay at %s...\n", ds.relayURL)
	// 1. Ping Relay
	_, err := netlink.Ping(ds.relayURL, 5*time.Second)
	if err != nil {
		return fmt.Errorf("relay ping failed: %w", err)
	}
	fmt.Println("[NetConnect Daemon] Ping relay successful.")

	// 2. Detect CIDR
	overlayCIDR, err := network.DetectCIDR()
	if err != nil {
		return fmt.Errorf("detect CIDR failed: %w", err)
	}
	targetSubnets, err := network.DetectSubnets()
	if err != nil {
		return fmt.Errorf("detect subnets failed: %w", err)
	}

	// 3. Create TUN Device
	dev, err := network.CreateVirtualDevice(overlayCIDR, targetSubnets)
	if err != nil {
		return fmt.Errorf("TUN device creation failed (%w). If kernel was updated, please reboot to load the 'tun' module.", err)
	}
	ds.dev = dev

	// 4. Start route manager
	routeMgr := network.NewDeviceRouteManager(dev.Name(), dev.OverlayIP, ds.relayURL)
	routeMgr.StartSyncLoop(5 * time.Minute)
	ds.routeMgr = routeMgr

	// 5. Start TUN router
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
