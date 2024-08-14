module gitea.homelab-antunes.duckdns.org/emu-socket

go 1.22.5

replace github.com/asavie/xdp => ./asavie

require (
	github.com/asavie/xdp v0.0.0-00010101000000-000000000000
	github.com/google/uuid v1.6.0
	github.com/vishvananda/netlink v1.1.0
	golang.org/x/sys v0.24.0
)

require (
	github.com/cilium/ebpf v0.4.0 // indirect
	github.com/vishvananda/netns v0.0.0-20191106174202-0a2b9b5464df // indirect
)
