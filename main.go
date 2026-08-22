package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/leonst036/NetConnect/network"
	netlink "github.com/leonst036/NetConnect/network/NetLink"
	"github.com/leonst036/NetConnect/utils"
)

func main() {
	if os.Geteuid() != 0 {
		fmt.Println("Please run this program as root")
		os.Exit(1)
	}

	overlayCIDR, err := network.DetectCIDR()
	if err != nil {
		log.Fatalf("Error detecting CIDR: %v", err)
	}
	targetSubnets, err := network.DetectSubnets()
	if err != nil {
		log.Fatalf("Error detecting subnets: %v", err)
	}
	if len(targetSubnets) == 0 {
		log.Fatalf("No target subnets found")
	}

	dev := network.CreateVirtualDevice(overlayCIDR, targetSubnets)
	defer dev.Close()

	// Start pinging NetLink relay
	relayURL := utils.GetEnv("NETLINK_RELAY_URL", "localhost:5173")
	netlink.StartPingLoop(relayURL, 5*time.Second)

	packet := make([]byte, 1500)
	for {
		n, err := dev.Read(packet)
		if err != nil {
			log.Fatalf("Error reading from interface: %v", err)
		}

		// Only parse IPv4 packets (version = 4)
		if (packet[0] >> 4) == 4 {
			srcIP := net.IP(packet[12:16])
			dstIP := net.IP(packet[16:20])
			protocol := packet[9]

			protoName := fmt.Sprintf("Proto(%d)", protocol)
			switch protocol {
			case 1:
				protoName = "ICMP (Ping)"
			case 6:
				protoName = "TCP"
			case 17:
				protoName = "UDP"
			}

			fmt.Printf("[IPv4] %s -> %s | %s | %d bytes\n", srcIP, dstIP, protoName, n)

			// Reply to ICMP Echo Request
			if protocol == 1 {
				network.HandleICMPEcho(dev.Interface, packet, n)
			}
		}
	}
}
