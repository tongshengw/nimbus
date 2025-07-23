package db

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"github.com/tongshengw/nimbus/backend/sectionleader/internal/models"
)

var DB *gorm.DB

func Init() error {
	var err error
	DB, err = gorm.Open(sqlite.Open("app.db"), &gorm.Config{})
	if err != nil {
		return err
	}
	
	// Auto-migrate the schema
	err = DB.AutoMigrate(&models.MachineData{})
	if err != nil {
		return err
	}
	err = DB.AutoMigrate(&models.ForwardedPort{})
	if err != nil {
		return err
	}

	return nil
}
