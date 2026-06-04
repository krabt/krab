package main

import (
	"fmt"
	"io"
	"krab/config"
	"krab/icon"
	"krab/keeper"
	"log/slog"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/mitchellh/go-homedir"
)

var (
	pm             *keeper.ProcessManager
	svcName        = "xray"
	workSpace      string
	cfgPath        string
	xray           *config.XRay
	cfgDataBinding = binding.NewString()
)

func main() {
	a := app.New()
	w := a.NewWindow("Krab")
	w.Resize(fyne.NewSize(300, 400))

	if desk, ok := a.(desktop.App); ok {
		m := fyne.NewMenu("Krab",
			fyne.NewMenuItem("启动服务", func() {
				// 在后台 goroutine 中执行
				go func() {
					err := pm.StartProcess(svcName)
					if err != nil {
						slog.Error("服务启动失败", slog.Any("error", err))
					}
					desk.SetSystemTrayIcon(icon.LogoGreen)
				}()
			}),
			fyne.NewMenuItem("停止服务", func() {
				// 在后台 goroutine 中执行
				go func() {
					err := pm.StopProcess(svcName)
					if err != nil {
						slog.Error("服务停止失败", slog.Any("error", err))
					}
					desk.SetSystemTrayIcon(icon.LogoActive)
				}()
			}),
			fyne.NewMenuItem("设置系统代理", func() {
				// 在后台 goroutine 中执行
				go func() {
					keeper.SetProxy(xray.GetInboundHttpPort())
				}()
			}),
			fyne.NewMenuItem("取消系统代理", func() {
				// 在后台 goroutine 中执行
				go func() {
					keeper.UnsetProxy()
				}()
			}),
			fyne.NewMenuItem("复制终端命令", func() {
				proxyCMD := fmt.Sprintf("export http_proxy=http://127.0.0.1:%v\nexport https_proxy=http://127.0.0.1:%v", xray.GetInboundHttpPort(), xray.GetInboundHttpPort())
				a.Clipboard().SetContent(proxyCMD)
			}),

			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("打开控制面板", func() { w.Show(); w.RequestFocus() }),
		)
		desk.SetSystemTrayIcon(icon.LogoActive)
		desk.SetSystemTrayMenu(m)
	}

	w.SetContent(container.NewAppTabs(
		container.NewTabItem("Settings", settings(w)),
		container.NewTabItem("Config", setConfig(w)),
	))
	w.SetCloseIntercept(func() {
		w.Hide()
	})
	w.Hide()
	a.Run()

}

func init() {

	workDir, err := homedir.Expand("~/.krab")
	if err != nil {
		fmt.Printf("无法获取用户主目录: %v", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(workDir, 0o755); err != nil {
		fmt.Printf("无法创建日志目录: %v", err)
		os.Exit(1)
	}

	logFilePath := filepath.Join(workDir, "log.txt")
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Printf("无法打开日志文件 %s: %v", logFilePath, err)
	} else {
		logLevel := "debug"
		handler := slog.NewTextHandler(io.MultiWriter(os.Stdout, logFile), &slog.HandlerOptions{
			AddSource: true,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.SourceKey {
					source := a.Value.Any().(*slog.Source)
					source.File = filepath.Base(source.File)
					return slog.Attr{Key: a.Key, Value: a.Value}
				}
				return a
			},
			Level: func() slog.Level {
				switch logLevel {
				case "debug":
					return slog.LevelDebug
				case "info":
					return slog.LevelInfo
				case "warn":
					return slog.LevelWarn
				case "error":
					return slog.LevelError
				default:
					return slog.LevelInfo
				}
			}(),
		})
		slog.SetDefault(slog.New(handler))
	}

	cfgPath = workDir + "/config.json"
	workSpace = workDir
	pm = keeper.NewProcessManager(cfgPath)

	cfg := &keeper.Config{
		Processes: []keeper.ProcessConfig{
			{
				Name:         svcName,
				Command:      workDir + "/xray",
				Args:         []string{"-c", "config.json"},
				WorkDir:      workDir,
				AutoRestart:  true,
				Enabled:      true,
				Environment:  map[string]string{"ENV": "production"},
				User:         "",
				MaxRestarts:  3,
				RestartDelay: 3,
				Description:  "xray",
			},
		},
	}
	pm.SetConfig(cfg)
	pm.SetStatus()
	_ = os.Setenv("FYNE_SCALE", "0.8")
	xray = &config.XRay{}
	xray.ReadCfg(cfgPath)

}

// Settings 设置界面
func settings(w fyne.Window) fyne.CanvasObject {

	port := &widget.Entry{PlaceHolder: "Proxy port", Text: xray.GetInboundHttpPort()}
	portFM := widget.NewForm(widget.NewFormItem("Port", port))

	serverAddr := &widget.Entry{PlaceHolder: "Remote server addr", Text: xray.GetRemoteServerIP()}
	remoteServerFM := widget.NewForm(widget.NewFormItem("Remote Server", serverAddr))

	saveBtn := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), func() {
		editFalg := false
		if port.Text != "" && port.Text != xray.GetInboundHttpPort() {
			proxyPortTmp := port.Text
			xray.ReplaceInboundHttpPort(proxyPortTmp)
			editFalg = true
		}
		if serverAddr.Text != "" && serverAddr.Text != xray.GetRemoteServerIP() {
			serverAddrTmp := serverAddr.Text
			xray.ReplaceRemoteServerIP(serverAddrTmp)
			editFalg = true
		}
		if editFalg {
			xray.SaveCfg(cfgPath)
			cfgDataBinding.Set(xray.GetCfg())
		}
	})

	svcStatus := binding.NewString()
	startBtn := widget.NewButtonWithIcon("Start", theme.MediaSkipNextIcon(), func() {
		// 在后台 goroutine 中执行，避免阻塞 UI
		go func() {
			err := pm.StartProcess(svcName)
			if err != nil {
				slog.Error("服务启动失败", slog.Any("error", err))
			}
			svcStatus.Set(pm.GetProcesseByName(svcName))
		}()
	})

	stopBtn := widget.NewButtonWithIcon("Stop", theme.MediaSkipPreviousIcon(), func() {
		// 在后台 goroutine 中执行，避免阻塞 UI
		go func() {
			err := pm.StopProcess(svcName)
			if err != nil {
				slog.Error("服务停止失败", slog.Any("error", err))
			}
			svcStatus.Set(pm.GetProcesseByName(svcName))
		}()
	})

	return container.NewBorder(nil, nil, nil, nil, container.NewVBox(
		portFM,
		remoteServerFM,
		container.NewCenter(saveBtn),
		container.NewGridWithColumns(2, startBtn, stopBtn),
		layout.NewSpacer(),
		container.NewCenter(widget.NewLabelWithData(svcStatus)),
		layout.NewSpacer(),
	))
}

func setConfig(w fyne.Window) fyne.CanvasObject {
	cfgDataBinding.Set(xray.GetCfg())
	cfgDataEntry := widget.NewMultiLineEntry()
	cfgDataEntry.Bind(cfgDataBinding)
	cfgDataEntry.Validator = nil

	saveBtn := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), func() {
		newCfgStr, _ := cfgDataBinding.Get()
		err := xray.UpdateCfgFromJson(newCfgStr)
		if err != nil {
			slog.Error("更新配置失败", slog.Any("error", err))
			return
		}
		xray.SaveCfg(cfgPath)
	})

	jsonScroll := container.NewScroll(cfgDataEntry)

	return container.NewBorder(nil, container.NewCenter(saveBtn), nil, nil, jsonScroll)
}
