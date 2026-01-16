package db

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/npinot/vibe/backend/internal/model"
)

func NewPostgresConnection(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	return db, nil
}

func RunMigrations(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{},
	)
}
