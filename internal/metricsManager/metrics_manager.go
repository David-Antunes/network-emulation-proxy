package metricsManager

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/David-Antunes/network-emulation-proxy/api"
	"github.com/David-Antunes/network-emulation-proxy/xdp"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"net"
	"syscall"
	"time"
)

type MetricsManager struct {
	iface           xdp.Isocket
	mac             net.HardwareAddr
	ip              net.IP
	port            int
	fd              int
	addr            *syscall.SockaddrLinklayer
	socketPath      string
	receiveLatency  time.Duration
	transmitLatency time.Duration
	tests           []api.RTTRequest
	currConnection  *api.StartTestRequest
}

func NewMetricsManager(unixPath string, iface xdp.Isocket, mac net.HardwareAddr, ip net.IP, port int) *MetricsManager {

	fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, int(htons(syscall.ETH_P_IP)))
	if err != nil {
		fmt.Println(err)
		syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		return nil
	}

	ifi, err := net.InterfaceByName("veth1")
	if err != nil {
		fmt.Println(err)
		syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		return nil
	}
	addr := &syscall.SockaddrLinklayer{
		Protocol: htons(syscall.ETH_P_IP),
		Ifindex:  ifi.Index,
	}
	return &MetricsManager{
		iface:           iface,
		mac:             mac,
		ip:              ip,
		port:            port,
		fd:              fd,
		addr:            addr,
		socketPath:      unixPath,
		receiveLatency:  0,
		transmitLatency: 0,
		tests:           make([]api.RTTRequest, 5),
		currConnection:  nil,
	}
}

func htons(i uint16) uint16 {
	return (i<<8)&0xff00 | i>>8
}

func (manager *MetricsManager) Close() {
	manager.iface.Close()
	err := syscall.Close(manager.fd)
	if err != nil {
		fmt.Println(err)
	}
}

func (manager *MetricsManager) Start() {

	frames, err := manager.iface.Receive()

	if err != nil {
		fmt.Println(err)
		syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		return
	}
	for {

		for _, frame := range frames {
			startTest := &api.StartTestRequest{}
			err = receive(frame.FramePointer[:frame.FrameSize], startTest)
			if err != nil {
				continue
			} else {
				manager.currConnection = startTest
				break
			}

		}
		for i := 5; i < 5; i++ {
			_, err = manager.sendTest()
			if err != nil {
				break
			}
			frames, err = manager.iface.Receive()
			req := &api.RTTRequest{}
			for _, frame := range frames {
				err = receive(frame.FramePointer[:frame.FrameSize], req)
				if err != nil {
					continue
				} else {
					break
				}
			}
			if req == nil {
				break
			}
			req.EndTime = time.Now()
			manager.tests = append(manager.tests, *req)
		}
		if len(manager.tests) == 5 {
			break
		}
	}

}

func (manager *MetricsManager) calculateAvg() {
	var accReceive time.Duration
	var accTransmit time.Duration

	for _, test := range manager.tests {
		accReceive += test.ReceiveTime.Sub(test.StartTime)
		accTransmit += test.EndTime.Sub(test.TransmitTime)
	}

	manager.receiveLatency = accReceive / 5
	manager.transmitLatency = accTransmit / 5
}

func (manager *MetricsManager) sendTest() (api.RTTRequest, error) {
	req, err := json.Marshal(&api.RTTRequest{
		StartTime:    time.Now(),
		ReceiveTime:  time.Time{},
		TransmitTime: time.Time{},
		EndTime:      time.Time{},
	})
	if err != nil {
		return api.RTTRequest{}, err
	}
	buf := gopacket.NewSerializeBuffer()
	var layersToSerialize []gopacket.SerializableLayer

	// Automatically include the Ethernet layer if MAC addresses are provided
	ethLayer := &layers.Ethernet{
		SrcMAC:       manager.mac,
		DstMAC:       manager.currConnection.Mac,
		EthernetType: layers.EthernetTypeIPv4,
	}
	layersToSerialize = append(layersToSerialize, ethLayer)

	// Set IP layer
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
	udpLayer.SetNetworkLayerForChecksum(ipLayer) // Important for checksum calculation
	layersToSerialize = append(layersToSerialize, udpLayer)

	// Optionally, fill the payload with data
	layersToSerialize = append(layersToSerialize, gopacket.Payload(req))

	if err = gopacket.SerializeLayers(buf, gopacket.SerializeOptions{ComputeChecksums: true, FixLengths: true}, layersToSerialize...); err != nil {
		fmt.Println(err)
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

func receive(payload []byte, request any) error {
	packet := gopacket.NewPacket(payload, layers.LayerTypeIPv4, gopacket.Default)
	if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil {
		udp := udpLayer.(*layers.UDP)

		d := json.NewDecoder(bytes.NewReader(udp.Payload))
		err := d.Decode(&request)
		if err != nil {
			return err
		}
		return nil
	} else {
		return errors.New("received tcp packet")
	}
}

func (manager *MetricsManager) Publish() {

}
