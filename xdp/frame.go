package xdp

type Frame struct {
	FramePointer   []byte `json:"framePointer"`
	FrameSize      int    `json:"frameSize"`
	Time           int64  `json:"time"`
	MacOrigin      string `json:"macOrigin"`
	MacDestination string `json:"macDestination"`
}

func CreateFrame(framePointer []byte, frameSize int, time int64, macOrigin, macDestination string) *Frame {
	return &Frame{
		FramePointer:   framePointer,
		FrameSize:      frameSize,
		Time:           time,
		MacOrigin:      macOrigin,
		MacDestination: macDestination,
	}
}

func (frame *Frame) GetTime() int64 {
	return frame.Time
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
