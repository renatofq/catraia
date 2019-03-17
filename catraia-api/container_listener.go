package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/renatofq/catraia/events"
	"github.com/renatofq/catraia/handlers"
	"github.com/renatofq/catraia/utils"
)

type containerListener struct {
	client http.Client
}

func NewContainerListener(address string) CreationListener {
	return &containerListener{
		client: http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial(utils.NetTypeFromAddr(address), address)
				},
			},
		},
	}
}

func (cl *containerListener) Created(id string, pid uint32) {

	event := events.ContainerEvent{
		Type:      events.ContainerCreated,
		ID:        id,
		Namespace: getNetns(pid),
	}

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Fail to generate json for event %v: %v\n", event, err)
		return
	}

	resp, err := cl.client.Post("http://unix/container", "application/json",
		bytes.NewReader(data))
	if err != nil {
		log.Printf("Event server returned error for event %s: %v\n", data, err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		errResp, err := handlers.ReadError(resp)
		if err != nil {
			log.Printf("ErrorResponse from EventServer is invalid: %v\n", err)
			return
		}

		log.Printf("Fail to notify container creation. status %s: %s\n",
			resp.Status, errResp.Message)
	}
}

func getNetns(pid uint32) string {
	return fmt.Sprintf("/proc/%d/ns/net", pid)
}
