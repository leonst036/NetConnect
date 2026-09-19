package network

import (
	"strings"
	"testing"
)

func TestRemoveMarkerBlock(t *testing.T) {
	initial := "127.0.0.1 localhost\n# BEGIN NETLINK MAGIC DNS\n10.0.0.5 pc.netlink pc\n# END NETLINK MAGIC DNS\n::1 ip6-localhost\n"
	cleaned := removeMarkerBlock(initial)

	if strings.Contains(cleaned, "pc.netlink") {
		t.Errorf("expected marked block to be removed, got: %s", cleaned)
	}
	if !strings.Contains(cleaned, "127.0.0.1 localhost") {
		t.Errorf("expected localhost line to be preserved, got: %s", cleaned)
	}
	if !strings.Contains(cleaned, "::1 ip6-localhost") {
		t.Errorf("expected ip6-localhost line to be preserved, got: %s", cleaned)
	}
}

func TestDNSManagerExtractRelayHost(t *testing.T) {
	dm := NewDNSManager("netconnect0", "http://192.168.1.50:4535")
	host := dm.extractRelayHost()
	if host != "192.168.1.50" {
		t.Errorf("expected 192.168.1.50, got %s", host)
	}

	dmLocal := NewDNSManager("netconnect0", "http://localhost:4535")
	hostLocal := dmLocal.extractRelayHost()
	if hostLocal != "127.0.0.1" {
		t.Errorf("expected 127.0.0.1 for localhost, got %s", hostLocal)
	}
}

func TestIsValidHostname(t *testing.T) {
	valid := []string{"example.com", "my-host", "node_1.netlink", "a", "sub.domain.local"}
	for _, h := range valid {
		if !isValidHostname(h) {
			t.Errorf("expected %q to be valid", h)
		}
	}

	invalid := []string{"", "bad host", "host\nname", "host\rname", "host\tname", "evil/path"}
	for _, h := range invalid {
		if isValidHostname(h) {
			t.Errorf("expected %q to be invalid", h)
		}
	}
}
