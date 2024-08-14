package main

import (
	"os"

	unixsocket "gitea.homelab-antunes.duckdns.org/emu-socket/unix-socket"
)

func main() {
	os.Remove("/tmp/emu.sock")
	// Create a Unix domain socket and listen for incoming connections.
	unixsocket.SetSocketPath("/tmp/emu.sock")
	unixsocket.StartSocket()
}
