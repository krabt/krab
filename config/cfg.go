package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/tailscale/hujson"
)

type XrayCfg struct {
	Log       Log         `json:"log"`
	Inbounds  []Inbounds  `json:"inbounds"`
	Outbounds []Outbounds `json:"outbounds"`
	Routing   Routing     `json:"routing"`
}
type Log struct {
}
type Settings struct {
	Auth string `json:"auth"`
	UDP  bool   `json:"udp"`
}

type Inbounds struct {
	Port     string    `json:"port"`
	Protocol string    `json:"protocol"`
	Settings *Settings `json:"settings,omitempty"`
	Tag      string    `json:"tag"`
}
type Users struct {
	ID         string `json:"id"`
	Encryption string `json:"encryption"`
}
type Vnext struct {
	Address string  `json:"address"`
	Port    int     `json:"port"`
	Users   []Users `json:"users"`
}
type OutSettings struct {
	Vnext []Vnext `json:"vnext"`
}
type XhttpSettings struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
}
type TLSSettings struct {
	AllowInsecure           bool `json:"allowInsecure"`
	VerifyClientCertificate bool `json:"verifyClientCertificate"`
}
type StreamSettings struct {
	Network       string         `json:"network"`
	XhttpSettings *XhttpSettings `json:"xhttpSettings"`
	Security      string         `json:"security"`
	TLSSettings   TLSSettings    `json:"tlsSettings"`
}
type Settings1 struct {
}
type Outbounds struct {
	Tag            string          `json:"tag"`
	Protocol       string          `json:"protocol"`
	Settings       *OutSettings    `json:"settings,omitempty"`
	StreamSettings *StreamSettings `json:"streamSettings,omitempty"`
}
type Rules struct {
	Type        string   `json:"type"`
	OutboundTag string   `json:"outboundTag"`
	Domain      []string `json:"domain,omitempty"`
	IP          []string `json:"ip,omitempty"`
}
type Routing struct {
	DomainStrategy string  `json:"domainStrategy"`
	Rules          []Rules `json:"rules"`
}

type XRay struct {
	cfg *XrayCfg
}

func NewXRay() *XRay {
	return &XRay{
		cfg: &XrayCfg{},
	}
}

func GetDefaultCfg() *XrayCfg {
	cfg := &XrayCfg{
		Log: Log{},
		Inbounds: []Inbounds{
			{
				Port:     "5888",
				Protocol: "socks",
				Settings: &Settings{
					Auth: "noauth",
					UDP:  true,
				},
				Tag: "socks-in",
			},
			{
				Port:     "5889",
				Protocol: "http",
				Tag:      "http-in",
			},
		},
		Outbounds: []Outbounds{
			{
				Tag:      "direct",
				Protocol: "freedom",
			},
			{
				Tag:      "blocked",
				Protocol: "blackhole",
			},
			{
				Tag:      "proxy",
				Protocol: "vless",
				Settings: &OutSettings{
					Vnext: []Vnext{
						{
							Address: "sg.local.io",
							Port:    443,
							Users: []Users{
								{
									ID:         "A00546EF-B224-473B-A0EF-E0AE4534AB46",
									Encryption: "none",
								},
							},
						},
					},
				},
				StreamSettings: &StreamSettings{
					Network: "xhttp",
					XhttpSettings: &XhttpSettings{
						Path: "pznhckhsiyxrtkkkkkk",
						Mode: "auto",
					},
					Security: "tls",
					TLSSettings: TLSSettings{
						AllowInsecure:           true,
						VerifyClientCertificate: false,
					},
				},
			},
		},
		Routing: Routing{
			DomainStrategy: "IPOnDemand",
			Rules: []Rules{
				{
					Type:        "field",
					OutboundTag: "blocked",
					Domain: []string{
						"geosite:sogou",
						"geosite:wps",
						"domain:dc.services.visualstudio.com",
						"domain:jianshu.com",
						"geosite:category-ads",
					},
				},
				{
					Type:        "field",
					OutboundTag: "proxy",
					Domain: []string{
						"geosite:github",
						"geosite:docker",
						"geosite:openai",
						"domain:googlecccccccccc.com",
						"domain:go.dev",
					},
				},
				{
					Type:        "field",
					OutboundTag: "direct",
					Domain: []string{
						"domain:gorm.io",
						"domain:jetbrains.com.cn",
						"domain:jetbrains.com",
						"domain:go.dev",
						"domain:kubernetes.docker.internal",
						"domain:macwk.com",
						"geosite:cn",
						"geosite:google-trust-services",
						"geosite:digicert",
						"geosite:letsencrypt",
						"geosite:mozilla",
						"geosite:apple",
						"geosite:private",
					},
					IP: []string{
						"geoip:cn",
						"geoip:private",
						"223.5.5.5",
						"223.6.6.6",
						"10.2.0.0/16",
						"10.1.0.0/16",
					},
				},
			},
		},
	}

	return cfg
}

