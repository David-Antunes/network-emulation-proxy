package xdp

import (
	"fmt"

	"github.com/Workiva/go-datastructures/queue"
)

var GC = createGC()

type SocketGc struct {
	sockets map[string]Isocket
	queues  map[string]*queue.RingBuffer
}

func createGC() *SocketGc {
	return &SocketGc{make(map[string]Isocket), make(map[string]*queue.RingBuffer)}
}

func (gc *SocketGc) AddSocket(socket Isocket, mac string) {
	gc.sockets[mac] = socket
	gc.queues[mac] = queue.NewRingBuffer(gcSize)
}

func (gc *SocketGc) RemoveSocket(mac string) {
	delete(gc.sockets, mac)
	delete(gc.queues, mac)
}

func (gc *SocketGc) RemoteSocket(frame Frame) {
	In.remoteSocket.SendFrame(frame)
}

func (gc *SocketGc) GCFrame(frame Frame) {
	q, ok := gc.queues[frame.macOrigin]

	if ok {
		if q.Len() < gcSize {
			err := q.Put(frame)
			if err != nil {
				panic(err)
			}
		} else {
			frames := make([]Frame, 0, gcSize+1)
			for i := 0; i < gcSize; i++ {
				element, err := q.Get()
				if err != nil {
					panic(err)
				}
				frames = append(frames, element.(Frame))
			}
			frames = append(frames, frame)
			socket, ok := gc.sockets[frame.macOrigin]
			if ok {
				socket.CleanFrameMem(frames)
				// fmt.Println("Cleaned frames")
			} else {
				fmt.Println("Failed to GC Frames.")
			}
		}
	}
}
