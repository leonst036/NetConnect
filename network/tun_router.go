package network

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/leonst036/NetConnect/network/NetLink"
	"github.com/songgao/water"
)

type tcpSessionKey struct {
	srcIP   string
	srcPort uint16
	dstIP   string
	dstPort uint16
}

type tcpSession struct {
	key        tcpSessionKey
	clientSeq  uint32 // next expected sequence from client
	serverSeq  uint32 // current sequence from server
	stream     *netlink.StreamConn
	pendingBuf []byte
	mu         sync.Mutex
	closed     bool
	ifce       *water.Interface
	relayURL   string
	targetID   string
}

// TUNRouter handles packet routing and WSS tunneling for intercepted TUN traffic.
type TUNRouter struct {
	ifce         *water.Interface
	routeManager *DeviceRouteManager
	relayURL     string
	targetID     string
	mu           sync.RWMutex
	sessions     map[tcpSessionKey]*tcpSession
	stopChan     chan struct{}
}

// NewTUNRouter creates a new TUNRouter.
func NewTUNRouter(ifce *water.Interface, routeManager *DeviceRouteManager, relayURL, targetID string) *TUNRouter {
	return &TUNRouter{
		ifce:         ifce,
		routeManager: routeManager,
		relayURL:     relayURL,
		targetID:     targetID,
		sessions:     make(map[tcpSessionKey]*tcpSession),
		stopChan:     make(chan struct{}),
	}
}

// Start begins processing packets from the TUN device.
func (tr *TUNRouter) Start() {
	buf := make([]byte, 65535)
	for {
		select {
		case <-tr.stopChan:
			return
		default:
		}

		n, err := tr.ifce.Read(buf)
		if err != nil {
			select {
			case <-tr.stopChan:
				return
			default:
				fmt.Printf("[TUNRouter] Error reading from TUN: %v\n", err)
				continue
			}
		}

		if n < 20 {
			continue
		}

		packet := make([]byte, n)
		copy(packet, buf[:n])

		// IPv4 check
		if (packet[0] >> 4) != 4 {
			continue
		}

		go tr.handlePacket(packet)
	}
}

// Stop stops the router and closes active sessions.
func (tr *TUNRouter) Stop() {
	close(tr.stopChan)
	tr.mu.Lock()
	defer tr.mu.Unlock()
	for _, sess := range tr.sessions {
		sess.close()
	}
	tr.sessions = make(map[tcpSessionKey]*tcpSession)
}

func (tr *TUNRouter) handlePacket(packet []byte) {
	ihl := int((packet[0] & 0x0F) * 4)
	if len(packet) < ihl {
		return
	}

	protocol := packet[9]
	srcIP := net.IP(packet[12:16]).String()
	dstIP := net.IP(packet[16:20]).String()

	switch protocol {
	case 1: // ICMP
		HandleICMPEcho(tr.ifce, packet, len(packet))
	case 6: // TCP
		tr.handleTCPPacket(packet, ihl, srcIP, dstIP)
	case 17: // UDP
		tr.handleUDPPacket(packet, ihl, srcIP, dstIP)
	}
}

func (tr *TUNRouter) handleUDPPacket(packet []byte, ihl int, srcIP, dstIP string) {
	udpBytes := packet[ihl:]
	if len(udpBytes) < 8 {
		return
	}

	srcPort := binary.BigEndian.Uint16(udpBytes[0:2])
	dstPort := binary.BigEndian.Uint16(udpBytes[2:4])
	udpLen := int(binary.BigEndian.Uint16(udpBytes[4:6]))

	if len(udpBytes) < udpLen || udpLen < 8 {
		return
	}

	// Intercept DNS queries (port 53)
	if dstPort == 53 {
		payload := udpBytes[8:udpLen]
		go tr.forwardDNSQuery(payload, packet, ihl, srcIP, dstIP, srcPort, dstPort)
	}
}

func (tr *TUNRouter) forwardDNSQuery(payload []byte, origPacket []byte, ihl int, srcIP, dstIP string, srcPort, dstPort uint16) {
	u, err := url.Parse(tr.relayURL)
	if err != nil {
		return
	}

	host := u.Hostname()
	if host == "" || host == "localhost" {
		host = "127.0.0.1"
	}

	dnsTarget := fmt.Sprintf("%s:53", host)
	raddr, err := net.ResolveUDPAddr("udp", dnsTarget)
	if err != nil {
		return
	}

	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(2500 * time.Millisecond))
	if _, err := conn.Write(payload); err != nil {
		return
	}

	respBuf := make([]byte, 4096)
	n, err := conn.Read(respBuf)
	if err != nil || n == 0 {
		return
	}

	dnsResponse := respBuf[:n]
	totalIPLen := 20 + 8 + len(dnsResponse)
	reply := make([]byte, totalIPLen)

	// IPv4 Header
	reply[0] = 0x45
	reply[1] = 0x00
	binary.BigEndian.PutUint16(reply[2:4], uint16(totalIPLen))
	binary.BigEndian.PutUint16(reply[4:6], uint16(rand.Intn(65535)))
	reply[6] = 0x40
	reply[7] = 0x00
	reply[8] = 64
	reply[9] = 17 // UDP
	reply[10] = 0
	reply[11] = 0
	copy(reply[12:16], origPacket[16:20]) // Src = original Dst
	copy(reply[16:20], origPacket[12:16]) // Dst = original Src
	binary.BigEndian.PutUint16(reply[10:12], Checksum(reply[:20]))

	// UDP Header
	binary.BigEndian.PutUint16(reply[20:22], dstPort) // 53
	binary.BigEndian.PutUint16(reply[22:24], srcPort)
	binary.BigEndian.PutUint16(reply[24:26], uint16(8+len(dnsResponse)))
	reply[26] = 0
	reply[27] = 0

	// Payload
	copy(reply[28:], dnsResponse)

	_, _ = tr.ifce.Write(reply)
}

