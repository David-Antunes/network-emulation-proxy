package xdp

import (
	"fmt"
	"log"
	"net"
	"os"
	"sync"
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
	sync.Mutex
	sockets  map[string]Isocket
	outbound *Outbound
	queue    chan *Frame
	running  bool
	gateway  chan *Frame
	ctx      chan struct{}
}

var inLog = log.New(os.Stdout, "inbound INFO: ", log.Ltime)

func CreateInbound(gateway chan *Frame, outbound *Outbound) *Inbound {
	return &Inbound{
		Mutex:    sync.Mutex{},
		sockets:  make(map[string]Isocket),
		outbound: outbound,
		queue:    make(chan *Frame, queueSize),
		running:  false,
		gateway:  gateway,
		ctx:      make(chan struct{}),
	}
}

func (inbound *Inbound) AddSocket(iface string, socket Isocket) {
	inbound.Lock()
	if _, ok := inbound.sockets[iface]; !ok {
		inbound.sockets[iface] = socket
		go inbound.pollSocket(socket)
		inLog.Println("Setup local socket for", iface, "interface")
	}
	inbound.Unlock()
}

func (inbound *Inbound) RemoveSocket(iface string) {
	inbound.Lock()
	if sock, ok := inbound.sockets[iface]; ok {
		delete(inbound.sockets, iface)
		sock.Close()
	}
	inbound.Unlock()
}

func (inbound *Inbound) pollSocket(socket Isocket) {
	//TODO: Pass this to somewhere else

	var frames []*Frame

	for len(frames) == 0 {
		frames = socket.Receive()
	}

	inLog.Println("Found MAC address: ", net.HardwareAddr(frames[0].MacOrigin))

	mac := frames[0].MacOrigin
	inbound.outbound.AddMac(mac, socket)

	for _, frame := range frames {
		inbound.queue <- frame
	}

	for {
		for _, frame := range socket.Receive() {
			if len(inbound.queue) < queueSize {
				inbound.queue <- frame
			} else {
				fmt.Println("Queue Full!")
			}
		}
	}
}

func (inbound *Inbound) Start() {
	if !inbound.running {
		inbound.running = true
		go inbound.send()
	}
}

func (inbound *Inbound) send() {
	for {
		select {
		case <-inbound.ctx:
			return

		case frame := <-inbound.queue:
			inbound.gateway <- frame
		}
	}
}

func (inbound *Inbound) Stop() {
	inbound.ctx <- struct{}{}
}

func (inbound *Inbound) Close() {
	inbound.Lock()
	inbound.running = false
	inbound.Stop()
	for _, sock := range inbound.sockets {
		sock.Close()
	}
	inbound.Unlock()
}
