package xrayconf

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf/serial"
	_ "github.com/xtls/xray-core/main/distro/all"

	"github.com/krabt/krab/pkg/profile"
)

func TestMultipleOutboundRules(t *testing.T) {
	s := profile.Server{Protocol: "vless", Address: "example.com", Port: 443, UUID: "b831381d-6324-4d53-ad4f-8cda48b30811", Extra: map[string]string{}}
	cfg, err := Build(s, Options{SOCKSPort: 1080, OutboundRules: []RoutingRule{
		{Enabled: true, BlacklistDomains: []string{"domain:ads.example"}, WhitelistIPs: []string{"10.0.0.0/8"}},
		{Enabled: false, BlacklistDomains: []string{"domain:disabled.example"}},
		{Enabled: true, WhitelistDomains: []string{"domain:direct.example"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Routing struct {
			Rules []struct {
				Domain      []string `json:"domain"`
				IP          []string `json:"ip"`
				OutboundTag string   `json:"outboundTag"`
			} `json:"rules"`
		} `json:"routing"`
	}
	if err := json.Unmarshal(cfg, &decoded); err != nil {
		t.Fatal(err)
	}
	rules := decoded.Routing.Rules
	if len(rules) != 3 {
		t.Fatalf("expected 3 routing entries, got %d: %s", len(rules), cfg)
	}
	if rules[0].OutboundTag != "blocked" || rules[0].Domain[0] != "domain:ads.example" {
		t.Fatalf("first rule should block the first group's domain: %#v", rules[0])
	}
	if rules[1].OutboundTag != "direct" || rules[1].IP[0] != "10.0.0.0/8" {
		t.Fatalf("second rule should direct the first group's IP: %#v", rules[1])
	}
	if rules[2].OutboundTag != "direct" || rules[2].Domain[0] != "domain:direct.example" {
		t.Fatalf("third rule should direct the last group's domain: %#v", rules[2])
	}
	if strings.Contains(string(cfg), "disabled.example") {
		t.Fatalf("disabled group leaked into xray config: %s", cfg)
	}
}

func TestMultipleHTTPInboundPorts(t *testing.T) {
	s := profile.Server{Protocol: "vless", Address: "example.com", Port: 443, UUID: "b831381d-6324-4d53-ad4f-8cda48b30811", Extra: map[string]string{}}
	cfg, err := Build(s, Options{HTTPPorts: []int{5889, 5890, 5889}, SOCKSPort: 5888})
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Inbounds []struct {
			Tag      string `json:"tag"`
			Port     int    `json:"port"`
			Protocol string `json:"protocol"`
		} `json:"inbounds"`
	}
	if err := json.Unmarshal(cfg, &decoded); err != nil {
		t.Fatal(err)
	}
	want := []struct {
		tag      string
		port     int
		protocol string
	}{{"http-in", 5889, "http"}, {"http-in-2", 5890, "http"}, {"socks-in", 5888, "socks"}}
	if len(decoded.Inbounds) != len(want) {
		t.Fatalf("inbounds = %#v, want %d entries", decoded.Inbounds, len(want))
	}
	for index, expected := range want {
		got := decoded.Inbounds[index]
		if got.Tag != expected.tag || got.Port != expected.port || got.Protocol != expected.protocol {
			t.Fatalf("inbound %d = %#v, want %#v", index, got, expected)
		}
	}
}

func TestOutboundRulePriorityAcrossGroups(t *testing.T) {
	s := profile.Server{Protocol: "vless", Address: "example.com", Port: 443, UUID: "b831381d-6324-4d53-ad4f-8cda48b30811", Extra: map[string]string{}}
	cfg, err := Build(s, Options{SOCKSPort: 1080, OutboundRules: []RoutingRule{
		{Enabled: true, WhitelistDomains: []string{"geosite:cn"}},
		{Enabled: true, ProxyDomains: []string{"domain:example.com"}},
		{Enabled: true, BlacklistDomains: []string{"domain:baidu.com"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		Routing struct {
			Rules []struct {
				OutboundTag string `json:"outboundTag"`
			} `json:"rules"`
		} `json:"routing"`
	}
	if err := json.Unmarshal(cfg, &parsed); err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(parsed.Routing.Rules))
	for _, rule := range parsed.Routing.Rules {
		got = append(got, rule.OutboundTag)
	}
	want := []string{"blocked", "proxy", "direct"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("routing priority = %v, want %v", got, want)
	}
}

func TestVLESSEncryption(t *testing.T) {
	enc := "mlkem768x25519plus.native.0rtt.o6NZ0a1EJo29bvlnOqDUQO2rjfiSFw91q2dBqrOqtxw"
	s, err := profile.ParseLink("vless://id@h.example:443?encryption=" + enc + "&security=reality&pbk=K&sid=ab&sni=www.example.com&fp=chrome&type=tcp#x")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := Build(s, Options{SOCKSPort: 1080})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfg), `"encryption":"`+enc+`"`) {
		t.Fatalf("encryption not passed through: %s", cfg)
	}

	plain, _ := profile.ParseLink("vless://id@h.example:443?security=tls&type=ws#y")
	cfg, _ = Build(plain, Options{SOCKSPort: 1080})
	if !strings.Contains(string(cfg), `"encryption":"none"`) {
		t.Fatalf("expected encryption none by default: %s", cfg)
	}
}

func TestHysteria2Outbound(t *testing.T) {
	s := profile.Server{Protocol: "hysteria2", Address: "example.com", Port: 443, Password: "pw",
		Extra: map[string]string{"obfs": "salamander", "obfs-password": "x", "mport": "20000-30000"}}
	cfg, err := Build(s, Options{SOCKSPort: 1080})
	if err != nil {
		t.Fatal(err)
	}
	config, err := serial.LoadJSONConfig(bytes.NewReader(cfg))
	if err != nil {
		t.Fatalf("xray rejected config: %v\n%s", err, cfg)
	}
	if _, err := core.New(config); err != nil {
		t.Fatalf("core.New: %v\n%s", err, cfg)
	}
}

func TestSSHConfig(t *testing.T) {
	s := profile.Server{Protocol: "ssh", Address: "1.2.3.4", Port: 22, Password: "pw"}
	cfg, err := Build(s, Options{SOCKSPort: 1080, TUN: false, SSHBridgePort: 5555})
	if err != nil {
		t.Fatal(err)
	}
	config, err := serial.LoadJSONConfig(bytes.NewReader(cfg))
	if err != nil {
		t.Fatalf("xray rejected config: %v\n%s", err, cfg)
	}
	if _, err := core.New(config); err != nil {
		t.Fatalf("core.New: %v\n%s", err, cfg)
	}
}

func TestLogLevelAndDNSConfig(t *testing.T) {
	s := profile.Server{Protocol: "vless", Address: "example.com", Port: 443, UUID: "b831381d-6324-4d53-ad4f-8cda48b30811", Extra: map[string]string{}}
	cfg, err := Build(s, Options{
		SOCKSPort: 1080,
		LogLevel:  "warning",
		DNSHosts:  map[string]string{"example.com": "1.2.3.4"},
		DNSServers: []string{
			"1.1.1.1",
			"https://dns.google/dns-query",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Log struct {
			Level string `json:"loglevel"`
		} `json:"log"`
		DNS struct {
			Hosts   map[string]string `json:"hosts"`
			Servers []string          `json:"servers"`
		} `json:"dns"`
	}
	if err := json.Unmarshal(cfg, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Log.Level != "warning" {
		t.Fatalf("unexpected log level: %q", decoded.Log.Level)
	}
	if decoded.DNS.Hosts["example.com"] != "1.2.3.4" || len(decoded.DNS.Servers) != 2 {
		t.Fatalf("unexpected DNS config: %#v", decoded.DNS)
	}
	if _, err := serial.LoadJSONConfig(bytes.NewReader(cfg)); err != nil {
		t.Fatalf("xray rejected DNS config: %v\n%s", err, cfg)
	}
}

func TestTransports(t *testing.T) {
	for _, extra := range []map[string]string{
		{"type": "httpupgrade", "path": "/up", "host": "h.example"},
		{"type": "xhttp", "path": "/x", "mode": "packet-up", "extra": `{"xPaddingBytes":"100-1000"}`},
		{"type": "h2", "path": "/h2", "host": "h.example", "security": "tls"},
		{"type": "kcp", "seed": "s", "headerType": "wechat-video"},
		{"type": "kcp"},
		{"type": "grpc", "serviceName": "svc", "mode": "multi", "security": "tls"},
		{"type": "ws", "security": "tls", "sni": "a.example", "ech": "cloudflare-ech.com+https://1.1.1.1/dns-query"},
	} {
		s := profile.Server{Protocol: "vless", Address: "example.com", Port: 443, UUID: "b831381d-6324-4d53-ad4f-8cda48b30811", Extra: extra}
		cfg, err := Build(s, Options{SOCKSPort: 1080})
		if err != nil {
			t.Fatal(err)
		}
		config, err := serial.LoadJSONConfig(bytes.NewReader(cfg))
		if err != nil {
			t.Fatalf("%v: xray rejected config: %v\n%s", extra, err, cfg)
		}
		if _, err := core.New(config); err != nil {
			t.Fatalf("%v: core.New: %v\n%s", extra, err, cfg)
		}
	}
}
