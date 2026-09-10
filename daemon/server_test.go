package daemon

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDomainRouteEndpoints(t *testing.T) {
	srv := NewDaemonServer("http://localhost:4535", "test-device")

	// 1. GET /api/domainroute/status
	req := httptest.NewRequest(http.MethodGet, "/api/domainroute/status", nil)
	w := httptest.NewRecorder()
	srv.handleDomainRouteStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var statusRes map[string]any
	if err := json.NewDecoder(w.Body).Decode(&statusRes); err != nil {
		t.Fatalf("failed to parse status json: %v", err)
	}

	if statusRes["enabled"] != true {
		t.Fatalf("expected enabled to be true, got %v", statusRes["enabled"])
	}

	// 2. POST /api/domainroute/toggle (toggle off)
	req = httptest.NewRequest(http.MethodPost, "/api/domainroute/toggle", bytes.NewReader([]byte(`{"enabled": false}`)))
	w = httptest.NewRecorder()
	srv.handleDomainRouteToggle(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var toggleRes map[string]any
	if err := json.NewDecoder(w.Body).Decode(&toggleRes); err != nil {
		t.Fatalf("failed to parse toggle json: %v", err)
	}

	if toggleRes["enabled"] != false {
		t.Fatalf("expected enabled to be false after toggle, got %v", toggleRes["enabled"])
	}

	// 3. POST /api/domainroute/toggle (toggle on via empty body)
	req = httptest.NewRequest(http.MethodPost, "/api/domainroute/toggle", bytes.NewReader([]byte(`{}`)))
	w = httptest.NewRecorder()
	srv.handleDomainRouteToggle(w, req)

	if err := json.NewDecoder(w.Body).Decode(&toggleRes); err != nil {
		t.Fatalf("failed to parse toggle json: %v", err)
	}
	if toggleRes["enabled"] != true {
		t.Fatalf("expected enabled to be true after toggle, got %v", toggleRes["enabled"])
	}

	// 4. GET /api/domainroute/rules
	req = httptest.NewRequest(http.MethodGet, "/api/domainroute/rules", nil)
	w = httptest.NewRecorder()
	srv.handleDomainRouteRules(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var rulesRes struct {
		Rules []string `json:"rules"`
	}
	if err := json.NewDecoder(w.Body).Decode(&rulesRes); err != nil {
		t.Fatalf("failed to parse rules json: %v", err)
	}

	if len(rulesRes.Rules) == 0 {
		t.Fatal("expected default rules, got empty list")
	}

	// 5. POST /api/domainroute/rules (add rule)
	req = httptest.NewRequest(http.MethodPost, "/api/domainroute/rules", bytes.NewReader([]byte(`{"action": "add", "rule": "*.customstream.net"}`)))
	w = httptest.NewRecorder()
	srv.handleDomainRouteRules(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var addRes struct {
		Success bool     `json:"success"`
		Rules   []string `json:"rules"`
	}
	if err := json.NewDecoder(w.Body).Decode(&addRes); err != nil {
		t.Fatalf("failed to parse add rule json: %v", err)
	}

	found := false
	for _, r := range addRes.Rules {
		if r == "*.customstream.net" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected added rule to be present in rules list")
	}

	// 6. POST /api/domainroute/rules (remove rule)
	req = httptest.NewRequest(http.MethodPost, "/api/domainroute/rules", bytes.NewReader([]byte(`{"action": "remove", "rule": "*.customstream.net"}`)))
	w = httptest.NewRecorder()
	srv.handleDomainRouteRules(w, req)

	var removeRes struct {
		Success bool     `json:"success"`
		Rules   []string `json:"rules"`
	}
	if err := json.NewDecoder(w.Body).Decode(&removeRes); err != nil {
		t.Fatalf("failed to parse remove rule json: %v", err)
	}

	for _, r := range removeRes.Rules {
		if r == "*.customstream.net" {
			t.Fatal("expected removed rule to be absent from rules list")
		}
	}
}
