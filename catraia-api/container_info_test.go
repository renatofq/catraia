package main

import (
	"testing"
	"strings"
)

func TestParseInfo(t *testing.T) {
	jsonData := `
        {
          "helloweb" : {
            "ref" : "docker.io/renatofq/helloweb:latest"
          },
          "helloworld" : {
            "ref" : "docker.io/renatofq/helloworld:latest"
          }
        }
`

	expected := infoMap{
		"helloweb": &ImageInfo{
			ID: "helloweb",
			Ref: "docker.io/renatofq/helloweb:latest",
		},
		"helloworld": &ImageInfo{
			ID: "helloworld",
			Ref: "docker.io/renatofq/helloworld:latest",
		},
	}

	result, err := parseInfoData(strings.NewReader(jsonData))
	if err != nil {
		t.Error(err)
	}

	for k, v := range expected {
		if *result[k] != *v {
			t.Errorf("at %s want %q got %q\n", k, v, result[k])
		}
	}

}
