package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/renatofq/catraia/servers"
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

func main() {
	log.Printf("catraia is starting\n")

	ctx := setupSignalHandling(context.Background())

	eventListener := NewContainerListener("unix", "/run/catraia/event.sock")

	containerService := NewContainerService(NewConfigService(), eventListener)

	tunnelServer := NewTunnelServer("Tunnel", "tcp", ":2020", "unix",
		"/run/catraia/proxy.sock")
	go servers.Run(tunnelServer)

	apiServer := NewAPIServer("API", ":2077", containerService)
	go servers.Run(apiServer)

	log.Printf("catraia is ready\n")

	// Wait until context is done by receiving a signal to terminate
	<-ctx.Done()

	log.Printf("catraia is shutting down\n")

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		servers.Shutdown(context.Background(), tunnelServer)
	}()

	go func() {
		defer wg.Done()
		servers.Shutdown(context.Background(), apiServer)
	}()

	wg.Wait()
	log.Printf("catraia is down\n")
}
