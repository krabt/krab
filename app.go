package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	goruntime "runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/krabt/krab/internal/database"
	"github.com/krabt/krab/internal/system"
	"github.com/krabt/krab/internal/tray"
	"github.com/krabt/krab/internal/update"
	"github.com/krabt/krab/internal/xray"
	"github.com/krabt/krab/pkg/probe"
	"github.com/krabt/krab/pkg/profile"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// App is the Wails bound struct: every exported method on it becomes
// callable from the frontend via the generated JS bridge.
type App struct {
	wails        *application.App
	window       *application.WebviewWindow
	systemTray   *application.SystemTray
	manager      *xray.Manager
	store        *profile.Store
	updateInfo   update.Info
	killSwitchOn bool
	proxyConfig  system.ProxyConfig
	trafficStore *xray.TrafficHistoryStore
	trafficMu    sync.Mutex
	trafficStop  chan struct{}
	trafficDone  chan struct{}
	// quitting distinguishes a real quit (from the tray's "Quit Krab")
	// from the window's own close button, which main.go's OnBeforeClose
	// intercepts to hide to the tray instead -- see main.go.
	quitting bool
}

func NewApp() *App {
	return &App{proxyConfig: system.ProxyConfig{HTTPHost: "127.0.0.1", HTTPPort: 5889, HTTPSHost: "127.0.0.1", HTTPSPort: 5889, SOCKSHost: "127.0.0.1", SOCKSPort: 5888}}
}

// startup runs once the Wails runtime is ready and the window exists.
func (a *App) startup(wails *application.App, window *application.WebviewWindow) {
	a.wails = wails
	a.window = window
	if err := database.EnsureDir(); err != nil {
		log.Printf("create Krab data directory: %v", err)
	}
	a.store = profile.NewStore()
	a.manager = xray.NewManager()
	a.trafficStore = xray.NewTrafficHistoryStore()
	if data, found, err := database.Get("system_proxy"); err == nil && found {
		_ = json.Unmarshal(data, &a.proxyConfig)
	}
	a.manager.SetInboundPorts(a.proxyConfig.HTTPPort, a.proxyConfig.HTTPSPort, a.proxyConfig.SOCKSPort)

	// Clean up connection-only state from a previous interrupted run. System
	// proxy settings are deliberately untouched and only change via the
	// explicit SetSystemProxy/ClearSystemProxy actions.
	update.CleanupOldBinary()
	system.RemoveStaleTUNRoutes("172.19.")
	if system.KillSwitchActive() {
		_ = system.DisableKillSwitch()
	}
}

// shutdown stops connection-only resources. Explicit system proxy settings
// remain unchanged until the user clicks "Clear proxy".
func (a *App) shutdown() {
	a.stopTrafficTracking()
	if a.manager != nil {
		a.manager.Stop()
	}
	if a.killSwitchOn {
		_ = system.DisableKillSwitch()
	}
}

// SetWindowTheme keeps the native window chrome in sync with the web UI.
// CSS alone cannot change the macOS title bar or the Windows window frame.
func (a *App) SetWindowTheme(theme string) {
	if a.window == nil {
		return
	}
	setNativeWindowTheme(a.window, theme == "dark")
}

// AutoStartEnabled reports whether Krab is registered to start when the
// current user signs in.
func (a *App) AutoStartEnabled() (bool, error) { return system.AutoStartEnabled() }

// SetAutoStart enables or disables starting Krab when the current user signs in.
func (a *App) SetAutoStart(enabled bool) error {
	if err := system.SetAutoStart(enabled); err != nil {
		return err
	}
	settings := xray.LoadGeoSettings()
	if settings.UI == nil {
		settings.UI = make(map[string]interface{})
	}
	settings.UI["autoStart"] = enabled
	if err := xray.SaveGeoSettings(settings); err != nil {
		// Keep the operating-system registration and persisted preference in sync.
		_ = system.SetAutoStart(!enabled)
		return err
	}
	return nil
}

// --- Server profile methods (bound to frontend) ---

func (a *App) ListProfiles() ([]profile.Server, error) {
	return a.store.List()
}

