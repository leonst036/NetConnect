package network

import (
	"encoding/binary"
	"log"

	"github.com/songgao/water"
)

// HandleICMPEcho handles incoming ICMP Echo Requests and sends an Echo Reply back to the TUN interface.
func HandleICMPEcho(ifce *water.Interface, packet []byte, n int) {
	if ifce == nil || n < 20 || len(packet) < 20 {
		return
	}
	ihl := int((packet[0] & 0x0F) * 4)
	if ihl < 20 || n < ihl+8 || len(packet) < ihl+8 || len(packet) < n {
		return
	}

	// Check for ICMP Echo Request (type 8, code 0)
	if packet[ihl] != 8 || packet[ihl+1] != 0 {
		return
	}

	// Swap source and destination IP addresses
	srcIP := make([]byte, 4)
	copy(srcIP, packet[12:16])
	copy(packet[12:16], packet[16:20])
	copy(packet[16:20], srcIP)

	// Recalculate IPv4 header checksum
	packet[10] = 0
	packet[11] = 0
	binary.BigEndian.PutUint16(packet[10:12], Checksum(packet[:ihl]))

	// Change ICMP Type to Echo Reply (type 0)
	packet[ihl] = 0

	// Recalculate ICMP checksum
	packet[ihl+2] = 0
	packet[ihl+3] = 0
	binary.BigEndian.PutUint16(packet[ihl+2:ihl+4], Checksum(packet[ihl:n]))

	// Send reply back into TUN device
	if _, err := ifce.Write(packet[:n]); err != nil {
		log.Printf("Error sending ICMP reply: %v", err)
	}
}

// Checksum calculates the standard 16-bit one's complement checksum.
func Checksum(data []byte) uint16 {
	var sum uint32
	for i := 0; i < len(data)-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(data[i : i+2]))
	}
	if len(data)%2 != 0 {
		sum += uint32(data[len(data)-1]) << 8
	}
	for sum > 0xffff {
		sum = (sum & 0xffff) + (sum >> 16)
	}
	return ^uint16(sum)
}
