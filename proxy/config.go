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
}

type Frontend struct {
	Listen string `json:"listen"`
}

type Upstream struct {
	Name    string `json:"name"`
	Url     string `json:"url"`
	Timeout string `json:"timeout"`
}
