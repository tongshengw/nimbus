package db

import (
	"github.com/tongshengw/nimbus/backend/sectionleader/internal/models"
	"github.com/tongshengw/nimbus/backend/sectionleader/internal/shared"
)

func CreateMachine(machine *models.MachineData) error {
	return DB.Create(machine).Error
}

func GetMachineByUUID(id shared.MachineUUID) ([]models.MachineData, error) {
	var machines []models.MachineData
	err := DB.Where("Id = ?", id).Find(&machines).Error
	return machines, err
}
