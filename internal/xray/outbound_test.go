package xray

import (
	"reflect"
	"testing"
)

func TestDefaultOutboundSettings(t *testing.T) {
	settings := DefaultOutboundSettings()
	if len(settings.Rules) != 1 {
		t.Fatalf("expected one default rule, got %d", len(settings.Rules))
	}
	rule := settings.Rules[0]
	if !rule.Enabled {
		t.Fatal("default rule must be enabled")
	}
	if len(rule.WhitelistDomains) != 1 || rule.WhitelistDomains[0] != "geosite:cn" {
		t.Fatalf("unexpected default domain whitelist: %#v", rule.WhitelistDomains)
	}
	if len(rule.WhitelistIPs) != 2 || rule.WhitelistIPs[0] != "geoip:private" || rule.WhitelistIPs[1] != "geoip:cn" {
		t.Fatalf("unexpected default IP whitelist: %#v", rule.WhitelistIPs)
	}
}

func TestNormalizeOutboundProxyEntries(t *testing.T) {
	settings := normalizeOutboundSettings(OutboundSettings{Rules: []OutboundRule{{
		ID: "proxy", Name: "Proxy", Enabled: true,
		ProxyDomains: []string{" domain:example.com ", ""},
		ProxyIPs:     []string{"203.0.113.8", "2001:db8::1"},
	}}})
	rule := settings.Rules[0]
	if len(rule.ProxyDomains) != 1 || rule.ProxyDomains[0] != "domain:example.com" {
		t.Fatalf("unexpected proxy domains: %#v", rule.ProxyDomains)
	}
	if len(rule.ProxyIPs) != 2 || rule.ProxyIPs[0] != "203.0.113.8/32" || rule.ProxyIPs[1] != "2001:db8::1/128" {
		t.Fatalf("unexpected proxy IPs: %#v", rule.ProxyIPs)
	}
}

func TestNormalizeOutboundSettingsSplitsMultilineDomainsAndIPs(t *testing.T) {
	settings := normalizeOutboundSettings(OutboundSettings{Rules: []OutboundRule{{
		ID:               "multiline",
		Name:             "Multiline",
		WhitelistDomains: []string{"example.com, domain:example.org\r\n\nfull:api.example.net"},
		WhitelistIPs:     []string{"192.0.2.1,198.51.100.0/24"},
	}}})

	rule := settings.Rules[0]
	wantDomains := []string{"example.com", "domain:example.org", "full:api.example.net"}
	if !reflect.DeepEqual(rule.WhitelistDomains, wantDomains) {
		t.Fatalf("unexpected multiline domains: got %#v, want %#v", rule.WhitelistDomains, wantDomains)
	}
	wantIPs := []string{"192.0.2.1/32", "198.51.100.0/24"}
	if !reflect.DeepEqual(rule.WhitelistIPs, wantIPs) {
		t.Fatalf("unexpected multiline IPs: got %#v, want %#v", rule.WhitelistIPs, wantIPs)
	}
}