func (tr *TUNRouter) handleTCPPacket(packet []byte, ihl int, srcIP, dstIP string) {
	tcpBytes := packet[ihl:]
	if len(tcpBytes) < 20 {
		return
	}

	srcPort := binary.BigEndian.Uint16(tcpBytes[0:2])
	dstPort := binary.BigEndian.Uint16(tcpBytes[2:4])
	seq := binary.BigEndian.Uint32(tcpBytes[4:8])
	ack := binary.BigEndian.Uint32(tcpBytes[8:12])
	dataOffset := int((tcpBytes[12] >> 4) * 4)
	flags := tcpBytes[13]

	if len(tcpBytes) < dataOffset {
		return
	}

	payload := tcpBytes[dataOffset:]
	key := tcpSessionKey{srcIP: srcIP, srcPort: srcPort, dstIP: dstIP, dstPort: dstPort}

	isSYN := (flags & 0x02) != 0
	isACK := (flags & 0x10) != 0
	isFIN := (flags & 0x01) != 0
	isRST := (flags & 0x04) != 0

	if isRST {
		tr.removeSession(key)
		return
	}

	tr.mu.Lock()
	sess, exists := tr.sessions[key]
	if !exists && isSYN {
		// New TCP Connection Request
		initialServerSeq := rand.Uint32()
		sess = &tcpSession{
			key:       key,
			clientSeq: seq + 1,
			serverSeq: initialServerSeq + 1,
			ifce:      tr.ifce,
			relayURL:  tr.relayURL,
			targetID:  tr.targetID,
		}
		tr.sessions[key] = sess
		tr.mu.Unlock()

		// Reply with SYN-ACK and MSS option (1400 bytes)
		mssOption := []byte{0x02, 0x04, 0x05, 0x78}
		sess.sendTCP(0x12, mssOption, nil, initialServerSeq, seq+1)

		// Connect to destination over NetLink WSS stream in background
		go sess.startStream(tr)
		return
	}
	tr.mu.Unlock()

	if sess == nil {
		return
	}

	if isFIN {
		sess.clientSeq = seq + 1
		sess.sendTCP(0x11, nil, nil, sess.serverSeq, sess.clientSeq) // FIN-ACK
		tr.removeSession(key)
		return
	}

	if len(payload) > 0 {
		sess.clientSeq = seq + uint32(len(payload))
		sess.handleClientData(payload)
		// ACK the received data
		sess.sendTCP(0x10, nil, nil, sess.serverSeq, sess.clientSeq)
	} else if isACK && ack > 0 {
		// Pure ACK
	}
}

func (tr *TUNRouter) removeSession(key tcpSessionKey) {
	tr.mu.Lock()
	sess, ok := tr.sessions[key]
	if ok {
		delete(tr.sessions, key)
	}
	tr.mu.Unlock()

	if sess != nil {
		sess.close()
	}
}

func (s *tcpSession) handleClientData(payload []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}

	if s.stream != nil {
		_, _ = s.stream.Write(payload)
	} else {
		// Buffer data until stream connection is established
		s.pendingBuf = append(s.pendingBuf, payload...)
	}
}

func (s *tcpSession) startStream(tr *TUNRouter) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := netlink.DialLANStream(ctx, s.relayURL, s.targetID, s.key.dstIP, int(s.key.dstPort))
	if err != nil {
		fmt.Printf("[TUNRouter] Failed to dial LAN stream for %s:%d: %v\n", s.key.dstIP, s.key.dstPort, err)
		s.mu.Lock()
		if !s.closed {
			s.sendTCP(0x04, nil, nil, s.serverSeq, s.clientSeq) // RST
			s.closed = true
		}
		s.mu.Unlock()
		tr.removeSession(s.key)
		return
	}

	s.mu.Lock()
	if s.closed {
		_ = stream.Close()
		s.mu.Unlock()
		return
	}
	s.stream = stream

	// Flush any pending data received before stream was ready
	if len(s.pendingBuf) > 0 {
		_, _ = stream.Write(s.pendingBuf)
		s.pendingBuf = nil
	}
	s.mu.Unlock()

	// Forward stream incoming bytes back into TUN as TCP packets
	// Chunk reads to fit within standard MTU (MSS 1400)
	buf := make([]byte, 1400)
	for {
		n, err := stream.Read(buf)
		if n > 0 {
			s.mu.Lock()
			if !s.closed {
				s.sendTCP(0x18, nil, buf[:n], s.serverSeq, s.clientSeq) // PSH-ACK
				s.serverSeq += uint32(n)
			}
			s.mu.Unlock()
		}
		if err != nil {
			if err != io.EOF {
				fmt.Printf("[TUNRouter] Stream read error: %v\n", err)
			}
			s.mu.Lock()
			if !s.closed {
				s.sendTCP(0x11, nil, nil, s.serverSeq, s.clientSeq) // FIN-ACK
				s.closed = true
			}
			s.mu.Unlock()
			tr.removeSession(s.key)
			return
		}
	}
}

