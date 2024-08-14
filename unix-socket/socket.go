package unixsocket

import (
	"errors"
	"log"
	"net"
	"os"
)

type socket struct {
	socketPath string
	sock       net.Listener
	read       chan []byte
	write      chan []byte
	conn       net.Conn
	closed     bool
}

var s = &socket{
	socketPath: "",
	sock:       nil,
	read:       make(chan []byte),
	write:      make(chan []byte),
	conn:       nil,
	closed:     false,
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
			s.read <- buf
		}
	}
}

func sendMsg(conn net.Conn) {
	for {
		select {
		case bytes := <-s.write:
			_, err := conn.Write(bytes)
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
