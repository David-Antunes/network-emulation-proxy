package outbound

import (
	"github.com/David-Antunes/network-emulation-proxy/internal"
	"github.com/David-Antunes/network-emulation-proxy/internal/proxy"
	"github.com/David-Antunes/network-emulation-proxy/xdp"
	"log"
	"os"
	"sync"
)

type Outbound struct {
	sync.Mutex
	sockets map[string]*proxy.ProxySocket
	gateway chan *xdp.Frame
	running bool
	queues  map[string]chan *xdp.Frame
}

var outLog = log.New(os.Stdout, "outbound INFO: ", log.Ltime)

func CreateOutbound(gateway chan *xdp.Frame) *Outbound {
	return &Outbound{
		Mutex:   sync.Mutex{},
		sockets: make(map[string]*proxy.ProxySocket),
		gateway: gateway,
		running: false,
		queues:  make(map[string]chan *xdp.Frame),
	}
}

func (outbound *Outbound) Start() {
	if !outbound.running {
		outLog.Println("Starting...")
		go outbound.receive()
	}
}

func (outbound *Outbound) AddSocket(sock *proxy.ProxySocket) {
	channel := make(chan *xdp.Frame, internal.QUEUE_SIZE)
	//outbound.queues[sock.GetMac()] = channel
	//outbound.sockets[sock.GetMac()] = sock
	go outbound.send(channel, sock)
}

func (outbound *Outbound) receive() {

	for {
		select {
		case frame := <-outbound.gateway:
			outbound.queues[frame.MacDestination] <- frame
		}
	}
}

func (outbound *Outbound) send(queue chan *xdp.Frame, sock *proxy.ProxySocket) {
	for {
		select {
		case frame := <-queue:
			batchSize := len(queue)
			frames := make([]*xdp.Frame, 0, batchSize+1)

			frames = append(frames, frame)

			for i := 1; i < batchSize; i++ {
				frames = append(frames, <-queue)
			}
			//sock.Send(frames)
		}
	}
}
