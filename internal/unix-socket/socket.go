package unixsocket

import (
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/David-Antunes/network-emulation-proxy/internal"
	"github.com/David-Antunes/network-emulation-proxy/internal/inbound"

	"github.com/David-Antunes/network-emulation-proxy/xdp"

	"net"
	"os"
)

type socket struct {
	socketPath string
	sock       net.Listener
	in         *inbound.Inbound
	read       chan *xdp.Frame
	conn       net.Conn
	closed     bool
}

var s = &socket{
	socketPath: "",
	sock:       nil,
	in:         nil,
	read:       make(chan *xdp.Frame, internal.GOB_QUEUESIZE),
	conn:       nil,
	closed:     false,
}

func SetInbound(in *inbound.Inbound) {
	s.in = in
}

func GetReadChannel() chan *xdp.Frame {
	return s.read
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
			fmt.Println(err)
			return nil
		}
		enc := gob.NewEncoder(conn)
		dec := gob.NewDecoder(conn)
		s.in.SetEnc(enc)
		for {
			var frame *xdp.Frame

			err = dec.Decode(&frame)
			if err != nil {
				fmt.Println(err)
				break
			}
			if len(s.read) < internal.GOB_QUEUESIZE {
				s.read <- frame
			}
		}
	}
}

func Close() {
	if s.conn != nil {
		err := s.conn.Close()
		if err != nil {
			return
		}
	}
	err := os.Remove(s.socketPath)
	if err != nil {
		return
	}
}
