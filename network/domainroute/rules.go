package domainroute

import (
	"net"
	"regexp"
	"strings"
	"sync"
)

var DefaultPatterns = []string{
	"*.netflix.com",
	"*.nflxvideo.net",
	"*.nflximg.net",
	"*.nflxext.com",
	"*.nflxso.net",
	"*.ipify.org",
}

type compiledRule struct {
	pattern string
	re      *regexp.Regexp
}

func compilePattern(pattern string) (*compiledRule, error) {
	clean := strings.TrimSpace(pattern)
	if clean == "" {
		return nil, nil
	}

	lower := strings.ToLower(clean)

	var regexStr string
	if strings.HasPrefix(lower, "regexp:") {
		regexStr = "(?i)" + clean[7:]
	} else if strings.HasPrefix(lower, "^") || strings.HasSuffix(lower, "$") {
		regexStr = "(?i)" + clean
	} else if strings.HasPrefix(lower, "*.") {
		base := clean[2:]
		regexStr = "(?i)^(.*\\.)?" + regexp.QuoteMeta(base) + "$"
	} else if strings.Contains(clean, "*") {
		parts := strings.Split(clean, "*")
		quoted := make([]string, len(parts))
		for i, p := range parts {
			quoted[i] = regexp.QuoteMeta(p)
		}
		regexStr = "(?i)^" + strings.Join(quoted, ".*") + "$"
	} else {
		regexStr = "(?i)^" + regexp.QuoteMeta(clean) + "$"
	}

	re, err := regexp.Compile(regexStr)
	if err != nil {
		return nil, err
	}

	return &compiledRule{
		pattern: clean,
		re:      re,
	}, nil
}

type RuleEngine struct {
	mu    sync.RWMutex
	rules []compiledRule
}

func NewRuleEngine(patterns ...string) *RuleEngine {
	engine := &RuleEngine{}
	if len(patterns) == 0 {
		engine.SetRules(DefaultPatterns)
	} else {
		engine.SetRules(patterns)
	}
	return engine
}

func (e *RuleEngine) Matches(host string) bool {
	cleanHost := strings.TrimSpace(host)
	if cleanHost == "" {
		return false
	}

	if strings.Contains(cleanHost, ":") {
		if h, _, err := net.SplitHostPort(cleanHost); err == nil {
			cleanHost = h
		}
	}
	cleanHost = strings.TrimSuffix(strings.ToLower(cleanHost), ".")

	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, rule := range e.rules {
		if rule.re.MatchString(cleanHost) {
			return true
		}
	}
	return false
}

func (e *RuleEngine) GetRules() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]string, len(e.rules))
	for i, r := range e.rules {
		result[i] = r.pattern
	}
	return result
}

func (e *RuleEngine) SetRules(patterns []string) {
	var compiled []compiledRule
	for _, p := range patterns {
		rule, err := compilePattern(p)
		if err == nil && rule != nil {
			compiled = append(compiled, *rule)
		}
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules = compiled
}

func (e *RuleEngine) AddRule(pattern string) bool {
	rule, err := compilePattern(pattern)
	if err != nil || rule == nil {
		return false
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	for _, r := range e.rules {
		if strings.EqualFold(r.pattern, rule.pattern) {
			return true
		}
	}

	e.rules = append(e.rules, *rule)
	return true
}

func (e *RuleEngine) RemoveRule(pattern string) bool {
	clean := strings.TrimSpace(pattern)

	e.mu.Lock()
	defer e.mu.Unlock()

	newRules := make([]compiledRule, 0, len(e.rules))
	found := false
	for _, r := range e.rules {
		if strings.EqualFold(r.pattern, clean) {
			found = true
		} else {
			newRules = append(newRules, r)
		}
	}
	e.rules = newRules
	return found
}
