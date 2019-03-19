package main

type ImageInfo struct {
	ID string
	ImageRef    string
}

type ImageInfoService interface {
	Get(id string) (*ImageInfo, error)
}

type infoMap map[string]*ImageInfo

func NewInfoService() ImageInfoService {
	return infoMap{
		"helloweb":   &ImageInfo{"helloweb", "docker.io/renatofq/helloweb:latest"},
	}
}

func (cMap infoMap) Get(id string) (*ImageInfo, error) {
	return cMap[id], nil
}
