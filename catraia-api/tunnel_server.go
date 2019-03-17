package main

import (
	"context"
	"io"
	"log"
	"net"
	"sync"

	"github.com/renatofq/catraia/servers"
	"github.com/renatofq/catraia/utils"
)

type tunServer struct {
	name string
	listenAddr string
	destAddr   string

	listener net.Listener
	waiting  sync.WaitGroup
}

func NewTunnelServer(name, listenAddr, destAddr string) servers.Server {
	return &tunServer{
		name: name,
		listenAddr: listenAddr,
		destAddr: destAddr,
	}
}

func (ts *tunServer) Name() string {
	return ts.name
}

func (ts *tunServer) ListenAndServe() error {
	l, err := net.Listen(utils.NetTypeFromAddr(ts.listenAddr), ts.listenAddr)
	if err != nil {
		return err
	}

	return ts.Serve(l)
}

func (ts *tunServer) Serve(l net.Listener) error {
	l = &onceCloseListener{Listener: l}
	defer l.Close()

	ts.listener = l

	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}

		ts.waiting.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			handleConnection(conn, utils.NetTypeFromAddr(ts.destAddr), ts.destAddr)
		}(&ts.waiting)
	}
}

func (ts *tunServer) Shutdown(ctx context.Context) error {
	ts.listener.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		ts.waiting.Wait()
	}()

	select {
	case <-done:
	case <-ctx.Done():
	}

	return ctx.Err()
}

func handleConnection(srcConn net.Conn, dstNet, dstAddr string) {
	defer srcConn.Close()

	dstConn, err := net.Dial(dstNet, dstAddr)
	if err != nil {
		log.Printf("Fail to connect to destination: %v\n", err)
		return
	}
	defer dstConn.Close()

	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		join(srcConn, dstConn)
	}()

	go func() {
		defer wg.Done()
		join(dstConn, srcConn)
	}()

	wg.Wait()
}

func join(src io.Reader, dst io.Writer) {
	if _, err := io.Copy(dst, src); err != nil {
		log.Printf("Error streaming data")
	}
}

// onceCloseListener wraps a net.Listener, protecting it from
// multiple Close calls.
type onceCloseListener struct {
	net.Listener
	once     sync.Once
	closeErr error
}

func (oc *onceCloseListener) Close() error {
	oc.once.Do(oc.close)
	return oc.closeErr
}

func (oc *onceCloseListener) close() {
	oc.closeErr = oc.Listener.Close()
}
