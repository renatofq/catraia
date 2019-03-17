package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/renatofq/catraia/handlers"
	"github.com/renatofq/catraia/servers"
)

func NewProxyServer(name, addr string, store EndpointStore) servers.Server {

	chain := handlers.NewChain(handlers.LogAdapter(), handlers.CORSAdapter())

	reverseProxyHandler := &httputil.ReverseProxy{
		Director: director(store),
	}

	return servers.NewHTTPServer(name, addr, chain.Then(reverseProxyHandler))
}


type directorFunc func(*http.Request)

func director(store EndpointStore) directorFunc {
	return func(r *http.Request) {
		path := r.URL.Path
		id, targetPath := splitTargetPath(path)
		targetHost, err := store.Load(id)
		if err != nil {
			// do something to abort
			log.Printf("Fail to get target from store: %v\n", err)
			r.URL = nil
		}

		log.Printf("Proxy to: %s\n", targetHost.String()+targetPath)

		target, err := url.Parse(targetHost.String() + targetPath)
		if err != nil {
			log.Printf("Fail mount target url: %v\n", err)
			r.URL = nil
		}

		r.URL = target
	}
}

func splitTargetPath(path string) (string, string) {
	result := strings.SplitN(path, "/", 3)

	if len(result) == 1 {
		return "", ""
	}

	if len(result) == 2 {
		return result[1], ""
	}

	return result[1], result[2]
}
