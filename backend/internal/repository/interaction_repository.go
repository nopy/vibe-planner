package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/npinot/vibe/backend/internal/model"
)

type InteractionRepository interface {
	Create(ctx context.Context, interaction *model.Interaction) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Interaction, error)
	FindByTaskID(ctx context.Context, taskID uuid.UUID) ([]model.Interaction, error)
	FindBySessionID(ctx context.Context, sessionID uuid.UUID) ([]model.Interaction, error)
	DeleteByTaskID(ctx context.Context, taskID uuid.UUID) error
}

type interactionRepository struct {
	db *gorm.DB
}

func NewInteractionRepository(db *gorm.DB) InteractionRepository {
	return &interactionRepository{db: db}
}

func (r *interactionRepository) Create(ctx context.Context, interaction *model.Interaction) error {
	if interaction.ID == uuid.Nil {
		interaction.ID = uuid.New()
	}

	now := time.Now()
	if interaction.CreatedAt.IsZero() {
		interaction.CreatedAt = now
	}
	if interaction.UpdatedAt.IsZero() {
		interaction.UpdatedAt = now
	}

	if err := r.db.WithContext(ctx).Create(interaction).Error; err != nil {
		return fmt.Errorf("failed to create interaction: %w", err)
	}

	return nil
}

func (r *interactionRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Interaction, error) {
	var interaction model.Interaction
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&interaction).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to find interaction: %w", err)
	}

	return &interaction, nil
}

func (r *interactionRepository) FindByTaskID(ctx context.Context, taskID uuid.UUID) ([]model.Interaction, error) {
	var interactions []model.Interaction
	if err := r.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Order("created_at ASC").
		Find(&interactions).Error; err != nil {
		return nil, fmt.Errorf("failed to find interactions by task ID: %w", err)
	}

	return interactions, nil
}

func (r *interactionRepository) FindBySessionID(ctx context.Context, sessionID uuid.UUID) ([]model.Interaction, error) {
	var interactions []model.Interaction
	if err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Find(&interactions).Error; err != nil {
		return nil, fmt.Errorf("failed to find interactions by session ID: %w", err)
	}

	return interactions, nil
}

func (r *interactionRepository) DeleteByTaskID(ctx context.Context, taskID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("task_id = ?", taskID).Delete(&model.Interaction{}).Error; err != nil {
		return fmt.Errorf("failed to delete interactions by task ID: %w", err)
	}

	return nil
}
