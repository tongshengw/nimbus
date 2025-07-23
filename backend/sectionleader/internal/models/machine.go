package models

import (
	"time"

	"github.com/tongshengw/nimbus/backend/sectionleader/internal/shared"
)

type MachineData struct {
	ID             uint               `gorm:"primaryKey;autoIncrement"`
	UUID           shared.MachineUUID `gorm:"not null"`
	Name           string             `gorm:"not null"`
	LocalIp        shared.Ipv4        `gorm:"not null"`
	SshPort        int                `gorm:"not null"`
	CreationTime   time.Time          `gorm:"not null"`
	ForwardedPorts []ForwardedPort    `gorm:"foreignKey:MachineID"`
}

type ForwardedPort struct {
	ID          uint   `gorm:"primaryKey;autoIncrement"`
	MachineID   uint   `gorm:"foreignKey:ID"`
	MachinePort int    `gorm:"not null"`
	RemotePort  int    `gorm:"not null;unique"`
	Protocol    string `gorm:"not null"`
}
