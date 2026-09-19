package netlink

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/leonst036/NetConnect/network/NetLink/auth"
)

// StreamConn wraps a WebSocket connection to implement net.Conn.
type StreamConn struct {
	ws        *websocket.Conn
	reader    io.Reader
	readMu    sync.Mutex
	writeMu   sync.Mutex
	localAddr net.Addr
	remAddr   net.Addr
	closed    atomic.Bool
}

// DialLANStream dials a destination IP:port on the remote LAN via the NetLink relay WSS tunnel.
func DialLANStream(ctx context.Context, relayURL, targetID, destIP string, destPort int) (*StreamConn, error) {
	client := auth.GetOrCreateClient(relayURL)

	header := http.Header{}
	var authQuery string

	ticket, err := client.GetTicket(ctx)
	if err == nil && ticket != "" {
		authQuery = "ticket=" + url.QueryEscape(ticket)
		header.Set("Authorization", "Ticket "+ticket)
	} else {
		token, tokenErr := client.GetToken(ctx)
		if tokenErr != nil {
			return nil, fmt.Errorf("failed to get ticket (%v) or token (%w) for stream", err, tokenErr)
		}
		authQuery = "token=" + url.QueryEscape(token)
		header.Set("Authorization", "Bearer "+token)
	}

	wsURL := relayURL
	if strings.HasPrefix(wsURL, "http://") {
		wsURL = "ws://" + strings.TrimPrefix(wsURL, "http://")
	} else if strings.HasPrefix(wsURL, "https://") {
		wsURL = "wss://" + strings.TrimPrefix(wsURL, "https://")
	} else if !strings.HasPrefix(wsURL, "ws://") && !strings.HasPrefix(wsURL, "wss://") {
		wsURL = "ws://" + wsURL
	}
	wsURL = strings.TrimRight(wsURL, "/")

	if targetID == "" {
		targetID = client.TargetID()
	}

	endpoint := fmt.Sprintf("%s/netconnect/stream?target=%s&destIP=%s&destPort=%d&%s",
		wsURL,
		url.QueryEscape(targetID),
		url.QueryEscape(destIP),
		destPort,
		authQuery,
	)

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	ws, resp, err := dialer.DialContext(ctx, endpoint, header)
	if err != nil {
		if resp != nil && resp.Body != nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("stream dial failed (status %d): %s (%w)", resp.StatusCode, string(body), err)
		}
		return nil, fmt.Errorf("stream dial failed: %w", err)
	}

	localAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	remAddr, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", destIP, destPort))

	return &StreamConn{
		ws:        ws,
		localAddr: localAddr,
		remAddr:   remAddr,
	}, nil
}

func (s *StreamConn) Read(b []byte) (n int, err error) {
	s.readMu.Lock()
	defer s.readMu.Unlock()

	if s.closed.Load() {
		return 0, io.EOF
	}

	for {
		if s.reader != nil {
			n, err = s.reader.Read(b)
			if err == io.EOF {
				s.reader = nil
				if n > 0 {
					return n, nil
				}
				continue
			}
			return n, err
		}

		messageType, reader, err := s.ws.NextReader()
		if err != nil {
			return 0, err
		}

		if messageType == websocket.BinaryMessage || messageType == websocket.TextMessage {
			s.reader = reader
			n, err = s.reader.Read(b)
			if err == io.EOF {
				s.reader = nil
			}
			return n, err
		}
	}
}

func (s *StreamConn) Write(b []byte) (n int, err error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if s.closed.Load() {
		return 0, io.ErrClosedPipe
	}

	err = s.ws.WriteMessage(websocket.BinaryMessage, b)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func (s *StreamConn) Close() error {
	if s.closed.CompareAndSwap(false, true) {
		s.writeMu.Lock()
		defer s.writeMu.Unlock()
		_ = s.ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		return s.ws.Close()
	}
	return nil
}

func (s *StreamConn) LocalAddr() net.Addr                { return s.localAddr }
func (s *StreamConn) RemoteAddr() net.Addr               { return s.remAddr }
func (s *StreamConn) SetDeadline(t time.Time) error      { return s.ws.SetReadDeadline(t) }
func (s *StreamConn) SetReadDeadline(t time.Time) error  { return s.ws.SetReadDeadline(t) }
func (s *StreamConn) SetWriteDeadline(t time.Time) error { return s.ws.SetWriteDeadline(t) }
