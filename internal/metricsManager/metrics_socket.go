package metricsManager

import (
	"encoding/gob"
	"fmt"
	"time"

	"github.com/David-Antunes/network-emulation-proxy/api"
	"github.com/David-Antunes/network-emulation-proxy/xdp"

	"net"
	"os"
)

type MetricsSocket struct {
	socketPath string
	sock       net.Listener
	read       chan *xdp.Frame
	write      chan *xdp.Frame
	conn       []net.Conn
	closed     bool
}

func NewMetricsSocket(socketPath string) (*MetricsSocket, error) {
	listen, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, err
	}
	return &MetricsSocket{
		socketPath: socketPath,
		sock:       listen,
		conn:       make([]net.Conn, 0),
		closed:     false,
	}, nil
}

func (s *MetricsSocket) StartSocket() error {

	for {
		conn, err := s.sock.Accept()
		if err != nil {
			fmt.Println(err)
			return nil
		}
		s.conn = append(s.conn, conn)
	}
}

func (s *MetricsSocket) sendRTT(receiveLatency time.Duration, transmitLatency time.Duration) {
	currentConnections := make([]net.Conn, 0)
	for _, conn := range s.conn {
		enc := gob.NewEncoder(conn)

		err := enc.Encode(&api.UpdateRTTRequest{
			ReceiveLatency:  receiveLatency,
			TransmitLatency: transmitLatency,
		})

		if err == nil {
			currentConnections = append(currentConnections, conn)
		} else {
			conn.Close()
			fmt.Println(err)
		}
	}
	s.conn = currentConnections
	metricsLog.Println("Published results")
}

func (s *MetricsSocket) Close() {

	for _, conn := range s.conn {
		err := conn.Close()
		if err != nil {
			continue
		}
	}
	err := os.Remove(s.socketPath)
	if err != nil {
		return
	}
}
