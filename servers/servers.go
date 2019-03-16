package servers

import (
	"context"
	"log"
	"net"
	"net/http"
	"time"
)

type Server interface {
	Name() string
	ListenAndServe() error
	Shutdown(context.Context) error
}

func Run(s Server) {
	err := s.ListenAndServe()

	log.Printf("%s Server shutdown: %v\n", s.Name(), err)
}

func Shutdown(ctx context.Context, s Server) {

	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := s.Shutdown(timeoutCtx); err != nil {
		log.Printf("Error during %s Server shutdown: %v\n", s.Name(), err)
	}
}

type httpServer struct {
	name       string
	listenNet  string
	listenAddr string
	httpServer *http.Server
}

func NewHTTPServer(name, listenNet, listenAddr string, handler http.Handler) Server {

	server := &http.Server{
		Handler: handler,
	}

	return &httpServer{
		name:       name,
		listenNet:  listenNet,
		listenAddr: listenAddr,
		httpServer: server,
	}
}

func (hs *httpServer) Name() string {
	return hs.name
}

func (hs *httpServer) ListenAndServe() error {
	l, err := net.Listen(hs.listenNet, hs.listenAddr)
	if err != nil {
		return err
	}

	return hs.httpServer.Serve(l)
}

func (hs *httpServer) Shutdown(ctx context.Context) error {
	return hs.httpServer.Shutdown(ctx)
}

