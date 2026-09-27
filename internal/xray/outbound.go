package xray

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/krabt/krab/internal/database"
)

type OutboundSettings struct {
	Rules []OutboundRule `json:"rules"`
}

type OutboundRule struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Enabled          bool     `json:"enabled"`
	WhitelistDomains []string `json:"whitelistDomains"`
	WhitelistIPs     []string `json:"whitelistIPs"`
	BlacklistDomains []string `json:"blacklistDomains"`
	BlacklistIPs     []string `json:"blacklistIPs"`
	ProxyDomains     []string `json:"proxyDomains"`
	ProxyIPs         []string `json:"proxyIPs"`
}

func DefaultOutboundSettings() OutboundSettings {
	return OutboundSettings{Rules: []OutboundRule{{
		ID:               "default-cn-private",
		Name:             "CN 和私有网络",
		Enabled:          true,
		WhitelistDomains: []string{"geosite:cn"},
		WhitelistIPs:     []string{"geoip:private", "geoip:cn"},
	}}}
}

func legacyOutboundSettingsPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "kite", "outbound-settings.json")
}

func LoadOutboundSettings() OutboundSettings {
	data, found, err := database.Get("outbound_settings")
	if err != nil {
		return DefaultOutboundSettings()
	}
	if !found {
		data, err = os.ReadFile(legacyOutboundSettingsPath())
		if err != nil {
			return DefaultOutboundSettings()
		}
		_ = database.Set("outbound_settings", data)
	}
	var settings OutboundSettings
	if json.Unmarshal(data, &settings) != nil {
		return DefaultOutboundSettings()
	}
	if len(settings.Rules) == 0 {
		return DefaultOutboundSettings()
	}
	return normalizeOutboundSettings(settings)
}

func SaveOutboundSettings(settings OutboundSettings) error {
	settings = normalizeOutboundSettings(settings)
	data, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	return database.Set("outbound_settings", data)
}

func normalizeOutboundSettings(settings OutboundSettings) OutboundSettings {
	cleanDomains := func(values []string) []string {
		result := make([]string, 0, len(values))
		for _, value := range splitOutboundLines(values) {
			value = strings.TrimSpace(value)
			if value != "" {
				result = append(result, value)
			}
		}
		return result
	}
	cleanIPs := func(values []string) []string {
		result := make([]string, 0, len(values))
		for _, value := range splitOutboundLines(values) {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			if net.ParseIP(value) != nil && !strings.Contains(value, "/") {
				if strings.Contains(value, ":") {
					value += "/128"
				} else {
					value += "/32"
				}
			}
			result = append(result, value)
		}
		return result
	}
	for index := range settings.Rules {
		rule := &settings.Rules[index]
		rule.ID = strings.TrimSpace(rule.ID)
		if rule.ID == "" {
			rule.ID = fmt.Sprintf("rule-%d", index+1)
		}
		rule.Name = strings.TrimSpace(rule.Name)
		if rule.Name == "" {
			rule.Name = fmt.Sprintf("规则 %d", index+1)
		}
		rule.WhitelistDomains = cleanDomains(rule.WhitelistDomains)
		rule.BlacklistDomains = cleanDomains(rule.BlacklistDomains)
		rule.ProxyDomains = cleanDomains(rule.ProxyDomains)
		rule.WhitelistIPs = cleanIPs(rule.WhitelistIPs)
		rule.BlacklistIPs = cleanIPs(rule.BlacklistIPs)
		rule.ProxyIPs = cleanIPs(rule.ProxyIPs)
	}
	return settings
}

func splitOutboundLines(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, strings.FieldsFunc(value, func(character rune) bool {
			return character == ',' || character == '\n' || character == '\r'
		})...)
	}
	return result
}
