package server

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/renatofq/catraia/container"

	jsoniter "github.com/json-iterator/go"
)

type serviceHandler struct {
	containerService container.Service
}

func newServiceHandler(containerService container.Service) http.Handler {
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
		writeError(w, http.StatusBadRequest, errors.New("invalid service id"))
		return
	}

	info, err := s.containerService.Info(r.Context(), id)
	if err != nil {
		log.Printf("Fail to get container %s info: %v\n", id, err)
		writeError(w, http.StatusInternalServerError,
			errors.New("fail to get container info"))
		return
	}

	writeEntity(w, http.StatusOK, info)
}

func (s *serviceHandler) putService(w http.ResponseWriter, r *http.Request) {

	id, ok := parseServiceID(r.URL.Path)
	if !ok {
		log.Printf("Invalid service id %s\n", id)
		writeError(w, http.StatusBadRequest, errors.New("invalid service id"))
		return
	}

	if err := s.containerService.Deploy(r.Context(), id); err != nil {
		log.Printf("Fail to deploy service %s: %v\n", id, err)
		writeError(w, http.StatusInternalServerError,
			errors.New("error deploying app"))
		return
	}

	writeEntity(w, http.StatusOK, "service deployed")
}

func (s *serviceHandler) deleteService(w http.ResponseWriter, r *http.Request) {

	id, ok := parseServiceID(r.URL.Path)
	if !ok {
		log.Printf("Invalid service id %s\n", id)
		writeError(w, http.StatusBadRequest, errors.New("invalid service id"))
		return
	}

	if err := s.containerService.Undeploy(r.Context(), id); err != nil {
		log.Printf("Fail to undeploy service %s: %v\n", id, err)
		writeError(w, http.StatusInternalServerError,
			errors.New("fail to undeploying service"))
		return
	}

	writeEntity(w, http.StatusOK, "service undeployed")
}

func NewAPI(addr string, containerService container.Service) Server {

	mux := http.NewServeMux()

	adapterChain := newChain(logAdapter(), corsAdapter())

	mux.Handle("/service/", adapterChain.then(newServiceHandler(containerService)))

	return &http.Server{
		Addr:    addr,
		Handler: mux,
	}
}

func writeEntity(w http.ResponseWriter, statusCode int, entity interface{}) {
	data, err := jsoniter.Marshal(entity)
	if err != nil {
		log.Printf("Fail to marshal entity reponse '%v': %v\n", entity, err)
		writeError(w, http.StatusInternalServerError,
			errors.New("fail to generate response"))
		return
	}

	writeResponse(w, statusCode, data)
}

type errorResponse struct {
	Message string
}

func writeError(w http.ResponseWriter, statusCode int, error error) {
	data, err := jsoniter.Marshal(&errorResponse{error.Error()})
	if err != nil {
		log.Printf("Fail to marshal error reponse '%v': %v\n", error, err)
		writeResponse(w, http.StatusInternalServerError,
			[]byte("Fail to generate error response"))
		return
	}

	writeResponse(w, statusCode, data)
}

func writeResponse(w http.ResponseWriter, statusCode int, data []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	_, err := w.Write(data)
	if err != nil {
		log.Printf("Fail to write response: %v\n", err)
	}
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
