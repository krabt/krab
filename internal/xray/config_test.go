package xray

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/krabt/krab/pkg/profile"
)

func TestBuildJSONMergesOnlyServerOutboundRules(t *testing.T) {
	server := profile.Server{
		Protocol: "vless", Address: "example.com", Port: 443,
		UUID:  "b831381d-6324-4d53-ad4f-8cda48b30811",
		Extra: map[string]string{}, OutboundRuleIDs: []string{"rule-b", "rule-c", "rule-d"},
	}
	settings := OutboundSettings{Rules: []OutboundRule{
		{ID: "rule-a", Enabled: true, BlacklistDomains: []string{"domain:not-linked.example"}},
		{ID: "rule-b", Enabled: true, WhitelistDomains: []string{"domain:direct.example"}},
		{ID: "rule-c", Enabled: true, BlacklistIPs: []string{"203.0.113.0/24"}},
		{ID: "rule-d", Enabled: true, ProxyDomains: []string{"domain:proxy.example"}},
	}}
	data, err := buildJSON(server, ModeProxy, "", 0, GeoSettings{}, settings)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "not-linked.example") {
		t.Fatalf("unlinked rule leaked into config: %s", data)
	}
	var config struct {
		Outbounds []struct {
			Tag string `json:"tag"`
		} `json:"outbounds"`
		Routing struct {
			Rules []json.RawMessage `json:"rules"`
		} `json:"routing"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	if len(config.Routing.Rules) != 3 {
		t.Fatalf("expected three merged routing entries, got %d: %s", len(config.Routing.Rules), data)
	}
	foundDirectTag := false
	foundBlockedTag := false
	foundProxyRuleTag := false
	for _, rawRule := range config.Routing.Rules {
		var rule struct {
			OutboundTag string   `json:"outboundTag"`
			Domains     []string `json:"domain"`
		}
		if err := json.Unmarshal(rawRule, &rule); err != nil {
			t.Fatal(err)
		}
		if len(rule.Domains) == 1 && rule.Domains[0] == "domain:direct.example" && rule.OutboundTag == "direct" {
			foundDirectTag = true
		}
		if rule.OutboundTag == "blocked" {
			foundBlockedTag = true
		}
		if len(rule.Domains) == 1 && rule.Domains[0] == "domain:proxy.example" && rule.OutboundTag == "proxy" {
			foundProxyRuleTag = true
		}
	}
	if !foundDirectTag {
		t.Fatalf("expected whitelist rule to use fixed direct outbound tag: %s", data)
	}
	if !foundBlockedTag {
		t.Fatalf("expected blacklist rule to use fixed blocked outbound tag: %s", data)
	}
	if !foundProxyRuleTag {
		t.Fatalf("expected proxy rule to use fixed proxy outbound tag: %s", data)
	}
	if len(config.Outbounds) == 0 || config.Outbounds[0].Tag != "proxy" {
		t.Fatalf("expected the server outbound to use fixed proxy tag: %s", data)
	}
}
