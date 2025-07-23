package models

import (
	"time"

	"github.com/tongshengw/nimbus/backend/sectionleader/internal/shared"
)

type MachineData struct {
	Id           shared.MachineUUID
	Name         string
	LocalIp      shared.Ipv4
	RemotePort   int
	CreationTime time.Time
}
