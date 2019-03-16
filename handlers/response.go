package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"bytes"
)

var httpLineBreak = []byte{ 13, 10 }

func WriteEntity(w http.ResponseWriter, statusCode int, entity interface{}) {
	data, err := json.Marshal(entity)
	if err != nil {
		log.Printf("Fail to marshal entity reponse '%v': %v\n", entity, err)
		WriteError(w, http.StatusInternalServerError,
			errors.New("fail to generate response"))
		return
	}

	WriteResponse(w, statusCode, data)
}

type ErrorResponse struct {
	Message string `json:"message"`
}

func ReadError(resp *http.Response) (*ErrorResponse, error) {
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		return nil, errors.New("response is not json")
	}

	decoder := json.NewDecoder(resp.Body)
	errResp := new(ErrorResponse)
	if err := decoder.Decode(errResp); err != nil {
		return nil, err
	}

	return errResp, nil
}

func WriteError(w http.ResponseWriter, statusCode int, error error) {
	data, err := json.Marshal(&ErrorResponse{error.Error()})
	if err != nil {
		log.Printf("Fail to marshal error reponse '%v': %v\n", error, err)
		WriteResponse(w, http.StatusInternalServerError,
			[]byte("Fail to generate error response"))
		return
	}

	WriteResponse(w, statusCode, data)
}

func WriteResponse(w http.ResponseWriter, statusCode int, data []byte) {

	if ! bytes.HasSuffix(data, httpLineBreak) {
		// append http line break to response
		data = append(data, httpLineBreak...)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	_, err := w.Write(data)
	if err != nil {
		log.Printf("Fail to write response: %v\n", err)
	}
}
