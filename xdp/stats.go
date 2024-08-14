package xdp

import (
	"sync"

	"github.com/google/uuid"
	"golang.org/x/sys/unix"
)

type Stats struct {
	Filled      uint64
	Received    uint64
	Transmitted uint64
	Completed   uint64
	KernelStats unix.XDPStatistics
}

type statsManager struct {
	sync.Mutex
	sockets map[string]Isocket
}

var stats = statsManager{sync.Mutex{}, make(map[string]Isocket)}

func registerSocket(socket Isocket) {
	stats.Lock()
	id := uuid.New().String()
	stats.sockets[id] = socket
	stats.Unlock()
}

func registerUniqSocket(id string, socket Isocket) {
	stats.Lock()
	if _, ok := stats.sockets[id]; !ok {
		stats.sockets[id] = socket
	}
	stats.Unlock()
}

func GetStatSocketList() []string {
	stats.Lock()
	ids := make([]string, 0, len(stats.sockets))

	for id := range stats.sockets {
		ids = append(ids, id)
	}
	stats.Unlock()
	return ids
}

func GetStatsFromSocket(id string) Stats {
	stats.Lock()
	if socket, ok := stats.sockets[id]; ok {
		stats.Unlock()
		return socket.Stats()
	} else {
		stats.Unlock()
		return Stats{}
	}
}

func GetStatsFromAllSockets() map[string]Stats {
	stats.Lock()
	statData := make(map[string]Stats)

	for id, socket := range stats.sockets {
		statData[id] = socket.Stats()
	}
	stats.Unlock()
	return statData
}
