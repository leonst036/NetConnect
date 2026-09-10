package domainroute

import (
	"testing"
)

func TestDefaultRulesMatching(t *testing.T) {
	engine := NewRuleEngine()

	tests := []struct {
		host  string
		match bool
	}{
		{"netflix.com", true},
		{"www.netflix.com", true},
		{"api.prod.netflix.com", true},
		{"netflix.com:443", true},
		{"WWW.NETFLIX.COM:443", true},
		{"fakenetflix.com", false},
		{"netflix.com.evil.org", false},
		{"nflxvideo.net", true},
		{"ipv4-c001-ams001-ix.1.oca.nflxvideo.net", true},
		{"nflximg.net", true},
		{"assets.nflximg.net", true},
		{"nflxext.com", true},
		{"nflxso.net", true},
		{"ipify.org", true},
		{"api.ipify.org", true},
		{"api4.ipify.org:80", true},
		{"google.com", false},
		{"github.com", false},
	}

	for _, tt := range tests {
		got := engine.Matches(tt.host)
		if got != tt.match {
			t.Errorf("Matches(%q) = %v; want %v", tt.host, got, tt.match)
		}
	}
}

func TestRegexAndCustomRules(t *testing.T) {
	engine := NewRuleEngine("regexp:^(.*\\.)?stream\\.(com|org)$", "api*.example.com", "exact.org")

	tests := []struct {
		host  string
		match bool
	}{
		{"my.stream.com", true},
		{"stream.org", true},
		{"stream.net", false},
		{"api1.example.com", true},
		{"api-v2.example.com", true},
		{"web.example.com", false},
		{"exact.org", true},
		{"sub.exact.org", false},
	}

	for _, tt := range tests {
		got := engine.Matches(tt.host)
		if got != tt.match {
			t.Errorf("Matches(%q) = %v; want %v", tt.host, got, tt.match)
		}
	}
}

func TestAddRemoveRules(t *testing.T) {
	engine := NewRuleEngine()
	if engine.Matches("foo.bar.com") {
		t.Fatalf("expected foo.bar.com not to match")
	}

	engine.AddRule("*.bar.com")
	if !engine.Matches("foo.bar.com") {
		t.Fatalf("expected foo.bar.com to match after add")
	}

	removed := engine.RemoveRule("*.bar.com")
	if !removed {
		t.Fatalf("expected rule to be removed")
	}
	if engine.Matches("foo.bar.com") {
		t.Fatalf("expected foo.bar.com not to match after remove")
	}
}
