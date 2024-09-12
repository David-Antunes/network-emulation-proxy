package inbound

import (
	"fmt"
	"github.com/David-Antunes/network-emulation-proxy/internal"
	"github.com/David-Antunes/network-emulation-proxy/xdp"
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
	sockets map[string]xdp.Isocket
	queue   chan *xdp.Frame
	running bool
	gateway chan *xdp.Frame
	ctx     chan struct{}
}

var inLog = log.New(os.Stdout, "INBOUND INFO: ", log.Ltime)

func CreateInbound(gateway chan *xdp.Frame) *Inbound {
	return &Inbound{
		Mutex:   sync.Mutex{},
		sockets: make(map[string]xdp.Isocket),
		queue:   make(chan *xdp.Frame, internal.QUEUE_SIZE),
		running: false,
		gateway: gateway,
		ctx:     make(chan struct{}),
	}
}

func (inbound *Inbound) AddSocket(iface string, socket xdp.Isocket) {
	inbound.Lock()
	if _, ok := inbound.sockets[iface]; !ok {
		inbound.sockets[iface] = socket
		go inbound.pollSocket(socket)
		inLog.Println("Registered socket for", iface)
	}
	inbound.Unlock()
}

func (inbound *Inbound) RemoveSocket(iface string) {
	inbound.Lock()
	if sock, ok := inbound.sockets[iface]; ok {
		delete(inbound.sockets, iface)
		inLog.Println("Removed socket from", iface)
		sock.Close()
	}
	inbound.Unlock()
}

func (inbound *Inbound) pollSocket(socket xdp.Isocket) {

	var frames []*xdp.Frame
	var err error
	for len(frames) == 0 {
		frames, err = socket.Receive(-1)
	}
	if err != nil {
		fmt.Println(err)
		return
	}

	inLog.Println("Received MAC:", net.HardwareAddr(frames[0].MacOrigin))

	for _, frame := range frames {
		inbound.queue <- frame
	}

	for {

		frames, err = socket.Receive(-1)
		if err != nil {
			fmt.Println(err)
			return
		}
		for _, frame := range frames {
			if len(inbound.queue) < internal.QUEUE_SIZE {
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
		inLog.Println("Started")
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
	inLog.Println("Stopped")
}

func (inbound *Inbound) Close() {
	inbound.Lock()
	inbound.running = false
	inbound.Stop()
	for _, sock := range inbound.sockets {
		sock.Close()
	}
	inbound.Unlock()
	inLog.Println("Closed")
}
