package main

import (
	"fmt"
	"github.com/David-Antunes/network-emulation-proxy/internal/daemon"
	"github.com/David-Antunes/network-emulation-proxy/internal/inbound"
	"github.com/David-Antunes/network-emulation-proxy/internal/outbound"
	"github.com/David-Antunes/network-emulation-proxy/internal/unix-socket"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/David-Antunes/network-emulation-proxy/xdp"
)

func cleanup(in *inbound.Inbound, out *outbound.Outbound) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		in.Close()
		out.Close()
		os.Exit(1)
	}()
}
func main() {
	err := os.Remove("/tmp/emu.sock")
	//if err != nil {
	//	return
	//}
	unixsocket.SetSocketPath("/tmp/emu.sock")
	outbound := outbound.CreateOutbound(unixsocket.GetReadChannel())
	outbound.SetSocket()
	inbound := inbound.CreateInbound(unixsocket.GetWriteChannel())

	server := daemon.NewDaemon(inbound, outbound, "/tmp/proxy-server.sock")

	server.Serve()

	go cleanup(inbound, outbound)

	go func() {

		for {

			time.Sleep(time.Second * 1)
			ifaces, err := net.Interfaces()

			if err != nil {
				panic(err)
			}

			newIfaces := make([]string, 0, len(ifaces))
			for _, iface := range ifaces {
				if _, ok := interfaces[iface.Name]; !ok {
					newIfaces = append(newIfaces, iface.Name)
					interfaces[iface.Name] = struct{}{}
				}
			}

			for _, iface := range newIfaces {
				fmt.Println("Found interface " + iface)
				sock := xdp.CreateXdpBpfSock(0, iface)
				inbound.AddSocket(iface, sock)
				fmt.Println("Found interface " + iface)
			}
		}

	}()

	inbound.Start()
	outbound.Start()

	err = unixsocket.StartSocket()
	if err != nil {
		return
	}
}
