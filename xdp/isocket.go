package xdp

type Isocket interface {
	ID() string
	SendFrame(*Frame)
	Send([]*Frame)
	Receive() ([]*Frame, error)
	Close()
}
