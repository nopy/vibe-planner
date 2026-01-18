package db

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/npinot/vibe/backend/internal/model"
)

func NewPostgresConnection(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  databaseURL,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	return db, nil
}

func RunMigrations(db *gorm.DB) error {
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		return fmt.Errorf("failed to create uuid extension: %w", err)
	}

	return db.AutoMigrate(
		&model.User{},
		&model.Project{},
	)
}
