package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/npinot/vibe/backend/internal/model"
)

type TaskRepository interface {
	Create(ctx context.Context, task *model.Task) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Task, error)
	FindByProjectID(ctx context.Context, projectID uuid.UUID) ([]model.Task, error)
	Update(ctx context.Context, task *model.Task) error
	UpdateStatus(ctx context.Context, id uuid.UUID, newStatus model.TaskStatus) error
	UpdatePosition(ctx context.Context, id uuid.UUID, newPosition int) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

type taskRepository struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &taskRepository{db: db}
}

func (r *taskRepository) Create(ctx context.Context, task *model.Task) error {
	if task.ID == uuid.Nil {
		task.ID = uuid.New()
	}

	if err := r.db.WithContext(ctx).Create(task).Error; err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	return nil
}

func (r *taskRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Task, error) {
	var task model.Task
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to find task: %w", err)
	}

	return &task, nil
}

func (r *taskRepository) FindByProjectID(ctx context.Context, projectID uuid.UUID) ([]model.Task, error) {
	var tasks []model.Task
	if err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("position ASC").
		Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to find tasks by project ID: %w", err)
	}

	return tasks, nil
}

func (r *taskRepository) Update(ctx context.Context, task *model.Task) error {
	if err := r.db.WithContext(ctx).Save(task).Error; err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

func (r *taskRepository) UpdateStatus(ctx context.Context, id uuid.UUID, newStatus model.TaskStatus) error {
	updates := map[string]interface{}{
		"status": newStatus,
	}

	if err := r.db.WithContext(ctx).
		Model(&model.Task{}).
		Where("id = ?", id).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	return nil
}

func (r *taskRepository) UpdatePosition(ctx context.Context, id uuid.UUID, newPosition int) error {
	updates := map[string]interface{}{
		"position": newPosition,
	}

	if err := r.db.WithContext(ctx).
		Model(&model.Task{}).
		Where("id = ?", id).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update task position: %w", err)
	}

	return nil
}

func (r *taskRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Task{}).Error; err != nil {
		return fmt.Errorf("failed to soft delete task: %w", err)
	}

	return nil
}
