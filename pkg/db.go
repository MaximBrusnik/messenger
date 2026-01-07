package pkg

import (
	"MessangerMax/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
)

func NewDB(config *config.Config) (*gorm.DB, error) {
	dsn := "host=" + config.DBHost +
		" user=" + config.DBUser +
		" password=" + config.DBPassword +
		" dbname=" + config.DBName +
		" port=" + config.DBPort +
		" sslmode=disable"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		log.Printf("Не удалось подключиться к базе данных: %v", err)
		return nil, err
	}

	log.Println("Подключение к базе данных установлено")
	return db, nil
}
