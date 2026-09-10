package domainroute

import (
	"bytes"
	"testing"
)

func TestEncodeParseConnect(t *testing.T) {
	channelID := uint32(42)
	port := uint16(443)
	domain := "netflix.com"

	encoded := EncodeConnect(channelID, port, domain)
	expectedLen := 9 + len(domain)
	if len(encoded) != expectedLen {
		t.Fatalf("expected len %d, got %d", expectedLen, len(encoded))
	}

	frame, err := ParseConnect(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if frame.ChannelID != channelID {
		t.Errorf("expected channelID %d, got %d", channelID, frame.ChannelID)
	}
	if frame.Port != port {
		t.Errorf("expected port %d, got %d", port, frame.Port)
	}
	if frame.Domain != domain {
		t.Errorf("expected domain %s, got %s", domain, frame.Domain)
	}
}

func TestEncodeParseData(t *testing.T) {
	channelID := uint32(101)
	payload := []byte("hello world")

	encoded := EncodeData(channelID, payload)
	if len(encoded) != 5+len(payload) {
		t.Fatalf("expected len %d, got %d", 5+len(payload), len(encoded))
	}

	chID, parsedPayload, err := ParseData(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if chID != channelID {
		t.Errorf("expected channelID %d, got %d", channelID, chID)
	}
	if !bytes.Equal(parsedPayload, payload) {
		t.Errorf("payload mismatch: expected %s, got %s", payload, parsedPayload)
	}
}

func TestEncodeParseClose(t *testing.T) {
	channelID := uint32(999)

	encoded := EncodeClose(channelID)
	if len(encoded) != 5 {
		t.Fatalf("expected len 5, got %d", len(encoded))
	}

	chID, err := ParseClose(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if chID != channelID {
		t.Errorf("expected channelID %d, got %d", channelID, chID)
	}
}
