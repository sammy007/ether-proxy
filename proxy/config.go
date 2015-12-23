package proxy

type Config struct {
	Proxy                 Proxy      `json:"proxy"`
	Frontend              Frontend   `json:"frontend"`
	Upstream              []Upstream `json:"upstream"`
	UpstreamCheckInterval string     `json:"upstreamCheckInterval"`

	Threads int `json:"threads"`

	NewrelicName    string `json:"newrelicName"`
	NewrelicKey     string `json:"newrelicKey"`
	NewrelicVerbose bool   `json:"newrelicVerbose"`
	NewrelicEnabled bool   `json:"newrelicEnabled"`
}

type Proxy struct {
	Listen               string `json:"listen"`
	ClientTimeout        string `json:"clientTimeout"`
	BlockRefreshInterval string `json:"blockRefreshInterval"`
	HashrateWindow       string `json:"hashrateWindow"`
	SubmitHashrate       bool   `json:"submitHashrate"`
	LuckWindow           string `json:"luckWindow"`
	LargeLuckWindow      string `json:"largeLuckWindow"`
}

type Frontend struct {
	Listen   string `json:"listen"`
	Login    string `json:"login"`
	Password string `json:"password"`
}

type Upstream struct {
	Name    string `json:"name"`
	Url     string `json:"url"`
	Timeout string `json:"timeout"`
	Pool    bool   `json:"pool"`
}
