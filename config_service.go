package main

type Config struct {
	ContainerID string
	ImageRef    string
}

type ConfigService interface {
	Get(id string) (*Config, error)
}

type configMap map[string]*Config

func NewConfigService() ConfigService {
	return configMap{
		"helloweb":   &Config{"helloweb", "docker.io/renatofq/helloweb:latest"},
	}
}

func (cMap configMap) Get(id string) (*Config, error) {
	return cMap[id], nil
}
