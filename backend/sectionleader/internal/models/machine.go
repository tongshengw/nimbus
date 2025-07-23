package models

import (
	"time"

	"github.com/tongshengw/nimbus/backend/sectionleader/internal/shared"
)

type MachineData struct {
	Id             shared.MachineUUID
	Name           string
	LocalIp        shared.Ipv4
	RemotePort     int
	CreationTime   time.Time
	ForwardedPorts []ForwardedPort
}

// ForwardedPort is a port that is forwarded from the machine to host, then FRP will proxy the port from Hostport to Remoteport
type ForwardedPort struct {
	MachinePort int
	HostPort    int
	RemotePort  int
	Protocol    string
}
