package server

import (
	"context"
	"log"
	"net/http"
)

type Server interface {
	ListenAndServe() error
	Shutdown(context.Context) error
}

type adapter func(http.Handler) http.Handler

type chain struct {
	adapters []adapter
}

func newChain(adapters ...adapter) chain {
	return chain{append(([]adapter)(nil), adapters...)}
}

func (c chain) then(h http.Handler) http.Handler {
	for i := len(c.adapters) - 1; i >= 0; i-- {
		h = c.adapters[i](h)
	}

	return h
}

func logAdapter() adapter {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Request %s %s %s\n", r.Method, r.URL.String(), r.Proto)
			next.ServeHTTP(w, r)
		})
	}
}

func corsAdapter() adapter {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			next.ServeHTTP(w, r)
		})
	}
}