func (a *App) refreshTrayServers() {
	servers, err := a.store.List()
	if err != nil {
		return
	}
	choices := make([]tray.ServerChoice, 0, len(servers))
	for _, server := range servers {
		choices = append(choices, tray.ServerChoice{ID: server.ID, Name: server.Name})
	}
	tray.SetServers(choices)
	if a.wails != nil {
		a.wails.Event.Emit("profiles:changed", servers)
	}
}

func (a *App) emitConnectionChanged() {
	if a.wails != nil && a.manager != nil {
		a.wails.Event.Emit("connection:changed", a.manager.Status())
	}
}

func (a *App) emitSystemProxyChanged() {
	if a.wails == nil {
		return
	}
	status, err := system.GetProxyStatus()
	if err == nil {
		a.wails.Event.Emit("system-proxy:changed", status)
	}
}

func (a *App) AddProfileFromLink(link string) (profile.Server, error) {
	server, err := profile.ParseLink(link)
	if err != nil {
		return profile.Server{}, err
	}
	server, err = a.store.Add(server)
	if err == nil {
		a.refreshTrayServers()
	}
	return server, err
}

// AddSubscription fetches a subscription URL (the standard V2RayN/
// V2RayNG/Shadowrocket-style base64 link list most providers publish)
// and adds every server it can parse out of it. It succeeds as long as
// at least one server was added, even if some entries in the
// subscription couldn't be parsed.
func (a *App) AddSubscription(subURL string) ([]profile.Server, error) {
	return a.importSubscription(subURL, uuid.NewString())
}

// RefreshSubscription re-fetches the subscription that a given server
// group was originally imported from, replacing that group's servers
// with the freshly parsed list (same group ID, so the UI's expand/
// collapse state and position don't reset).
func (a *App) RefreshSubscription(groupID string) ([]profile.Server, error) {
	all, err := a.store.List()
	if err != nil {
		return nil, err
	}
	var subURL string
	for _, s := range all {
		if s.Extra["subGroup"] == groupID {
			subURL = s.Extra["subURL"]
			break
		}
	}
	if subURL == "" {
		return nil, fmt.Errorf("subscription group not found")
	}
	// importSubscription only drops the group's old servers once the new
	// list has been fetched and parsed, so a failed refresh (offline,
	// provider down) leaves the existing servers untouched.
	return a.importSubscription(subURL, groupID)
}

// DeleteSubscriptionGroup removes every server that was imported from
// the same subscription (same extra.subGroup id).
func (a *App) DeleteSubscriptionGroup(groupID string) error {
	_, err := a.store.ReplaceWhere(inGroup(groupID), nil)
	if err == nil {
		a.refreshTrayServers()
	}
	return err
}

// EditSubscription renames a subscription group and/or changes its URL.
// A changed URL takes effect on the next sync.
func (a *App) EditSubscription(groupID, name, subURL string) error {
	name, subURL = strings.TrimSpace(name), strings.TrimSpace(subURL)
	if name == "" {
		return fmt.Errorf("name can't be empty")
	}
	if u, err := url.Parse(subURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("subscription URL must be an http(s):// link")
	}
	n, err := a.store.UpdateWhere(inGroup(groupID), func(s *profile.Server) {
		s.Extra["subGroupName"] = name
		s.Extra["subURL"] = subURL
	})
	if err == nil && n == 0 {
		err = fmt.Errorf("subscription group not found")
	}
	if err == nil {
		a.refreshTrayServers()
	}
	return err
}

// SaveProfile stores a server from the editor: a new one when it has no ID
// yet (added manually), otherwise an update of the existing one.
func (a *App) SaveProfile(server profile.Server) (profile.Server, error) {
	server.Name = strings.TrimSpace(server.Name)
	server.Address = strings.TrimSpace(server.Address)
	if server.Address == "" {
		return profile.Server{}, fmt.Errorf("address is required")
	}
	if server.Port < 1 || server.Port > 65535 {
		return profile.Server{}, fmt.Errorf("port must be between 1 and 65535")
	}
	if server.Name == "" {
		server.Name = server.Address
	}
	if server.ID == "" {
		saved, err := a.store.Add(server)
		if err == nil {
			a.refreshTrayServers()
		}
		return saved, err
	}
	err := a.store.Update(server)
	if err == nil {
		a.refreshTrayServers()
	}
	return server, err
}

