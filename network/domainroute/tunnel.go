package domainroute

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/leonst036/NetConnect/network/NetLink/auth"
)

type TunnelClient struct {
	mu           sync.RWMutex
	relayURL     string
	targetID     string
	client       *auth.Client
	stats        *Stats
	ws           *websocket.Conn
	wsMu         sync.Mutex
	channels     sync.Map
	nextChanID   atomic.Uint32
	closed       atomic.Bool
	stopCh       chan struct{}
	isConnected  atomic.Bool
	customDialer func(ctx context.Context) (*websocket.Conn, error)
}

func NewTunnelClient(relayURL, targetID string, client *auth.Client, stats *Stats) *TunnelClient {
	if stats == nil {
		stats = &Stats{}
	}
	return &TunnelClient{
		relayURL: relayURL,
		targetID: targetID,
		client:   client,
		stats:    stats,
		stopCh:   make(chan struct{}),
	}
}

func (t *TunnelClient) SetCustomDialer(fn func(ctx context.Context) (*websocket.Conn, error)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.customDialer = fn
}

func (t *TunnelClient) Start() {
	if t.closed.Load() {
		return
	}
	go t.connectLoop()
}

func (t *TunnelClient) Stop() {
	if t.closed.CompareAndSwap(false, true) {
		close(t.stopCh)
		t.wsMu.Lock()
		t.isConnected.Store(false)
		if t.ws != nil {
			_ = t.ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			_ = t.ws.Close()
			t.ws = nil
		}
		t.wsMu.Unlock()
		t.closeAllChannels()
	}
}

func (t *TunnelClient) IsConnected() bool {
	return t.isConnected.Load()
}

func (t *TunnelClient) UpdateConfig(relayURL, targetID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.relayURL = relayURL
	t.targetID = targetID
	if relayURL != "" {
		t.client = auth.GetOrCreateClient(relayURL)
	}
}

func (t *TunnelClient) AllocateChannelID() uint32 {
	for {
		id := t.nextChanID.Add(1)
		if id != 0 {
			return id
		}
	}
}

func (t *TunnelClient) RegisterChannel(id uint32, ch *Channel) {
	t.channels.Store(id, ch)
}

func (t *TunnelClient) GetChannel(id uint32) (*Channel, bool) {
	val, ok := t.channels.Load(id)
	if !ok {
		return nil, false
	}
	return val.(*Channel), true
}

func (t *TunnelClient) RemoveChannel(id uint32) {
	t.channels.Delete(id)
}

func (t *TunnelClient) ActiveChannelsCount() int {
	count := 0
	t.channels.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}

func (t *TunnelClient) SendConnect(channelID uint32, port uint16, domain string) error {
	t.wsMu.Lock()
	defer t.wsMu.Unlock()

	if t.ws == nil || !t.isConnected.Load() {
		return errors.New("tunnel not connected")
	}

	frame := EncodeConnect(channelID, port, domain)
	if err := t.ws.WriteMessage(websocket.BinaryMessage, frame); err != nil {
		return err
	}

	t.stats.AddBytesSent(uint64(len(frame)))
	return nil
}

func (t *TunnelClient) SendData(channelID uint32, payload []byte) error {
	t.wsMu.Lock()
	defer t.wsMu.Unlock()

	if t.ws == nil || !t.isConnected.Load() {
		return errors.New("tunnel not connected")
	}

	w, err := t.ws.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return err
	}

	var hdr [5]byte
	hdr[0] = CmdData
	binary.BigEndian.PutUint32(hdr[1:5], channelID)

	if _, err := w.Write(hdr[:]); err != nil {
		_ = w.Close()
		return err
	}
	if _, err := w.Write(payload); err != nil {
		_ = w.Close()
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	t.stats.AddBytesSent(uint64(len(payload) + 5))
	return nil
}

func (t *TunnelClient) SendClose(channelID uint32) error {
	t.wsMu.Lock()
	defer t.wsMu.Unlock()

	if t.ws == nil || !t.isConnected.Load() {
		return nil
	}

	frame := EncodeClose(channelID)
	if err := t.ws.WriteMessage(websocket.BinaryMessage, frame); err != nil {
		return err
	}

	t.stats.AddBytesSent(uint64(len(frame)))
	return nil
}

func (t *TunnelClient) closeAllChannels() {
	t.channels.Range(func(key, val any) bool {
		ch := val.(*Channel)
		ch.Close()
		t.channels.Delete(key)
		return true
	})
}

func (t *TunnelClient) dial(ctx context.Context) (*websocket.Conn, error) {
	t.mu.RLock()
	customDialer := t.customDialer
	relayURL := t.relayURL
	targetID := t.targetID
	t.mu.RUnlock()

	if customDialer != nil {
		return customDialer(ctx)
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

	if targetID == "" && t.client != nil {
		targetID = t.client.TargetID()
	}

	var ticket string
	if t.client != nil {
		ticket, _ = t.client.GetTicket(ctx)
	}

	q := url.Values{}
	if targetID != "" {
		q.Set("target", targetID)
	}
	if ticket != "" {
		q.Set("ticket", ticket)
	} else if t.client != nil && t.client.Token() != "" {
		q.Set("token", t.client.Token())
	}

	endpoint := fmt.Sprintf("%s/netconnect/domainroute", wsURL)
	if encoded := q.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}

	header := http.Header{}
	if ticket != "" {
		header.Set("Authorization", "Ticket "+ticket)
	} else if t.client != nil && t.client.Token() != "" {
		header.Set("Authorization", "Bearer "+t.client.Token())
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}

	ws, resp, err := dialer.DialContext(ctx, endpoint, header)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusUnauthorized && t.client != nil {
			t.client.InvalidateTicket()
		}
		if resp != nil && resp.Body != nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("domainroute dial status %d: %s (%w)", resp.StatusCode, string(body), err)
		}
		return nil, fmt.Errorf("domainroute dial failed: %w", err)
	}

	return ws, nil
}

func (t *TunnelClient) connectLoop() {
	backoff := 1 * time.Second
	maxBackoff := 15 * time.Second

	for !t.closed.Load() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		ws, err := t.dial(ctx)
		cancel()

		if err != nil {
			select {
			case <-time.After(backoff):
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				continue
			case <-t.stopCh:
				return
			}
		}

		backoff = 1 * time.Second

		t.wsMu.Lock()
		t.ws = ws
		t.isConnected.Store(true)
		t.wsMu.Unlock()

		t.readLoop(ws)

		t.wsMu.Lock()
		t.isConnected.Store(false)
		if t.ws == ws {
			_ = t.ws.Close()
			t.ws = nil
		}
		t.wsMu.Unlock()

		t.closeAllChannels()
	}
}

func (t *TunnelClient) readLoop(ws *websocket.Conn) {
	for {
		msgType, msg, err := ws.ReadMessage()
		if err != nil {
			return
		}
		if msgType != websocket.BinaryMessage || len(msg) < 5 {
			continue
		}

		cmd := msg[0]
		channelID := binary.BigEndian.Uint32(msg[1:5])

		switch cmd {
		case CmdData:
			payload := msg[5:]
			t.stats.AddBytesReceived(uint64(len(payload)))
			if val, ok := t.channels.Load(channelID); ok {
				ch := val.(*Channel)
				ch.PushIncoming(payload)
			}
		case CmdClose:
			if val, ok := t.channels.Load(channelID); ok {
				ch := val.(*Channel)
				ch.Close()
				t.channels.Delete(channelID)
			}
		}
	}
}
