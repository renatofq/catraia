package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/renatofq/catraia/handlers"
	"github.com/renatofq/catraia/servers"
)

func NewAPIServer(name, addr string, ctrService ContainerService) servers.Server {

	mux := http.NewServeMux()

	chain := handlers.NewChain(handlers.LogAdapter(), handlers.CORSAdapter())

	mux.Handle("/service/", chain.Then(newServiceHandler(ctrService)))

	return servers.NewHTTPServer(name, addr, mux)
}

type serviceHandler struct {
	containerService ContainerService
}

func newServiceHandler(containerService ContainerService) http.Handler {
	return &serviceHandler{containerService}
}

func (s *serviceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		s.optionsService(w, r)
	case "GET":
		s.getService(w, r)
	case "PUT":
		s.putService(w, r)
	case "DELETE":
		s.deleteService(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *serviceHandler) optionsService(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", "OPTIONS, GET, PUT, DELETE")
	w.WriteHeader(http.StatusOK)
}

func (s *serviceHandler) getService(w http.ResponseWriter, r *http.Request) {

	id, ok := parseServiceID(r.URL.Path)
	if !ok {
		log.Printf("Invalid service id %s\n", id)
		handlers.WriteError(w, http.StatusBadRequest, errors.New("invalid service id"))
		return
	}

	info, err := s.containerService.Info(r.Context(), id)
	if err != nil {
		log.Printf("Fail to get container %s info: %v\n", id, err)
		handlers.WriteError(w, http.StatusInternalServerError,
			errors.New("fail to get container info"))
		return
	}

	handlers.WriteEntity(w, http.StatusOK, info)
}

func (s *serviceHandler) putService(w http.ResponseWriter, r *http.Request) {

	id, ok := parseServiceID(r.URL.Path)
	if !ok {
		log.Printf("Invalid service id %s\n", id)
		handlers.WriteError(w, http.StatusBadRequest, errors.New("invalid service id"))
		return
	}

	if err := s.containerService.Deploy(r.Context(), id); err != nil {
		log.Printf("Fail to deploy service %s: %v\n", id, err)
		handlers.WriteError(w, http.StatusInternalServerError,
			errors.New("error deploying app"))
		return
	}

	handlers.WriteEntity(w, http.StatusOK, "service deployed")
}

func (s *serviceHandler) deleteService(w http.ResponseWriter, r *http.Request) {

	id, ok := parseServiceID(r.URL.Path)
	if !ok {
		log.Printf("Invalid service id %s\n", id)
		handlers.WriteError(w, http.StatusBadRequest, errors.New("invalid service id"))
		return
	}

	if err := s.containerService.Undeploy(r.Context(), id); err != nil {
		log.Printf("Fail to undeploy service %s: %v\n", id, err)
		handlers.WriteError(w, http.StatusInternalServerError,
			errors.New("fail to undeploying service"))
		return
	}

	handlers.WriteEntity(w, http.StatusOK, "service undeployed")
}

func parseServiceID(path string) (string, bool) {
	var id string

	fmt.Sscanf(path, "/service/%s", &id)

	if len(id) == 0 {
		return "", false
	}

	if strings.ContainsRune(id, '/') {
		return "", false
	}

	return id, true
}
