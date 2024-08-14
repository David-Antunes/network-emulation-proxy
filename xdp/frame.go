package xdp

import (
	"time"

	"github.com/asavie/xdp"
)

type Frame struct {
	framePointer   []byte
	time           time.Time
	macOrigin      string
	macDestination string
	umemAddr       xdp.Desc
}

func (frame *Frame) Time() time.Time {
	return frame.time
}

func (frame *Frame) AddTime(time time.Duration) {
	frame.time = frame.time.Add(time)
}

func (frame *Frame) Frame() []byte {
	return frame.framePointer
}

func (frame *Frame) MacOrigin() string {
	return frame.macOrigin
}
func (frame *Frame) MacDestination() string {
	return frame.macDestination
}

func (frame *Frame) UmemAddr() xdp.Desc {
	return frame.umemAddr
}
