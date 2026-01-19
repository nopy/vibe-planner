package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/npinot/vibe/backend/internal/model"
)

type SessionRepository interface {
	Create(ctx context.Context, session *model.Session) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Session, error)
	FindByTaskID(ctx context.Context, taskID uuid.UUID) ([]model.Session, error)
	FindActiveSessionsForProject(ctx context.Context, projectID uuid.UUID) ([]model.Session, error)
	FindAllActiveSessions(ctx context.Context) ([]model.Session, error)
	Update(ctx context.Context, session *model.Session) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.SessionStatus) error
	UpdateOutput(ctx context.Context, id uuid.UUID, output string) error
	UpdateLastEventID(ctx context.Context, id uuid.UUID, lastEventID string) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

type sessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) SessionRepository {
	return &sessionRepository{db: db}
}

func (r *sessionRepository) Create(ctx context.Context, session *model.Session) error {
	if session.ID == uuid.Nil {
		session.ID = uuid.New()
	}

	if err := r.db.WithContext(ctx).Create(session).Error; err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

func (r *sessionRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Session, error) {
	var session model.Session
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to find session: %w", err)
	}

	return &session, nil
}

func (r *sessionRepository) FindByTaskID(ctx context.Context, taskID uuid.UUID) ([]model.Session, error) {
	var sessions []model.Session
	if err := r.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Order("created_at DESC").
		Find(&sessions).Error; err != nil {
		return nil, fmt.Errorf("failed to find sessions by task ID: %w", err)
	}

	return sessions, nil
}

func (r *sessionRepository) FindActiveSessionsForProject(ctx context.Context, projectID uuid.UUID) ([]model.Session, error) {
	var sessions []model.Session
	if err := r.db.WithContext(ctx).
		Where("project_id = ? AND status IN ?", projectID, []model.SessionStatus{
			model.SessionStatusPending,
			model.SessionStatusRunning,
		}).
		Order("created_at DESC").
		Find(&sessions).Error; err != nil {
		return nil, fmt.Errorf("failed to find active sessions for project: %w", err)
	}

	return sessions, nil
}

func (r *sessionRepository) Update(ctx context.Context, session *model.Session) error {
	if err := r.db.WithContext(ctx).Save(session).Error; err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

func (r *sessionRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.SessionStatus) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if err := r.db.WithContext(ctx).
		Model(&model.Session{}).
		Where("id = ?", id).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update session status: %w", err)
	}

	return nil
}

func (r *sessionRepository) UpdateOutput(ctx context.Context, id uuid.UUID, output string) error {
	updates := map[string]interface{}{
		"output": output,
	}

	if err := r.db.WithContext(ctx).
		Model(&model.Session{}).
		Where("id = ?", id).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update session output: %w", err)
	}

	return nil
}

func (r *sessionRepository) UpdateLastEventID(ctx context.Context, id uuid.UUID, lastEventID string) error {
	updates := map[string]interface{}{
		"last_event_id": lastEventID,
	}

	if err := r.db.WithContext(ctx).
		Model(&model.Session{}).
		Where("id = ?", id).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update last event ID: %w", err)
	}

	return nil
}

func (r *sessionRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Session{}).Error; err != nil {
		return fmt.Errorf("failed to soft delete session: %w", err)
	}

	return nil
}

func (r *sessionRepository) FindAllActiveSessions(ctx context.Context) ([]model.Session, error) {
	var sessions []model.Session

	if err := r.db.WithContext(ctx).
		Where("status IN ?", []string{"pending", "running", "waiting_input"}).
		Find(&sessions).Error; err != nil {
		return nil, fmt.Errorf("failed to find all active sessions: %w", err)
	}

	return sessions, nil
}
