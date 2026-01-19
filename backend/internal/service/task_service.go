package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/npinot/vibe/backend/internal/model"
	"github.com/npinot/vibe/backend/internal/repository"
)

var (
	ErrTaskNotFound           = errors.New("task not found")
	ErrInvalidTaskTitle       = errors.New("invalid task title")
	ErrInvalidTaskPriority    = errors.New("invalid task priority")
	ErrInvalidStateTransition = errors.New("invalid state transition")
)

// validTransitions defines the state machine for task status transitions
var validTransitions = map[model.TaskStatus][]model.TaskStatus{
	model.TaskStatusTodo:        {model.TaskStatusInProgress},
	model.TaskStatusInProgress:  {model.TaskStatusAIReview, model.TaskStatusTodo},
	model.TaskStatusAIReview:    {model.TaskStatusHumanReview, model.TaskStatusInProgress},
	model.TaskStatusHumanReview: {model.TaskStatusDone, model.TaskStatusInProgress},
	model.TaskStatusDone:        {model.TaskStatusTodo}, // Allow reopening
}

// TaskService defines business logic operations for task management
type TaskService interface {
	// CreateTask creates a new task with validation and authorization
	CreateTask(ctx context.Context, projectID, userID uuid.UUID, title, description string, priority model.TaskPriority) (*model.Task, error)

	// GetTask retrieves a task by ID with authorization check
	GetTask(ctx context.Context, id, userID uuid.UUID) (*model.Task, error)

	// ListProjectTasks retrieves all tasks for a project with authorization
	ListProjectTasks(ctx context.Context, projectID, userID uuid.UUID) ([]model.Task, error)

	// UpdateTask updates task fields with authorization check
	UpdateTask(ctx context.Context, id, userID uuid.UUID, updates map[string]interface{}) (*model.Task, error)

	// MoveTask moves a task to a new state and/or position with state machine validation
	MoveTask(ctx context.Context, id, userID uuid.UUID, newState model.TaskStatus, newPosition int) (*model.Task, error)

	// DeleteTask soft deletes a task with authorization check
	DeleteTask(ctx context.Context, id, userID uuid.UUID) error

	// ExecuteTask starts execution of a task via OpenCode session
	ExecuteTask(ctx context.Context, id, userID uuid.UUID) (*model.Session, error)

	// StopTask stops execution of a task
	StopTask(ctx context.Context, id, userID uuid.UUID) error

	// GetTaskSessions returns execution history for a task
	GetTaskSessions(ctx context.Context, id, userID uuid.UUID) ([]model.Session, error)
}

type taskService struct {
	taskRepo       repository.TaskRepository
	projectRepo    repository.ProjectRepository
	sessionService SessionService
}

// NewTaskService creates a new task service
func NewTaskService(taskRepo repository.TaskRepository, projectRepo repository.ProjectRepository, sessionService SessionService) TaskService {
	return &taskService{
		taskRepo:       taskRepo,
		projectRepo:    projectRepo,
		sessionService: sessionService,
	}
}

// CreateTask creates a new task with validation and authorization
func (s *taskService) CreateTask(ctx context.Context, projectID, userID uuid.UUID, title, description string, priority model.TaskPriority) (*model.Task, error) {
	// Validate input
	if err := validateTaskTitle(title); err != nil {
		return nil, err
	}

	if err := validateTaskPriority(priority); err != nil {
		return nil, err
	}

	// Check user owns project
	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to retrieve project: %w", err)
	}

	if project.UserID != userID {
		return nil, ErrUnauthorized
	}

	// Get current tasks to determine position (append to TODO column)
	tasks, err := s.taskRepo.FindByProjectID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks for position calculation: %w", err)
	}

	// Calculate position as max position + 1 for TODO status
	position := 0
	for _, task := range tasks {
		if task.Status == model.TaskStatusTodo && task.Position >= position {
			position = task.Position + 1
		}
	}

	// Create task entity
	task := &model.Task{
		ProjectID:   projectID,
		Title:       title,
		Description: description,
		Status:      model.TaskStatusTodo,
		Position:    position,
		Priority:    priority,
		CreatedBy:   userID,
	}

	// Save to database
	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create task in database: %w", err)
	}

	return task, nil
}

// GetTask retrieves a task with authorization check
func (s *taskService) GetTask(ctx context.Context, id, userID uuid.UUID) (*model.Task, error) {
	task, err := s.taskRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("failed to retrieve task: %w", err)
	}

	// Authorization check - verify user owns the project
	project, err := s.projectRepo.FindByID(ctx, task.ProjectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to retrieve project for authorization: %w", err)
	}

	if project.UserID != userID {
		return nil, ErrUnauthorized
	}

	return task, nil
}

// ListProjectTasks retrieves all tasks for a project with authorization
func (s *taskService) ListProjectTasks(ctx context.Context, projectID, userID uuid.UUID) ([]model.Task, error) {
	// Check user owns project
	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to retrieve project: %w", err)
	}

	if project.UserID != userID {
		return nil, ErrUnauthorized
	}

	// Fetch tasks
	tasks, err := s.taskRepo.FindByProjectID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	return tasks, nil
}

