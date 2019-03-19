package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

type ImageInfo struct {
	ID  string `json:"-"`
	Ref string `json:ref`
}

type ImageInfoService interface {
	Get(id string) (*ImageInfo, error)
}

type infoMap map[string]*ImageInfo

func NewInfoService(infoFile string) (ImageInfoService, error) {
	file, err := os.Open(infoFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return parseInfoData(file)
}

func parseInfoData(reader io.Reader) (infoMap, error) {
	data := make(infoMap)
	decoder := json.NewDecoder(reader)

	// read openning brace
	if _, err := decoder.Token(); err != nil {
		return nil, parseError(err)
	}

	for decoder.More() {
		var info ImageInfo

		t, err := decoder.Token()
		if err != nil {
			return nil, parseError(err)
		}

		id, ok := t.(string)
		if !ok {
			return nil, parseError(errors.New("id key expected"))
		}

		if err := decoder.Decode(&info); err != nil {
			return nil, parseError(err)
		}

		info.ID = id
		data[id] = &info
	}

	// read closing brace
	if _, err := decoder.Token(); err != nil {
		return nil, parseError(err)
	}

	return data, nil

}

func parseError(err error) error {
	return fmt.Errorf("invalid info data: %v", err)
}

func (cMap infoMap) Get(id string) (*ImageInfo, error) {
	return cMap[id], nil
}
