package main

import (
	"r-vBackend/internal/app/ds"
	"r-vBackend/internal/app/dsn"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	_ = godotenv.Load()
	db, err := gorm.Open(postgres.Open(dsn.PostgresFromEnv()), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	err = db.AutoMigrate(
		&ds.Stock{},
		&ds.User{},
	)
	if err != nil {
		panic("cant migrate db")
	}

	// Check if there is a default user row and add one if it's missing
	var count int64

	db.Model(&ds.User{}).Where("login = ?", "test").Count(&count)
	if count == 0 {
		defaultUser := ds.User{Login: "test", Password: "test123", IsModerator: false}

		err := db.Create(&defaultUser).Error

		if err != nil {
			panic("error creating default user")
		}
	}
}
