package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDeviceFlow(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/device/code":
			if r.Method != http.MethodPost {
				http.Error(w, "bad method", http.StatusMethodNotAllowed)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(DeviceCodeResponse{
				DeviceCode:              "dev-code-123",
				UserCode:                "NET-9999",
				VerificationURI:         "http://example.com/devices/authorize",
				VerificationURIComplete: "http://example.com/devices/authorize?code=NET-9999",
				ExpiresIn:               300,
				Interval:                1,
			})

		case "/api/auth/device/token":
			if r.Method != http.MethodPost {
				http.Error(w, "bad method", http.StatusMethodNotAllowed)
				return
			}
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["device_code"] == "dev-code-123" {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(DeviceTokenResponse{
					Status:   "approved",
					Token:    "jwt-mock-user-token",
					TargetID: "target-server-alpha",
					Username: "alice",
				})
				return
			}
			http.Error(w, "invalid device code", http.StatusBadRequest)

		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	client := NewClientWithTarget(ts.URL, "test-device")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Start device flow
	codeRes, err := client.StartDeviceAuth(ctx, "test-device")
	if err != nil {
		t.Fatalf("StartDeviceAuth failed: %v", err)
	}
	if codeRes.UserCode != "NET-9999" {
		t.Errorf("expected user code NET-9999, got %s", codeRes.UserCode)
	}
	if codeRes.DeviceCode != "dev-code-123" {
		t.Errorf("expected device code dev-code-123, got %s", codeRes.DeviceCode)
	}

	// 2. Poll device token
	tokenRes, err := client.PollDeviceAuth(ctx, codeRes.DeviceCode)
	if err != nil {
		t.Fatalf("PollDeviceAuth failed: %v", err)
	}
	if tokenRes.Token != "jwt-mock-user-token" {
		t.Errorf("expected token jwt-mock-user-token, got %s", tokenRes.Token)
	}
	if tokenRes.Username != "alice" {
		t.Errorf("expected username alice, got %s", tokenRes.Username)
	}
	if tokenRes.TargetID != "target-server-alpha" {
		t.Errorf("expected target target-server-alpha, got %s", tokenRes.TargetID)
	}

	// Verify client state was updated
	if client.Token() != "jwt-mock-user-token" {
		t.Errorf("client token mismatch")
	}
	if client.Username() != "alice" {
		t.Errorf("client username mismatch")
	}
	if client.TargetID() != "target-server-alpha" {
		t.Errorf("client target mismatch")
	}

	// 3. Logout
	client.Logout()
	if client.Token() != "" || client.Username() != "" {
		t.Errorf("client logout did not clear state")
	}
}
