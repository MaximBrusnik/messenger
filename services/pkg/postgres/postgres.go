package postgres

import (
	"errors"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"messengermax/pkg/config"
)

func New(cfg config.Postgres) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable connect_timeout=5",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name)

	const maxRetries = 10
	var db *gorm.DB
	var err error
	for i := 1; i <= maxRetries; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Info)})
		if err == nil {
			sql, perr := db.DB()
			if perr == nil {
				if perr = sql.Ping(); perr == nil {
					break
				}
				err = perr
			}
		}
		log.Printf("postgres: retry %d/%d connecting: %v", i, maxRetries, err)
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		return nil, errors.New("postgres: failed to connect: " + err.Error())
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Minute)
	return db, nil
}
