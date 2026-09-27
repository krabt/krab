//go:build darwin

package system

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// networkServices lists every enabled network service (Wi-Fi, Ethernet,
// USB tethering, ...) -- the proxy is per-service on macOS, so setting it
// on "Wi-Fi" alone leaves wired connections unproxied.
func networkServices() ([]string, error) {
	out, err := exec.Command("networksetup", "-listallnetworkservices").Output()
	if err != nil {
		return nil, err
	}
	var services []string
	for i, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		// First line is an explanatory header; "*" marks disabled services.
		if i == 0 || line == "" || strings.HasPrefix(line, "*") {
			continue
		}
		services = append(services, line)
	}
	return services, nil
}

func SetProxy(host string, httpPort, socksPort int) error {
	return SetProxyConfig(ProxyConfig{host, httpPort, host, httpPort, host, socksPort})
}

func SetProxyConfig(config ProxyConfig) error {
	services, err := networkServices()
	if err != nil {
		return err
	}
	for _, svc := range services {
		for _, args := range [][]string{
			{"-setwebproxy", svc, config.HTTPHost, fmt.Sprint(config.HTTPPort)},
			{"-setsecurewebproxy", svc, config.HTTPSHost, fmt.Sprint(config.HTTPSPort)},
			{"-setsocksfirewallproxy", svc, config.SOCKSHost, fmt.Sprint(config.SOCKSPort)},
		} {
			if err := exec.Command("networksetup", args...).Run(); err != nil {
				return fmt.Errorf("networksetup %s %s: %w", args[0], svc, err)
			}
		}
	}
	return nil
}

func ClearProxy() error {
	services, err := networkServices()
	if err != nil {
		return err
	}
	for _, svc := range services {
		for _, flag := range []string{"-setwebproxystate", "-setsecurewebproxystate", "-setsocksfirewallproxystate"} {
			_ = exec.Command("networksetup", flag, svc, "off").Run()
		}
	}
	return nil
}

func ClearStaleProxy(host string, port int) error {
	return nil
}

func GetProxyStatus() (ProxyStatus, error) {
	out, err := exec.Command("scutil", "--proxy").Output()
	if err != nil {
		return ProxyStatus{}, err
	}
	values := make(map[string]string)
	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), ":", 2)
		if len(parts) == 2 {
			values[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	port := func(key string) int {
		value, _ := strconv.Atoi(values[key])
		return value
	}
	return ProxyStatus{
		Enabled: values["HTTPEnable"] == "1" || values["HTTPSEnable"] == "1" || values["SOCKSEnable"] == "1",
		Config: ProxyConfig{
			HTTPHost: values["HTTPProxy"], HTTPPort: port("HTTPPort"),
			HTTPSHost: values["HTTPSProxy"], HTTPSPort: port("HTTPSPort"),
			SOCKSHost: values["SOCKSProxy"], SOCKSPort: port("SOCKSPort"),
		},
	}, nil
}
