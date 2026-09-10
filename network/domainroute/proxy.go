package domainroute

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type bufferedConn struct {
	net.Conn
	r *bufio.Reader
}

func (b *bufferedConn) Read(p []byte) (int, error) {
	return b.r.Read(p)
}

type ProxyListener struct {
	bindAddr string
	listener net.Listener
	rules    *RuleEngine
	tunnel   *TunnelClient
	stats    *Stats
	enabled  atomic.Bool
	closed   atomic.Bool
	mu       sync.Mutex
}

func NewProxyListener(bindAddr string, rules *RuleEngine, tunnel *TunnelClient, stats *Stats) *ProxyListener {
	if stats == nil {
		stats = &Stats{}
	}
	p := &ProxyListener{
		bindAddr: bindAddr,
		rules:    rules,
		tunnel:   tunnel,
		stats:    stats,
	}
	p.enabled.Store(true)
	return p
}

func (p *ProxyListener) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.listener != nil {
		return nil
	}

	ln, err := net.Listen("tcp", p.bindAddr)
	if err != nil {
		return err
	}

	p.listener = ln
	p.closed.Store(false)

	go p.acceptLoop(ln)
	return nil
}

func (p *ProxyListener) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed.CompareAndSwap(false, true) {
		if p.listener != nil {
			_ = p.listener.Close()
			p.listener = nil
		}
	}
}

func (p *ProxyListener) IsEnabled() bool {
	return p.enabled.Load()
}

func (p *ProxyListener) SetEnabled(enabled bool) {
	p.enabled.Store(enabled)
}

func (p *ProxyListener) Addr() net.Addr {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.listener != nil {
		return p.listener.Addr()
	}
	return nil
}

func (p *ProxyListener) acceptLoop(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if p.closed.Load() {
				return
			}
			continue
		}
		go p.handleConn(conn)
	}
}

func (p *ProxyListener) handleConn(conn net.Conn) {
	p.stats.IncTotal()

	br := bufio.NewReader(conn)
	firstByte, err := br.Peek(1)
	if err != nil {
		_ = conn.Close()
		return
	}

	var targetHost string
	var targetPort uint16

	if firstByte[0] == 0x05 {
		targetHost, targetPort, err = p.handshakeSOCKS5(conn, br)
	} else if firstByte[0] == 'C' {
		targetHost, targetPort, err = p.handshakeHTTP(conn, br)
	} else {
		_ = conn.Close()
		return
	}

	if err != nil {
		_ = conn.Close()
		return
	}

	clientConn := &bufferedConn{Conn: conn, r: br}
	p.dispatch(clientConn, targetHost, targetPort)
}

