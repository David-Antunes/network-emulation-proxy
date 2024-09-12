package main

import (
	"github.com/David-Antunes/network-emulation-proxy/internal/conn"
	"github.com/David-Antunes/network-emulation-proxy/internal/daemon"
	"github.com/David-Antunes/network-emulation-proxy/internal/inbound"
	"github.com/David-Antunes/network-emulation-proxy/internal/metricsManager"
	"github.com/David-Antunes/network-emulation-proxy/internal/outbound"
	"github.com/David-Antunes/network-emulation-proxy/internal/unix-socket"
	"github.com/David-Antunes/network-emulation-proxy/xdp"
	"github.com/spf13/viper"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var proxyLog = log.New(os.Stdout, "PROXY INFO: ", log.Ltime)

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

	viper.SetConfigFile(".env")
	if err := viper.ReadInConfig(); err != nil {
		viper.Set("PROXY_SOCKET", "/tmp/proxy.sock")
		viper.Set("PROXY_SERVER", "/tmp/proxy-server.sock")
		viper.Set("PROXY_RTT_SOCKET", "/tmp/proxy-rtt.sock")
		viper.Set("TIMEOUT", time.Second*60)
		viper.Set("NUM_TESTS", 5)
		viper.SetConfigType("env")
		viper.WriteConfigAs(".env")
	} else {
		if !viper.IsSet("PROXY_SOCKET") {
			viper.Set("PROXY_SOCKET", "/tmp/proxy.sock")
		}
		if !viper.IsSet("PROXY_SERVER") {
			viper.Set("PROXY_SERVER", "/tmp/proxy-server.sock")
		}
		if !viper.IsSet("PROXY_RTT_SOCKET") {
			viper.Set("PROXY_RTT_SOCKET", "/tmp/proxy-rtt.sock")
		}
		if !viper.IsSet("TIMEOUT") {
			viper.Set("TIMEOUT", 60000)
		}
		if !viper.IsSet("NUM_TESTS") {
			viper.Set("NUM_TESTS", 5)
		}
		viper.SetConfigType("env")
		viper.WriteConfigAs(".env")
	}

	for id, value := range viper.AllSettings() {
		proxyLog.Println(id, value)
	}

	os.Remove(viper.GetString("PROXY_SOCKET"))
	os.Remove(viper.GetString("PROXY_SERVER"))
	os.Remove(viper.GetString("PROXY_RTT_SOCKET"))

	unixsocket.SetSocketPath(viper.GetString("PROXY_SOCKET"))

	out := outbound.CreateOutbound(unixsocket.GetReadChannel())
	out.SetSocket()
	in := inbound.CreateInbound(unixsocket.GetWriteChannel())

	server := daemon.NewDaemon(in, out, viper.GetString("PROXY_SERVER"))

	metricsIp, metricsMac, broadcastIP := GetIfaceInformation()

	rtt, err := xdp.CreateXdpBpfSock(0, "veth1")
	if err != nil {
		panic(err)
	}

	rttConn := &conn.RttConnection{
		Mac:  []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		IP:   broadcastIP,
		Port: 8000,
	}
	metrics := metricsManager.NewMetricsManager(rtt, metricsMac, metricsIp, 8000, rttConn, viper.GetString("PROXY_RTT_SOCKET"), time.Duration(viper.GetInt("TIMEOUT"))*time.Millisecond, viper.GetInt("NUM_TESTS"))
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
	if len(splitAddr) != 4 {
		panic("something went wrong with Ip address")
	}

	broadcastIp := splitAddr[0] + "." + splitAddr[1] + "." + splitAddr[2] + ".255"
	return net.ParseIP(ip[0]), ief.HardwareAddr, net.ParseIP(broadcastIp)

}
