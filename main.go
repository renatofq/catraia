package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/renatofq/catraia/config"
	"github.com/renatofq/catraia/container"
	"github.com/renatofq/catraia/endpoint"
	"github.com/renatofq/catraia/server"
)

func setupSignalHandling(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(context.Background())

	// we may loose signals with a buffer of 1. But, as we quit after receiving
	// any of the handled signals it makes no difference
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Signal(syscall.SIGINT),
		os.Signal(syscall.SIGTERM))
	go func() {
		defer cancel()
		sig := <-sigChan
		log.Printf("canceled by signal: %v\n", sig)
	}()

	return ctx
}

func runServer(s server.Server, alias string) {
	err := s.ListenAndServe()

	log.Printf("%s Server shutdown: %v\n", alias, err)
}

func shutdownServer(ctx context.Context, s server.Server, alias string) {

	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := s.Shutdown(timeoutCtx); err != nil {
		log.Printf("Error during %s Server shutdown: %v\n", alias, err)
	}
}

func main() {
	log.Printf("Catraia is starting\n")

	ctx := setupSignalHandling(context.Background())

	store := endpoint.NewStore()
	containerService := container.New(config.NewGetter(), store)

	proxyServer := server.NewProxy(":2020", store)
	go runServer(proxyServer, "Proxy")

	apiServer := server.NewAPI(":2077", containerService)
	go runServer(apiServer, "API")

	log.Printf("Catraia is ready\n")

	// Wait until context is done
	<-ctx.Done()

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		shutdownServer(context.Background(), proxyServer, "Proxy")
	}()

	go func() {
		defer wg.Done()
		shutdownServer(context.Background(), apiServer, "API")
	}()

	log.Printf("Waiting services to shutdown\n")
	wg.Wait()
	log.Printf("Catraia is down\n")
}
