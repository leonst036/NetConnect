package domainroute

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func TestManagerLifecycleAndStatus(t *testing.T) {
	mgr := NewManager("http://localhost:4535", "test-dev", "127.0.0.1:0", nil)

	status := mgr.GetStatus()
	if status.Running {
		t.Fatal("expected manager not to be running before Start()")
	}
	if !status.Enabled {
		t.Fatal("expected manager to be enabled by default")
	}

	rules := mgr.GetRules()
	if len(rules) != len(DefaultPatterns) {
		t.Fatalf("expected %d default rules, got %d", len(DefaultPatterns), len(rules))
	}

	toggled := mgr.ToggleEnabled()
	if toggled || mgr.IsEnabled() {
		t.Fatal("expected manager to be disabled after toggle")
	}

	mgr.SetEnabled(true)
	if !mgr.IsEnabled() {
		t.Fatal("expected manager to be enabled after SetEnabled(true)")
	}

	mgr.AddRule("*.custom.org")
	if !mgr.rules.Matches("sub.custom.org") {
		t.Fatal("expected sub.custom.org to match")
	}

	mgr.RemoveRule("*.custom.org")
	if mgr.rules.Matches("sub.custom.org") {
		t.Fatal("expected sub.custom.org not to match after remove")
	}
}

func TestManagerTunnelEndToEnd(t *testing.T) {
	// Mock NetLink WebSocket Relay
	connectReceived := make(chan ConnectFrame, 1)
	dataReceived := make(chan []byte, 1)

	relayServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/netconnect/domainroute") {
			http.NotFound(w, r)
			return
		}
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()

		for {
			msgType, msg, err := ws.ReadMessage()
			if err != nil {
				return
			}
			if msgType != websocket.BinaryMessage || len(msg) < 5 {
				continue
			}

			cmd := msg[0]
			switch cmd {
			case CmdConnect:
				cf, err := ParseConnect(msg)
				if err == nil {
					connectReceived <- cf
				}
			case CmdData:
				chID, payload, err := ParseData(msg)
				if err == nil {
					dataReceived <- payload
					// Echo back data: CMD_DATA
					echoFrame := EncodeData(chID, append([]byte("echo: "), payload...))
					_ = ws.WriteMessage(websocket.BinaryMessage, echoFrame)
				}
			case CmdClose:
				return
			}
		}
	}))
	defer relayServer.Close()

	mgr := NewManager(relayServer.URL, "test-device", "127.0.0.1:0", nil)
	mgr.rules.SetRules([]string{"*.netflix.com"})

	// Setup custom dialer pointing to relayServer
	mgr.Tunnel().SetCustomDialer(func(ctx context.Context) (*websocket.Conn, error) {
		wsURL := "ws://" + relayServer.Listener.Addr().String() + "/netconnect/domainroute"
		ws, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
		return ws, err
	})

	if err := mgr.Start(); err != nil {
		t.Fatalf("failed to start manager: %v", err)
	}
	defer mgr.Stop()

	// Wait for tunnel to connect
	deadline := time.Now().Add(3 * time.Second)
	for !mgr.Tunnel().IsConnected() {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for tunnel to connect")
		}
		time.Sleep(20 * time.Millisecond)
	}

	proxyAddr := mgr.listener.Addr().String()

	// Connect client to SOCKS5 proxy requesting a matching domain
	client, err := net.Dial("tcp", proxyAddr)
	if err != nil {
		t.Fatalf("dial proxy failed: %v", err)
	}
	defer client.Close()

	// Handshake
	_, _ = client.Write([]byte{0x05, 0x01, 0x00})
	greetingReply := make([]byte, 2)
	_, _ = io.ReadFull(client, greetingReply)

	domain := "api.netflix.com"
	req := []byte{0x05, 0x01, 0x00, 0x03, byte(len(domain))}
	req = append(req, []byte(domain)...)
	var portBuf [2]byte
	binary.BigEndian.PutUint16(portBuf[:], 443)
	req = append(req, portBuf[:]...)
	_, _ = client.Write(req)

	connReply := make([]byte, 10)
	_, _ = io.ReadFull(client, connReply)
	if connReply[1] != 0x00 {
		t.Fatalf("socks5 connect failed: %v", connReply)
	}

	// Verify relay received CMD_CONNECT
	select {
	case cf := <-connectReceived:
		if cf.Domain != domain {
			t.Errorf("expected domain %s, got %s", domain, cf.Domain)
		}
		if cf.Port != 443 {
			t.Errorf("expected port 443, got %d", cf.Port)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for CMD_CONNECT")
	}

	// Send payload from client through SOCKS5 connection
	testPayload := []byte("hello netflix stream")
	if _, err := client.Write(testPayload); err != nil {
		t.Fatalf("failed to write payload: %v", err)
	}

	// Verify relay received CMD_DATA
	select {
	case p := <-dataReceived:
		if !bytes.Equal(p, testPayload) {
			t.Errorf("expected payload %q, got %q", testPayload, p)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for CMD_DATA on relay")
	}

	// Verify client receives echo response
	expectedEcho := []byte("echo: hello netflix stream")
	echoBuf := make([]byte, len(expectedEcho))
	if _, err := io.ReadFull(client, echoBuf); err != nil {
		t.Fatalf("failed to read echo from client: %v", err)
	}
	if !bytes.Equal(echoBuf, expectedEcho) {
		t.Errorf("expected %q, got %q", expectedEcho, echoBuf)
	}

	// Verify stats
	status := mgr.GetStatus()
	if status.Stats.TunneledConnections != 1 {
		t.Errorf("expected 1 tunneled connection, got %d", status.Stats.TunneledConnections)
	}
}
