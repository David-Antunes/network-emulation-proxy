package daemon

import (
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/David-Antunes/network-emulation-proxy/internal"
	"github.com/David-Antunes/network-emulation-proxy/internal/inbound"
	"github.com/David-Antunes/network-emulation-proxy/internal/outbound"
	"github.com/David-Antunes/network-emulation-proxy/xdp"
)

var serverLog = log.New(os.Stdout, "SERVER INFO: ", log.Ltime)

type Daemon struct {
	sync.Mutex
	in         *inbound.Inbound
	out        *outbound.Outbound
	unixPath   string
	httpServer *http.Server
	socket     net.Listener
	interfaces map[string]struct{}
}

func NewDaemon(in *inbound.Inbound, out *outbound.Outbound, unixPath string) *Daemon {

	s, err := net.Listen("unix", unixPath)
	if err != nil {
		internal.ShutdownAndLog(err)
		return nil
	}

	interfaces := make(map[string]struct{})

	// Ignore Interfaces
	interfaces["veth0"] = struct{}{}
	interfaces["veth1"] = struct{}{}
	interfaces["vxlan0"] = struct{}{}
	interfaces["br0"] = struct{}{}
	interfaces["lo"] = struct{}{}

	d := &Daemon{
		Mutex:      sync.Mutex{},
		in:         in,
		out:        out,
		unixPath:   unixPath,
		httpServer: nil,
		socket:     nil,
		interfaces: interfaces,
	}

	m := http.NewServeMux()
	m.HandleFunc("/refresh", d.SearchInterfaces)

	httpServer := http.Server{
		Handler: m,
	}

	d.httpServer = &httpServer
	d.socket = s
	return d

}
func (d *Daemon) Serve() {

	serverLog.Println("socketPath:", d.unixPath)
	serverLog.Println("Serving...")
	if err := d.httpServer.Serve(d.socket); err != nil {
		internal.ShutdownAndLog(err)
	}
}

func (d *Daemon) SearchInterfaces(w http.ResponseWriter, r *http.Request) {
	serverLog.Println("Searching for network interfaces")
	d.Lock()
	ifaces, err := net.Interfaces()

	if err != nil {
		internal.ShutdownAndLog(err)
		return
	}

	if len(ifaces) == 1 {
		internal.ShutdownAndLog(errors.New("something went wrong with the network"))
		return
	}

	newIfaces := make([]string, 0, len(ifaces))
	for _, iface := range ifaces {
		if _, ok := d.interfaces[iface.Name]; !ok {
			newIfaces = append(newIfaces, iface.Name)
			d.interfaces[iface.Name] = struct{}{}
		}
	}

	for _, iface := range newIfaces {
		serverLog.Println("Found interface:", iface)
		sock, err := xdp.CreateXdpBpfSock(0, iface)
		if err != nil {
			serverLog.Println("Error creating socket: " + err.Error())
			continue
		}
		d.in.AddSocket(iface, sock)
	}
	d.Unlock()
}

func (d *Daemon) Cleanup() {
	d.in.Close()
	d.socket.Close()
	d.httpServer.Close()
	os.Remove(d.unixPath)
	d.out.Close()
	serverLog.Println("Closed")
}
