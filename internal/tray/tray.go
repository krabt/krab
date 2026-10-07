// Package tray implements Krab's system tray using Wails v3's native API.
package tray

import (
	"fmt"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

var (
	serviceToggleItem *application.MenuItem
	statusItem        *application.MenuItem
	proxyToggleItem   *application.MenuItem
	systemTray        *application.SystemTray
	serverMenu        *application.Menu
	selectServer      func(string)
	serverItems       = map[string]serverMenuItem{}
	selectedID        string
	serviceConnected  bool
	proxyEnabled      bool
	mu                sync.Mutex
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
	serverMenu = menu.AddSubmenu("选择服务器")
	selectServer = onSelectServer
	rebuildServerMenuLocked(servers)
	menu.AddSeparator()
	serviceToggleItem = menu.Add(serviceToggleLabel(false))
	serviceToggleItem.SetEnabled(initialServerID != "")
	serviceToggleItem.OnClick(func(*application.Context) {
		mu.Lock()
		id := selectedID
		connected := serviceConnected
		mu.Unlock()
		if connected {
			onDisconnect()
		} else if id != "" {
			onConnect(id)
		}
	})
	menu.AddSeparator()
	proxyToggleItem = menu.Add(proxyToggleLabel(initialProxyEnabled))
	proxyToggleItem.OnClick(func(*application.Context) {
		mu.Lock()
		enabled := proxyEnabled
		mu.Unlock()
		if enabled {
			onClearProxy()
		} else {
			onSetProxy()
		}
	})
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

// SetServers rebuilds the server submenu from the current profile store.
// It keeps a valid selection when possible and falls back to the first server.
func SetServers(servers []ServerChoice) {
	mu.Lock()
	defer mu.Unlock()
	if serverMenu == nil {
		return
	}

	selectionExists := false
	for _, server := range servers {
		if server.ID == selectedID {
			selectionExists = true
			break
		}
	}
	if !selectionExists {
		selectedID = ""
		if len(servers) > 0 {
			selectedID = servers[0].ID
		}
	}
	rebuildServerMenuLocked(servers)
	serverMenu.Update()
	if serviceToggleItem != nil {
		serviceToggleItem.SetEnabled(serviceConnected || selectedID != "")
	}
}

func rebuildServerMenuLocked(servers []ServerChoice) {
	serverMenu.Clear()
	serverItems = make(map[string]serverMenuItem, len(servers))
	if len(servers) == 0 {
		serverMenu.Add("暂无服务器").SetEnabled(false)
		return
	}
	for _, server := range servers {
		choice := server
		label := choice.Name
		if choice.ID == selectedID {
			label += " *"
		}
		item := serverMenu.Add(label)
		item.OnClick(func(*application.Context) {
			SetSelectedServer(choice.ID)
			if selectServer != nil {
				selectServer(choice.ID)
			}
		})
		serverItems[choice.ID] = serverMenuItem{name: choice.Name, item: item}
	}
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

func proxyToggleLabel(enabled bool) string {
	if enabled {
		return "取消代理"
	}
	return "设置代理"
}

func serviceToggleLabel(connected bool) string {
	if connected {
		return "断开服务"
	}
	return "连接服务"
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
	if serviceToggleItem != nil {
		serviceToggleItem.SetEnabled(serviceConnected || id != "")
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
	if serviceToggleItem != nil {
		serviceToggleItem.SetLabel(serviceToggleLabel(connected))
		serviceToggleItem.SetEnabled(connected || selectedID != "")
	}
	refreshTooltipLocked()
}

// SetProxyEnabled updates the tray's system-proxy indicator.
func SetProxyEnabled(enabled bool) {
	mu.Lock()
	defer mu.Unlock()
	proxyEnabled = enabled
	if proxyToggleItem != nil {
		proxyToggleItem.SetLabel(proxyToggleLabel(enabled))
	}
	if statusItem != nil {
		statusItem.SetBitmap(statusBitmap(serviceConnected, enabled))
		statusItem.SetTooltip(statusTooltip(serviceConnected, enabled))
	}
	refreshTooltipLocked()
}
