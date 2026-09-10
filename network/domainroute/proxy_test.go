package domainroute

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"strconv"
	"testing"
	"time"
)

func TestSOCKS5HandshakeDomain(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	proxy := &ProxyListener{
		rules: NewRuleEngine(),
		stats: &Stats{},
	}

	errChan := make(chan error, 1)
	var host string
	var port uint16

	go func() {
		br := bufio.NewReader(serverConn)
		var err error
		host, port, err = proxy.handshakeSOCKS5(serverConn, br)
		errChan <- err
	}()

	// Client sends greeting: VER=5, NMETHODS=1, METHOD=0
	_, err := clientConn.Write([]byte{0x05, 0x01, 0x00})
	if err != nil {
		t.Fatalf("client write failed: %v", err)
	}

	// Server reply: VER=5, METHOD=0
	reply := make([]byte, 2)
	if _, err := io.ReadFull(clientConn, reply); err != nil {
		t.Fatalf("client read greeting reply failed: %v", err)
	}
	if reply[0] != 0x05 || reply[1] != 0x00 {
		t.Fatalf("unexpected greeting reply: %v", reply)
	}

	// Client sends CONNECT to domain: netflix.com:443
	domain := "netflix.com"
	req := []byte{0x05, 0x01, 0x00, 0x03, byte(len(domain))}
	req = append(req, []byte(domain)...)
	portBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(portBuf, 443)
	req = append(req, portBuf...)

	if _, err := clientConn.Write(req); err != nil {
		t.Fatalf("client write request failed: %v", err)
	}

	// Server reply: 0x05 0x00 0x00 0x01 0 0 0 0 0 0
	resp := make([]byte, 10)
	if _, err := io.ReadFull(clientConn, resp); err != nil {
		t.Fatalf("client read response failed: %v", err)
	}
	if !bytes.Equal(resp, []byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0}) {
		t.Fatalf("unexpected connect reply: %v", resp)
	}

	select {
	case err := <-errChan:
		if err != nil {
			t.Fatalf("handshake error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handshake timeout")
	}

	if host != "netflix.com" {
		t.Errorf("expected host netflix.com, got %s", host)
	}
	if port != 443 {
		t.Errorf("expected port 443, got %d", port)
	}
}

