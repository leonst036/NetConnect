package domainroute

import "sync/atomic"

type StatsSnapshot struct {
	TotalConnections    uint64 `json:"totalConnections"`
	TunneledConnections uint64 `json:"tunneledConnections"`
	BypassedConnections uint64 `json:"bypassedConnections"`
	BytesSent           uint64 `json:"bytesSent"`
	BytesReceived       uint64 `json:"bytesReceived"`
}

type Stats struct {
	totalConnections    atomic.Uint64
	tunneledConnections atomic.Uint64
	bypassedConnections atomic.Uint64
	bytesSent           atomic.Uint64
	bytesReceived       atomic.Uint64
}

func (s *Stats) IncTotal() {
	s.totalConnections.Add(1)
}

func (s *Stats) IncTunneled() {
	s.tunneledConnections.Add(1)
}

func (s *Stats) IncBypassed() {
	s.bypassedConnections.Add(1)
}

func (s *Stats) AddBytesSent(n uint64) {
	s.bytesSent.Add(n)
}

func (s *Stats) AddBytesReceived(n uint64) {
	s.bytesReceived.Add(n)
}

func (s *Stats) Snapshot() StatsSnapshot {
	return StatsSnapshot{
		TotalConnections:    s.totalConnections.Load(),
		TunneledConnections: s.tunneledConnections.Load(),
		BypassedConnections: s.bypassedConnections.Load(),
		BytesSent:           s.bytesSent.Load(),
		BytesReceived:       s.bytesReceived.Load(),
	}
}
