package xdp

import (
	"time"

	"github.com/asavie/xdp"
	"github.com/google/uuid"
	"github.com/vishvananda/netlink"
)

type XdpSock struct {
	id    uuid.UUID
	sock  xdp.Socket
	link  netlink.Link
	descs []xdp.Desc
}

func (socket XdpSock) ID() string {
	return socket.id.String()
}

func (socket XdpSock) Stats() Stats {
	stats, err := socket.sock.Stats()
	if err != nil {
		panic(err)
	}

	return Stats{stats.Filled, stats.Received, stats.Transmitted, stats.Completed, stats.KernelStats}
}

func CreateXdpSock(queue int, ifname string) XdpSock {

	link, err := netlink.LinkByName(ifname)

	if err != nil {
		panic(err)
	}

	xsk, err := xdp.NewSocket(link.Attrs().Index, queue, &DefaultSocketOptions)
	if err != nil {
		panic(err)
	}
	xsk.Fill(xsk.GetDescs(xsk.NumFreeFillSlots(), true))
	return XdpSock{uuid.New(), *xsk, link, []xdp.Desc{}}
}

func (socket XdpSock) Receive() []Frame {
	numRx, _, err := socket.sock.Poll(1)

	if err != nil {
		panic(err)
	}

	if numRx == 0 {
		return []Frame{}
	}

	rxDescs := socket.sock.Receive(numRx)
	if len(rxDescs) > 0 {
		frames := make([]Frame, 0, len(rxDescs))
		for i := 0; i < len(rxDescs); i++ {
			framePointer := socket.sock.GetFrame(rxDescs[i])
			macDest := string(framePointer[0:6])
			macOrig := string(framePointer[6:12])
			frame := Frame{framePointer, time.Now(), macOrig, macDest, rxDescs[i]}
			frames = append(frames, frame)
		}
		return frames
	}
	return []Frame{}
}

func (socket XdpSock) SendFrame(frame Frame) {

	_, _, err := socket.sock.Poll(1)

	if err != nil {
		panic(err)
	}
	txDescs := socket.getTransmitDescs(1)

	if len(txDescs) > 0 {
		outFrame := socket.sock.GetFrame(txDescs[0])
		txDescs[0].Len = uint32(copy(outFrame, frame.framePointer))
		socket.sock.Transmit(txDescs)
	}
}

func (socket XdpSock) Send(frames []Frame) {
	_, _, err := socket.sock.Poll(1)

	if err != nil {
		panic(err)
	}
	txDescs := socket.getTransmitDescs(len(frames))

	for i := 0; i < len(txDescs); i++ {
		outFrame := socket.sock.GetFrame(txDescs[i])
		txDescs[i].Len = uint32(copy(outFrame, frames[i].framePointer))
	}
	socket.sock.Transmit(txDescs)
}

func (socket XdpSock) getTransmitDescs(number int) []xdp.Desc {
	if len(socket.descs) < number {
		socket.descs = socket.sock.GetDescs(socket.sock.NumFreeTxSlots(), false)
	}
	if len(socket.descs) < number {
		return socket.descs
	} else {
		descs := socket.descs[0:number]
		socket.descs = socket.descs[number:]
		return descs
	}
}

func (socket XdpSock) Close() {
	err := socket.sock.Close()
	if err != nil {
		panic(err)
	}
}

func (socket XdpSock) CleanFrameMem(frames []Frame) {
	descs := make([]xdp.Desc, 0, len(frames))

	for _, frame := range frames {
		descs = append(descs, frame.umemAddr)
	}
	socket.sock.Fill(descs)
}
