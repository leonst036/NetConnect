package network

import (
	"fmt"
	"net"
)

// DetectCIDR returns the primary local network CIDR
func DetectCIDR() (string, error) {
	// Try routing lookup via UDP to find the outbound interface IP
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		defer conn.Close()
		localIP := conn.LocalAddr().(*net.UDPAddr).IP

		if cidr, ok := findCIDRForIP(localIP); ok {
			return cidr, nil
		}
	}

	// Fallback to first available local subnet
	subnets, err := DetectSubnets()
	if err != nil {
		return "", err
	}
	if len(subnets) == 0 {
		return "", fmt.Errorf("no active IPv4 network interface found")
	}

	return subnets[0], nil
}

// DetectSubnets returns all active local IPv4 subnets in CIDR format
func DetectSubnets() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to list network interfaces: %w", err)
	}

	var subnets []string
	seen := make(map[string]bool)

	for _, iface := range interfaces {
		if (iface.Flags&net.FlagUp) == 0 || (iface.Flags&net.FlagLoopback) != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ip4 := ipNet.IP.To4()
			if ip4 == nil || ip4.IsLoopback() {
				continue
			}

			networkIP := ip4.Mask(ipNet.Mask)
			ones, _ := ipNet.Mask.Size()
			cidr := fmt.Sprintf("%s/%d", networkIP.String(), ones)

			if !seen[cidr] {
				seen[cidr] = true
				subnets = append(subnets, cidr)
			}
		}
	}

	if len(subnets) == 0 {
		return nil, fmt.Errorf("no active IPv4 subnets found")
	}

	return subnets, nil
}

func findCIDRForIP(targetIP net.IP) (string, bool) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", false
	}

	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			if ipNet.IP.Equal(targetIP) {
				ip4 := ipNet.IP.To4()
				if ip4 == nil {
					continue
				}
				networkIP := ip4.Mask(ipNet.Mask)
				ones, _ := ipNet.Mask.Size()
				return fmt.Sprintf("%s/%d", networkIP.String(), ones), true
			}
		}
	}

	return "", false
}

// IsSubnetLocal checks if any active physical interface is already part of targetCIDR.
func IsSubnetLocal(targetCIDR string, excludeIface string) (bool, error) {
	_, targetNet, err := net.ParseCIDR(targetCIDR)
	if err != nil {
		return false, fmt.Errorf("invalid CIDR %s: %w", targetCIDR, err)
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

			if targetNet.Contains(ip) {
				return true, nil
			}
		}
	}

	return false, nil
}
