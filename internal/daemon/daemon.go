package daemon

import (
	"fmt"
	"github.com/David-Antunes/network-emulation-proxy/internal/inbound"
	"github.com/David-Antunes/network-emulation-proxy/internal/outbound"
	"github.com/David-Antunes/network-emulation-proxy/xdp"
	"net"
	"sync"
	"syscall"
	"time"
)

type Daemon struct {
	sync.Mutex
	in         *inbound.Inbound
	out        *outbound.Outbound
	unixPath   string
	interfaces map[string]struct{}
}

func NewDaemon(in *inbound.Inbound, out *outbound.Outbound, unixPath string) *Daemon {
	interfaces := make(map[string]struct{})

	// Ignore Interfaces
	interfaces["veth0"] = struct{}{}
	interfaces["veth1"] = struct{}{}
	interfaces["vxlan0"] = struct{}{}
	interfaces["br0"] = struct{}{}
	interfaces["lo"] = struct{}{}

	return &Daemon{
		Mutex:      sync.Mutex{},
		in:         in,
		out:        out,
		unixPath:   unixPath,
		interfaces: interfaces,
	}
}
func (d *Daemon) Serve() error {
	return nil
}
func (d *Daemon) SearchInterfaces() {
	for {
		time.Sleep(time.Second * 10)
		ifaces, err := net.Interfaces()

		if err != nil {
			fmt.Println(err)
			syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
			continue

		}

		if len(ifaces) == 1 {
			fmt.Println("something went wrong with the network")
			syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
			continue
		}

		newIfaces := make([]string, 0, len(ifaces))
		for _, iface := range ifaces {
			if _, ok := d.interfaces[iface.Name]; !ok {
				newIfaces = append(newIfaces, iface.Name)
				d.interfaces[iface.Name] = struct{}{}
			}
		}

		for _, iface := range newIfaces {
			fmt.Println("Found interface " + iface)
			sock, err := xdp.CreateXdpBpfSock(0, iface)
			if err != nil {
				fmt.Println("Error creating socket: " + err.Error())
				continue
			}
			d.in.AddSocket(iface, sock)
			fmt.Println("Found interface " + iface)
		}
	}
}
