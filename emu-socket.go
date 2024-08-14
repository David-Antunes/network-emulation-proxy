package main

import (
	"gitea.homelab-antunes.duckdns.org/emu-socket/xdp"
	"os"

	unixsocket "gitea.homelab-antunes.duckdns.org/emu-socket/unix-socket"
)

func main() {
	os.Remove("/tmp/emu.sock")
	unixsocket.SetSocketPath("/tmp/emu.sock")
	unixsocket.StartSocket()
	outbound := xdp.CreateOutbound(unixsocket.GetReadChannel())
	inbound := xdp.CreateInbound(unixsocket.GetWriteChannel(), outbound)

	inbound.Start()
	outbound.Start()
}
