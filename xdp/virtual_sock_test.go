package xdp

import (
	"testing"
	"time"
)

func TestSendFrame(t *testing.T) {
	mac_a := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x0a}
	mac_b := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x0b}
	sock := CreateVirtSocket(string(mac_a))

	sock.InjectFrame(string(mac_b))

	frame := sock.Receive()[0]

	if frame.MacOrigin != string(mac_a) {
		t.Fatal("Wrong origin mac!")
	} else if frame.MacDestination != string(mac_b) {
		t.Fatal("Wrong destination mac")
	}
}

func TestWithInbound(t *testing.T) {
	mac_a := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x0a}
	mac_b := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x0b}

	sock_a := CreateVirtSocket(string(mac_a))
	In.Start()

	In.AddLocalSocket(mac_a, sock_a)
	sock_chan, ok := In.GetIncomingChannel(mac_a)

	if !ok {
		t.Fatal("Failed to retrieve Frame channel from inbound!")
	}

	sock_a.InjectFrame(string(mac_b))
	frame := <-sock_chan

	if frame.macOrigin != string(mac_a) {
		t.Fatal("Wrong origin mac!")
	} else if frame.macDestination != string(mac_b) {
		t.Fatal("Wrong destination mac")
	}
}

func TestWithOutbound(t *testing.T) {
	mac_a := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x0a}
	mac_b := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x0b}
	mac_nil := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	sock_a := CreateVirtSocket(string(mac_a))
	out_sock := CreateVirtSocket(string(mac_nil))

	Out.SetSocket(out_sock)
	In.Start()
	Out.Start()

	In.AddLocalSocket(mac_a, sock_a)

	sock_chan, ok := In.GetIncomingChannel(mac_a)

	if !ok {
		t.Fatal("Failed to retrieve Frame channel from inbound!")
	}

	sock_a.InjectFrame(string(mac_b))

	frame := <-sock_chan

	Out.channel <- frame

	time.Sleep(time.Second)

	result := out_sock.ReceivedFrames[0]

	if result.MacOrigin != frame.macOrigin || result.MacDestination != frame.macDestination {
		t.Fatal("Frames don't match!")
	}
}