func (p *ProxyListener) handshakeSOCKS5(conn net.Conn, br *bufio.Reader) (string, uint16, error) {
	ver, err := br.ReadByte()
	if err != nil || ver != 0x05 {
		return "", 0, errors.New("invalid socks5 version")
	}

	nmethods, err := br.ReadByte()
	if err != nil {
		return "", 0, err
	}

	methods := make([]byte, int(nmethods))
	if _, err := io.ReadFull(br, methods); err != nil {
		return "", 0, err
	}

	hasNoAuth := false
	for _, m := range methods {
		if m == 0x00 {
			hasNoAuth = true
			break
		}
	}

	if !hasNoAuth {
		_, _ = conn.Write([]byte{0x05, 0xFF})
		return "", 0, errors.New("no acceptable auth methods")
	}

	if _, err := conn.Write([]byte{0x05, 0x00}); err != nil {
		return "", 0, err
	}

	reqHdr := make([]byte, 4)
	if _, err := io.ReadFull(br, reqHdr); err != nil {
		return "", 0, err
	}

	reqVer := reqHdr[0]
	cmd := reqHdr[1]
	atyp := reqHdr[3]

	if reqVer != 0x05 {
		return "", 0, errors.New("invalid socks5 request version")
	}

	if cmd != 0x01 {
		_, _ = conn.Write([]byte{0x05, 0x07, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return "", 0, fmt.Errorf("unsupported socks5 command: %d", cmd)
	}

	var targetHost string
	switch atyp {
	case 0x01:
		ip := make([]byte, 4)
		if _, err := io.ReadFull(br, ip); err != nil {
			return "", 0, err
		}
		targetHost = net.IP(ip).String()
	case 0x03:
		dLen, err := br.ReadByte()
		if err != nil {
			return "", 0, err
		}
		domain := make([]byte, int(dLen))
		if _, err := io.ReadFull(br, domain); err != nil {
			return "", 0, err
		}
		targetHost = string(domain)
	case 0x04:
		ip := make([]byte, 16)
		if _, err := io.ReadFull(br, ip); err != nil {
			return "", 0, err
		}
		targetHost = net.IP(ip).String()
	default:
		_, _ = conn.Write([]byte{0x05, 0x08, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return "", 0, fmt.Errorf("unsupported socks5 address type: %d", atyp)
	}

	var portBuf [2]byte
	if _, err := io.ReadFull(br, portBuf[:]); err != nil {
		return "", 0, err
	}
	targetPort := binary.BigEndian.Uint16(portBuf[:])

	reply := []byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0}
	if _, err := conn.Write(reply); err != nil {
		return "", 0, err
	}

	return targetHost, targetPort, nil
}

func (p *ProxyListener) handshakeHTTP(conn net.Conn, br *bufio.Reader) (string, uint16, error) {
	reqLine, err := br.ReadString('\n')
	if err != nil {
		return "", 0, err
	}

	reqLine = strings.TrimRight(reqLine, "\r\n")
	parts := strings.Fields(reqLine)
	if len(parts) < 2 || !strings.EqualFold(parts[0], "CONNECT") {
		return "", 0, fmt.Errorf("invalid http connect request line: %s", reqLine)
	}

	target := parts[1]
	var targetHost string
	var targetPort uint16 = 443

	if strings.Contains(target, ":") {
		host, portStr, err := net.SplitHostPort(target)
		if err == nil {
			targetHost = host
			if pNum, err := strconv.Atoi(portStr); err == nil && pNum > 0 && pNum <= 65535 {
				targetPort = uint16(pNum)
			}
		} else {
			targetHost = target
		}
	} else {
		targetHost = target
	}

	for {
		headerLine, err := br.ReadString('\n')
		if err != nil {
			return "", 0, err
		}
		if strings.TrimRight(headerLine, "\r\n") == "" {
			break
		}
	}

	reply := []byte("HTTP/1.1 200 Connection Established\r\n\r\n")
	if _, err := conn.Write(reply); err != nil {
		return "", 0, err
	}

	return targetHost, targetPort, nil
}

func (p *ProxyListener) dispatch(clientConn net.Conn, targetHost string, targetPort uint16) {
	if !p.enabled.Load() || !p.rules.Matches(targetHost) {
		p.stats.IncBypassed()
		p.handleBypass(clientConn, targetHost, targetPort)
		return
	}

	p.stats.IncTunneled()
	p.handleTunnel(clientConn, targetHost, targetPort)
}

func (p *ProxyListener) handleTunnel(clientConn net.Conn, targetHost string, targetPort uint16) {
	if p.tunnel == nil || !p.tunnel.IsConnected() {
		p.handleBypass(clientConn, targetHost, targetPort)
		return
	}

	channelID := p.tunnel.AllocateChannelID()
	ch := NewChannel(channelID, clientConn)
	p.tunnel.RegisterChannel(channelID, ch)
	defer func() {
		_ = p.tunnel.SendClose(channelID)
		ch.Close()
		p.tunnel.RemoveChannel(channelID)
	}()

	if err := p.tunnel.SendConnect(channelID, targetPort, targetHost); err != nil {
		return
	}

	go ch.RunEgress()

	bufPtr := GetBuffer()
	buf := *bufPtr
	defer PutBuffer(bufPtr)

	for {
		n, err := clientConn.Read(buf)
		if n > 0 {
			if err := p.tunnel.SendData(channelID, buf[:n]); err != nil {
				break
			}
		}
		if err != nil {
			break
		}
	}
}

func (p *ProxyListener) handleBypass(clientConn net.Conn, targetHost string, targetPort uint16) {
	dest := net.JoinHostPort(targetHost, strconv.Itoa(int(targetPort)))
	directConn, err := net.DialTimeout("tcp", dest, 10*time.Second)
	if err != nil {
		_ = clientConn.Close()
		return
	}
	defer directConn.Close()
	defer clientConn.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		bufPtr := GetBuffer()
		defer PutBuffer(bufPtr)
		n, _ := io.CopyBuffer(directConn, clientConn, *bufPtr)
		p.stats.AddBytesSent(uint64(n))
		if tc, ok := directConn.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		}
	}()

	go func() {
		defer wg.Done()
		bufPtr := GetBuffer()
		defer PutBuffer(bufPtr)
		n, _ := io.CopyBuffer(clientConn, directConn, *bufPtr)
		p.stats.AddBytesReceived(uint64(n))
		if tc, ok := clientConn.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		}
	}()

	wg.Wait()
}
