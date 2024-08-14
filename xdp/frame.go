package xdp

import (
	"time"
)

type Frame struct {
	framePointer   []byte
	time           time.Time
	macOrigin      string
	macDestination string
}

func CreateFrame(framePointer []byte, time time.Time, macOrigin, macDestination string) *Frame {
	return &Frame{
		framePointer:   framePointer,
		time:           time,
		macOrigin:      macOrigin,
		macDestination: macDestination,
	}
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
