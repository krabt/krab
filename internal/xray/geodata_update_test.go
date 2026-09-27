package xray

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestGeoDataDownloadURL(t *testing.T) {
	if got := geoDataDownloadURL(geoIPDownloadURL, false); got != geoIPDownloadURL {
		t.Fatalf("direct URL = %q", got)
	}
	want := "https://gh-proxy.com/" + geoIPDownloadURL
	if got := geoDataDownloadURL(geoIPDownloadURL, true); got != want {
		t.Fatalf("proxy URL = %q, want %q", got, want)
	}
}

func TestDownloadGeoFileWritesNewFile(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader("geodata")),
		}, nil
	})}

	path := filepath.Join(t.TempDir(), "geoip.dat.new")
	if err := downloadGeoFile(client, "geoip.dat", "https://example.test/geoip.dat", path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "geodata" {
		t.Fatalf("downloaded data = %q", data)
	}
}
