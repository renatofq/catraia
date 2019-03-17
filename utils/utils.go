package utils

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func NetTypeFromAddr(addr string) string {
	if strings.HasSuffix(addr, ".sock") {
		return "unix"
	}

	return "tcp"
}

func SignalHandling(ctx context.Context) context.Context {
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
