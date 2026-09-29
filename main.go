package main

import (
	"embed"
	"log"
	"os"
	"runtime"
	"slices"
	"time"

	"github.com/krabt/krab/internal/tray"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var trayIconPNG []byte

const relaunchWaitFlag = "--krab-relaunch-wait"

var singleInstanceKey = [32]byte{
	0x6b, 0x69, 0x74, 0x65, 0xa1, 0x5e, 0x9f, 0x0a,
	0x9e, 0x3b, 0x4a, 0x2e, 0x8c, 0x7c, 0x3a, 0x6b,
	0x2b, 0x3f, 0x6a, 0x41, 0x6b, 0x69, 0x74, 0x65,
	0xa1, 0x5e, 0x9f, 0x0a, 0x9e, 0x3b, 0x4a, 0x2e,
}

func init() {
	application.RegisterEvent[map[string]int64]("update:progress")
	application.RegisterEvent[string]("profile:selected")
}

func main() {
	if slices.Contains(os.Args[1:], relaunchWaitFlag) {
		time.Sleep(3 * time.Second)
	}

	backend := NewApp()
	var window *application.WebviewWindow

	app := application.New(application.Options{
		Name:        "Krab",
		Description: "A modern Xray-core client",
		Icon:        trayIconPNG,
		Services: []application.Service{
			application.NewService(backend),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		OnShutdown: backend.shutdown,
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID:      "krab-a15e9f0a-9e3b-4a2e-8c7c-3a6b2b3f6a41",
			EncryptionKey: singleInstanceKey,
			OnSecondInstanceLaunch: func(_ application.SecondInstanceData) {
				if window != nil {
					window.Show()
					window.Restore()
					window.Focus()
				}
			},
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
			ActivationPolicy: application.ActivationPolicyAccessory,
		},
	})

	window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:                      "Krab",
		Width:                      800,
		Height:                     500,
		URL:                        "/",
		BackgroundColour:           application.NewRGB(28, 28, 30),
		DefaultContextMenuDisabled: false,
		Mac: application.MacWindow{
			Appearance: application.NSAppearanceNameDarkAqua,
			TitleBar: application.MacTitleBar{
				AppearsTransparent: true,
			},
		},
		Windows: application.WindowsWindow{
			Theme: application.Dark,
		},
	})

	backend.startup(app, window)
	profiles, _ := backend.ListProfiles()
	choices := make([]tray.ServerChoice, 0, len(profiles))
	for _, server := range profiles {
		choices = append(choices, tray.ServerChoice{ID: server.ID, Name: server.Name})
	}
	selectedServerID := ""
	if settings := backend.GeoSettings(); settings.UI != nil {
		selectedServerID, _ = settings.UI["selectedServerId"].(string)
	}
	selectedServerExists := false
	for _, choice := range choices {
		if choice.ID == selectedServerID {
			selectedServerExists = true
			break
		}
	}
	if !selectedServerExists && len(choices) > 0 {
		selectedServerID = choices[0].ID
	}
	proxyStatus, _ := backend.SystemProxyStatus()
	backend.systemTray = tray.SetupSystemTray(app, trayIconPNG, choices, selectedServerID, proxyStatus.Enabled,
		func() {
			window.Show()
			window.Restore()
			window.Focus()
		},
		func(id string) { _ = backend.Connect(id, "proxy", false) },
		func() { _ = backend.Disconnect() },
		func() { _ = backend.SetSystemProxy(backend.currentProxyConfig()) },
		func() { _ = backend.ClearSystemProxy() },
		func() { app.Clipboard.SetText(backend.terminalProxyCommand()) },
		func(id string) { app.Event.Emit("profile:selected", id); window.Show(); window.Focus() },
		func() {
			backend.quitting = true
			app.Quit()
		},
	)

	menu := app.NewMenu()
	if runtime.GOOS == "darwin" {
		menu.AddRole(application.AppMenu)
	}
	menu.AddRole(application.FileMenu)
	menu.AddRole(application.EditMenu)
	menu.AddRole(application.ViewMenu)
	menu.AddRole(application.WindowMenu)
	window.SetMenu(menu)

	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if !backend.quitting {
			event.Cancel()
			window.Hide()
		}
	})

	if err := app.Run(); err != nil {
		log.Printf("Krab exited with an error: %v", err)
	}
}
