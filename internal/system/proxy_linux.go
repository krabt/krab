//go:build linux

package system

import (
	"fmt"
	"os/exec"
	"strings"
)

// SetProxy uses gsettings, which covers GNOME and most GTK-based desktops.
// Other desktop environments (KDE, etc.) will need their own backend later.
func SetProxy(host string, httpPort, socksPort int) error {
	return SetProxyConfig(ProxyConfig{host, httpPort, host, httpPort, host, socksPort})
}

func SetProxyConfig(config ProxyConfig) error {
	proxies := map[string]struct {
		host string
		port int
	}{
		"http": {config.HTTPHost, config.HTTPPort}, "https": {config.HTTPSHost, config.HTTPSPort}, "socks": {config.SOCKSHost, config.SOCKSPort},
	}
	for kind, proxy := range proxies {
		key := "org.gnome.system.proxy." + kind
		if err := exec.Command("gsettings", "set", key, "host", proxy.host).Run(); err != nil {
			return err
		}
		if err := exec.Command("gsettings", "set", key, "port", fmt.Sprint(proxy.port)).Run(); err != nil {
			return err
		}
	}
	return exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "manual").Run()
}

func ClearProxy() error {
	return exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "none").Run()
}

func ClearStaleProxy(host string, port int) error {
	return nil
}

func GetProxyStatus() (ProxyStatus, error) {
	out, err := exec.Command("gsettings", "get", "org.gnome.system.proxy", "mode").Output()
	if err != nil {
		return ProxyStatus{}, err
	}
	return ProxyStatus{Enabled: strings.TrimSpace(string(out)) == "'manual'"}, nil
}
