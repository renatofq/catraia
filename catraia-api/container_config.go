package main

type ContainerConfig struct {
	ContainerID string
	ImageRef    string
}

type ContainerConfigService interface {
	Get(id string) (*ContainerConfig, error)
}

type configMap map[string]*ContainerConfig

func NewConfigService() ContainerConfigService {
	return configMap{
		"helloweb":   &ContainerConfig{"helloweb", "docker.io/renatofq/helloweb:latest"},
	}
}

func (cMap configMap) Get(id string) (*ContainerConfig, error) {
	return cMap[id], nil
}
