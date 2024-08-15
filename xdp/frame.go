package xdp

import (
	"time"
)

type Frame struct {
	FramePointer   []byte    `json:"framePointer"`
	Time           time.Time `json:"time"`
	MacOrigin      string    `json:"macOrigin"`
	MacDestination string    `json:"macDestination"`
}

func CreateFrame(framePointer []byte, time time.Time, macOrigin, macDestination string) *Frame {
	return &Frame{
		FramePointer:   framePointer,
		Time:           time,
		MacOrigin:      macOrigin,
		MacDestination: macDestination,
	}
}

func (frame *Frame) GetTime() time.Time {
	return frame.Time
}

func (frame *Frame) AddTime(time time.Duration) {
	frame.Time = frame.Time.Add(time)
}

func (frame *Frame) Frame() []byte {
	return frame.FramePointer
}

func (frame *Frame) GetMacOrigin() string {
	return frame.MacOrigin
}
func (frame *Frame) GetMacDestination() string {
	return frame.MacDestination
}
