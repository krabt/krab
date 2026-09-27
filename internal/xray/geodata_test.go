package xray

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveGeoAssetDirExpandsHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	for _, input := range []string{"~/.krab", `~\.krab`} {
		got, err := resolveGeoAssetDir(input)
		if err != nil {
			t.Fatalf("resolve %q: %v", input, err)
		}
		want := filepath.Join(home, ".krab")
		if got != want {
			t.Fatalf("resolve %q: got %q, want %q", input, got, want)
		}
	}
}

func TestValidateGeoSettingsWithHomeRelativePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	assetDir := filepath.Join(home, ".krab")
	if err := os.MkdirAll(assetDir, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"geoip.dat", "geosite.dat"} {
		if err := os.WriteFile(filepath.Join(assetDir, name), []byte("test"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	if err := ValidateGeoSettings(GeoSettings{Enabled: true, AssetDir: "~/.krab"}); err != nil {
		t.Fatalf("validate home-relative GeoData path: %v", err)
	}
}

func TestNormalizeCoreSettings(t *testing.T) {
	settings := normalizeGeoSettings(GeoSettings{
		LogLevel:   " WARNING ",
		DNSHosts:   map[string]string{" example.com ": " 1.2.3.4 ", "empty": ""},
		DNSServers: []string{" 1.1.1.1 ", "", "https://dns.google/dns-query"},
	})
	if settings.LogLevel != "warning" {
		t.Fatalf("unexpected log level: %q", settings.LogLevel)
	}
	if settings.AssetDir != "~/.krab" {
		t.Fatalf("unexpected default GeoData directory: %q", settings.AssetDir)
	}
	if len(settings.DNSHosts) != 1 || settings.DNSHosts["example.com"] != "1.2.3.4" {
		t.Fatalf("unexpected DNS hosts: %#v", settings.DNSHosts)
	}
	if len(settings.DNSServers) != 2 || settings.DNSServers[0] != "1.1.1.1" {
		t.Fatalf("unexpected DNS servers: %#v", settings.DNSServers)
	}
}

func TestValidateCoreSettingsRejectsUnknownLogLevel(t *testing.T) {
	settings := normalizeGeoSettings(GeoSettings{LogLevel: "verbose"})
	if err := validateCoreSettings(settings); err == nil {
		t.Fatal("expected unknown log level to be rejected")
	}
}