// PingServer measures a server's delay in ms. mode is "tcp", "http" or
// "real" (a real request through a temporary xray-core instance).
func (a *App) PingServer(id, mode string) (int, error) {
	server, err := a.store.Get(id)
	if err != nil {
		return 0, err
	}
	return probe.Ping(server, mode)
}

// ShareLink returns a server's standard share link (vless://, vmess://,
// trojan://, ss://) for copying into another client.
func (a *App) ShareLink(id string) (string, error) {
	server, err := a.store.Get(id)
	if err != nil {
		return "", err
	}
	return profile.ShareLink(server)
}

// ShareSubscription returns the share links of every server in a
// subscription group, one per line -- a plain link list any client can
// import, even when the original subscription URL is private.
func (a *App) ShareSubscription(groupID string) (string, error) {
	all, err := a.store.List()
	if err != nil {
		return "", err
	}
	var links []string
	for _, s := range all {
		if s.Extra["subGroup"] == groupID {
			if link, err := profile.ShareLink(s); err == nil {
				links = append(links, link)
			}
		}
	}
	if len(links) == 0 {
		return "", fmt.Errorf("subscription group not found")
	}
	return strings.Join(links, "\n"), nil
}

func inGroup(groupID string) func(profile.Server) bool {
	return func(s profile.Server) bool { return s.Extra["subGroup"] == groupID }
}

func (a *App) importSubscription(subURL, groupID string) ([]profile.Server, error) {
	req, err := http.NewRequest(http.MethodGet, subURL, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid subscription URL: %w", err)
	}
	// Some subscription providers gate on a client-looking User-Agent.
	req.Header.Set("User-Agent", "Krab/1.0 (compatible; v2rayN/6.0)")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching subscription: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("subscription server returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return nil, fmt.Errorf("reading subscription: %w", err)
	}

	parsed, notes, errs := profile.ParseSubscription(string(body))
	if len(parsed) == 0 {
		if len(errs) > 0 {
			return nil, fmt.Errorf("no valid servers found in subscription: %w", errs[0])
		}
		return nil, fmt.Errorf("no valid servers found in subscription")
	}

	groupName := subURL
	if u, err := url.Parse(subURL); err == nil && u.Host != "" {
		groupName = u.Host
	}
	// Panels like 3x-ui and Marzban name the subscription via
	// Profile-Title, optionally base64-encoded as "base64:...".
	if title := strings.TrimSpace(resp.Header.Get("Profile-Title")); title != "" {
		if enc, ok := strings.CutPrefix(title, "base64:"); ok {
			if dec, err := base64.StdEncoding.DecodeString(enc); err == nil {
				title = string(dec)
			}
		}
		if title = strings.TrimSpace(title); title != "" {
			groupName = title
		}
	}
	// Refresh interval (hours) the provider asks for; the UI auto-syncs
	// subscriptions whose interval has passed.
	updateHours := strings.TrimSpace(resp.Header.Get("Profile-Update-Interval"))
	updatedAt := strconv.FormatInt(time.Now().Unix(), 10)

	// Usage/expiry, when the provider reports it via the informal but
	// widely-adopted Subscription-Userinfo response header, plus any
	// human-readable "info" lines mixed into the link list itself
	// (see profile.isInfoNode) -- both get attached to every server in
	// the group so the UI can show plan/expiry/traffic without a
	// separate subscriptions store.
	var subMeta string
	if usage, ok := profile.ParseSubscriptionUserinfo(resp.Header.Get("Subscription-Userinfo")); ok {
		if b, err := json.Marshal(usage); err == nil {
			subMeta = string(b)
		}
	}
	var subNotes string
	if len(notes) > 0 {
		if b, err := json.Marshal(notes); err == nil {
			subNotes = string(b)
		}
	}

	for i := range parsed {
		if parsed[i].Extra == nil {
			parsed[i].Extra = map[string]string{}
		}
		parsed[i].Extra["subGroup"] = groupID
		parsed[i].Extra["subGroupName"] = groupName
		parsed[i].Extra["subURL"] = subURL
		parsed[i].Extra["subUpdatedAt"] = updatedAt
		if updateHours != "" {
			parsed[i].Extra["subUpdateHours"] = updateHours
		}
		if subMeta != "" {
			parsed[i].Extra["subUsage"] = subMeta
		}
		if subNotes != "" {
			parsed[i].Extra["subNotes"] = subNotes
		}
	}
	servers, err := a.store.ReplaceWhere(inGroup(groupID), parsed)
	if err == nil {
		a.refreshTrayServers()
	}
	return servers, err
}

