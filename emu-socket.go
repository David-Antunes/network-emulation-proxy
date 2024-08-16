package main

import (
	"fmt"
	"gitea.homelab-antunes.duckdns.org/david-antunes/network-emulation-socket/xdp"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	unixsocket "gitea.homelab-antunes.duckdns.org/david-antunes/network-emulation-socket/unix-socket"
)

func cleanup(in *xdp.Inbound, out *xdp.Outbound) {
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
	outbound := xdp.CreateOutbound(unixsocket.GetReadChannel())
	inbound := xdp.CreateInbound(unixsocket.GetWriteChannel(), outbound)

	go cleanup(inbound, outbound)
	interfaces := make(map[string]struct{})

	interfaces["veth0"] = struct{}{}
	interfaces["veth1"] = struct{}{}
	interfaces["vxlan0"] = struct{}{}
	interfaces["br0"] = struct{}{}
	interfaces["lo"] = struct{}{}

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
