package network

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	hostsBeginMarker = "# BEGIN NETLINK MAGIC DNS"
	hostsEndMarker   = "# END NETLINK MAGIC DNS"
	dnsLogPrefix     = "[NetConnect DNS]"
)

// DnsConfigResponse represents the JSON returned by /api/dns/config.
type DnsConfigResponse struct {
	Enabled bool              `json:"enabled"`
	Server  string            `json:"server"`
	Port    int               `json:"port"`
	Suffix  string            `json:"suffix"`
	Records map[string]string `json:"records"`
}

// DNSManager configures OS-level DNS and syncs MagicDNS records.
type DNSManager struct {
	ifceName             string
	relayURL             string
	stopChan             chan struct{}
	mu                   sync.Mutex
	configuredResolvectl bool
	configuredHosts      bool
	lastRecords          map[string]string
}

// NewDNSManager creates a new DNSManager instance.
func NewDNSManager(ifceName, relayURL string) *DNSManager {
	return &DNSManager{
		ifceName:    ifceName,
		relayURL:    relayURL,
		stopChan:    make(chan struct{}),
		lastRecords: make(map[string]string),
	}
}

// Start begins DNS configuration and periodic record synchronization.
func (dm *DNSManager) Start() {
	go func() {
		if err := dm.syncDNS(); err != nil {
			fmt.Printf("%s Initial DNS configuration failed: %v\n", dnsLogPrefix, err)
		}

		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-dm.stopChan:
				return
			case <-ticker.C:
				if err := dm.syncDNS(); err != nil {
					fmt.Printf("%s DNS sync failed: %v\n", dnsLogPrefix, err)
				}
			}
		}
	}()
}

// Stop reverts all DNS changes and cleans up system configuration.
func (dm *DNSManager) Stop() {
	close(dm.stopChan)
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.revertDNS()
}

func (dm *DNSManager) syncDNS() error {
	cfg, err := dm.fetchConfig()
	if err != nil {
		return err
	}

	dm.mu.Lock()
	defer dm.mu.Unlock()

	dnsServerIP := dm.resolveServerIP(cfg.Server)
	dnsPort := cfg.Port
	if dnsPort <= 0 {
		dnsPort = 53
	}
	suffix := cfg.Suffix
	if suffix == "" {
		suffix = "netlink"
	}

	if !dm.configuredResolvectl && dnsServerIP != "" {
		if err := dm.applyResolvectl(dnsServerIP, dnsPort, suffix); err != nil {
			fmt.Printf("%s resolvectl not available or failed (%v). Using /etc/hosts fallback.\n", dnsLogPrefix, err)
		} else {
			dm.configuredResolvectl = true
			fmt.Printf("%s Configured resolvectl split-DNS (~%s -> %s:%d)\n", dnsLogPrefix, suffix, dnsServerIP, dnsPort)
		}
	}

	if cfg.Records != nil && len(cfg.Records) > 0 {
		if dm.recordsChanged(cfg.Records) || !dm.configuredHosts {
			if err := dm.updateHostsFile(cfg.Records, suffix); err != nil {
				fmt.Printf("%s Failed to update /etc/hosts: %v\n", dnsLogPrefix, err)
			} else {
				dm.configuredHosts = true
				dm.lastRecords = cfg.Records
				fmt.Printf("%s Updated /etc/hosts with %d MagicDNS records\n", dnsLogPrefix, len(cfg.Records))
			}
		}
	}

	return nil
}

func (dm *DNSManager) fetchConfig() (*DnsConfigResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	endpoint := strings.TrimRight(dm.relayURL, "/") + "/api/dns/config"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return &DnsConfigResponse{
			Enabled: true,
			Server:  dm.extractRelayHost(),
			Port:    53,
			Suffix:  "netlink",
		}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var config DnsConfigResponse
	if err := json.Unmarshal(body, &config); err != nil {
		return nil, err
	}

	if config.Server == "" || config.Server == "0.0.0.0" {
		config.Server = dm.extractRelayHost()
	}

	return &config, nil
}

func (dm *DNSManager) extractRelayHost() string {
	u, err := url.Parse(dm.relayURL)
	if err != nil {
		return "127.0.0.1"
	}
	host := u.Hostname()
	if host == "" || host == "localhost" {
		return "127.0.0.1"
	}
	return host
}

