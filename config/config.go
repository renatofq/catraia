package config

import "os"

type Config struct {
	RuntimeDir          string
	APIServerAddr       string
	NetServerAddr       string
	TunnelAddr          string
	ProxyAddr           string
	Bridge              string
	ContainerdNamespace string
	ContainerdSocket    string
}

func New() *Config {
	return &Config{
		RuntimeDir:          getEnv("CATRAIA_RUNTIME_DIR", "/run/catraia"),
		APIServerAddr:       getEnv("CATRAIA_API_SERVER_ADDR", "/run/catraia/event.sock"),
		NetServerAddr:       getEnv("CATRAIA_NET_SERVER_ADDR", "/run/catraia/event.sock"),
		TunnelAddr:          getEnv("CATRAIA_TUNNEL_ADDR", ":2020"),
		ProxyAddr:           getEnv("CATRAIA_PROXY_ADDR", "/run/catraia/proxy.sock"),
		Bridge:              getEnv("CATRAIA_BRIDGE", "catraia0"),
		ContainerdNamespace: getEnv("CATRAIA_CONTAINERD_NAMESPACE", "default"),
		ContainerdSocket:    getEnv("CATRAIA_CONTAINERD_SOCKET", "/run/containerd/containerd.sock"),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultValue
}
