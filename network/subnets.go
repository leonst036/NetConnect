package network

import (
	"fmt"
	"net"
	"os"
)

// DetectCIDR returns a virtual overlay CIDR for the TUN interface that does not conflict with local networks.
func DetectCIDR() (string, error) {
	if envCIDR := os.Getenv("NETLINK_OVERLAY_CIDR"); envCIDR != "" {
		return envCIDR, nil
	}

	candidates := []string{
		"10.200.0.2/24",
		"100.96.0.2/24",
		"172.28.0.2/24",
		"10.88.0.2/24",
	}

	localSubnets, _ := DetectSubnets()
	for _, candidate := range candidates {
		_, candNet, err := net.ParseCIDR(candidate)
		if err != nil {
			continue
		}
		conflict := false
		for _, local := range localSubnets {
			_, locNet, err := net.ParseCIDR(local)
			if err != nil {
				continue
			}
			if candNet.Contains(locNet.IP) || locNet.Contains(candNet.IP) {
				conflict = true
				break
			}
		}
		if !conflict {
			return candidate, nil
		}
	}

	return "10.200.0.2/24", nil
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
