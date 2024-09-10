package main

import (
	"github.com/David-Antunes/network-emulation-proxy/internal/daemon"
	"github.com/David-Antunes/network-emulation-proxy/internal/inbound"
	"github.com/David-Antunes/network-emulation-proxy/internal/outbound"
	"github.com/David-Antunes/network-emulation-proxy/internal/unix-socket"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func cleanup(d *daemon.Daemon) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		d.Cleanup()
		unixsocket.Close()
		os.Exit(1)
	}()
}
func main() {
	err := os.Remove("/tmp/emu.sock")
	//if err != nil {
	//	return
	//}
	unixsocket.SetSocketPath("/tmp/emu.sock")
	out := outbound.CreateOutbound(unixsocket.GetReadChannel())
	out.SetSocket()
	in := inbound.CreateInbound(unixsocket.GetWriteChannel())

	server := daemon.NewDaemon(in, out, "/tmp/proxy-server.sock")

	go server.Serve()

	go cleanup(server)

	in.Start()
	out.Start()
	go func() {
		time.Sleep(10 * time.Second)
		server.SearchInterfaces(nil, nil)
	}()

	err = unixsocket.StartSocket()
	if err != nil {
		return
	}

}
