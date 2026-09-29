package xray

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/krabt/krab/internal/database"
)

type GeoSettings struct {
	Enabled    bool              `json:"enabled"`
	AssetDir   string            `json:"assetDir"`
	UseGHProxy bool              `json:"useGHProxy,omitempty"`
	LogLevel   string            `json:"logLevel,omitempty"`
	DNSHosts   map[string]string `json:"dnsHosts,omitempty"`
	DNSServers []string          `json:"dnsServers,omitempty"`
	UI         map[string]any    `json:"ui,omitempty"`
}

func LoadGeoSettings() GeoSettings {
	data, found, err := database.Get("xray_settings")
	if err != nil || !found {
		return normalizeGeoSettings(GeoSettings{})
	}
	var settings GeoSettings
	_ = json.Unmarshal(data, &settings)
	return normalizeGeoSettings(settings)
}

func SaveGeoSettings(settings GeoSettings) error {
	settings = normalizeGeoSettings(settings)
	if err := validateCoreSettings(settings); err != nil {
		return err
	}
	data, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	return database.Set("xray_settings", data)
}

func normalizeGeoSettings(settings GeoSettings) GeoSettings {
	settings.AssetDir = strings.TrimSpace(settings.AssetDir)
	if settings.AssetDir == "" {
		settings.AssetDir = "~/.krab"
	}
	settings.LogLevel = strings.ToLower(strings.TrimSpace(settings.LogLevel))
	if settings.LogLevel == "" {
		settings.LogLevel = "debug"
	}
	hosts := make(map[string]string, len(settings.DNSHosts))
	for host, target := range settings.DNSHosts {
		host, target = strings.TrimSpace(host), strings.TrimSpace(target)
		if host != "" && target != "" {
			hosts[host] = target
		}
	}
	settings.DNSHosts = hosts
	servers := make([]string, 0, len(settings.DNSServers))
	for _, server := range settings.DNSServers {
		if server = strings.TrimSpace(server); server != "" {
			servers = append(servers, server)
		}
	}
	settings.DNSServers = servers
	return settings
}

func validateCoreSettings(settings GeoSettings) error {
	switch settings.LogLevel {
	case "debug", "info", "warning", "error", "none":
		return nil
	default:
		return fmt.Errorf("invalid Xray log level %q", settings.LogLevel)
	}
}

func ValidateGeoSettings(settings GeoSettings) error {
	dir, err := resolveGeoAssetDir(settings.AssetDir)
	if err != nil {
		return err
	}
	if dir == "" {
		return fmt.Errorf("GeoData resource directory is required")
	}
	for _, name := range []string{"geoip.dat", "geosite.dat"} {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil || info.IsDir() {
			return fmt.Errorf("%s was not found in %s", name, dir)
		}
	}
	return nil
}

// resolveGeoAssetDir expands a user-entered home-relative path before it is
// passed to xray-core. Both slash forms are accepted so a saved configuration
// remains portable across macOS, Linux, and Windows.
func resolveGeoAssetDir(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if value != "~" && !strings.HasPrefix(value, "~/") && !strings.HasPrefix(value, `~\`) {
		return filepath.Clean(value), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	if value == "~" {
		return filepath.Clean(home), nil
	}
	relative := strings.TrimLeft(value[1:], `/\`)
	relative = strings.ReplaceAll(relative, `\`, "/")
	return filepath.Join(home, filepath.FromSlash(relative)), nil
}