func (a *App) DeleteProfile(id string) error {
	err := a.store.Delete(id)
	if err == nil {
		a.refreshTrayServers()
	}
	return err
}

func (a *App) RenameProfile(id string, name string) (profile.Server, error) {
	server, err := a.store.Get(id)
	if err != nil {
		return profile.Server{}, err
	}
	server.Name = name
	if err := a.store.Update(server); err != nil {
		return profile.Server{}, err
	}
	a.refreshTrayServers()
	return server, nil
}

// --- Proxy control methods (bound to frontend) ---

// Connect starts xray-core against the given server profile in the given
// mode ("proxy" or "tun"; anything else is treated as "proxy"). TUN mode
// is currently Windows-only and requires the process to already be
// running elevated -- see Platform/IsElevated/RestartElevated below.
// killSwitch, when true, blocks all outbound traffic except Krab's own
// (which is how xray itself reaches the VPN server) and loopback while
// connected -- see internal/system/killswitch_windows.go. It's currently
// Windows-only and needs the same elevation as TUN mode.
func (a *App) Connect(serverID string, mode string, killSwitch bool) error {
	server, err := a.store.Get(serverID)
	if err != nil {
		return err
	}

	xrayMode := xray.ModeProxy
	if mode == string(xray.ModeTUN) {
		if goruntime.GOOS != "windows" {
			return fmt.Errorf("TUN mode is currently only supported on Windows")
		}
		if !system.IsElevated() {
			return fmt.Errorf("TUN mode needs administrator privileges -- use \"Restart as admin\" and try again")
		}
		xrayMode = xray.ModeTUN
	}

	if killSwitch {
		if goruntime.GOOS != "windows" {
			return fmt.Errorf("kill switch is currently only supported on Windows")
		}
		if !system.IsElevated() {
			return fmt.Errorf("kill switch needs administrator privileges -- use \"Restart as admin\" and try again")
		}
	}

	if err := a.manager.Start(server, xrayMode); err != nil {
		a.emitConnectionChanged()
		return err
	}

	if killSwitch {
		if err := system.EnableKillSwitch(); err != nil {
			_ = a.manager.Stop()
			a.emitConnectionChanged()
			return fmt.Errorf("enable kill switch: %w", err)
		}
	}
	a.startTrafficTracking(server)
	a.killSwitchOn = killSwitch
	settings := xray.LoadGeoSettings()
	if settings.UI == nil {
		settings.UI = make(map[string]interface{})
	}
	settings.UI["lastConnectedServerId"] = server.ID
	settings.UI["selectedServerId"] = server.ID
	_ = xray.SaveGeoSettings(settings)
	tray.SetConnected(true)
	a.emitConnectionChanged()
	return nil
}

func (a *App) Disconnect() error {
	a.stopTrafficTracking()
	if a.killSwitchOn {
		_ = system.DisableKillSwitch()
		a.killSwitchOn = false
	}
	err := a.manager.Stop()
	tray.SetConnected(false)
	a.emitConnectionChanged()
	return err
}

// SetSystemProxy applies proxy settings only on explicit user request.
func (a *App) SetSystemProxy(config system.ProxyConfig) error {
	if err := validateProxyConfig(config); err != nil {
		return err
	}
	if err := system.SetProxyConfig(config); err != nil {
		return err
	}
	tray.SetProxyEnabled(true)
	err := a.saveSystemProxyConfig(config)
	a.emitSystemProxyChanged()
	return err
}

// SaveSystemProxyConfig persists the ports used by the next Xray connection
// without changing the operating system's current proxy state.
func (a *App) SaveSystemProxyConfig(config system.ProxyConfig) error {
	if err := validateProxyConfig(config); err != nil {
		return err
	}
	return a.saveSystemProxyConfig(config)
}

func (a *App) saveSystemProxyConfig(config system.ProxyConfig) error {
	a.proxyConfig = config
	if a.manager != nil {
		a.manager.SetInboundPorts(config.HTTPPort, config.HTTPSPort, config.SOCKSPort)
	}
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	return database.Set("system_proxy", data)
}

