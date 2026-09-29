<div align="center">
  <img src=".github/logo.png" alt="Krab" width="120" />

  # Krab

  **跨平台桌面版 V2Ray / 代理客户端。**
  Wails 3 + Go 后端，React/Tailwind 前端，xray-core 作为 Go 库嵌入。

  Fork 自 [freeb5d/kite](https://github.com/freeb5d/kite)。

  [![Release](https://img.shields.io/github/v/release/krabt/krab?label=release&color=6366f1)](https://github.com/krabt/krab/releases/latest)
  [![Build](https://img.shields.io/github/actions/workflow/status/krabt/krab/release.yml?label=build)](https://github.com/krabt/krab/actions/workflows/release.yml)
  [![Platforms](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-6366f1)](#下载)
  [![Go Report Card](https://goreportcard.com/badge/github.com/krabt/krab)](https://goreportcard.com/report/github.com/krabt/krab)
  [![License: MIT](https://img.shields.io/badge/license-MIT-6366f1)](LICENSE)

  [下载](#下载) · [功能](#功能) · [从源码构建](#从源码构建) · [架构](#架构)

  [English](README.md) · 中文
</div>

<p align="center">
  <img src=".github/screenshots/light.png" alt="Krab, light mode" width="49%" />
  <img src=".github/screenshots/dark.png" alt="Krab, dark mode" width="49%" />
</p>

---

## 下载

从 **[发布页面](https://github.com/krabt/krab/releases/latest)** 获取最新版本：

| 平台 | 下载文件 |
| --- | --- |
| Windows (x64) | `krab-windows-amd64.exe` |
| Linux (x64) | `krab-linux-amd64` |
| macOS (Apple Silicon) | `krab-macos-arm64` |

每个平台都是单个便携可执行文件——无需安装程序，无需管理员权限（macOS 版本是从 Wails 3 生成的 `.app` 包中提取出的原始二进制文件；关于这带来的 Gatekeeper 提示以及如何绕过它，见[已知限制](#已知限制--后续计划)）。安装后，程序会在启动时检查新版本，一键即可自我更新。

## 功能

- **链接格式**：`vmess://`、`vless://`、`trojan://`、`ss://`, `hysteria2://`, `ssh://` —— 粘贴分享链接即可解析并保存
- **订阅链接** —— 支持 V2RayN/V2RayNG/Shadowrocket 的 base64 链接列表、Clash/Mihomo YAML、sing-box JSON、Xray JSON 和 Shadowsocks SIP008。订阅按分组展示套餐、流量和到期信息，并支持同步、分享、编辑、删除及按服务商间隔自动刷新
- **分享服务器** —— 可复制标准的 `vless://`、`vmess://`、`trojan://`、`ss://`、`hysteria2://` 或 `ssh://` 链接
- **xray-core**，作为 Go 库嵌入（而非外部调用的二进制文件）——完整的生命周期控制和真实的流量统计，无需解析标准输出
- **传输方式**：TCP（包括 `headerType=http`）、WebSocket、gRPC、mKCP、HTTPUpgrade、XHTTP 和 HTTP/2，并支持 TLS、REALITY、ECH 与证书固定
- **出站路由规则** —— 可为每个服务器关联独立的黑名单、代理和白名单规则组，支持域名、IP/CIDR、`geosite:` 和 `geoip:`；优先级固定为黑名单、代理、白名单
- **系统代理集成** —— 可分别配置 HTTP、HTTPS 与 SOCKS5 地址，并显式设置或清除操作系统代理；支持一键复制终端代理环境变量
- **GeoData 与 DNS 设置** —— 可在界面中管理 `geoip.dat`、`geosite.dat`、Hosts 映射和上游 DNS
- **TUN 模式（Windows）** —— 通过虚拟网络适配器（xray-core 的 TUN，配合内置的 WinTun 驱动）路由所有系统流量，而不仅仅是遵循代理设置的应用。支持所有服务器类型，包括 `headerType=http`，DNS 也经由隧道。需要管理员权限；Krab 可以一键以提升权限的方式重启自身
- **Kill Switch（仅限 Windows）** —— 连接期间通过一对 Windows 防火墙规则阻止除 Krab 自身外的所有出站流量，这样即使某个应用忽略系统代理（或引擎进程崩溃），也无法在隧道之外泄漏流量。与 TUN 模式有相同的管理员权限要求
- **系统托盘** —— 关闭窗口会将其隐藏到托盘而不是退出程序，因此活动连接会继续保持；托盘菜单包含“显示 Krab”、“断开连接”和“退出 Krab”
- **开机自动启动** —— 使用 macOS、Linux 和 Windows 的原生机制注册登录后自动启动
- **内置诊断** —— “测试”按钮会通过隧道发起一次真实请求并报告实际结果；日志查看器可内联显示 xray-core 自身的调试日志
- **流量历史** —— 显示实时上传/下载速度，并按日期和服务器持久化累计流量
- **自我更新** —— 启动时检查 GitHub Releases，一键下载、替换并重启（“关于”面板显示 Krab 和内置 xray-core 的版本）
- **深色/浅色主题**，配有可搜索的服务器列表、内联重命名和一键移除
- **2 种语言**，可在侧边栏切换：
  - 🇬🇧 English（默认）
  - 🇨🇳 中文
- **SQLite 持久化** —— 服务器、应用设置、出站规则和流量历史统一保存在 `~/.krab/krab.db`

## 从源码构建

### 一次性工具链准备

1. **Go 1.27+** —— https://go.dev/dl/ 。Windows 上还需要一个支持 CGO 的 C 编译器（Wails 3 及其部分依赖会用到）——安装 [TDM-GCC](https://jmeubank.github.io/tdm-gcc/) 或 `winget install -e --id GoLang.Go` 加上 MSYS2 的 `mingw-w64-x86_64-gcc`。
2. **Node.js（LTS）** —— https://nodejs.org ，或 `winget install OpenJS.NodeJS.LTS`。
3. **Wails 3 CLI**：
   ```bash
   go install github.com/wailsapp/wails/v3/cmd/wails3@latest
   wails3 doctor
   ```

### 运行

```bash
go mod tidy
corepack enable
cd frontend && pnpm install && cd ..
wails3 dev
```

### 构建发布版二进制文件

```bash
wails3 build
```

## 架构

```
krab/
├── main.go / app.go        Wails 3 入口与提供给前端调用的 Go 方法
├── internal/
│   ├── database/             位于 ~/.krab/krab.db 的 SQLite 存储
│   ├── xray/                 xray-core 生命周期、配置、流量统计及 TUN
│   ├── system/               各平台系统代理、开机启动及 Windows TUN 支持
│   │   ├── proxy_windows.go   HKCU Internet Settings（无需管理员权限）
│   │   ├── proxy_darwin.go    networksetup
│   │   ├── proxy_linux.go     gsettings（GNOME）
│   │   ├── autostart_*.go     各平台原生开机启动注册
│   │   ├── elevate_windows.go Windows 管理员权限检测与申请
│   │   ├── route_windows.go   Windows TUN 上游例外路由
│   │   └── wintun/            内置 Wintun 驱动
│   └── update/                检查、下载、替换并重启应用
├── pkg/
│   ├── profile/               配置模型、链接/订阅解析与 SQLite 存储
│   ├── xrayconf/              共用的 xray-core 配置生成器
│   ├── certpin/               TLS 证书固定
│   ├── probe/                 TCP、HTTP 与真实延迟检测
│   └── sshbridge/             本地 SOCKS 到 SSH 桥接
├── frontend/                 Vite + React + Tailwind 前端
└── .github/workflows/        CI 构建与 GitHub Release 发布
```

`App`（见 [app.go](app.go)）上每一个绑定的 Go 方法，会由 `wails3 generate bindings` 生成为 `frontend/bindings/github.com/krabt/krab/app.js` 中可供前端调用的函数。

## 已知限制 / 后续计划

- **macOS 仅支持 Apple Silicon（arm64）** —— 没有 Intel 构建版本。如有需要，可在 `.github/workflows/release.yml` 中添加与 `darwin/arm64` 并列的 `darwin/amd64` 矩阵项。
- **TUN 模式目前仅限 Windows** 且仅支持 IPv4 —— 用于防止引擎自身上游连接循环回 TUN 适配器的例外路由（`internal/system/route_windows.go`）目前只覆盖已解析的 IPv4 地址；只能通过 IPv6 访问的服务器暂时无法在 TUN 模式下使用。Linux/macOS 的 TUN 支持是可行的（xray-core 自带的 `tun` 包已支持两者），只是尚未接入。
- **Linux 系统代理仅支持 GNOME**（`gsettings`）—— 其他桌面环境需要在 `internal/system/proxy_linux.go` 中实现各自的后端。
- **Kill Switch 仅限 Windows** —— 通过 `netsh advfirewall` 规则实现（`internal/system/killswitch_windows.go`）；Linux/macOS 需要各自的后端（`iptables`/`pfctl`），目前尚未实现（该开关在这些平台上被隐藏，与 TUN 模式相同）。
- **原生最小化按钮仍然只是最小化到任务栏** —— 关闭窗口会隐藏到托盘，最小化则保持操作系统的默认行为。
- **没有代码签名** —— Windows SmartScreen 和 macOS Gatekeeper 都会对未签名的二进制文件发出警告；目前属于预期情况。在 macOS 上，首次运行下载的二进制文件需要执行 `xattr -d com.apple.quarantine krab-macos-arm64`（或右键 → 打开）来绕过 Gatekeeper，因为它没有经过公证。在 Windows 上，杀毒软件（包括 Defender）有时会在 Krab 将 `wintun.dll` 写入自身旁边后立即将其隔离，因为这类与内核相关的网络 DLL 很容易被启发式检测标记——WireGuard、v2rayN 等基于 Wintun 的应用也会遇到同样的问题。如果 TUN 模式因类似“找不到文件”的错误而失败，请将 Krab 所在文件夹加入杀毒软件的排除列表。

## 贡献

欢迎提交 Issue 和 PR。实际待办事项见上方[已知限制](#已知限制--后续计划)。

## 许可证

[MIT](LICENSE)
