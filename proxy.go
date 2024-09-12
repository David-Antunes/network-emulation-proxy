package main

import (
	"fmt"
	"github.com/David-Antunes/network-emulation-proxy/api"
	"github.com/David-Antunes/network-emulation-proxy/internal/daemon"
	"github.com/David-Antunes/network-emulation-proxy/internal/inbound"
	"github.com/David-Antunes/network-emulation-proxy/internal/metricsManager"
	"github.com/David-Antunes/network-emulation-proxy/internal/outbound"
	"github.com/David-Antunes/network-emulation-proxy/internal/unix-socket"
	"github.com/David-Antunes/network-emulation-proxy/xdp"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func cleanup(d *daemon.Daemon, m *metricsManager.MetricsManager) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		d.Cleanup()
		unixsocket.Close()
		m.Close()
		os.Exit(1)
	}()
}
func main() {
	err := os.Remove("/tmp/emu.sock")

	unixsocket.SetSocketPath("/tmp/emu.sock")
	out := outbound.CreateOutbound(unixsocket.GetReadChannel())
	out.SetSocket()
	in := inbound.CreateInbound(unixsocket.GetWriteChannel())

	server := daemon.NewDaemon(in, out, "/tmp/proxy-server.sock")

	metricsIp, metricsMac, broadcastIP := GetIfaceInformation()
	fmt.Println(metricsIp)
	rtt, err := xdp.CreateXdpBpfSock(0, "veth1")
	if err != nil {
		panic(err)
	}

	metrics := metricsManager.NewMetricsManager(rtt, metricsMac, metricsIp, 8000, &api.StartTestRequest{
		Mac:  []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		IP:   broadcastIP,
		Port: 8000,
	})
	go metrics.Start()
	go server.Serve()

	go cleanup(server, metrics)

	in.Start()
	out.Start()
	server.SearchInterfaces(nil, nil)
	go func() {
		time.Sleep(10 * time.Second)
		server.SearchInterfaces(nil, nil)
	}()

	err = unixsocket.StartSocket()
	if err != nil {
		return
	}

}

func GetIfaceInformation() (net.IP, net.HardwareAddr, net.IP) {
	ief, err := net.InterfaceByName("br0")
	if err != nil {
		panic(err)
	}
	addrs, err := ief.Addrs()
	if err != nil {
		panic(err)
	}
	ip := strings.Split(addrs[0].String(), "/")
	splitAddr := strings.Split(ip[0], ".")
	fmt.Println(addrs)
	if len(splitAddr) != 4 {
		panic("something went wrong with Ip address")
	}

	broadcastIp := splitAddr[0] + "." + splitAddr[1] + "." + splitAddr[2] + ".255"
	fmt.Println(broadcastIp)
	return net.ParseIP(ip[0]), ief.HardwareAddr, net.ParseIP(broadcastIp)

}
