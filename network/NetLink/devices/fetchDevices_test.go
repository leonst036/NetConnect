package devices

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/leonst036/NetConnect/network/NetLink/auth"
)

func TestFetchDevices(t *testing.T) {
	mockDevices := []Device{
		{IP: "192.168.1.10", Hostname: "printer"},
		{IP: "192.168.1.20", Hostname: "desktop"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/validate-target":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(auth.ValidateTargetResponse{
				Valid: true,
				Token: "mock-token-123",
			})
		case "/api/auth/ticket":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(auth.TicketResponse{
				Success: true,
				Ticket:  "mock-ticket-abc",
			})
		case "/api/net-graph/scan":
			authHeader := r.Header.Get("Authorization")
			if authHeader != "Ticket mock-ticket-abc" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(mockDevices)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	devices, err := FetchDevices(server.URL, 5*time.Second)
	if err != nil {
		t.Fatalf("FetchDevices failed: %v", err)
	}

	if len(devices) != len(mockDevices) {
		t.Fatalf("expected %d devices, got %d", len(mockDevices), len(devices))
	}
	if devices[0].IP != mockDevices[0].IP || devices[0].Hostname != mockDevices[0].Hostname {
		t.Errorf("device mismatch: got %+v, want %+v", devices[0], mockDevices[0])
	}
}

func TestFetchDevicesError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/validate-target":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(auth.ValidateTargetResponse{
				Valid: true,
				Token: "mock-token-123",
			})
		case "/api/auth/ticket":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(auth.TicketResponse{
				Success: true,
				Ticket:  "mock-ticket-abc",
			})
		case "/api/net-graph/scan":
			http.Error(w, "internal server error", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	_, err := FetchDevices(server.URL, 5*time.Second)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
