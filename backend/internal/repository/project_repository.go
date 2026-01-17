package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/npinot/vibe/backend/internal/model"
)

type ProjectRepository interface {
	Create(ctx context.Context, project *model.Project) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Project, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Project, error)
	Update(ctx context.Context, project *model.Project) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	UpdatePodStatus(ctx context.Context, id uuid.UUID, status string, podError string) error
}

type projectRepository struct {
	db *gorm.DB
}

func NewProjectRepository(db *gorm.DB) ProjectRepository {
	return &projectRepository{db: db}
}

func (r *projectRepository) Create(ctx context.Context, project *model.Project) error {
	if project.ID == uuid.Nil {
		project.ID = uuid.New()
	}

	if err := r.db.WithContext(ctx).Create(project).Error; err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	return nil
}

func (r *projectRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Project, error) {
	var project model.Project
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to find project: %w", err)
	}

	return &project, nil
}

func (r *projectRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Project, error) {
	var projects []model.Project
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&projects).Error; err != nil {
		return nil, fmt.Errorf("failed to find projects by user ID: %w", err)
	}

	return projects, nil
}

func (r *projectRepository) Update(ctx context.Context, project *model.Project) error {
	if err := r.db.WithContext(ctx).Save(project).Error; err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	return nil
}

func (r *projectRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Project{}).Error; err != nil {
		return fmt.Errorf("failed to soft delete project: %w", err)
	}

	return nil
}

func (r *projectRepository) UpdatePodStatus(ctx context.Context, id uuid.UUID, status string, podError string) error {
	updates := map[string]interface{}{
		"pod_status": status,
	}

	if podError != "" {
		updates["pod_error"] = podError
	}

	if err := r.db.WithContext(ctx).
		Model(&model.Project{}).
		Where("id = ?", id).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update pod status: %w", err)
	}

	return nil
}