func (s *tcpSession) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		s.closed = true
		if s.stream != nil {
			_ = s.stream.Close()
		}
		s.pendingBuf = nil
	}
}

func (s *tcpSession) sendTCP(flags byte, options, payload []byte, seq, ack uint32) {
	if s.closed && flags != 0x04 {
		return
	}

	srcIP := net.ParseIP(s.key.dstIP).To4()
	dstIP := net.ParseIP(s.key.srcIP).To4()
	if srcIP == nil || dstIP == nil {
		return
	}

	// Calculate TCP header length with options (padded to 4-byte boundary)
	optLen := len(options)
	if optLen%4 != 0 {
		pad := 4 - (optLen % 4)
		options = append(options, make([]byte, pad)...)
		optLen = len(options)
	}

	tcpHeaderLen := 20 + optLen
	ipTotalLen := 20 + tcpHeaderLen + len(payload)
	packet := make([]byte, ipTotalLen)

	// IP Header (20 bytes)
	packet[0] = 0x45 // Version 4, IHL 5
	packet[1] = 0x00 // TOS
	binary.BigEndian.PutUint16(packet[2:4], uint16(ipTotalLen))
	binary.BigEndian.PutUint16(packet[4:6], uint16(rand.Intn(65535)))
	binary.BigEndian.PutUint16(packet[6:8], 0x4000) // DF set
	packet[8] = 64                                 // TTL
	packet[9] = 6                                  // Protocol: TCP
	copy(packet[12:16], srcIP)
	copy(packet[16:20], dstIP)

	// IP Checksum
	ipChecksum := calcChecksum(packet[:20])
	binary.BigEndian.PutUint16(packet[10:12], ipChecksum)

	// TCP Header
	tcpOffset := 20
	binary.BigEndian.PutUint16(packet[tcpOffset:tcpOffset+2], s.key.dstPort)
	binary.BigEndian.PutUint16(packet[tcpOffset+2:tcpOffset+4], s.key.srcPort)
	binary.BigEndian.PutUint32(packet[tcpOffset+4:tcpOffset+8], seq)
	binary.BigEndian.PutUint32(packet[tcpOffset+8:tcpOffset+12], ack)
	dataOffsetWords := byte(tcpHeaderLen / 4)
	packet[tcpOffset+12] = dataOffsetWords << 4
	packet[tcpOffset+13] = flags
	binary.BigEndian.PutUint16(packet[tcpOffset+14:tcpOffset+16], 65535) // Window size

	// Copy TCP options if any
	if optLen > 0 {
		copy(packet[tcpOffset+20:tcpOffset+20+optLen], options)
	}

	// Copy payload if any
	if len(payload) > 0 {
		copy(packet[tcpOffset+tcpHeaderLen:], payload)
	}

	// TCP Checksum (Pseudo-header + TCP segment)
	tcpChecksum := calcTCPChecksum(srcIP, dstIP, packet[tcpOffset:])
	binary.BigEndian.PutUint16(packet[tcpOffset+16:tcpOffset+18], tcpChecksum)

	_, _ = s.ifce.Write(packet)
}

func calcChecksum(b []byte) uint16 {
	var sum uint32
	for i := 0; i < len(b)-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(b[i : i+2]))
	}
	if len(b)%2 == 1 {
		sum += uint32(b[len(b)-1]) << 8
	}
	for sum > 0xffff {
		sum = (sum & 0xffff) + (sum >> 16)
	}
	return ^uint16(sum)
}

func calcTCPChecksum(srcIP, dstIP net.IP, tcpSegment []byte) uint16 {
	var sum uint32

	// Pseudo Header
	for i := 0; i < 4; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(srcIP[i : i+2]))
		sum += uint32(binary.BigEndian.Uint16(dstIP[i : i+2]))
	}
	sum += 6 // Protocol TCP
	sum += uint32(len(tcpSegment))

	// TCP segment
	for i := 0; i < len(tcpSegment)-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(tcpSegment[i : i+2]))
	}
	if len(tcpSegment)%2 == 1 {
		sum += uint32(tcpSegment[len(tcpSegment)-1]) << 8
	}

	for sum > 0xffff {
		sum = (sum & 0xffff) + (sum >> 16)
	}
	return ^uint16(sum)
}
