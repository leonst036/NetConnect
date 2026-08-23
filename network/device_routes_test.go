package network

import (
	"net"
	"testing"
)

func TestIsIPLocal(t *testing.T) {
	// 127.0.0.1 should not be matched as physical local network
	isLocal, err := IsIPLocal("127.0.0.1", "netconnect0")
	if err != nil {
		t.Fatalf("unexpected error checking 127.0.0.1: %v", err)
	}
	if isLocal {
		t.Errorf("expected 127.0.0.1 not to be matched as a physical non-loopback local network")
	}

	// Invalid IP should return error
	_, err = IsIPLocal("invalid-ip", "netconnect0")
	if err == nil {
		t.Errorf("expected error for invalid IP")
	}
}

func TestDeviceRouteManagerInit(t *testing.T) {
	overlayIP := net.ParseIP("10.0.0.1")
	drm := NewDeviceRouteManager("netconnect0", overlayIP, "http://localhost:4535")
	if drm == nil {
		t.Fatalf("expected non-nil DeviceRouteManager")
	}

	if drm.IsMapped("192.168.55.10") {
		t.Errorf("expected IP not to be mapped initially")
	}
}
