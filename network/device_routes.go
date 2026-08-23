package network

import (
	"fmt"
	"net"
	"os/exec"
	"sync"
	"time"

	"github.com/leonst036/NetConnect/network/NetLink/devices"
)

const devPrefix = "[NetConnect DeviceRoutes]"

// DeviceRouteManager manages dynamic /32 host routes for discovered NetLink devices.
type DeviceRouteManager struct {
	devName      string
	overlayIP    net.IP
	relayURL     string
	mu           sync.RWMutex
	mappedRoutes map[string]bool
	stopChan     chan struct{}
}

// NewDeviceRouteManager creates a new DeviceRouteManager.
func NewDeviceRouteManager(devName string, overlayIP net.IP, relayURL string) *DeviceRouteManager {
	return &DeviceRouteManager{
		devName:      devName,
		overlayIP:    overlayIP,
		relayURL:     relayURL,
		mappedRoutes: make(map[string]bool),
		stopChan:     make(chan struct{}),
	}
}

// StartSyncLoop starts periodic route synchronization every interval (default 5 minutes).
func (drm *DeviceRouteManager) StartSyncLoop(interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Minute
	}

	go func() {
		// Run initial sync immediately
		if err := drm.Sync(); err != nil {
			fmt.Printf("%s Initial sync failed: %v\n", devPrefix, err)
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-drm.stopChan:
				return
			case <-ticker.C:
				if err := drm.Sync(); err != nil {
					fmt.Printf("%s Route sync error: %v\n", devPrefix, err)
				}
			}
		}
	}()
}

// Stop stops the sync loop and cleans up mapped routes.
func (drm *DeviceRouteManager) Stop() {
	close(drm.stopChan)
	drm.ClearAllRoutes()
}

// Sync fetches devices from NetLink and synchronizes the TUN device routing table.
func (drm *DeviceRouteManager) Sync() error {
	devList, err := devices.FetchDevices(drm.relayURL, 15*time.Second)
	if err != nil {
		return fmt.Errorf("failed to fetch devices from %s: %w", drm.relayURL, err)
	}

	newDeviceIPs := make(map[string]bool)

	for _, d := range devList {
		if d.IP == "" {
			continue
		}

		// Check if the device IP is already local to physical interfaces
		isLocal, err := IsIPLocal(d.IP, drm.devName)
		if err != nil {
			fmt.Printf("%s Warning: failed to check if %s is local: %v\n", devPrefix, d.IP, err)
		}
		if isLocal {
			// Skip tunneling if the destination is on the local physical network
			continue
		}

		newDeviceIPs[d.IP] = true
	}

	drm.mu.Lock()
	defer drm.mu.Unlock()

	// Add routes for newly discovered devices
	for ip := range newDeviceIPs {
		if !drm.mappedRoutes[ip] {
			if err := addHostRoute(drm.devName, drm.overlayIP, ip); err == nil {
				drm.mappedRoutes[ip] = true
				fmt.Printf("%s Added route: %s/32 -> %s (src %s)\n", devPrefix, ip, drm.devName, drm.overlayIP.String())
			} else {
				fmt.Printf("%s Failed to add route for %s: %v\n", devPrefix, ip, err)
			}
		}
	}

	// Remove routes for devices that disappeared
	for oldIP := range drm.mappedRoutes {
		if !newDeviceIPs[oldIP] {
			if err := delHostRoute(drm.devName, oldIP); err == nil {
				delete(drm.mappedRoutes, oldIP)
				fmt.Printf("%s Removed route: %s/32 from %s\n", devPrefix, oldIP, drm.devName)
			} else {
				fmt.Printf("%s Failed to remove route for %s: %v\n", devPrefix, oldIP, err)
			}
		}
	}

	fmt.Printf("%s Synchronized %d active device routes on %s\n", devPrefix, len(drm.mappedRoutes), drm.devName)
	return nil
}

// ClearAllRoutes removes all active routes managed by this manager.
func (drm *DeviceRouteManager) ClearAllRoutes() {
	drm.mu.Lock()
	defer drm.mu.Unlock()

	for ip := range drm.mappedRoutes {
		_ = delHostRoute(drm.devName, ip)
		delete(drm.mappedRoutes, ip)
	}
}

// IsMapped returns whether an IP is currently mapped through the TUN device.
func (drm *DeviceRouteManager) IsMapped(ip string) bool {
	drm.mu.RLock()
	defer drm.mu.RUnlock()
	return drm.mappedRoutes[ip]
}

// IsIPLocal checks if an IP belongs to any local physical subnet.
func IsIPLocal(targetIP string, excludeIface string) (bool, error) {
	parsedIP := net.ParseIP(targetIP)
	if parsedIP == nil {
		return false, fmt.Errorf("invalid IP %s", targetIP)
	}

	interfaces, err := net.Interfaces()
	if err != nil {
		return false, err
	}

	for _, iface := range interfaces {
		if iface.Name == excludeIface || (iface.Flags&net.FlagUp) == 0 || (iface.Flags&net.FlagLoopback) != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok {
				if ipNet.Contains(parsedIP) {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

func addHostRoute(ifaceName string, srcIP net.IP, targetIP string) error {
	cmd := exec.Command("ip", "route", "replace", targetIP+"/32", "dev", ifaceName, "src", srcIP.String())
	return cmd.Run()
}

func delHostRoute(ifaceName string, targetIP string) error {
	cmd := exec.Command("ip", "route", "del", targetIP+"/32", "dev", ifaceName)
	return cmd.Run()
}
