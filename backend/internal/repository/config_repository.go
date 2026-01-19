package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/npinot/vibe/backend/internal/model"
)

// ConfigRepository defines the interface for OpenCode configuration persistence
type ConfigRepository interface {
	// GetActiveConfig retrieves the active configuration for a project
	GetActiveConfig(ctx context.Context, projectID uuid.UUID) (*model.OpenCodeConfig, error)

	// CreateConfig creates a new configuration version (deactivates old configs in transaction)
	CreateConfig(ctx context.Context, config *model.OpenCodeConfig) error

	// GetConfigVersions lists all configuration versions for a project (newest first)
	GetConfigVersions(ctx context.Context, projectID uuid.UUID) ([]model.OpenCodeConfig, error)

	// GetConfigByVersion retrieves a specific version of configuration
	GetConfigByVersion(ctx context.Context, projectID uuid.UUID, version int) (*model.OpenCodeConfig, error)

	// DeleteConfig deletes a configuration version (only if not active)
	DeleteConfig(ctx context.Context, id uuid.UUID) error
}

type configRepository struct {
	db *gorm.DB
}

// NewConfigRepository creates a new instance of ConfigRepository
func NewConfigRepository(db *gorm.DB) ConfigRepository {
	return &configRepository{db: db}
}

// GetActiveConfig retrieves the active configuration for a project
func (r *configRepository) GetActiveConfig(ctx context.Context, projectID uuid.UUID) (*model.OpenCodeConfig, error) {
	var config model.OpenCodeConfig
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND is_active = true", projectID).
		First(&config).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get active config: %w", err)
	}

	return &config, nil
}

// CreateConfig creates a new configuration version
// This method handles version incrementing and deactivating old configs in a transaction
func (r *configRepository) CreateConfig(ctx context.Context, config *model.OpenCodeConfig) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Deactivate all existing configs for this project
		if err := tx.Model(&model.OpenCodeConfig{}).
			Where("project_id = ?", config.ProjectID).
			Update("is_active", false).Error; err != nil {
			return fmt.Errorf("failed to deactivate old configs: %w", err)
		}

		// Get next version number
		var maxVersion int
		tx.Model(&model.OpenCodeConfig{}).
			Where("project_id = ?", config.ProjectID).
			Select("COALESCE(MAX(version), 0)").
			Scan(&maxVersion)

		config.Version = maxVersion + 1
		config.IsActive = true

		// Generate ID if not set
		if config.ID == uuid.Nil {
			config.ID = uuid.New()
		}

		// Create the new config
		if err := tx.Create(config).Error; err != nil {
			return fmt.Errorf("failed to create config: %w", err)
		}

		return nil
	})
}

// GetConfigVersions lists all configuration versions for a project (newest first)
func (r *configRepository) GetConfigVersions(ctx context.Context, projectID uuid.UUID) ([]model.OpenCodeConfig, error) {
	var configs []model.OpenCodeConfig
	err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("version DESC").
		Find(&configs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get config versions: %w", err)
	}

	return configs, nil
}

// GetConfigByVersion retrieves a specific version of configuration
func (r *configRepository) GetConfigByVersion(ctx context.Context, projectID uuid.UUID, version int) (*model.OpenCodeConfig, error) {
	var config model.OpenCodeConfig
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND version = ?", projectID, version).
		First(&config).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get config by version: %w", err)
	}

	return &config, nil
}

// DeleteConfig deletes a configuration version (only if not active)
func (r *configRepository) DeleteConfig(ctx context.Context, id uuid.UUID) error {
	// Check if config is active first
	var config model.OpenCodeConfig
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return gorm.ErrRecordNotFound
		}
		return fmt.Errorf("failed to find config: %w", err)
	}

	if config.IsActive {
		return errors.New("cannot delete active configuration")
	}

	// Delete the config
	if err := r.db.WithContext(ctx).Delete(&model.OpenCodeConfig{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete config: %w", err)
	}

	return nil
}
