package domainroute

import (
	"net"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/leonst036/NetConnect/network/NetLink/auth"
)

type Status struct {
	Running        bool          `json:"running"`
	Enabled        bool          `json:"enabled"`
	Port           int           `json:"port"`
	BindAddress    string        `json:"bindAddress"`
	ActiveChannels int           `json:"activeChannels"`
	RelayConnected bool          `json:"relayConnected"`
	Stats          StatsSnapshot `json:"stats"`
}

type Manager struct {
	mu          sync.Mutex
	bindAddr    string
	port        int
	rules       *RuleEngine
	tunnel      *TunnelClient
	listener    *ProxyListener
	stats       *Stats
	enabled     atomic.Bool
	running     atomic.Bool
}

func NewManager(relayURL, targetID, bindAddr string, client *auth.Client) *Manager {
	if bindAddr == "" {
		bindAddr = "127.0.0.1:1080"
	}

	port := 1080
	if _, portStr, err := net.SplitHostPort(bindAddr); err == nil {
		if p, err := strconv.Atoi(portStr); err == nil && p > 0 {
			port = p
		}
	}

	stats := &Stats{}
	rules := NewRuleEngine()
	tunnel := NewTunnelClient(relayURL, targetID, client, stats)
	listener := NewProxyListener(bindAddr, rules, tunnel, stats)

	m := &Manager{
		bindAddr: bindAddr,
		port:     port,
		rules:    rules,
		tunnel:   tunnel,
		listener: listener,
		stats:    stats,
	}
	m.enabled.Store(true)
	return m
}

func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running.Load() {
		return nil
	}

	m.tunnel.Start()

	if err := m.listener.Start(); err != nil {
		m.tunnel.Stop()
		return err
	}

	m.running.Store(true)
	return nil
}

func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running.Load() {
		return
	}

	m.listener.Stop()
	m.tunnel.Stop()
	m.running.Store(false)
}

func (m *Manager) IsRunning() bool {
	return m.running.Load()
}

func (m *Manager) IsEnabled() bool {
	return m.enabled.Load()
}

func (m *Manager) SetEnabled(enabled bool) {
	m.enabled.Store(enabled)
	m.listener.SetEnabled(enabled)
}

func (m *Manager) ToggleEnabled() bool {
	newState := !m.enabled.Load()
	m.SetEnabled(newState)
	return newState
}

func (m *Manager) GetStatus() Status {
	activeChannels := 0
	relayConnected := false
	if m.tunnel != nil {
		activeChannels = m.tunnel.ActiveChannelsCount()
		relayConnected = m.tunnel.IsConnected()
	}

	return Status{
		Running:        m.running.Load(),
		Enabled:        m.enabled.Load(),
		Port:           m.port,
		BindAddress:    m.bindAddr,
		ActiveChannels: activeChannels,
		RelayConnected: relayConnected,
		Stats:          m.stats.Snapshot(),
	}
}

func (m *Manager) GetRules() []string {
	return m.rules.GetRules()
}

func (m *Manager) SetRules(rules []string) {
	m.rules.SetRules(rules)
}

func (m *Manager) AddRule(rule string) bool {
	return m.rules.AddRule(rule)
}

func (m *Manager) RemoveRule(rule string) bool {
	return m.rules.RemoveRule(rule)
}

func (m *Manager) UpdateConfig(relayURL, targetID string) {
	m.tunnel.UpdateConfig(relayURL, targetID)
}

func (m *Manager) BindAddress() string {
	return m.bindAddr
}

func (m *Manager) Port() int {
	return m.port
}

func (m *Manager) Tunnel() *TunnelClient {
	return m.tunnel
}
