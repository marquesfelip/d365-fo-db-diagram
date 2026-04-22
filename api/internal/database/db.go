package database

import (
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect(dsn string) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("database connection failed: %s\n", err.Error())
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("error: %s\n", err.Error())
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	DB = db

	log.Println("Postgres connected")
}
