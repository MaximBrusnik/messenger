package pkg

import (
	"MessangerMax/internal/config"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"time"
)

const (
	maxRetries      = 5
	retryInterval   = 5 * time.Second
	connMaxLifetime = 30 * time.Minute
	connMaxIdleTime = 5 * time.Minute
	maxOpenConns    = 25
	maxIdleConns    = 10
)

func NewDB(config *config.Config) (*gorm.DB, error) {
	dsn := "host=" + config.DBHost +
		" user=" + config.DBUser +
		" password=" + config.DBPassword +
		" dbname=" + config.DBName +
		" port=" + config.DBPort +
		" sslmode=disable"

	var db *gorm.DB
	var err error

	for i := 0; i < maxRetries; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})

		if err == nil {
			sqlDB, _ := db.DB()
			sqlDB.SetConnMaxLifetime(connMaxLifetime)
			sqlDB.SetConnMaxIdleTime(connMaxIdleTime)
			sqlDB.SetMaxOpenConns(maxOpenConns)
			sqlDB.SetMaxIdleConns(maxIdleConns)
			err = sqlDB.Ping()
		}

		if err == nil {
			log.Println("Подключение к базе данных установлено")
			return db, nil
		}

		log.Printf("База не готова (попытка %d/%d): %v", i+1, maxRetries, err)
		time.Sleep(retryInterval)
	}

	return nil, fmt.Errorf("не удалось подключиться к БД после %d попыток: %w", maxRetries, err)
}
