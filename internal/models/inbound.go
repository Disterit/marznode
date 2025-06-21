package models

type Inbound struct {
	Tag      string
	Protocol string
	Config   hysteriaConfig
}

type hysteriaConfig struct {
	ApiPort      int
	StatsPort    int
	StatsSecret  string
	Port         int
	TLS          string
	LoadedConfig loadedConfig
}

type loadedConfig struct {
	Auth         auth
	TrafficStats trafficStats
}

type auth struct {
	PortType string // http
	HttpUrl  string //http://127.0.0.1: + ApiPort
}

type trafficStats struct {
	Listen string
	Secret string
}
