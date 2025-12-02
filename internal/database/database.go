package database

import (
	"log"

	"forumapp/internal/config"
	"forumapp/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

// Initialize sets up the database connection and runs migrations
func Initialize(cfg *config.Config) *gorm.DB {
	var err error
	DB, err = gorm.Open(sqlite.Open(cfg.DatabasePath), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database: ", err)
	}

	// AutoMigrate the schema
	err = DB.AutoMigrate(
		&models.User{},
		&models.Game{},
		&models.Tag{},
		&models.Post{},
		&models.Comment{},
	)
	if err != nil {
		log.Fatal("failed to migrate database: ", err)
	}

	return DB
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return DB
}
