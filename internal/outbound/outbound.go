package outbound

import (
	"fmt"
	"github.com/David-Antunes/network-emulation-proxy/internal"
	"github.com/David-Antunes/network-emulation-proxy/xdp"
	"log"
	"net"
	"os"
	"sync"
	"syscall"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

type Outbound struct {
	sync.Mutex
	gateway chan *xdp.Frame
	running bool
	queue   chan *xdp.Frame
	ctx     chan struct{}
	fd      int
	addr    *syscall.SockaddrLinklayer
}

var outLog = log.New(os.Stdout, "outbound INFO: ", log.Ltime)

func CreateOutbound(gateway chan *xdp.Frame) *Outbound {
	return &Outbound{
		Mutex:   sync.Mutex{},
		gateway: gateway,
		running: false,
		queue:   make(chan *xdp.Frame, internal.QUEUE_SIZE),
		ctx:     make(chan struct{}),
	}
}

func (outbound *Outbound) SetSocket() {
	fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, int(htons(syscall.ETH_P_IP)))
	if err != nil {
		panic(err)
	}
	outbound.fd = fd

	ifi, err := net.InterfaceByName("br0")
	if err != nil {
		panic(err)
	}
	outbound.addr = &syscall.SockaddrLinklayer{
		Protocol: htons(syscall.ETH_P_IP),
		Ifindex:  ifi.Index,
	}
}
func htons(i uint16) uint16 {
	return (i<<8)&0xff00 | i>>8
}

func (outbound *Outbound) Stop() {
	if outbound.running {
		outbound.ctx <- struct{}{}
		outbound.ctx <- struct{}{}
		outbound.running = false
	}
}

func (outbound *Outbound) Close() {
	outbound.ctx <- struct{}{}
	outbound.ctx <- struct{}{}
	outbound.running = false
	err := syscall.Close(outbound.fd)
	if err != nil {
		fmt.Println(err)
	}
}

func (outbound *Outbound) Start() {
	if !outbound.running {
		outLog.Println("Starting...")
		outLog.Println("Spawned 4 send routines")
		outbound.running = true
		go outbound.send()
		go outbound.send()
		go outbound.send()
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
			outbound.queue <- frame
		}
	}
}

func (outbound *Outbound) send() {

	var packet gopacket.Packet
	var eth *layers.Ethernet
	var ip *layers.IPv4
	var tcpLayer gopacket.Layer
	var tcp *layers.TCP
	for {
		select {
		case <-outbound.ctx:
			return

		case frame := <-outbound.queue:

			packet = gopacket.NewPacket(frame.FramePointer[:frame.FrameSize], layers.LayerTypeEthernet, gopacket.Default)
			eth = packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
			if packet.Layer(layers.LayerTypeIPv4) != nil {
				ip = packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
				if tcpLayer = packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
					tcp = tcpLayer.(*layers.TCP)
					err := tcp.SetNetworkLayerForChecksum(ip)

					if err != nil {
						internal.ShutdownAndLog(err)
						continue
					}
				} else {
					if err := syscall.Sendto(outbound.fd, frame.FramePointer, 0, outbound.addr); err != nil {
						internal.ShutdownAndLog(err)
						continue
					}
					continue
				}
			} else {
				if err := syscall.Sendto(outbound.fd, frame.FramePointer, 0, outbound.addr); err != nil {
					internal.ShutdownAndLog(err)
					continue
				}
				continue
			}

			buf := gopacket.NewSerializeBuffer()
			if err := gopacket.SerializeLayers(buf, gopacket.SerializeOptions{ComputeChecksums: true, FixLengths: true}, eth, ip, tcp, gopacket.Payload(tcp.LayerPayload())); err != nil {
				internal.ShutdownAndLog(err)
				continue
			}

			if err := syscall.Sendto(outbound.fd, buf.Bytes(), 0, outbound.addr); err != nil {
				internal.ShutdownAndLog(err)
				continue
			}

		}
	}
}