func (a *App) currentProxyConfig() system.ProxyConfig { return a.proxyConfig }

func (a *App) terminalProxyCommand() string {
	c := a.proxyConfig
	httpProxy := fmt.Sprintf("http://%s:%d", c.HTTPHost, c.HTTPPort)
	httpsProxy := fmt.Sprintf("http://%s:%d", c.HTTPSHost, c.HTTPSPort)
	socksProxy := fmt.Sprintf("socks5://%s:%d", c.SOCKSHost, c.SOCKSPort)
	return fmt.Sprintf("export http_proxy=%s https_proxy=%s all_proxy=%s HTTP_PROXY=%s HTTPS_PROXY=%s ALL_PROXY=%s", httpProxy, httpsProxy, socksProxy, httpProxy, httpsProxy, socksProxy)
}

func (a *App) ClearSystemProxy() error {
	if err := system.ClearProxy(); err != nil {
		return err
	}
	tray.SetProxyEnabled(false)
	a.emitSystemProxyChanged()
	return nil
}

func (a *App) SystemProxyStatus() (system.ProxyStatus, error) { return system.GetProxyStatus() }

func (a *App) GeoSettings() xray.GeoSettings { return xray.LoadGeoSettings() }

// UpdateGeoData downloads geoip.dat and geosite.dat into the configured
// asset directory, optionally through gh-proxy.com.
func (a *App) UpdateGeoData(assetDir string, useGHProxy bool) error {
	return xray.UpdateGeoData(assetDir, useGHProxy)
}

func (a *App) SaveGeoSettings(settings xray.GeoSettings) error {
	if err := xray.SaveGeoSettings(settings); err != nil {
		return err
	}
	a.manager.SetGeoSettings(settings)
	if selectedID, ok := settings.UI["selectedServerId"].(string); ok {
		tray.SetSelectedServer(selectedID)
	}
	return nil
}

func (a *App) OutboundSettings() xray.OutboundSettings { return xray.LoadOutboundSettings() }

func (a *App) SaveOutboundSettings(settings xray.OutboundSettings) error {
	if err := xray.SaveOutboundSettings(settings); err != nil {
		return err
	}
	a.manager.SetOutboundSettings(settings)
	return nil
}

func (a *App) SaveOutboundRule(rule xray.OutboundRule) error {
	settings := xray.LoadOutboundSettings()
	found := false
	for index := range settings.Rules {
		if settings.Rules[index].ID == rule.ID {
			settings.Rules[index] = rule
			found = true
			break
		}
	}
	if !found {
		settings.Rules = append(settings.Rules, rule)
	}
	if err := xray.SaveOutboundSettings(settings); err != nil {
		return err
	}
	a.manager.SetOutboundSettings(settings)
	return nil
}

func (a *App) DeleteOutboundRule(id string) error {
	settings := xray.LoadOutboundSettings()
	rules := make([]xray.OutboundRule, 0, len(settings.Rules))
	for _, rule := range settings.Rules {
		if rule.ID != id {
			rules = append(rules, rule)
		}
	}
	settings.Rules = rules
	if err := xray.SaveOutboundSettings(settings); err != nil {
		return err
	}
	if _, err := a.store.UpdateWhere(func(server profile.Server) bool {
		return slices.Contains(server.OutboundRuleIDs, id)
	}, func(server *profile.Server) {
		server.OutboundRuleIDs = slices.DeleteFunc(server.OutboundRuleIDs, func(ruleID string) bool { return ruleID == id })
	}); err != nil {
		return err
	}
	a.manager.SetOutboundSettings(settings)
	return nil
}

func validateProxyConfig(config system.ProxyConfig) error {
	for name, endpoint := range map[string]struct {
		host string
		port int
	}{
		"HTTP": {config.HTTPHost, config.HTTPPort}, "HTTPS": {config.HTTPSHost, config.HTTPSPort}, "SOCKS": {config.SOCKSHost, config.SOCKSPort},
	} {
		if strings.TrimSpace(endpoint.host) == "" || endpoint.port < 1 || endpoint.port > 65535 {
			return fmt.Errorf("%s proxy host or port is invalid", name)
		}
	}
	if config.SOCKSPort == config.HTTPPort || config.SOCKSPort == config.HTTPSPort {
		return fmt.Errorf("SOCKS proxy port must differ from HTTP and HTTPS proxy ports")
	}
	return nil
}

