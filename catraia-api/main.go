package main

import (
	"context"
	"log"
	"sync"

	"github.com/renatofq/catraia/config"
	"github.com/renatofq/catraia/servers"
	"github.com/renatofq/catraia/utils"
)

func init() {
	if err := utils.LoadDotEnv(); err != nil {
		log.Printf("No .env file found\n")
	}
}

func setupContainerService(conf *config.Config) ContainerService {
	ctrdConf := &ContainerdConfig{
		Namespace: conf.ContainerdNamespace,
		Socket:    conf.ContainerdSocket,
	}

	configService := NewConfigService()

	eventListener := NewContainerListener(conf.NetServerAddr)

	return NewContainerService(ctrdConf, configService, eventListener)
}

func setupTunnelServer(conf *config.Config) servers.Server {
	tunnelServer := NewTunnelServer("Tunnel", conf.TunnelAddr, conf.ProxyAddr)
	go servers.Run(tunnelServer)

	return tunnelServer
}

func setupAPIServer(conf *config.Config) servers.Server {
	containerService := setupContainerService(conf)

	apiServer := NewAPIServer("API", conf.APIServerAddr, containerService)
	go servers.Run(apiServer)

	return apiServer
}

func main() {
	log.Printf("catraia is starting\n")

	ctx := utils.SignalHandling(context.Background())

	conf := config.New()

	tunnelServer := setupTunnelServer(conf)

	apiServer := setupAPIServer(conf)

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
