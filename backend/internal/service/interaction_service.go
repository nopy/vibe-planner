package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/npinot/vibe/backend/internal/model"
	"github.com/npinot/vibe/backend/internal/repository"
)

var (
	ErrInteractionNotFound   = errors.New("interaction not found")
	ErrInvalidMessageType    = errors.New("invalid message type")
	ErrInvalidMessageContent = errors.New("invalid message content")
	ErrTaskNotOwnedByUser    = errors.New("task not owned by user")
	// ErrSessionNotFound is imported from session_service.go (shared sentinel error)
)

// Valid message types
const (
	MessageTypeUser   = "user_message"
	MessageTypeAgent  = "agent_response"
	MessageTypeSystem = "system_notification"
)

type InteractionService interface {
	// CreateUserMessage stores a user message and returns the created interaction
	CreateUserMessage(ctx context.Context, taskID, userID uuid.UUID, content string, metadata model.JSONB) (*model.Interaction, error)

	// CreateAgentResponse stores an agent response message
	CreateAgentResponse(ctx context.Context, taskID, userID uuid.UUID, sessionID uuid.UUID, content string, metadata model.JSONB) (*model.Interaction, error)

	// CreateSystemNotification stores a system notification message
	CreateSystemNotification(ctx context.Context, taskID, userID uuid.UUID, sessionID *uuid.UUID, content string, metadata model.JSONB) (*model.Interaction, error)

	// GetTaskHistory retrieves all interactions for a task with authorization
	GetTaskHistory(ctx context.Context, taskID, userID uuid.UUID) ([]model.Interaction, error)

	// GetSessionHistory retrieves all interactions for a session with authorization
	GetSessionHistory(ctx context.Context, sessionID, userID uuid.UUID) ([]model.Interaction, error)

	// DeleteTaskHistory deletes all interactions for a task with authorization
	DeleteTaskHistory(ctx context.Context, taskID, userID uuid.UUID) error

	// ValidateTaskOwnership validates that a user owns the task
	ValidateTaskOwnership(ctx context.Context, taskID, userID uuid.UUID) error
}

type interactionService struct {
	interactionRepo repository.InteractionRepository
	taskRepo        repository.TaskRepository
	projectRepo     repository.ProjectRepository
	sessionRepo     repository.SessionRepository
}

func NewInteractionService(
	interactionRepo repository.InteractionRepository,
	taskRepo repository.TaskRepository,
	projectRepo repository.ProjectRepository,
	sessionRepo repository.SessionRepository,
) InteractionService {
	return &interactionService{
		interactionRepo: interactionRepo,
		taskRepo:        taskRepo,
		projectRepo:     projectRepo,
		sessionRepo:     sessionRepo,
	}
}

func (s *interactionService) CreateUserMessage(ctx context.Context, taskID, userID uuid.UUID, content string, metadata model.JSONB) (*model.Interaction, error) {
	if err := s.ValidateTaskOwnership(ctx, taskID, userID); err != nil {
		return nil, err
	}

	if err := validateMessageContent(content, MessageTypeUser); err != nil {
		return nil, err
	}

	interaction := &model.Interaction{
		TaskID:      taskID,
		UserID:      userID,
		SessionID:   nil,
		MessageType: MessageTypeUser,
		Content:     content,
		Metadata:    metadata,
	}

	if err := s.interactionRepo.Create(ctx, interaction); err != nil {
		return nil, fmt.Errorf("failed to create user message: %w", err)
	}

	return interaction, nil
}

// SECURITY WARNING: Only call from trusted internal components (session-proxy sidecar).
// External user-facing API endpoints must NOT expose this directly.
// Phase 7.3 will add internal authentication for agent response creation.
func (s *interactionService) CreateAgentResponse(ctx context.Context, taskID, userID uuid.UUID, sessionID uuid.UUID, content string, metadata model.JSONB) (*model.Interaction, error) {
	if err := s.ValidateTaskOwnership(ctx, taskID, userID); err != nil {
		return nil, err
	}

	if err := validateMessageContent(content, MessageTypeAgent); err != nil {
		return nil, err
	}

	if err := s.validateSessionBelongsToTask(ctx, sessionID, taskID); err != nil {
		return nil, err
	}

	interaction := &model.Interaction{
		TaskID:      taskID,
		UserID:      userID,
		SessionID:   &sessionID,
		MessageType: MessageTypeAgent,
		Content:     content,
		Metadata:    metadata,
	}

	if err := s.interactionRepo.Create(ctx, interaction); err != nil {
		return nil, fmt.Errorf("failed to create agent response: %w", err)
	}

	return interaction, nil
}

