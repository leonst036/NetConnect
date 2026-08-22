package network

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os/exec"
	"syscall"
	"time"

	"github.com/songgao/water"
)

// Device wraps a virtual TUN network device with routing, HTTP server, and HTTP client capabilities.
type Device struct {
	Interface     *water.Interface
	OverlayCIDR   string
	OverlayIP     net.IP
	TargetSubnets []string
	httpMux       *http.ServeMux
	httpServer    *http.Server
}

// CreateVirtualDevice initializes the TUN device, assigns an overlay IP, and configures routes.
func CreateVirtualDevice(overlayCIDR string, targetSubnets []string) *Device {
	config := water.Config{
		DeviceType: water.TUN,
		PlatformSpecificParams: water.PlatformSpecificParams{
			Name: "netconnect0",
		},
	}

	ifce, err := water.New(config)
	if err != nil {
		log.Fatalf("Error creating TUN device: %v\n", err)
	}

	fmt.Printf("TUN device created: %s\n", ifce.Name())

	// Bring the interface up and assign virtual overlay IP
	runCmd("ip", "link", "set", "dev", ifce.Name(), "up")
	runCmd("ip", "addr", "add", overlayCIDR, "dev", ifce.Name())

	// Extract IP without subnet mask for preferred source address selection
	srcIP, _, err := net.ParseCIDR(overlayCIDR)
	if err != nil {
		log.Fatalf("Invalid overlay CIDR: %v\n", err)
	}

	// Add routes for target remote subnets if not locally present
	for _, subnet := range targetSubnets {
		isLocal, err := IsSubnetLocal(subnet, ifce.Name())
		if err != nil {
			log.Printf("Warning: failed to check local subnet for %s: %v\n", subnet, err)
			continue
		}

		if isLocal {
			fmt.Printf("Skipping route for %s (already connected to local network)\n", subnet)
			continue
		}

		runCmd("ip", "route", "add", subnet, "dev", ifce.Name(), "src", srcIP.String())
		fmt.Printf("Route added: %s -> %s (src %s)\n", subnet, ifce.Name(), srcIP.String())
	}

	return &Device{
		Interface:     ifce,
		OverlayCIDR:   overlayCIDR,
		OverlayIP:     srcIP,
		TargetSubnets: targetSubnets,
		httpMux:       http.NewServeMux(),
	}
}

// Name returns the network interface name.
func (d *Device) Name() string {
	return d.Interface.Name()
}

// Read reads a packet from the interface.
func (d *Device) Read(b []byte) (int, error) {
	return d.Interface.Read(b)
}

// Write writes a packet to the interface.
func (d *Device) Write(b []byte) (int, error) {
	return d.Interface.Write(b)
}

// Close closes the virtual interface and any active HTTP server.
func (d *Device) Close() error {
	if d.httpServer != nil {
		_ = d.httpServer.Close()
	}
	return d.Interface.Close()
}

// HandleFunc registers an HTTP route on the device.
func (d *Device) HandleFunc(pattern string, handler http.HandlerFunc) {
	d.httpMux.HandleFunc(pattern, handler)
}

// StartHTTPServer starts an HTTP server bound to the overlay IP.
func (d *Device) StartHTTPServer(port int) {
	d.httpServer = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", d.OverlayIP.String(), port),
		Handler: d.httpMux,
	}

	go func() {
		log.Printf("Device HTTP server listening on %s:%d", d.OverlayIP.String(), port)
		if err := d.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Device HTTP server error: %v", err)
		}
	}()
}

// StartPingServer registers the /api/netconnect/ping route and starts the server.
func (d *Device) StartPingServer(port int) {
	d.HandleFunc("/api/netconnect/ping", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong\n"))
	})
	d.StartHTTPServer(port)
}

// HTTPClient returns an http.Client bound to this virtual device.
func (d *Device) HTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout: timeout,
		LocalAddr: &net.TCPAddr{
			IP: d.OverlayIP,
		},
		Control: func(network, address string, c syscall.RawConn) error {
			var bindErr error
			err := c.Control(func(fd uintptr) {
				bindErr = syscall.BindToDevice(int(fd), d.Interface.Name())
			})
			if err != nil {
				return err
			}
			return bindErr
		},
	}

	return &http.Client{
		Transport: &http.Transport{
			DialContext: dialer.DialContext,
		},
		Timeout: timeout,
	}
}

// Ping sends an HTTP GET ping request through this virtual device.
func (d *Device) Ping(targetURL string, timeout time.Duration) (string, error) {
	client := d.HTTPClient(timeout)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ping failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func runCmd(name string, args ...string) {
	cmd := exec.Command(name, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Fatalf("Failed to run %s %v: %s (%v)", name, args, string(out), err)
	}
}
