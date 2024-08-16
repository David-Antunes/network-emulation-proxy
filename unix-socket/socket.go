package unixsocket

import (
	"encoding/json"
	"errors"
	"fmt"
	"gitea.homelab-antunes.duckdns.org/emu-socket/xdp"
	"log"
	"net"
	"os"
)

type socket struct {
	socketPath string
	sock       net.Listener
	read       chan *xdp.Frame
	write      chan *xdp.Frame
	conn       net.Conn
	closed     bool
}

var s = &socket{
	socketPath: "",
	sock:       nil,
	read:       make(chan *xdp.Frame, 1000),
	write:      make(chan *xdp.Frame, 1000),
	conn:       nil,
	closed:     false,
}

func GetReadChannel() chan *xdp.Frame {
	return s.read
}
func GetWriteChannel() chan *xdp.Frame {
	return s.write
}

func SetSocketPath(path string) {

	s.socketPath = path
	listen, err := net.Listen("unix", path)
	s.sock = listen
	if err != nil {
		panic(err)
	}
}

func StartSocket() error {
	if s.socketPath == "" {
		return errors.New("socket path not defined")
	}

	for {
		if s.closed {
			return nil
		}
		conn, err := s.sock.Accept()
		s.conn = conn
		if err != nil {
			log.Fatal(err)
		}
		go sendMsg(conn)
		for {
			buf := make([]byte, 2868)

			// Read data from the connection.
			n, err := conn.Read(buf)
			buf = buf[:n]
			if err != nil {
				break
			}
			frame := &xdp.Frame{}
			err = json.Unmarshal(buf, frame)
			if err != nil {
				break
			}
			s.read <- frame
			fmt.Println(net.HardwareAddr(frame.MacOrigin), net.HardwareAddr(frame.MacDestination), frame.Time)
		}
	}
}

func sendMsg(conn net.Conn) {
	for {
		select {
		case frame := <-s.write:
			bytes, err := json.Marshal(frame)
			if err != nil {
				continue
			}
			_, err = conn.Write(bytes)
			if err != nil {
				return
			}
		}
	}
}

func Close() {
	s.conn.Close()
	os.Remove(s.socketPath)
}
