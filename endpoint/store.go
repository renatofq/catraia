package endpoint

import (
	"errors"
	"net/url"
	"sync"
)

type Store interface {
	Load(id string) (url.URL, error)
	Store(id string, endpoint url.URL) error
	Delete(id string) error
}

type endpointMap struct {
	m sync.Map
}

func NewStore() Store {
	return &endpointMap{}
}

func (em *endpointMap) Load(id string) (url.URL, error) {
	value, _ := em.m.Load(id)
	if value == nil {
		return url.URL{}, errors.New("endpoint not found")
	}

	strValue := value.(url.URL)
	return strValue, nil
}

func (em *endpointMap) Store(id string, endpoint url.URL) error {
	em.m.Store(id, endpoint)
	return nil
}

func (em *endpointMap) Delete(id string) error {
	em.m.Delete(id)
	return nil
}
