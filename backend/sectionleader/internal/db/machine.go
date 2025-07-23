package db

import (
	"github.com/tongshengw/nimbus/backend/sectionleader/internal/models"
	"github.com/tongshengw/nimbus/backend/sectionleader/internal/shared"
)

func CreateMachine(machine *models.MachineData) error {
	return DB.Create(machine).Error
}

func GetMachineByUUID(uuid shared.MachineUUID) ([]models.MachineData, error) {
	var machines []models.MachineData
	err := DB.Where("UUID = ?", uuid).Find(&machines).Error
	return machines, err
}

func GetAllMachines() ([]models.MachineData, error) {
	var machines []models.MachineData
	err := DB.Find(&machines).Error
	return machines, err
}

func UpdateMachine(machine *models.MachineData) error {
	return DB.Save(machine).Error
}

func DeleteMachine(id shared.MachineUUID) error {
	return DB.Delete(&models.MachineData{}, id).Error
}