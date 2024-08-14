package xdp

import (
	"fmt"
	"log"
	"os"
)

type Outbound struct {
	socket      Isocket
	channel     chan Frame
	running     bool
	queue       chan Frame
	ctx         chan struct{}
	nullChannel chan Frame
}

var Out = createOutbound()
var outLog = log.New(os.Stdout, "outbound INFO: ", log.Ltime)

func createOutbound() Outbound {
	return Outbound{nil, make(chan Frame, queueSize), false, make(chan Frame, queueSize), make(chan struct{}), make(chan Frame)}
}

func (outbound *Outbound) GetNullChannel() chan Frame {
	return outbound.nullChannel
}

func (outbound *Outbound) Channel() chan Frame {
	return outbound.channel
}

func (outbound *Outbound) Stop() {
	outbound.ctx <- struct{}{}
}

func (outbound *Outbound) SetSocket(socket Isocket) {
	outbound.socket = socket
}

func (outbound *Outbound) Start() {
	if !outbound.running {
		outLog.Println("Starting outbound routines.")
		outbound.running = true
		registerUniqSocket("outbound", outbound.socket)
		go outbound.receive()
		go outbound.send()
	}
}

func (outbound *Outbound) receive() {
	for frame := range outbound.channel {
		if len(outbound.queue) < queueSize {
			outbound.queue <- frame
		} else {
			GC.GCFrame(frame)
			fmt.Println("Queue Full!")
		}
	}
}

func (outbound *Outbound) send() {

	frames := make([]Frame, 0, batchSize)
	var queueLen int
	for {
		select {
		case <-outbound.ctx:
			return

		case frame := <-outbound.nullChannel:
			GC.GCFrame(frame)

		case frame := <-outbound.queue:
			frames = frames[:0]
			frames = append(frames, frame)
			queueLen = len(outbound.queue)
			for i := 0; queueLen < batchSize-1 && queueLen > 0 && i < queueLen; i++ {
				frame = <-outbound.queue
				frames = append(frames, frame)
			}
			outbound.socket.Send(frames)
			for _, frame := range frames {
				GC.GCFrame(frame)
			}

		}
	}
}

func (outbound *Outbound) Close() {
	outbound.running = false
	outbound.ctx <- struct{}{}
	outbound.socket.Close()
}
