package xdp

import (
	"fmt"
	"log"
	"net"
	"os"
	"sync"
)

type Outbound struct {
	sync.Mutex
	sockets map[string]Isocket
	gateway chan *Frame
	running bool
	queue   chan *Frame
	ctx     chan struct{}
}

var outLog = log.New(os.Stdout, "outbound INFO: ", log.Ltime)

func CreateOutbound(gateway chan *Frame) *Outbound {
	return &Outbound{
		Mutex:   sync.Mutex{},
		sockets: map[string]Isocket{},
		gateway: gateway,
		running: false,
		queue:   make(chan *Frame, queueSize),
		ctx:     make(chan struct{}),
	}
}

func (outbound *Outbound) AddMac(mac string, socket Isocket) {
	outbound.Lock()
	if _, ok := outbound.sockets[mac]; !ok {
		outbound.sockets[mac] = socket
	}
	outbound.Unlock()
}

func (outbound *Outbound) Stop() {
	outbound.ctx <- struct{}{}
}

func (outbound *Outbound) Start() {
	if !outbound.running {
		outLog.Println("Starting outbound routines.")
		outbound.running = true
		go outbound.send()
		go outbound.receive()
	}
}

func (outbound *Outbound) receive() {

	for {
		select {
		case <-outbound.ctx:
			return
		case frame := <-outbound.gateway:
			fmt.Println("yooo")
			outbound.queue <- frame
		}
	}
}

func (outbound *Outbound) send() {

	for {
		select {
		case <-outbound.ctx:
			return

		case frame := <-outbound.queue:
			fmt.Println(len(outbound.sockets))
			fmt.Println("before", net.HardwareAddr(frame.MacOrigin), net.HardwareAddr(frame.MacDestination), frame.Time)
			if socket, ok := outbound.sockets[frame.MacDestination]; ok {

				fmt.Println("outbound", net.HardwareAddr(frame.MacOrigin), net.HardwareAddr(frame.MacDestination), frame.Time)
				socket.SendFrame(frame)
			}

		}
	}
}

func (outbound *Outbound) Close() {
	outbound.running = false
	outbound.ctx <- struct{}{}
	outbound.ctx <- struct{}{}
}
