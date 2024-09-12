package metricsManager

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/David-Antunes/network-emulation-proxy/api"
	"github.com/David-Antunes/network-emulation-proxy/internal"
	"github.com/David-Antunes/network-emulation-proxy/internal/conn"
	"github.com/David-Antunes/network-emulation-proxy/xdp"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"net"
	"runtime/debug"
	"syscall"
	"time"
)

// Magic number to align json
var magicNumber = 34

type MetricsManager struct {
	iface           xdp.Isocket
	mac             net.HardwareAddr
	ip              net.IP
	port            int
	fd              int
	addr            *syscall.SockaddrLinklayer
	receiveLatency  time.Duration
	transmitLatency time.Duration
	tests           []api.RTTRequest
	currConnection  *conn.RttConnection
	metricsSocket   *MetricsSocket
	timeout         time.Duration
	numTests        int
}

func NewMetricsManager(iface xdp.Isocket, mac net.HardwareAddr, ip net.IP, port int, endpoint *conn.RttConnection, socketPath string, timeout time.Duration, numTests int) *MetricsManager {

	fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, int(htons(syscall.ETH_P_IP)))
	if err != nil {
		internal.ShutdownAndLog(err)
		return nil
	}

	ifi, err := net.InterfaceByName("veth1")
	if err != nil {
		internal.ShutdownAndLog(err)
		return nil
	}
	addr := &syscall.SockaddrLinklayer{
		Protocol: htons(syscall.ETH_P_IP),
		Ifindex:  ifi.Index,
	}

	metricsSocket, err := NewMetricsSocket(socketPath)
	if err != nil {
		internal.ShutdownAndLog(err)
	}

	return &MetricsManager{
		iface:           iface,
		mac:             mac,
		ip:              ip,
		port:            port,
		fd:              fd,
		addr:            addr,
		receiveLatency:  0,
		transmitLatency: 0,
		tests:           make([]api.RTTRequest, 5),
		currConnection:  endpoint,
		metricsSocket:   metricsSocket,
		timeout:         timeout,
		numTests:        numTests,
	}
}

func htons(i uint16) uint16 {
	return (i<<8)&0xff00 | i>>8
}

func (manager *MetricsManager) Close() {
	manager.iface.Close()
	manager.metricsSocket.Close()
	err := syscall.Close(manager.fd)
	if err != nil {
		fmt.Println(err)
	}
}

func (manager *MetricsManager) Start() {

	var frames []*xdp.Frame
	var err error

	for {
		manager.tests = make([]api.RTTRequest, 0, manager.numTests)

		for i := 0; i < 5; i++ {
			_, err = manager.sendTest()
			if err != nil {
				fmt.Println(err)
				break
			}
			frames, err = manager.iface.Receive(int(time.Second.Milliseconds()))
			if len(frames) == 0 {
				fmt.Println("No frames received")
				break
			}
			req := &api.RTTRequest{}
			for _, frame := range frames {
				err = receive(frame.FramePointer[:frame.FrameSize], req)
				if err != nil {
					fmt.Println(err)
					continue
				} else {
					break
				}
			}
			req.EndTime = time.Now()

			//fmt.Println("StartTime:", req.StartTime)
			//fmt.Println("ReceiveTime:", req.ReceiveTime)
			//fmt.Println("TransmitTime:", req.TransmitTime)
			//fmt.Println("EndTime:", req.EndTime)
			manager.tests = append(manager.tests, *req)
		}
		if len(manager.tests) == manager.numTests {
			manager.calculateAvg()
		}

		time.Sleep(manager.timeout)
	}

}

func (manager *MetricsManager) calculateAvg() {
	var accReceive time.Duration
	var accTransmit time.Duration

	for _, test := range manager.tests {
		accReceive += test.ReceiveTime.Sub(test.StartTime)
		accTransmit += test.EndTime.Sub(test.TransmitTime)
	}

	manager.receiveLatency = accReceive / time.Duration(manager.numTests)
	manager.transmitLatency = accTransmit / time.Duration(manager.numTests)
	fmt.Println("receiveLtency:", manager.receiveLatency)
	fmt.Println("transmitLatency:", manager.transmitLatency)
}

func (manager *MetricsManager) sendTest() (api.RTTRequest, error) {
	req, err := json.Marshal(&api.RTTRequest{
		StartTime:    time.Now(),
		ReceiveTime:  time.Time{},
		TransmitTime: time.Time{},
		EndTime:      time.Time{},
	})
	if err != nil {
		fmt.Println(req)
		return api.RTTRequest{}, err
	}
	buf := gopacket.NewSerializeBuffer()
	var layersToSerialize []gopacket.SerializableLayer

	ethLayer := &layers.Ethernet{
		SrcMAC:       manager.mac,
		DstMAC:       manager.currConnection.Mac,
		EthernetType: layers.EthernetTypeIPv4,
	}
	layersToSerialize = append(layersToSerialize, ethLayer)

	ipLayer := &layers.IPv4{
		Version:  4,
		TTL:      64,
		SrcIP:    manager.ip,
		DstIP:    manager.currConnection.IP,
		Protocol: layers.IPProtocolUDP,
	}
	layersToSerialize = append(layersToSerialize, ipLayer)
	udpLayer := &layers.UDP{
		SrcPort: layers.UDPPort(manager.port),
		DstPort: layers.UDPPort(manager.currConnection.Port),
	}
	udpLayer.SetNetworkLayerForChecksum(ipLayer)
	layersToSerialize = append(layersToSerialize, udpLayer)

	layersToSerialize = append(layersToSerialize, gopacket.Payload(req))

	if err = gopacket.SerializeLayers(buf, gopacket.SerializeOptions{ComputeChecksums: true, FixLengths: true}, layersToSerialize...); err != nil {
		fmt.Println(err)
		debug.PrintStack()
		syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		return api.RTTRequest{}, err
	}

	if err = syscall.Sendto(manager.fd, buf.Bytes(), 0, manager.addr); err != nil {
		fmt.Println(err)
		syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		return api.RTTRequest{}, err
	}
	return api.RTTRequest{}, nil
}

func receive(payload []byte, request *api.RTTRequest) error {
	packet := gopacket.NewPacket(payload, layers.LayerTypeUDP, gopacket.Default)

	if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil {
		udp := udpLayer.(*layers.UDP)

		d := json.NewDecoder(bytes.NewReader(udp.Payload[magicNumber:]))
		err := d.Decode(request)
		if err != nil {
			return err
		}
		return nil
	} else {
		return errors.New("received tcp packet")
	}
}

func (manager *MetricsManager) Publish() {
	manager.metricsSocket.sendRTT(manager.receiveLatency, manager.transmitLatency)
}
