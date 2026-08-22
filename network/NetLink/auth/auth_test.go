package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientTokenAndTicketFlow(t *testing.T) {
	var validateCalls int32
	var ticketCalls int32
	var protectedCalls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/validate-target":
			atomic.AddInt32(&validateCalls, 1)
			target := r.URL.Query().Get("target")
			if target == "" {
				http.Error(w, "missing target", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(ValidateTargetResponse{
				Valid: true,
				Token: "mock-jwt-token-123",
			})

		case "/api/auth/ticket":
			atomic.AddInt32(&ticketCalls, 1)
			auth := r.Header.Get("Authorization")
			if auth != "Bearer mock-jwt-token-123" {
				http.Error(w, "invalid bearer token", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(TicketResponse{
				Success: true,
				Ticket:  "mock-ticket-abc",
			})

		case "/api/netconnect/ping":
			atomic.AddInt32(&protectedCalls, 1)
			auth := r.Header.Get("Authorization")
			if auth != "Ticket mock-ticket-abc" {
				http.Error(w, "invalid ticket", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"message":"pong"}`))

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClientWithTarget(server.URL, "test-target-device")

	// 1. Verify token retrieval
	ctx := context.Background()
	token, err := client.GetToken(ctx)
	if err != nil {
		t.Fatalf("unexpected GetToken error: %v", err)
	}
	if token != "mock-jwt-token-123" {
		t.Fatalf("expected token mock-jwt-token-123, got: %s", token)
	}
	if atomic.LoadInt32(&validateCalls) != 1 {
		t.Fatalf("expected 1 validate-target call, got: %d", validateCalls)
	}

	// 2. Verify ticket retrieval
	ticket, err := client.GetTicket(ctx)
	if err != nil {
		t.Fatalf("unexpected GetTicket error: %v", err)
	}
	if ticket != "mock-ticket-abc" {
		t.Fatalf("expected ticket mock-ticket-abc, got: %s", ticket)
	}
	if atomic.LoadInt32(&ticketCalls) != 1 {
		t.Fatalf("expected 1 ticket call, got: %d", ticketCalls)
	}

	// 3. Verify cached ticket is reused
	ticket2, err := client.GetTicket(ctx)
	if err != nil {
		t.Fatalf("unexpected GetTicket second call error: %v", err)
	}
	if ticket2 != ticket {
		t.Fatalf("expected cached ticket, got: %s", ticket2)
	}
	if atomic.LoadInt32(&ticketCalls) != 1 {
		t.Fatalf("expected ticket call count to still be 1, got: %d", ticketCalls)
	}

	// 4. Verify authorized GET request
	resp, err := client.Get(ctx, "/api/netconnect/ping")
	if err != nil {
		t.Fatalf("unexpected Get error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got: %d", resp.StatusCode)
	}
	if atomic.LoadInt32(&protectedCalls) != 1 {
		t.Fatalf("expected 1 protected call, got: %d", protectedCalls)
	}
}

func TestClientAutomaticTicketRenewalOn401(t *testing.T) {
	var ticketCount int32
	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/validate-target":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(ValidateTargetResponse{
				Valid: true,
				Token: "mock-jwt-token-123",
			})

		case "/api/auth/ticket":
			count := atomic.AddInt32(&ticketCount, 1)
			w.Header().Set("Content-Type", "application/json")
			if count == 1 {
				_ = json.NewEncoder(w).Encode(TicketResponse{
					Success: true,
					Ticket:  "expired-ticket-1",
				})
			} else {
				_ = json.NewEncoder(w).Encode(TicketResponse{
					Success: true,
					Ticket:  "fresh-ticket-2",
				})
			}

		case "/api/protected":
			reqNum := atomic.AddInt32(&requestCount, 1)
			auth := r.Header.Get("Authorization")
			if auth == "Ticket expired-ticket-1" {
				http.Error(w, "ticket expired", http.StatusUnauthorized)
				return
			}
			if auth == "Ticket fresh-ticket-2" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"status":"ok"}`))
				return
			}
			t.Errorf("unexpected authorization header on request %d: %s", reqNum, auth)
			http.Error(w, "unauthorized", http.StatusUnauthorized)

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClientWithTarget(server.URL, "test-device")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Get(ctx, "/api/protected")
	if err != nil {
		t.Fatalf("unexpected request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 after retry, got: %d", resp.StatusCode)
	}

	if atomic.LoadInt32(&ticketCount) != 2 {
		t.Fatalf("expected 2 ticket requests (initial + refresh), got: %d", ticketCount)
	}
	if atomic.LoadInt32(&requestCount) != 2 {
		t.Fatalf("expected 2 requests (failed + retried), got: %d", requestCount)
	}
}
