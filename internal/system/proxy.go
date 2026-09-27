package system

// ProxyConfig describes the three system proxy endpoints independently.
type ProxyConfig struct {
	HTTPHost  string `json:"httpHost"`
	HTTPPort  int    `json:"httpPort"`
	HTTPSHost string `json:"httpsHost"`
	HTTPSPort int    `json:"httpsPort"`
	SOCKSHost string `json:"socksHost"`
	SOCKSPort int    `json:"socksPort"`
}

// ProxyStatus is the current operating-system proxy state.
type ProxyStatus struct {
	Enabled bool        `json:"enabled"`
	Config  ProxyConfig `json:"config"`
}
