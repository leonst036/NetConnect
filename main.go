package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/leonst036/NetConnect/network"
)

func main() {
	if os.Geteuid() != 0 {
		fmt.Println("Please run this program as root")
		os.Exit(1)
	}

	overlayCIDR := "10.200.0.2/24"
	targetSubnets := []string{"192.168.99.0/24"}

	ifce := network.CreateVirtualDevice(overlayCIDR, targetSubnets)
	defer ifce.Close()

	packet := make([]byte, 1500)
	for {
		n, err := ifce.Read(packet)
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
		}
	}
}
