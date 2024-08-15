package xdp

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/david-antunes/xdp"
)

const queueSize = 10000

const gcSize = 10

const batchSize = 256

const broadcastMacAddress = "ff:ff:ff:ff:ff:ff"

var DefaultSocketOptions = xdp.SocketOptions{
	NumFrames:              8192,
	FrameSize:              4096,
	FillRingNumDescs:       4096,
	CompletionRingNumDescs: 4096,
	RxRingNumDescs:         4096,
	TxRingNumDescs:         4096,
}

//var DefaultXdpFlags = int(unix.XDP_FLAGS_SKB_MODE)

func ConvertMacStringToBytes(macAddr string) []byte {
	parts := strings.Split(macAddr, ":")
	var macBytes []byte

	for _, part := range parts {
		b, err := strconv.ParseUint(part, 16, 8)
		if err != nil {
			fmt.Printf("Error parsing %s: %v\n", part, err)
			panic(err)
		}
		macBytes = append(macBytes, byte(b))

	}
	return macBytes
}