func (dm *DNSManager) resolveServerIP(server string) string {
	if server == "" || server == "0.0.0.0" {
		server = dm.extractRelayHost()
	}

	if ip := net.ParseIP(server); ip != nil {
		return ip.String()
	}

	ips, err := net.LookupIP(server)
	if err == nil && len(ips) > 0 {
		for _, ip := range ips {
			if ip4 := ip.To4(); ip4 != nil {
				return ip4.String()
			}
		}
		return ips[0].String()
	}

	return "127.0.0.1"
}

func (dm *DNSManager) applyResolvectl(serverIP string, port int, suffix string) error {
	dnsTarget := serverIP
	if port != 53 {
		dnsTarget = fmt.Sprintf("%s:%d", serverIP, port)
	}

	if err := runCmd("resolvectl", "dns", dm.ifceName, dnsTarget); err != nil {
		if err2 := runCmd("systemd-resolve", "-i", dm.ifceName, fmt.Sprintf("--set-dns=%s", serverIP), fmt.Sprintf("--set-domain=~%s", suffix)); err2 != nil {
			return err
		}
		return nil
	}

	_ = runCmd("resolvectl", "domain", dm.ifceName, fmt.Sprintf("~%s", suffix))
	_ = runCmd("resolvectl", "default-route", dm.ifceName, "false")
	return nil
}

func (dm *DNSManager) revertDNS() {
	if dm.configuredResolvectl {
		_ = runCmd("resolvectl", "revert", dm.ifceName)
		_ = runCmd("systemd-resolve", "--revert", "-i", dm.ifceName)
		dm.configuredResolvectl = false
		fmt.Printf("%s Reverted resolvectl configuration for %s\n", dnsLogPrefix, dm.ifceName)
	}

	if dm.configuredHosts {
		if err := dm.cleanHostsFile(); err != nil {
			fmt.Printf("%s Failed to clean /etc/hosts: %v\n", dnsLogPrefix, err)
		} else {
			dm.configuredHosts = false
			fmt.Printf("%s Cleaned /etc/hosts entries for %s\n", dnsLogPrefix, dm.ifceName)
		}
	}
}

func (dm *DNSManager) recordsChanged(newRecords map[string]string) bool {
	if len(newRecords) != len(dm.lastRecords) {
		return true
	}
	for k, v := range newRecords {
		if dm.lastRecords[k] != v {
			return true
		}
	}
	return false
}

func (dm *DNSManager) updateHostsFile(records map[string]string, suffix string) error {
	content, err := os.ReadFile("/etc/hosts")
	if err != nil {
		return err
	}

	text := string(content)
	cleanText := removeMarkerBlock(text)

	var sb strings.Builder
	sb.WriteString(cleanText)
	if !strings.HasSuffix(cleanText, "\n") {
		sb.WriteString("\n")
	}

	sb.WriteString(hostsBeginMarker + "\n")
	for domain, ip := range records {
		cleanDomain := strings.TrimSuffix(domain, ".")
		shortName := strings.TrimSuffix(cleanDomain, "."+suffix)
		if shortName != cleanDomain {
			sb.WriteString(fmt.Sprintf("%s\t%s\t%s\n", ip, cleanDomain, shortName))
		} else {
			sb.WriteString(fmt.Sprintf("%s\t%s\n", ip, cleanDomain))
		}
	}
	sb.WriteString(hostsEndMarker + "\n")

	return os.WriteFile("/etc/hosts", []byte(sb.String()), 0644)
}

func (dm *DNSManager) cleanHostsFile() error {
	content, err := os.ReadFile("/etc/hosts")
	if err != nil {
		return err
	}

	cleaned := removeMarkerBlock(string(content))
	return os.WriteFile("/etc/hosts", []byte(cleaned), 0644)
}

func removeMarkerBlock(text string) string {
	beginIdx := strings.Index(text, hostsBeginMarker)
	if beginIdx == -1 {
		return text
	}

	endIdx := strings.Index(text, hostsEndMarker)
	if endIdx == -1 {
		return text[:beginIdx]
	}

	endIdx += len(hostsEndMarker)
	if endIdx < len(text) && text[endIdx] == '\n' {
		endIdx++
	}

	return strings.TrimRight(text[:beginIdx], "\r\n") + "\n" + strings.TrimLeft(text[endIdx:], "\r\n")
}