func (x *XRay) ToJson() ([]byte, error) {
	return json.MarshalIndent(x.cfg, "", "  ")
}

func (x *XRay) GetRemoteServerIP() (serverIP string) {
	for _, outbound := range x.cfg.Outbounds {
		if outbound.Tag == "proxy" {
			if len(outbound.Settings.Vnext) > 0 {
				serverIP = outbound.Settings.Vnext[0].Address
				return serverIP
			}
		}
	}
	return
}

func (x *XRay) ReplaceRemoteServerIP(serverIP string) (err error) {
	for idx, outbound := range x.cfg.Outbounds {
		if outbound.Tag == "proxy" {
			if len(outbound.Settings.Vnext) > 0 {
				x.cfg.Outbounds[idx].Settings.Vnext[0].Address = serverIP
			}
		}
	}
	return
}

func (x *XRay) ReplaceInboundHttpPort(port string) (err error) {
	for idx, inbound := range x.cfg.Inbounds {
		if inbound.Tag == "http-in" {
			x.cfg.Inbounds[idx].Port = port
		}
	}
	return
}

func (x *XRay) GetInboundHttpPort() (port string) {
	for idx, inbound := range x.cfg.Inbounds {
		if inbound.Tag == "http-in" {
			port = x.cfg.Inbounds[idx].Port
		}
	}
	return
}

func (x *XRay) ReadCfg(cfgPath string) (err error) {
	fData, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("文件不存在: %v", cfgPath)
		}
		err = os.WriteFile(cfgPath, []byte(""), 0644)
		if err != nil {
			fmt.Println("创建文件失败:", err)
			return
		}
		fmt.Printf("文件创建成功: %v", cfgPath)
		x.cfg = GetDefaultCfg()
		x.SaveCfg(cfgPath)
		return nil
	}
	cfgData := &XrayCfg{}
	// 处理带注释的json
	ast, err := hujson.Parse(fData)
	if err != nil {
		log.Fatalf("解析注释失败: %v", err)
	}
	ast.Standardize()
	standardJSON := ast.Pack()

	err = json.Unmarshal(standardJSON, cfgData)
	if err != nil {
		fmt.Printf("解析JSON失败: %v", err)
		return nil
	}
	x.cfg = cfgData
	return nil

}

func (x *XRay) SaveCfg(cfgPath string) (err error) {

	cfgData, err := x.ToJson()
	if err != nil {
		fmt.Printf("配置转换为JSON失败: %v", err)
		return
	}
	err = os.WriteFile(cfgPath, cfgData, 0655)
	if err != nil {
		fmt.Printf("写入配置文件失败: %v", err)
		return
	}
	return
}

func (x *XRay) GetCfg() string {
	data, err := x.ToJson()
	if err != nil {
		fmt.Printf("配置转换为JSON失败: %v", err)
		return ""
	}
	return string(data)
}

func (x *XRay) UpdateCfgFromJson(jsonData string) (err error) {
	cfgData := &XrayCfg{}
	err = json.Unmarshal([]byte(jsonData), cfgData)
	if err != nil {
		fmt.Printf("解析JSON失败: %v", err)
		return nil
	}
	x.cfg = cfgData
	return nil
}