func (a *App) Status() xray.Status {
	return a.manager.Status()
}

// Traffic returns the current session's cumulative uplink/downlink byte
// counters, for the live traffic display. The frontend polls this and
// diffs successive calls to derive a speed.
func (a *App) Traffic() xray.Traffic {
	traffic := a.manager.Traffic()
	if a.trafficStore != nil {
		history := a.trafficStore.Snapshot()
		traffic.History = &history
	}
	return traffic
}

func (a *App) startTrafficTracking(server profile.Server) {
	a.stopTrafficTracking()
	stop, done := make(chan struct{}), make(chan struct{})
	a.trafficMu.Lock()
	a.trafficStop, a.trafficDone = stop, done
	a.trafficMu.Unlock()
	go func() {
		defer close(done)
		last := a.manager.Traffic()
		sampleTicker := time.NewTicker(time.Second)
		saveTicker := time.NewTicker(10 * time.Second)
		defer sampleTicker.Stop()
		defer saveTicker.Stop()
		record := func() {
			current := a.manager.Traffic()
			proxy := xray.TrafficTotals{Uplink: current.Proxy.Uplink - last.Proxy.Uplink, Downlink: current.Proxy.Downlink - last.Proxy.Downlink}
			direct := xray.TrafficTotals{Uplink: current.Direct.Uplink - last.Direct.Uplink, Downlink: current.Direct.Downlink - last.Direct.Downlink}
			if proxy.Uplink < 0 {
				proxy.Uplink = 0
			}
			if proxy.Downlink < 0 {
				proxy.Downlink = 0
			}
			if direct.Uplink < 0 {
				direct.Uplink = 0
			}
			if direct.Downlink < 0 {
				direct.Downlink = 0
			}
			a.trafficStore.RecordOutbound(server.ID, server.Name, proxy, direct, time.Now())
			last = current
		}
		for {
			select {
			case <-sampleTicker.C:
				record()
			case <-saveTicker.C:
				_ = a.trafficStore.Save()
			case <-stop:
				record()
				_ = a.trafficStore.Save()
				return
			}
		}
	}()
}

func (a *App) stopTrafficTracking() {
	a.trafficMu.Lock()
	stop, done := a.trafficStop, a.trafficDone
	a.trafficStop, a.trafficDone = nil, nil
	a.trafficMu.Unlock()
	if stop != nil {
		close(stop)
		<-done
	}
}

// Platform reports the OS Krab is running on, so the frontend can hide
// the TUN mode option where it isn't supported yet.
func (a *App) Platform() string {
	return goruntime.GOOS
}

// IsElevated reports whether Krab is currently running with administrator
// privileges (relevant to TUN mode, which needs them).
func (a *App) IsElevated() bool {
	return system.IsElevated()
}

// RestartElevated relaunches Krab with a UAC prompt and exits this
// process on success. Only returns when the relaunch itself failed
// (including the user cancelling the prompt).
func (a *App) RestartElevated() error {
	// Disconnect (stop xray/TUN, clear the proxy, remove kill switch
	// firewall rules) *before* relaunching, not left for OnShutdown to do
	// after Quit() -- that cleanup can take real time (TUN adapter
	// teardown, two netsh calls for the kill switch), and every bit of it
	// happening after Quit() eats into the freshly-relaunched process's
	// fixed wait-for-old-instance-to-die buffer below.
	if a.manager != nil && a.manager.Status().State == xray.StateRunning {
		_ = a.Disconnect()
	}

	if err := system.RelaunchElevated(); err != nil {
		return err
	}
	// Quit via Wails' own shutdown path (which releases the
	// SingleInstanceLock's IPC listener as part of tearing down), not a
	// raw os.Exit -- an abrupt kill left that listener around long enough
	// that the freshly-elevated process (launched just above) could still
	// see this instance as "already running" and get treated as a second
	// instance instead of starting for real, leaving two windows open
	// with neither one actually elevated.
	a.quitting = true
	a.wails.Quit()
	return nil
}

