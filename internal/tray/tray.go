// Package tray implements Krab's system tray using Wails v3's native API.
package tray

import (
	"fmt"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

var (
	connectItem      *application.MenuItem
	disconnectItem   *application.MenuItem
	statusItem       *application.MenuItem
	systemTray       *application.SystemTray
	serverItems      = map[string]serverMenuItem{}
	selectedID       string
	serviceConnected bool
	proxyEnabled     bool
	mu               sync.Mutex
)

type ServerChoice struct{ ID, Name string }
type serverMenuItem struct {
	name string
	item *application.MenuItem
}

// SetupSystemTray creates and configures Krab's native tray icon and menu.
// Wails owns the native event loop and starts the registered tray from Run.
func SetupSystemTray(app *application.App, iconPNG []byte, servers []ServerChoice, initialServerID string, initialProxyEnabled bool, onShow func(), onConnect func(string), onDisconnect, onSetProxy, onClearProxy, onCopyProxy func(), onSelectServer func(string), onQuit func()) *application.SystemTray {
	mu.Lock()
	serverItems = make(map[string]serverMenuItem, len(servers))
	selectedID = initialServerID
	serviceConnected = false
	proxyEnabled = initialProxyEnabled
	mu.Unlock()

	menu := app.NewMenu()
	menu.Add("显示 Krab").OnClick(func(*application.Context) { onShow() })
	statusItem = menu.Add(" ").SetTooltip(statusTooltip(false, initialProxyEnabled))
	statusItem.SetBitmap(statusBitmap(false, initialProxyEnabled))
	menu.AddSeparator()
	serverMenu := menu.AddSubmenu("选择服务器")
	if len(servers) == 0 {
		serverMenu.Add("暂无服务器").SetEnabled(false)
	}
	for _, server := range servers {
		choice := server
		label := choice.Name
		if choice.ID == initialServerID {
			label += " *"
		}
		item := serverMenu.Add(label)
		item.OnClick(func(*application.Context) {
			SetSelectedServer(choice.ID)
			onSelectServer(choice.ID)
		})
		serverItems[choice.ID] = serverMenuItem{name: choice.Name, item: item}
	}
	menu.AddSeparator()
	connectItem = menu.Add("连接服务")
	connectItem.SetEnabled(initialServerID != "")
	connectItem.OnClick(func(*application.Context) {
		mu.Lock()
		id := selectedID
		mu.Unlock()
		if id != "" {
			onConnect(id)
		}
	})
	disconnectItem = menu.Add("断开服务").SetEnabled(false)
	disconnectItem.OnClick(func(*application.Context) { onDisconnect() })
	menu.AddSeparator()
	menu.Add("设置代理").OnClick(func(*application.Context) { onSetProxy() })
	menu.Add("取消代理").OnClick(func(*application.Context) { onClearProxy() })
	menu.Add("复制代理命令").OnClick(func(*application.Context) { onCopyProxy() })
	menu.AddSeparator()

	menu.Add("退出 Krab").OnClick(func(*application.Context) { onQuit() })

	systemTray = app.SystemTray.New()
	systemTray.SetIcon(iconPNG)
	systemTray.SetMenu(menu)
	systemTray.OnClick(onShow)
	SetSelectedServer(initialServerID)
	mu.Lock()
	refreshTooltipLocked()
	mu.Unlock()
	return systemTray
}

func statusTooltip(connected, proxyEnabled bool) string {
	serviceText, proxyText := "服务未连接", "代理已关闭"
	if connected {
		serviceText = "服务已连接"
	}
	if proxyEnabled {
		proxyText = "代理已开启"
	}
	return serviceText + " · " + proxyText
}

func statusBitmap(connected, proxyEnabled bool) []byte {
	stateColor := func(active bool) string {
		if active {
			return "#22c55e"
		}
		return "#9ca3af"
	}
	return []byte(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="40" height="16" viewBox="0 0 40 16" fill="none">
<g stroke="%s" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
  <path d="M8 1.25v6"/>
  <path d="M3.93 3.93a5.75 5.75 0 1 0 8.14 0"/>
</g>
<g stroke="%s" stroke-width="1.35" stroke-linecap="round" stroke-linejoin="round">
  <circle cx="30" cy="8" r="5.75"/>
  <ellipse cx="30" cy="8" rx="2.65" ry="5.75"/>
  <path d="M24.25 8h11.5"/>
</g>
</svg>`, stateColor(connected), stateColor(proxyEnabled)))
}

func refreshTooltipLocked() {
	if systemTray == nil {
		return
	}
	systemTray.SetTooltip("Krab · " + statusTooltip(serviceConnected, proxyEnabled))
}

// SetSelectedServer marks the active tray choice and makes it the target of
// the Connect action. A plain-text marker works consistently across the
// native macOS, Linux, and Windows tray implementations.
func SetSelectedServer(id string) {
	mu.Lock()
	defer mu.Unlock()
	selectedID = id
	for serverID, entry := range serverItems {
		label := entry.name
		if serverID == id {
			label += " *"
		}
		entry.item.SetLabel(label)
	}
	if connectItem != nil {
		connectItem.SetEnabled(id != "")
	}
}

// SetConnected keeps both service actions visible and enables only the one
// that is valid for the current state.
func SetConnected(connected bool) {
	mu.Lock()
	defer mu.Unlock()
	serviceConnected = connected
	if statusItem != nil {
		statusItem.SetBitmap(statusBitmap(connected, proxyEnabled))
		statusItem.SetTooltip(statusTooltip(connected, proxyEnabled))
	}
	if connectItem != nil {
		connectItem.SetEnabled(!connected && selectedID != "")
	}
	if disconnectItem != nil {
		disconnectItem.SetEnabled(connected)
	}
	refreshTooltipLocked()
}

// SetProxyEnabled updates the tray's system-proxy indicator.
func SetProxyEnabled(enabled bool) {
	mu.Lock()
	defer mu.Unlock()
	proxyEnabled = enabled
	if statusItem != nil {
		statusItem.SetBitmap(statusBitmap(serviceConnected, enabled))
		statusItem.SetTooltip(statusTooltip(serviceConnected, enabled))
	}
	refreshTooltipLocked()
}
