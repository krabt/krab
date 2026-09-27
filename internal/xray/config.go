// Package xray runs xray-core (github.com/xtls/xray-core) as an in-process
// Go library: it builds a config from a server profile and starts/stops an
// xray-core instance, in proxy mode (local HTTP/SOCKS) or TUN mode.
package xray

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/krabt/krab/pkg/profile"
	"github.com/krabt/krab/pkg/xrayconf"
)

const (
	HTTPInboundPort  = 5889
	SOCKSInboundPort = 5888

	// TUN adapter settings. Each connection gets its own 172.19.N.1/30
	// (see manager.go); the kill switch allows all of 172.19.0.0/16.
	TUNMask = "255.255.255.252"
	TUNDNS  = "1.1.1.1"
)

// Mode selects what xray-core listens on: a local HTTP/SOCKS proxy
// ("proxy") or a TUN adapter capturing all system traffic ("tun").
type Mode = string

const (
	ModeProxy Mode = "proxy"
	ModeTUN   Mode = "tun"
)

// CoreVersion returns the embedded xray-core version, for the About panel.
// xray-core doesn't expose a runtime version constant, so this reads the
// resolved module version straight from the Go module graph instead.
func CoreVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == "github.com/xtls/xray-core" {
				return strings.TrimPrefix(dep.Version, "v")
			}
		}
	}
	return "unknown"
}

// LogFilePath returns where xray-core's own error log is written.
func LogFilePath() string {
	dir, err := os.UserHomeDir()
	if err != nil {
		dir = "."
	}
	logDir := filepath.Join(dir, ".krab")
	_ = os.MkdirAll(logDir, 0o700)
	return filepath.Join(logDir, "xray.log")
}

func buildJSON(server profile.Server, mode Mode, tunName string, sshBridgePort int, settings GeoSettings, outbound OutboundSettings) ([]byte, error) {
	return buildJSONWithInboundPorts(server, mode, tunName, sshBridgePort, settings, outbound, []int{HTTPInboundPort}, SOCKSInboundPort)
}

func buildJSONWithInboundPorts(server profile.Server, mode Mode, tunName string, sshBridgePort int, settings GeoSettings, outbound OutboundSettings, httpPorts []int, socksPort int) ([]byte, error) {
	selectedRules := make(map[string]struct{}, len(server.OutboundRuleIDs))
	for _, id := range server.OutboundRuleIDs {
		selectedRules[id] = struct{}{}
	}
	rules := make([]xrayconf.RoutingRule, 0, len(server.OutboundRuleIDs))
	for _, rule := range outbound.Rules {
		if _, selected := selectedRules[rule.ID]; !selected {
			continue
		}
		rules = append(rules, xrayconf.RoutingRule{
			Enabled:          rule.Enabled,
			WhitelistDomains: rule.WhitelistDomains,
			WhitelistIPs:     rule.WhitelistIPs,
			BlacklistDomains: rule.BlacklistDomains,
			BlacklistIPs:     rule.BlacklistIPs,
			ProxyDomains:     rule.ProxyDomains,
			ProxyIPs:         rule.ProxyIPs,
		})
	}
	return xrayconf.Build(server, xrayconf.Options{
		HTTPPorts:     httpPorts,
		SOCKSPort:     socksPort,
		TUN:           mode == ModeTUN,
		TUNName:       tunName,
		LogPath:       LogFilePath(),
		SSHBridgePort: sshBridgePort,
		UseGeoData:    settings.Enabled,
		LogLevel:      settings.LogLevel,
		DNSHosts:      settings.DNSHosts,
		DNSServers:    settings.DNSServers,
		OutboundRules: rules,
	})
}

var firstNonEmpty = xrayconf.FirstNonEmpty
