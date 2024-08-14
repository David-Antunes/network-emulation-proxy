package unixsocket

import (
	"errors"
	"gitea.homelab-antunes.duckdns.org/emu-socket/xdp"
	"log"
	"net"
	"os"
	"time"
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
	read:       make(chan *xdp.Frame),
	write:      make(chan *xdp.Frame),
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
			buf := make([]byte, 2048)

			// Read data from the connection.
			n, err := conn.Read(buf)
			buf = buf[:n]
			if err != nil {
				break
			}

			s.read <- xdp.CreateFrame(buf, time.Now(), "", "")
		}
	}
}

func sendMsg(conn net.Conn) {
	for {
		select {
		case frame := <-s.write:
			_, err := conn.Write(frame.Frame())
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