func TestSOCKS5HandshakeIPv4(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	proxy := &ProxyListener{
		rules: NewRuleEngine(),
		stats: &Stats{},
	}

	errChan := make(chan error, 1)
	var host string
	var port uint16

	go func() {
		br := bufio.NewReader(serverConn)
		var err error
		host, port, err = proxy.handshakeSOCKS5(serverConn, br)
		errChan <- err
	}()

	// Client sends greeting
	_, _ = clientConn.Write([]byte{0x05, 0x01, 0x00})
	reply := make([]byte, 2)
	_, _ = io.ReadFull(clientConn, reply)

	// Client sends CONNECT to IPv4: 1.1.1.1:53
	req := []byte{0x05, 0x01, 0x00, 0x01, 1, 1, 1, 1, 0x00, 0x35}
	_, _ = clientConn.Write(req)

	resp := make([]byte, 10)
	_, _ = io.ReadFull(clientConn, resp)

	select {
	case err := <-errChan:
		if err != nil {
			t.Fatalf("handshake error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handshake timeout")
	}

	if host != "1.1.1.1" {
		t.Errorf("expected host 1.1.1.1, got %s", host)
	}
	if port != 53 {
		t.Errorf("expected port 53, got %d", port)
	}
}

func TestHTTPConnectHandshake(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	proxy := &ProxyListener{
		rules: NewRuleEngine(),
		stats: &Stats{},
	}

	errChan := make(chan error, 1)
	var host string
	var port uint16

	go func() {
		br := bufio.NewReader(serverConn)
		var err error
		host, port, err = proxy.handshakeHTTP(serverConn, br)
		errChan <- err
	}()

	req := "CONNECT example.com:8443 HTTP/1.1\r\nHost: example.com:8443\r\nUser-Agent: test\r\n\r\n"
	if _, err := clientConn.Write([]byte(req)); err != nil {
		t.Fatalf("client write failed: %v", err)
	}

	resp := make([]byte, 39)
	if _, err := io.ReadFull(clientConn, resp); err != nil {
		t.Fatalf("client read failed: %v", err)
	}

	expectedReply := "HTTP/1.1 200 Connection Established\r\n\r\n"
	if string(resp) != expectedReply {
		t.Fatalf("expected reply %q, got %q", expectedReply, string(resp))
	}

	select {
	case err := <-errChan:
		if err != nil {
			t.Fatalf("handshake error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handshake timeout")
	}

	if host != "example.com" {
		t.Errorf("expected host example.com, got %s", host)
	}
	if port != 8443 {
		t.Errorf("expected port 8443, got %d", port)
	}
}

func TestProxyListenerBypass(t *testing.T) {
	// Start a local echo server
	echoLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start echo listener: %v", err)
	}
	defer echoLn.Close()

	go func() {
		for {
			conn, err := echoLn.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				_, _ = io.Copy(c, c)
			}(conn)
		}
	}()

	echoAddr := echoLn.Addr().(*net.TCPAddr)

	// Start ProxyListener on random port
	proxy := NewProxyListener("127.0.0.1:0", NewRuleEngine(), nil, nil)
	if err := proxy.Start(); err != nil {
		t.Fatalf("failed to start proxy: %v", err)
	}
	defer proxy.Stop()

	proxyAddr := proxy.Addr().String()

	// Connect to proxy using SOCKS5
	conn, err := net.Dial("tcp", proxyAddr)
	if err != nil {
		t.Fatalf("dial proxy failed: %v", err)
	}
	defer conn.Close()

	// SOCKS5 greeting
	_, _ = conn.Write([]byte{0x05, 0x01, 0x00})
	greetingResp := make([]byte, 2)
	_, _ = io.ReadFull(conn, greetingResp)

	// CONNECT to local echo server (not in default ruleset -> bypass)
	req := []byte{0x05, 0x01, 0x00, 0x01, 127, 0, 0, 1}
	var pBuf [2]byte
	binary.BigEndian.PutUint16(pBuf[:], uint16(echoAddr.Port))
	req = append(req, pBuf[:]...)
	_, _ = conn.Write(req)

	connectResp := make([]byte, 10)
	_, _ = io.ReadFull(conn, connectResp)
	if connectResp[1] != 0x00 {
		t.Fatalf("connect failed with status %d", connectResp[1])
	}

	// Send data through echo tunnel
	message := []byte("ping pong test")
	if _, err := conn.Write(message); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	buf := make([]byte, len(message))
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatalf("read failed: %v", err)
	}

	if !bytes.Equal(buf, message) {
		t.Fatalf("expected %q, got %q", message, buf)
	}

	stats := proxy.stats.Snapshot()
	if stats.TotalConnections != 1 {
		t.Errorf("expected 1 total connection, got %d", stats.TotalConnections)
	}
	if stats.BypassedConnections != 1 {
		t.Errorf("expected 1 bypassed connection, got %d", stats.BypassedConnections)
	}
}

func TestProxyListenerHTTPConnectBypass(t *testing.T) {
	// Start a local echo server
	echoLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start echo listener: %v", err)
	}
	defer echoLn.Close()

	go func() {
		for {
			conn, err := echoLn.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				_, _ = io.Copy(c, c)
			}(conn)
		}
	}()

	echoAddr := echoLn.Addr().(*net.TCPAddr)

	proxy := NewProxyListener("127.0.0.1:0", NewRuleEngine(), nil, nil)
	if err := proxy.Start(); err != nil {
		t.Fatalf("failed to start proxy: %v", err)
	}
	defer proxy.Stop()

	conn, err := net.Dial("tcp", proxy.Addr().String())
	if err != nil {
		t.Fatalf("dial proxy failed: %v", err)
	}
	defer conn.Close()

	req := "CONNECT 127.0.0.1:" + strconv.Itoa(echoAddr.Port) + " HTTP/1.1\r\n\r\n"
	_, _ = conn.Write([]byte(req))

	resp := make([]byte, 39)
	_, _ = io.ReadFull(conn, resp)
	if !bytes.Contains(resp, []byte("200 Connection Established")) {
		t.Fatalf("unexpected connect response: %s", string(resp))
	}

	// Echo test
	testData := []byte("testing http connect bypass")
	_, _ = conn.Write(testData)
	recv := make([]byte, len(testData))
	_, _ = io.ReadFull(conn, recv)

	if !bytes.Equal(recv, testData) {
		t.Fatalf("expected %q, got %q", testData, recv)
	}
}
