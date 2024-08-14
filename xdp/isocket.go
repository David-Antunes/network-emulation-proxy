package xdp

type Isocket interface {
	ID() string
	Stats() Stats
	SendFrame(Frame)
	Send([]Frame)
	Receive() []Frame
	Close()
	CleanFrameMem([]Frame)
}
