package network

import (
	"fmt"
	"log"
	"net"
	"os/exec"

	"github.com/songgao/water"
)

// CreateVirtualDevice initializes the TUN device, assigns an overlay IP, and configures routes.
func CreateVirtualDevice(overlayCIDR string, targetSubnets []string) *water.Interface {
	config := water.Config{
		DeviceType: water.TUN,
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
		isLocal, err := isSubnetLocal(subnet, ifce.Name())
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

	return ifce
}

// isSubnetLocal checks if any active physical interface is already part of targetCIDR.
func isSubnetLocal(targetCIDR string, excludeIface string) (bool, error) {
	_, targetNet, err := net.ParseCIDR(targetCIDR)
	if err != nil {
		return false, fmt.Errorf("invalid CIDR %s: %w", targetCIDR, err)
	}

	interfaces, err := net.Interfaces()
	if err != nil {
		return false, err
	}

	for _, iface := range interfaces {
		if iface.Name == excludeIface || (iface.Flags&net.FlagUp) == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() {
				continue
			}

			// Check if local interface IP is inside target subnet
			if targetNet.Contains(ip) {
				return true, nil
			}
		}
	}

	return false, nil
}

func runCmd(name string, args ...string) {
	cmd := exec.Command(name, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Fatalf("Failed to run %s %v: %s (%v)", name, args, string(out), err)
	}
}

