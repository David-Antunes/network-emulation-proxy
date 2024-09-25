package inbound

import (
	"encoding/gob"
	"github.com/David-Antunes/network-emulation-proxy/internal"
	"github.com/David-Antunes/network-emulation-proxy/internal/outbound"
	"github.com/David-Antunes/network-emulation-proxy/internal/proxy"
	"github.com/David-Antunes/network-emulation-proxy/xdp"
	"log"
	"os"
)

/*
Inbound singleton is responsible for managing all incoming network packets
that enter the network emulator.
Handles the management of mac addresses of new sockets by listening for the
first packets and registering the seen mac address.
Is also responsible for network packets that come from the outside and insert
them into the emulation.
The singleton is automatically created as soon as the emulator starts.
Requires Garbage collection logic to be configured before starting its main go
routine.
*/
type Inbound struct {
	sockets map[string]*proxy.ProxySocket
	queue   chan *xdp.Frame
	out     *outbound.Outbound
	enc     *gob.Encoder
}

var inLog = log.New(os.Stdout, "INBOUND INFO: ", log.Ltime)

func CreateInbound(out *outbound.Outbound) *Inbound {
	return &Inbound{
		sockets: make(map[string]*proxy.ProxySocket),
		queue:   make(chan *xdp.Frame, internal.QUEUE_SIZE),
		out:     out,
		enc:     nil,
	}
}
func (inbound *Inbound) SetEnc(enc *gob.Encoder) {
	inbound.enc = enc
}

func (inbound *Inbound) AddSocket(iface string) {
	if _, ok := inbound.sockets[iface]; !ok {
		//socket, err := proxy.NewReceiveSocket(0, iface)
		//if err != nil {
		//	internal.ShutdownAndLog(err)
		//	return
		//}
		//socket.SetEnc(inbound.enc)
		//inbound.sockets[iface] = socket
		//go func() {
		//
		//	socket.FindMac()
		//	inbound.out.AddSocket(socket)
		//	socket.Receive(-1)
		//}()
		inLog.Println("Registered socket for", iface)
	}
}

func (inbound *Inbound) RemoveSocket(iface string) {
	if sock, ok := inbound.sockets[iface]; ok {
		sock.Close()
		delete(inbound.sockets, iface)
		inLog.Println("Removed socket from", iface)
		sock.Close()
	}
}

func (inbound *Inbound) Close() {
	for _, sock := range inbound.sockets {
		sock.Close()
	}
	inLog.Println("Stopped")
}
