package handlers

import (
	"log"
	"net/http"
)

type Adapter func(http.Handler) http.Handler

type Chain struct {
	adapters []Adapter
}

func NewChain(adapters ...Adapter) Chain {
	return Chain{append(([]Adapter)(nil), adapters...)}
}

func (c Chain) Then(h http.Handler) http.Handler {
	for i := len(c.adapters) - 1; i >= 0; i-- {
		h = c.adapters[i](h)
	}

	return h
}

func LogAdapter() Adapter {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Request %s %s %s\n", r.Method, r.URL.String(), r.Proto)
			next.ServeHTTP(w, r)
		})
	}
}

func CORSAdapter() Adapter {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			next.ServeHTTP(w, r)
		})
	}
}
