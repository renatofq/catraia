package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"

	"github.com/renatofq/catraia/events"
	"github.com/renatofq/catraia/handlers"
	"github.com/renatofq/catraia/servers"
)

func NewEventServer(name, addr string, store EndpointStore) servers.Server {

	mux := http.NewServeMux()

	chain := handlers.NewChain(handlers.LogAdapter())

	mux.Handle("/container", chain.Then(newEventHandler(store)))

	return servers.NewHTTPServer(name, addr, mux)
}

type eventHandler struct {
	store EndpointStore
}

func newEventHandler(store EndpointStore) http.Handler {
	return &eventHandler{store}
}

func (s *eventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodOptions:
		s.optionsService(w, r)
	case http.MethodPost:
		s.postService(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *eventHandler) optionsService(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", "OPTIONS, POST")
	w.WriteHeader(http.StatusOK)
}

func (s *eventHandler) postService(w http.ResponseWriter, r *http.Request) {

	evt, err := readEvent(r.Body)
	if err != nil {
		log.Printf("Invalid event: %v\n", err)
		handlers.WriteError(w, http.StatusBadRequest,
			errors.New("invalid event"))
		return
	}

	if evt.Type != events.ContainerCreated {
		handlers.WriteEntity(w, http.StatusOK, "Ok")
		return
	}

	addrs, err := setupNetworkIf(evt.Namespace)
	if err != nil {
		log.Printf("Failt to setup network: %v\n", err)
		handlers.WriteError(w, http.StatusInternalServerError,
			errors.New("fail to setup network"))
		return
	}

	if len(addrs) == 0 {
		log.Printf("No network interface were created\n")
		handlers.WriteError(w, http.StatusInternalServerError,
			errors.New("no network interface were created"))
		return
	}

	ep, err := toEndpoint(addrs[0])
	if err != nil {
		log.Printf("Fail to convert addres to endpoint: %v\n", err)
		handlers.WriteError(w, http.StatusInternalServerError,
			errors.New("fail to convert addres to endpoint"))
		return

	}

	s.store.Store(evt.ID, *ep)

	handlers.WriteEntity(w, http.StatusOK, "Network setup ok")
}

func readEvent(r io.Reader) (*events.ContainerEvent, error) {
	var evt events.ContainerEvent
	decoder := json.NewDecoder(r)

	if err := decoder.Decode(&evt); err != nil && err != io.EOF {
		return nil, err
	}

	return &evt, nil
}

func toEndpoint(addr net.IP) (*url.URL, error) {
	epStr := fmt.Sprintf("http://%s:2080/", addr.String())

	log.Printf("addr: %v - %s\n", addr, addr.String())
	ep, err := url.Parse(epStr)
	if err != nil {
		return nil, err
	}

	return ep, nil
}
