package main

import (
	"fmt"
	"log"

	"github.com/ziomciopoziomcio/digital-music-stand/server/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	dsn := "host=localhost user=devuser password=devpassword dbname=musicstand port=5432 sslmode=disable"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to the database: %v", err)
	}

	fmt.Println("Connected to the database")

	err = db.AutoMigrate(
		&models.User{},
		&models.Score{},
		&models.Band{},
		&models.BandMember{},
		&models.SharedBandScore{},
		&models.SharedUserScore{},
		&models.Concert{},
		&models.ConcertItem{},
	)

	if err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}
	fmt.Println("Migrated database")
}