func (s *interactionService) CreateSystemNotification(ctx context.Context, taskID, userID uuid.UUID, sessionID *uuid.UUID, content string, metadata model.JSONB) (*model.Interaction, error) {
	if err := s.ValidateTaskOwnership(ctx, taskID, userID); err != nil {
		return nil, err
	}

	if err := validateMessageContent(content, MessageTypeSystem); err != nil {
		return nil, err
	}

	if sessionID != nil {
		if err := s.validateSessionBelongsToTask(ctx, *sessionID, taskID); err != nil {
			return nil, err
		}
	}

	interaction := &model.Interaction{
		TaskID:      taskID,
		UserID:      userID,
		SessionID:   sessionID,
		MessageType: MessageTypeSystem,
		Content:     content,
		Metadata:    metadata,
	}

	if err := s.interactionRepo.Create(ctx, interaction); err != nil {
		return nil, fmt.Errorf("failed to create system notification: %w", err)
	}

	return interaction, nil
}

// GetTaskHistory retrieves all interactions for a task with authorization
func (s *interactionService) GetTaskHistory(ctx context.Context, taskID, userID uuid.UUID) ([]model.Interaction, error) {
	// Validate task ownership
	if err := s.ValidateTaskOwnership(ctx, taskID, userID); err != nil {
		return nil, err
	}

	// Fetch interactions
	interactions, err := s.interactionRepo.FindByTaskID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve task history: %w", err)
	}

	return interactions, nil
}

// GetSessionHistory retrieves all interactions for a session with authorization
func (s *interactionService) GetSessionHistory(ctx context.Context, sessionID, userID uuid.UUID) ([]model.Interaction, error) {
	// Validate session exists
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to retrieve session: %w", err)
	}

	// Validate task ownership via session's task
	if err := s.ValidateTaskOwnership(ctx, session.TaskID, userID); err != nil {
		return nil, err
	}

	// Fetch interactions
	interactions, err := s.interactionRepo.FindBySessionID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve session history: %w", err)
	}

	return interactions, nil
}

// DeleteTaskHistory deletes all interactions for a task with authorization
func (s *interactionService) DeleteTaskHistory(ctx context.Context, taskID, userID uuid.UUID) error {
	// Validate task ownership
	if err := s.ValidateTaskOwnership(ctx, taskID, userID); err != nil {
		return err
	}

	// Delete interactions
	if err := s.interactionRepo.DeleteByTaskID(ctx, taskID); err != nil {
		return fmt.Errorf("failed to delete task history: %w", err)
	}

	return nil
}

// ValidateTaskOwnership validates that a user owns the task
func (s *interactionService) ValidateTaskOwnership(ctx context.Context, taskID, userID uuid.UUID) error {
	// Retrieve task
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTaskNotFound
		}
		return fmt.Errorf("failed to retrieve task: %w", err)
	}

	// Retrieve project to check ownership
	project, err := s.projectRepo.FindByID(ctx, task.ProjectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrProjectNotFound
		}
		return fmt.Errorf("failed to retrieve project: %w", err)
	}

	// Check ownership
	if project.UserID != userID {
		return ErrTaskNotOwnedByUser
	}

	return nil
}

// validateSessionBelongsToTask validates that a session belongs to the specified task
func (s *interactionService) validateSessionBelongsToTask(ctx context.Context, sessionID, taskID uuid.UUID) error {
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSessionNotFound
		}
		return fmt.Errorf("failed to retrieve session: %w", err)
	}

	if session.TaskID != taskID {
		return fmt.Errorf("session does not belong to task")
	}

	return nil
}

// User messages: 2,000 char limit (TODO.md spec)
// Agent/System messages: 50,000 char limit (prevent abuse while allowing verbose AI responses)
func validateMessageContent(content string, messageType string) error {
	trimmed := strings.TrimSpace(content)

	if trimmed == "" {
		return fmt.Errorf("%w: content cannot be empty", ErrInvalidMessageContent)
	}

	maxLength := 50000
	if messageType == MessageTypeUser {
		maxLength = 2000
	}

	if len(content) > maxLength {
		return fmt.Errorf("%w: content exceeds maximum length of %d characters", ErrInvalidMessageContent, maxLength)
	}

	return nil
}
