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
	sockets       map[string]Isocket
	remoteSocket  Isocket
	pendingSock   []Isocket
	channels      map[string]chan Frame
	queue         chan Frame
	bootstrapSock map[string]Isocket
	running       bool
	ctx           chan struct{}
}

// In Inbound Singleton
var In = createInbound()
var inLog = log.New(os.Stdout, "inbound INFO: ", log.Ltime)

func createInbound() Inbound {
	return Inbound{sync.Mutex{}, make(map[string]Isocket), nil, make([]Isocket, 0, 10), make(map[string]chan Frame), make(chan Frame, queueSize), make(map[string]Isocket), false, make(chan struct{})}
}

func (inbound *Inbound) createChannel() chan Frame {
	return make(chan Frame, queueSize)
}

func (inbound *Inbound) AddLocalSocket(mac []byte, socket Isocket) chan Frame {

	macAddr := string(mac)
	var channel chan Frame

	inbound.Lock()

	if inChannel, ok := inbound.channels[macAddr]; ok {

		if _, ok := inbound.sockets[macAddr]; !ok {
			channel = inChannel
			inbound.sockets[macAddr] = socket
			GC.AddSocket(socket, macAddr)
			go inbound.pollSocket(socket)
		}
	} else {
		channel = inbound.createChannel()
		inbound.channels[macAddr] = channel
		inbound.sockets[macAddr] = socket
		GC.AddSocket(socket, macAddr)
		go inbound.pollSocket(socket)
	}

	registerUniqSocket(net.HardwareAddr(mac).String(), socket)

	inbound.Unlock()
	inLog.Println("Setup local socket for ", net.HardwareAddr(mac), ".")
	return channel
}

func (inbound *Inbound) SetRemoteSocket(socket Isocket) {
	inbound.Lock()
	inbound.remoteSocket = socket
	registerUniqSocket("vxlan0", socket)
	inbound.Unlock()
	go inbound.pollSocket(socket)
}

func (inbound *Inbound) addPendingSocket(sock Isocket) {
	inbound.Lock()
	inbound.pendingSock = append(inbound.pendingSock, sock)
	inbound.Unlock()
}

func (inbound *Inbound) removePendingSocket(sock Isocket) {
	inbound.Lock()

	index := -1

	for i, auxSock := range inbound.pendingSock {
		if auxSock.ID() == sock.ID() {
			index = i
			break
		}
	}
	// Removes the element from the list by replacing with the last element of the list
	if index != -1 {
		inbound.pendingSock[index] = inbound.pendingSock[len(inbound.pendingSock)-1]
		inbound.pendingSock = inbound.pendingSock[:len(inbound.pendingSock)-1]
	}

	inbound.Unlock()
}

func (inbound *Inbound) RemoveSocket(mac []byte) {
	if sock, ok := inbound.sockets[string(mac)]; ok {
		delete(inbound.sockets, string(mac))
		sock.Close()
		delete(inbound.channels, string(mac))
		GC.RemoveSocket(string(mac))
	}
}

func (inbound *Inbound) BootstrapSocket(sock Isocket) chan Frame {

	var frames []Frame
	inbound.addPendingSocket(sock)

	for len(frames) == 0 {
		frames = sock.Receive()
	}

	inLog.Println("Found MAC address: ", net.HardwareAddr(frames[0].macOrigin))

	mac := frames[0].macOrigin

	broadcast := string(ConvertMacStringToBytes(broadcastMacAddress))

	for _, i := range frames {
		if i.macDestination != broadcast {
			inbound.queue <- i
		} else {
			Out.channel <- i
		}
	}

	newChan := inbound.AddLocalSocket([]byte(mac), sock)

	inbound.removePendingSocket(sock)

	return newChan
}

func (inbound *Inbound) GetIncomingChannel(mac []byte) (chan Frame, bool) {
	inbound.Lock()
	socket, ok := inbound.channels[string(mac)]
	inbound.Unlock()
	return socket, ok
}

func (inbound *Inbound) AddRemoteMac(mac []byte) chan Frame {
	inbound.Lock()
	macAddr := string(mac)
	if channel, ok := inbound.channels[macAddr]; ok {
		inbound.Unlock()
		return channel
	} else {
		channel := inbound.createChannel()
		inbound.channels[macAddr] = channel
		GC.AddSocket(inbound.remoteSocket, string(mac))
		inbound.Unlock()
		return channel
	}
}

func (inbound *Inbound) AddMac(mac []byte) chan Frame {
	inbound.Lock()
	macAddr := string(mac)
	if channel, ok := inbound.channels[macAddr]; ok {
		inbound.Unlock()
		return channel
	} else {
		channel := inbound.createChannel()
		inbound.channels[macAddr] = channel
		inbound.Unlock()
		return channel
	}
}

func (inbound *Inbound) pollSocket(socket Isocket) {
	//TODO: Pass this to somewhere else
	for {
		for _, frame := range socket.Receive() {
			if len(inbound.queue) < queueSize {
				inbound.queue <- frame
			} else {
				GC.GCFrame(frame)
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

			inbound.Lock()
			channel, ok := inbound.channels[frame.macOrigin]
			inbound.Unlock()
			if ok {
				channel <- frame
			} else {
				GC.GCFrame(frame)
				fmt.Println("Something stupid is going on")
			}
		}
	}
}

func (inbound *Inbound) Stop() {
	inbound.ctx <- struct{}{}
}

func (inbound *Inbound) CloseSockets() {
	inbound.Lock()
	inbound.running = false
	inbound.Stop()
	for _, sock := range inbound.sockets {
		sock.Close()
	}
	for _, sock := range inbound.pendingSock {
		sock.Close()
	}
	inbound.remoteSocket.Close()

	inbound.Unlock()
}