// TestResult is what the Test button reports: the exit IP the target site
// actually sees (i.e. the VPN server's IP, not Krab's), the two-letter
// country code Cloudflare's edge resolved it to, and the real round-trip
// time of the whole request (DNS/TCP/TLS/HTTP) made through the tunnel.
type TestResult struct {
	IP      string `json:"ip"`
	Country string `json:"country"`
	DelayMs int64  `json:"delayMs"`
}

// TestConnection makes an actual HTTP request through the local xray HTTP
// inbound so a "running" status that isn't really routing traffic (bad
// outbound handshake, unreachable server, etc.) surfaces a concrete error
// instead of looking like nothing is wrong. It hits Cloudflare's own
// trace endpoint rather than a generic 204 check, since that response
// includes the client IP and country as Cloudflare's edge sees them --
// i.e. exactly what a site would see you connecting from.
func (a *App) TestConnection() (TestResult, error) {
	if a.manager.Status().State != xray.StateRunning {
		return TestResult{}, fmt.Errorf("not connected")
	}

	proxyURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", a.proxyConfig.HTTPPort))
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
	}

	// Two requests over one kept-alive connection: the first opens the
	// tunnel (handshakes), the second is the round trip that's reported --
	// the same way "real delay" is measured.
	var body []byte
	var delay time.Duration
	for i := 0; i < 2; i++ {
		start := time.Now()
		resp, err := client.Get("https://www.cloudflare.com/cdn-cgi/trace")
		if err != nil {
			return TestResult{}, fmt.Errorf("request through proxy failed: %w", err)
		}
		body, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return TestResult{}, fmt.Errorf("read response: %w", err)
		}
		delay = time.Since(start)
	}

	result := TestResult{DelayMs: delay.Milliseconds()}
	for _, line := range strings.Split(string(body), "\n") {
		if ip, ok := strings.CutPrefix(line, "ip="); ok {
			result.IP = strings.TrimSpace(ip)
		}
		if loc, ok := strings.CutPrefix(line, "loc="); ok {
			result.Country = strings.TrimSpace(loc)
		}
	}
	if result.IP == "" {
		return TestResult{}, fmt.Errorf("unexpected response from connectivity check")
	}

	return result, nil
}

// RecentLog returns the tail of xray-core's own error log, for diagnosing a
// connection that reports "running" but isn't actually routing traffic.
func (a *App) RecentLog() (string, error) {
	data, err := os.ReadFile(xray.LogFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	const maxBytes = 8000
	if len(data) > maxBytes {
		data = data[len(data)-maxBytes:]
	}
	return string(data), nil
}

// version is set at build time via -ldflags "-X main.version=v1.2.3"
// (see .github/workflows/release.yml); "dev" otherwise.
var version = "dev"

func (a *App) Version() string {
	return version
}

// XrayVersion returns the embedded xray-core version, for the About panel.
func (a *App) XrayVersion() string {
	return xray.CoreVersion()
}

// --- Self-update methods (bound to frontend) ---

// CheckForUpdate queries GitHub Releases for a newer build. The result is
// cached on the App so a subsequent ApplyUpdate doesn't need the frontend
// to round-trip the (unexported) download URL back to us.
func (a *App) CheckForUpdate() (update.Info, error) {
	info, err := update.Check(version)
	if err != nil {
		return update.Info{}, err
	}
	a.updateInfo = info
	return info, nil
}

// ApplyUpdate downloads and installs the build found by the most recent
// CheckForUpdate, then relaunches. On success this does not return: the
// new process has already started and this one is about to exit. It only
// returns when something failed before that point.
func (a *App) ApplyUpdate() error {
	if !a.updateInfo.Available {
		return fmt.Errorf("no update available; call CheckForUpdate first")
	}

	if a.manager != nil {
		_ = a.manager.Stop()
	}
	if a.killSwitchOn {
		_ = system.DisableKillSwitch()
		a.killSwitchOn = false
	}

	err := update.Apply(a.updateInfo, func(p update.Progress) {
		a.wails.Event.Emit("update:progress", map[string]int64{
			"downloaded": p.Downloaded,
			"total":      p.Total,
		})
	})
	if err != nil {
		return err
	}

	// See RestartElevated's comment: quit via Wails' own shutdown path,
	// not a raw os.Exit, so the SingleInstanceLock's IPC listener is
	// actually released before the already-started new process gets far
	// enough to check it.
	a.quitting = true
	a.wails.Quit()
	return nil
}