// UpdateTask updates task fields with authorization check
func (s *taskService) UpdateTask(ctx context.Context, id, userID uuid.UUID, updates map[string]interface{}) (*model.Task, error) {
	// Retrieve and authorize
	task, err := s.GetTask(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	// Validate and apply updates
	if title, ok := updates["title"].(string); ok {
		if err := validateTaskTitle(title); err != nil {
			return nil, err
		}
		task.Title = title
	}

	if description, ok := updates["description"].(string); ok {
		task.Description = description
	}

	if priorityStr, ok := updates["priority"].(string); ok {
		priority := model.TaskPriority(priorityStr)
		if err := validateTaskPriority(priority); err != nil {
			return nil, err
		}
		task.Priority = priority
	}

	// Update in database
	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	return task, nil
}

// MoveTask moves a task to a new state and/or position with state machine validation
func (s *taskService) MoveTask(ctx context.Context, id, userID uuid.UUID, newState model.TaskStatus, newPosition int) (*model.Task, error) {
	// Retrieve and authorize
	task, err := s.GetTask(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	// Validate state transition
	if task.Status != newState {
		if !isValidTransition(task.Status, newState) {
			return nil, fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidStateTransition, task.Status, newState)
		}
	}

	// Update status and position
	task.Status = newState
	task.Position = newPosition

	// Update in database
	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to move task: %w", err)
	}

	return task, nil
}

// DeleteTask soft deletes a task with authorization check
func (s *taskService) DeleteTask(ctx context.Context, id, userID uuid.UUID) error {
	// Retrieve and authorize
	_, err := s.GetTask(ctx, id, userID)
	if err != nil {
		return err
	}

	// Soft delete in database
	if err := s.taskRepo.SoftDelete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete task from database: %w", err)
	}

	return nil
}

// validateTaskTitle validates task title constraints
func validateTaskTitle(title string) error {
	if title == "" {
		return fmt.Errorf("%w: title cannot be empty", ErrInvalidTaskTitle)
	}

	if len(title) > 255 {
		return fmt.Errorf("%w: title cannot exceed 255 characters", ErrInvalidTaskTitle)
	}

	return nil
}

// validateTaskPriority validates task priority is a valid enum value
func validateTaskPriority(priority model.TaskPriority) error {
	switch priority {
	case model.TaskPriorityLow, model.TaskPriorityMedium, model.TaskPriorityHigh:
		return nil
	default:
		return fmt.Errorf("%w: must be 'low', 'medium', or 'high'", ErrInvalidTaskPriority)
	}
}

// isValidTransition checks if a state transition is allowed by the state machine
func isValidTransition(currentState, newState model.TaskStatus) bool {
	allowed, exists := validTransitions[currentState]
	if !exists {
		return false
	}

	for _, s := range allowed {
		if s == newState {
			return true
		}
	}

	return false
}

// ExecuteTask starts execution of a task via OpenCode session
func (s *taskService) ExecuteTask(ctx context.Context, id, userID uuid.UUID) (*model.Session, error) {
	task, err := s.GetTask(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if task.Status != model.TaskStatusTodo {
		return nil, fmt.Errorf("%w: can only execute tasks in TODO state, current state: %s", ErrInvalidStateTransition, task.Status)
	}

	prompt := fmt.Sprintf("Task: %s\n\nDescription:\n%s", task.Title, task.Description)

	session, err := s.sessionService.StartSession(ctx, task.ID, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to start session: %w", err)
	}

	task.Status = model.TaskStatusInProgress
	if err := s.taskRepo.UpdateStatus(ctx, task.ID, task.Status); err != nil {
		return nil, fmt.Errorf("failed to update task status: %w", err)
	}

	return session, nil
}

// StopTask stops execution of a task
func (s *taskService) StopTask(ctx context.Context, id, userID uuid.UUID) error {
	task, err := s.GetTask(ctx, id, userID)
	if err != nil {
		return err
	}

	if task.Status != model.TaskStatusInProgress {
		return fmt.Errorf("%w: can only stop tasks in IN_PROGRESS state, current state: %s", ErrInvalidStateTransition, task.Status)
	}

	activeSessions, err := s.sessionService.GetSessionsByTaskID(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("failed to get task sessions: %w", err)
	}

	var activeSessionID *uuid.UUID
	for _, session := range activeSessions {
		if session.Status == model.SessionStatusRunning || session.Status == model.SessionStatusPending {
			activeSessionID = &session.ID
			break
		}
	}

	if activeSessionID == nil {
		return fmt.Errorf("no active session found for task")
	}

	if err := s.sessionService.StopSession(ctx, *activeSessionID); err != nil {
		return fmt.Errorf("failed to stop session: %w", err)
	}

	task.Status = model.TaskStatusTodo
	if err := s.taskRepo.UpdateStatus(ctx, task.ID, task.Status); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	return nil
}

func (s *taskService) GetTaskSessions(ctx context.Context, id, userID uuid.UUID) ([]model.Session, error) {
	task, err := s.GetTask(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	sessions, err := s.sessionService.GetSessionsByTaskID(ctx, task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task sessions: %w", err)
	}

	return sessions, nil
}
