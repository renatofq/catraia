package main

import (
	"log"
	"net"
	"syscall"

	"fmt"

	gocni "github.com/containerd/go-cni"
	"github.com/google/uuid"
	"github.com/vishvananda/netlink"
)

func setupNetworkIf(netns, cniConfDir, cniPluginDir string) ([]net.IP, error) {
	id := uuid.New().String()

	cni, err := gocni.New(gocni.WithPluginConfDir(cniConfDir),
		gocni.WithPluginDir([]string{cniPluginDir}))
	if err != nil {
		return nil, err
	}

	// Load the cni configuration
	if err := cni.Load(gocni.WithLoNetwork, gocni.WithDefaultConf); err != nil {
		return nil, fmt.Errorf("fail to load cni configuration: %v", err)
	}

	result, err := cni.Setup(id, netns)
	if err != nil {
		return nil, fmt.Errorf("fail to setup network for namespace %q: %v",
			id, err)
	}

	var ips []net.IP
	for name, ifConfig := range result.Interfaces {
		log.Printf("Config of interface %s: %v\n",
			name, ifConfig)

		if ifConfig.Sandbox == netns {
			for _, ipConfig := range ifConfig.IPConfigs {
				ips = append(ips, ipConfig.IP)
			}
		}
	}

	return ips, nil
}

func setupBridge(name string) error {
	bridge, err := ensureBridge(name)
	if err != nil {
		return err
	}

	return netlink.LinkSetUp(bridge)
}

func getBridge(name string) (*netlink.Bridge, error) {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return nil, err
	}

	bridge, ok := link.(*netlink.Bridge)
	if !ok {
		return nil, fmt.Errorf("interface %s already exists but is not a bridge", name)
	}

	return bridge, nil
}

func ensureBridge(name string) (*netlink.Bridge, error) {
	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name:   name,
			TxQLen: -1,
		},
	}

	err := netlink.LinkAdd(bridge)
	if err != nil {
		if err == syscall.EEXIST {
			return getBridge(name)
		} else {
			return nil, err
		}
	}

	return bridge, nil
}
