package main

import (
	"context"
	"log"
	"os"
	"sync"

	"github.com/renatofq/catraia/config"
	"github.com/renatofq/catraia/servers"
	"github.com/renatofq/catraia/utils"
)

func setupRuntimeDir(conf *config.Config) error {
	return os.MkdirAll(conf.RuntimeDir, os.ModePerm)
}

func setupProxyServer(conf *config.Config, store EndpointStore) servers.Server {
	proxyServer := NewProxyServer("Proxy", conf.ProxyAddr, store)
	go servers.Run(proxyServer)

	return proxyServer
}

func setupNetworkServer(conf *config.Config, store EndpointStore) servers.Server {
	eventServer := NewEventServer("EventListener", conf.NetServerAddr, store)
	go servers.Run(eventServer)

	return eventServer
}

func main() {
	log.Printf("catraia-net is starting\n")
	defer log.Printf("catraia-net is down\n")

	ctx := utils.SignalHandling(context.Background())

	conf := config.New()

	if err := setupRuntimeDir(conf); err != nil {
		log.Fatalf("Fail to setup runtime: %v\n", err)
	}

	if err := setupBridge(conf.Bridge); err != nil {
		log.Fatalf("Fail to setup bridge: %v\n", err)
	}

	log.Printf("bridge interface %s is up\n", conf.Bridge)

	store := NewStore()

	proxyServer := setupProxyServer(conf, store)

	eventServer := setupNetworkServer(conf, store)

	log.Printf("catraia-net is ready\n")

	// Wait until context is done by receiving a signal to terminate
	<-ctx.Done()

	log.Printf("catraia-net is shutting down\n")

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		servers.Shutdown(context.Background(), eventServer)
	}()

	go func() {
		defer wg.Done()
		servers.Shutdown(context.Background(), proxyServer)
	}()

	wg.Wait()
}
