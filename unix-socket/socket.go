package unixsocket

import (
	"encoding/gob"
	"errors"
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
		dec := gob.NewDecoder(conn)
		for {
			var frame *xdp.Frame

			err = dec.Decode(&frame)
			if err != nil {
				break
			}
			s.read <- frame
			//fmt.Println(net.HardwareAddr(frame.MacOrigin), net.HardwareAddr(frame.MacDestination), frame.Time)
		}
	}
}

func sendMsg(conn net.Conn) {
	enc := gob.NewEncoder(conn)
	for {
		select {
		case frame := <-s.write:

			err := enc.Encode(frame)
			if err != nil {
				return
			}
		}
	}
}

func Close() {
	err := s.conn.Close()
	if err != nil {
		return
	}
	err = os.Remove(s.socketPath)
	if err != nil {
		return
	}
}
