package api

import (
	"net"
)

type StartTestRequest struct {
	Mac  net.HardwareAddr `json:"mac"`
	IP   net.IP           `json:"ip"`
	Port int              `json:"port"`
}
