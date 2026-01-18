package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"

	"github.com/npinot/vibe/backend/internal/model"
	"github.com/npinot/vibe/backend/internal/repository"
)

type MockTaskRepository struct {
	mock.Mock
}

func (m *MockTaskRepository) Create(ctx context.Context, task *model.Task) error {
	args := m.Called(ctx, task)
	return args.Error(0)
}

func (m *MockTaskRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Task, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Task), args.Error(1)
}

func (m *MockTaskRepository) FindByProjectID(ctx context.Context, projectID uuid.UUID) ([]model.Task, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Task), args.Error(1)
}

func (m *MockTaskRepository) Update(ctx context.Context, task *model.Task) error {
	args := m.Called(ctx, task)
	return args.Error(0)
}

func (m *MockTaskRepository) UpdateStatus(ctx context.Context, id uuid.UUID, newStatus model.TaskStatus) error {
	args := m.Called(ctx, id, newStatus)
	return args.Error(0)
}

func (m *MockTaskRepository) UpdatePosition(ctx context.Context, id uuid.UUID, newPosition int) error {
	args := m.Called(ctx, id, newPosition)
	return args.Error(0)
}

func (m *MockTaskRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

var _ repository.TaskRepository = (*MockTaskRepository)(nil)

func TestTaskService_CreateTask(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	projectID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mockTaskRepo := new(MockTaskRepository)
		mockProjectRepo := new(MockProjectRepository)

		project := &model.Project{
			ID:     projectID,
			UserID: userID,
			Name:   "Test Project",
		}

		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)
		mockTaskRepo.On("FindByProjectID", ctx, projectID).Return([]model.Task{}, nil)
		mockTaskRepo.On("Create", ctx, mock.AnythingOfType("*model.Task")).Return(nil)

		svc := NewTaskService(mockTaskRepo, mockProjectRepo)
		task, err := svc.CreateTask(ctx, projectID, userID, "Test Task", "Description", model.TaskPriorityMedium)

		assert.NoError(t, err)
		assert.NotNil(t, task)
		assert.Equal(t, "Test Task", task.Title)
		assert.Equal(t, "Description", task.Description)
		assert.Equal(t, model.TaskStatusTodo, task.Status)
		assert.Equal(t, model.TaskPriorityMedium, task.Priority)
		assert.Equal(t, 0, task.Position)
		assert.Equal(t, userID, task.CreatedBy)
		mockProjectRepo.AssertExpectations(t)
		mockTaskRepo.AssertExpectations(t)
	})

	t.Run("empty title", func(t *testing.T) {
		mockTaskRepo := new(MockTaskRepository)
		mockProjectRepo := new(MockProjectRepository)

		svc := NewTaskService(mockTaskRepo, mockProjectRepo)
		task, err := svc.CreateTask(ctx, projectID, userID, "", "Description", model.TaskPriorityMedium)

		assert.Error(t, err)
		assert.Nil(t, task)
		assert.ErrorIs(t, err, ErrInvalidTaskTitle)
		assert.Contains(t, err.Error(), "title cannot be empty")
		mockProjectRepo.AssertNotCalled(t, "FindByID")
		mockTaskRepo.AssertNotCalled(t, "Create")
	})

	t.Run("title exceeds max length", func(t *testing.T) {
		mockTaskRepo := new(MockTaskRepository)
		mockProjectRepo := new(MockProjectRepository)

		longTitle := string(make([]byte, 256))
		for i := range longTitle {
			longTitle = string(append([]byte(longTitle[:i]), 'a'))
		}

		svc := NewTaskService(mockTaskRepo, mockProjectRepo)
		task, err := svc.CreateTask(ctx, projectID, userID, longTitle, "Description", model.TaskPriorityMedium)

		assert.Error(t, err)
		assert.Nil(t, task)
		assert.ErrorIs(t, err, ErrInvalidTaskTitle)
		assert.Contains(t, err.Error(), "cannot exceed 255 characters")
	})

	t.Run("invalid priority", func(t *testing.T) {
		mockTaskRepo := new(MockTaskRepository)
		mockProjectRepo := new(MockProjectRepository)

		svc := NewTaskService(mockTaskRepo, mockProjectRepo)
		task, err := svc.CreateTask(ctx, projectID, userID, "Test Task", "Description", "invalid")

		assert.Error(t, err)
		assert.Nil(t, task)
		assert.ErrorIs(t, err, ErrInvalidTaskPriority)
		assert.Contains(t, err.Error(), "must be 'low', 'medium', or 'high'")
	})

	t.Run("project not found", func(t *testing.T) {
		mockTaskRepo := new(MockTaskRepository)
		mockProjectRepo := new(MockProjectRepository)

		mockProjectRepo.On("FindByID", ctx, projectID).Return(nil, gorm.ErrRecordNotFound)

		svc := NewTaskService(mockTaskRepo, mockProjectRepo)
		task, err := svc.CreateTask(ctx, projectID, userID, "Test Task", "Description", model.TaskPriorityMedium)

		assert.Error(t, err)
		assert.Nil(t, task)
		assert.ErrorIs(t, err, ErrProjectNotFound)
		mockProjectRepo.AssertExpectations(t)
		mockTaskRepo.AssertNotCalled(t, "Create")
	})

	t.Run("unauthorized - user does not own project", func(t *testing.T) {
		mockTaskRepo := new(MockTaskRepository)
		mockProjectRepo := new(MockProjectRepository)

		otherUserID := uuid.New()
		project := &model.Project{
			ID:     projectID,
			UserID: otherUserID,
			Name:   "Other User Project",
		}

		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)

		svc := NewTaskService(mockTaskRepo, mockProjectRepo)
		task, err := svc.CreateTask(ctx, projectID, userID, "Test Task", "Description", model.TaskPriorityMedium)

		assert.Error(t, err)
		assert.Nil(t, task)
		assert.ErrorIs(t, err, ErrUnauthorized)
		mockProjectRepo.AssertExpectations(t)
		mockTaskRepo.AssertNotCalled(t, "Create")
	})

	t.Run("position calculation with existing tasks", func(t *testing.T) {
		mockTaskRepo := new(MockTaskRepository)
		mockProjectRepo := new(MockProjectRepository)

		project := &model.Project{
			ID:     projectID,
			UserID: userID,
			Name:   "Test Project",
		}

		existingTasks := []model.Task{
			{ID: uuid.New(), Status: model.TaskStatusTodo, Position: 0},
			{ID: uuid.New(), Status: model.TaskStatusTodo, Position: 1},
			{ID: uuid.New(), Status: model.TaskStatusInProgress, Position: 0},
		}

		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)
		mockTaskRepo.On("FindByProjectID", ctx, projectID).Return(existingTasks, nil)
		mockTaskRepo.On("Create", ctx, mock.AnythingOfType("*model.Task")).Return(nil)

		svc := NewTaskService(mockTaskRepo, mockProjectRepo)
		task, err := svc.CreateTask(ctx, projectID, userID, "New Task", "Description", model.TaskPriorityLow)

		assert.NoError(t, err)
		assert.NotNil(t, task)
		assert.Equal(t, 2, task.Position)
	})

	t.Run("database error on create", func(t *testing.T) {
		mockTaskRepo := new(MockTaskRepository)
		mockProjectRepo := new(MockProjectRepository)

		project := &model.Project{
			ID:     projectID,
			UserID: userID,
			Name:   "Test Project",
		}

		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)
		mockTaskRepo.On("FindByProjectID", ctx, projectID).Return([]model.Task{}, nil)
		mockTaskRepo.On("Create", ctx, mock.AnythingOfType("*model.Task")).Return(errors.New("db error"))

		svc := NewTaskService(mockTaskRepo, mockProjectRepo)
		task, err := svc.CreateTask(ctx, projectID, userID, "Test Task", "Description", model.TaskPriorityHigh)

		assert.Error(t, err)
		assert.Nil(t, task)
		assert.Contains(t, err.Error(), "failed to create task in database")
	})
}

func TestTaskService_GetTask(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	projectID := uuid.New()
	taskID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mockTaskRepo := new(MockTaskRepository)
		mockProjectRepo := new(MockProjectRepository)

		task := &model.Task{
			ID:        taskID,
			ProjectID: projectID,
			Title:     "Test Task",
			Status:    model.TaskStatusTodo,
		}

		project := &model.Project{
			ID:     projectID,
			UserID: userID,
		}

		mockTaskRepo.On("FindByID", ctx, taskID).Return(task, nil)
		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)

		svc := NewTaskService(mockTaskRepo, mockProjectRepo)
		result, err := svc.GetTask(ctx, taskID, userID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, taskID, result.ID)
		assert.Equal(t, "Test Task", result.Title)
		mockTaskRepo.AssertExpectations(t)
		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("task not found", func(t *testing.T) {
		mockTaskRepo := new(MockTaskRepository)
		mockProjectRepo := new(MockProjectRepository)

		mockTaskRepo.On("FindByID", ctx, taskID).Return(nil, gorm.ErrRecordNotFound)

		svc := NewTaskService(mockTaskRepo, mockProjectRepo)
		result, err := svc.GetTask(ctx, taskID, userID)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrTaskNotFound)
		mockTaskRepo.AssertExpectations(t)
		mockProjectRepo.AssertNotCalled(t, "FindByID")
	})

	t.Run("unauthorized - user does not own project", func(t *testing.T) {
		mockTaskRepo := new(MockTaskRepository)
		mockProjectRepo := new(MockProjectRepository)

		task := &model.Task{
			ID:        taskID,
			ProjectID: projectID,
			Title:     "Test Task",
		}

		otherUserID := uuid.New()
		project := &model.Project{
			ID:     projectID,
			UserID: otherUserID,
		}

		mockTaskRepo.On("FindByID", ctx, taskID).Return(task, nil)
		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)

		svc := NewTaskService(mockTaskRepo, mockProjectRepo)
		result, err := svc.GetTask(ctx, taskID, userID)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrUnauthorized)
	})
}

func TestTaskService_ListProjectTasks(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	projectID := uuid.New()

	t.Run("success with tasks", func(t *testing.T) {
		mockTaskRepo := new(MockTaskRepository)
		mockProjectRepo := new(MockProjectRepository)

		project := &model.Project{
			ID:     projectID,
			UserID: userID,
		}

		tasks := []model.Task{
			{ID: uuid.New(), ProjectID: projectID, Title: "Task 1", Status: model.TaskStatusTodo},
			{ID: uuid.New(), ProjectID: projectID, Title: "Task 2", Status: model.TaskStatusInProgress},
		}

		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)
		mockTaskRepo.On("FindByProjectID", ctx, projectID).Return(tasks, nil)

		svc := NewTaskService(mockTaskRepo, mockProjectRepo)
		result, err := svc.ListProjectTasks(ctx, projectID, userID)

		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "Task 1", result[0].Title)
		assert.Equal(t, "Task 2", result[1].Title)
	})

	t.Run("success with no tasks", func(t *testing.T) {
		mockTaskRepo := new(MockTaskRepository)
		mockProjectRepo := new(MockProjectRepository)

		project := &model.Project{
			ID:     projectID,
			UserID: userID,
		}

		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)
		mockTaskRepo.On("FindByProjectID", ctx, projectID).Return([]model.Task{}, nil)

		svc := NewTaskService(mockTaskRepo, mockProjectRepo)
		result, err := svc.ListProjectTasks(ctx, projectID, userID)

		assert.NoError(t, err)
		assert.Len(t, result, 0)
	})

	t.Run("project not found", func(t *testing.T) {
		mockTaskRepo := new(MockTaskRepository)
		mockProjectRepo := new(MockProjectRepository)

		mockProjectRepo.On("FindByID", ctx, projectID).Return(nil, gorm.ErrRecordNotFound)

		svc := NewTaskService(mockTaskRepo, mockProjectRepo)
		result, err := svc.ListProjectTasks(ctx, projectID, userID)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrProjectNotFound)
	})

	t.Run("unauthorized", func(t *testing.T) {
		mockTaskRepo := new(MockTaskRepository)
		mockProjectRepo := new(MockProjectRepository)

		otherUserID := uuid.New()
		project := &model.Project{
			ID:     projectID,
			UserID: otherUserID,
		}

		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)

		svc := NewTaskService(mockTaskRepo, mockProjectRepo)
		result, err := svc.ListProjectTasks(ctx, projectID, userID)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrUnauthorized)
	})
}

func TestTaskService_UpdateTask(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	projectID := uuid.New()
	taskID := uuid.New()

	t.Run("update title", func(t *testing.T) {
		mockTaskRepo := new(MockTaskRepository)
		mockProjectRepo := new(MockProjectRepository)

		task := &model.Task{
			ID:        taskID,
			ProjectID: projectID,
			Title:     "Old Title",
		}

		project := &model.Project{
			ID:     projectID,
			UserID: userID,
		}

		mockTaskRepo.On("FindByID", ctx, taskID).Return(task, nil)
		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)
		mockTaskRepo.On("Update", ctx, mock.AnythingOfType("*model.Task")).Return(nil)

		svc := NewTaskService(mockTaskRepo, mockProjectRepo)
		updates := map[string]interface{}{"title": "New Title"}
		result, err := svc.UpdateTask(ctx, taskID, userID, updates)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "New Title", result.Title)
	})

	t.Run("update priority", func(t *testing.T) {
		mockTaskRepo := new(MockTaskRepository)
		mockProjectRepo := new(MockProjectRepository)

		task := &model.Task{
			ID:        taskID,
			ProjectID: projectID,
			Priority:  model.TaskPriorityMedium,
		}

		project := &model.Project{
			ID:     projectID,
			UserID: userID,
		}

		mockTaskRepo.On("FindByID", ctx, taskID).Return(task, nil)
		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)
		mockTaskRepo.On("Update", ctx, mock.AnythingOfType("*model.Task")).Return(nil)

		svc := NewTaskService(mockTaskRepo, mockProjectRepo)
		updates := map[string]interface{}{"priority": "high"}
		result, err := svc.UpdateTask(ctx, taskID, userID, updates)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, model.TaskPriorityHigh, result.Priority)
	})

	t.Run("invalid title", func(t *testing.T) {
		mockTaskRepo := new(MockTaskRepository)
		mockProjectRepo := new(MockProjectRepository)

		task := &model.Task{
			ID:        taskID,
			ProjectID: projectID,
			Title:     "Valid Title",
		}

		project := &model.Project{
			ID:     projectID,
			UserID: userID,
		}

		mockTaskRepo.On("FindByID", ctx, taskID).Return(task, nil)
		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)

		svc := NewTaskService(mockTaskRepo, mockProjectRepo)
		updates := map[string]interface{}{"title": ""}
		result, err := svc.UpdateTask(ctx, taskID, userID, updates)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrInvalidTaskTitle)
		mockTaskRepo.AssertNotCalled(t, "Update")
	})

	t.Run("invalid priority", func(t *testing.T) {
		mockTaskRepo := new(MockTaskRepository)
		mockProjectRepo := new(MockProjectRepository)

		task := &model.Task{
			ID:        taskID,
			ProjectID: projectID,
			Priority:  model.TaskPriorityMedium,
		}

		project := &model.Project{
			ID:     projectID,
			UserID: userID,
		}

		mockTaskRepo.On("FindByID", ctx, taskID).Return(task, nil)
		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)

		svc := NewTaskService(mockTaskRepo, mockProjectRepo)
		updates := map[string]interface{}{"priority": "invalid"}
		result, err := svc.UpdateTask(ctx, taskID, userID, updates)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrInvalidTaskPriority)
	})
}

func TestTaskService_MoveTask(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	projectID := uuid.New()
	taskID := uuid.New()

	t.Run("valid transition todo to in_progress", func(t *testing.T) {
		mockTaskRepo := new(MockTaskRepository)
		mockProjectRepo := new(MockProjectRepository)

		task := &model.Task{
			ID:        taskID,
			ProjectID: projectID,
			Status:    model.TaskStatusTodo,
			Position:  0,
		}

		project := &model.Project{
			ID:     projectID,
			UserID: userID,
		}

		mockTaskRepo.On("FindByID", ctx, taskID).Return(task, nil)
		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)
		mockTaskRepo.On("Update", ctx, mock.AnythingOfType("*model.Task")).Return(nil)

		svc := NewTaskService(mockTaskRepo, mockProjectRepo)
		result, err := svc.MoveTask(ctx, taskID, userID, model.TaskStatusInProgress, 0)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, model.TaskStatusInProgress, result.Status)
		assert.Equal(t, 0, result.Position)
	})

	t.Run("invalid transition todo to done", func(t *testing.T) {
		mockTaskRepo := new(MockTaskRepository)
		mockProjectRepo := new(MockProjectRepository)

		task := &model.Task{
			ID:        taskID,
			ProjectID: projectID,
			Status:    model.TaskStatusTodo,
			Position:  0,
		}

		project := &model.Project{
			ID:     projectID,
			UserID: userID,
		}

		mockTaskRepo.On("FindByID", ctx, taskID).Return(task, nil)
		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)

		svc := NewTaskService(mockTaskRepo, mockProjectRepo)
		result, err := svc.MoveTask(ctx, taskID, userID, model.TaskStatusDone, 0)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrInvalidStateTransition)
		assert.Contains(t, err.Error(), "cannot transition from todo to done")
		mockTaskRepo.AssertNotCalled(t, "Update")
	})

	t.Run("position change without state change", func(t *testing.T) {
		mockTaskRepo := new(MockTaskRepository)
		mockProjectRepo := new(MockProjectRepository)

		task := &model.Task{
			ID:        taskID,
			ProjectID: projectID,
			Status:    model.TaskStatusTodo,
			Position:  0,
		}

		project := &model.Project{
			ID:     projectID,
			UserID: userID,
		}

		mockTaskRepo.On("FindByID", ctx, taskID).Return(task, nil)
		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)
		mockTaskRepo.On("Update", ctx, mock.AnythingOfType("*model.Task")).Return(nil)

		svc := NewTaskService(mockTaskRepo, mockProjectRepo)
		result, err := svc.MoveTask(ctx, taskID, userID, model.TaskStatusTodo, 2)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, model.TaskStatusTodo, result.Status)
		assert.Equal(t, 2, result.Position)
	})
}

func TestTaskService_DeleteTask(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	projectID := uuid.New()
	taskID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mockTaskRepo := new(MockTaskRepository)
		mockProjectRepo := new(MockProjectRepository)

		task := &model.Task{
			ID:        taskID,
			ProjectID: projectID,
		}

		project := &model.Project{
			ID:     projectID,
			UserID: userID,
		}

		mockTaskRepo.On("FindByID", ctx, taskID).Return(task, nil)
		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)
		mockTaskRepo.On("SoftDelete", ctx, taskID).Return(nil)

		svc := NewTaskService(mockTaskRepo, mockProjectRepo)
		err := svc.DeleteTask(ctx, taskID, userID)

		assert.NoError(t, err)
		mockTaskRepo.AssertExpectations(t)
		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("task not found", func(t *testing.T) {
		mockTaskRepo := new(MockTaskRepository)
		mockProjectRepo := new(MockProjectRepository)

		mockTaskRepo.On("FindByID", ctx, taskID).Return(nil, gorm.ErrRecordNotFound)

		svc := NewTaskService(mockTaskRepo, mockProjectRepo)
		err := svc.DeleteTask(ctx, taskID, userID)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrTaskNotFound)
		mockTaskRepo.AssertNotCalled(t, "SoftDelete")
	})

	t.Run("unauthorized", func(t *testing.T) {
		mockTaskRepo := new(MockTaskRepository)
		mockProjectRepo := new(MockProjectRepository)

		task := &model.Task{
			ID:        taskID,
			ProjectID: projectID,
		}

		otherUserID := uuid.New()
		project := &model.Project{
			ID:     projectID,
			UserID: otherUserID,
		}

		mockTaskRepo.On("FindByID", ctx, taskID).Return(task, nil)
		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)

		svc := NewTaskService(mockTaskRepo, mockProjectRepo)
		err := svc.DeleteTask(ctx, taskID, userID)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrUnauthorized)
		mockTaskRepo.AssertNotCalled(t, "SoftDelete")
	})
}

func TestValidateTaskTitle(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		wantErr bool
	}{
		{"valid title", "Test Task", false},
		{"valid max length", string(make([]byte, 255)), false},
		{"empty title", "", true},
		{"exceeds max length", string(make([]byte, 256)), true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTaskTitle(tc.title)
			if tc.wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidTaskTitle)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTaskPriority(t *testing.T) {
	tests := []struct {
		name     string
		priority model.TaskPriority
		wantErr  bool
	}{
		{"low priority", model.TaskPriorityLow, false},
		{"medium priority", model.TaskPriorityMedium, false},
		{"high priority", model.TaskPriorityHigh, false},
		{"invalid priority", "invalid", true},
		{"empty priority", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTaskPriority(tc.priority)
			if tc.wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidTaskPriority)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsValidTransition(t *testing.T) {
	tests := []struct {
		name         string
		currentState model.TaskStatus
		newState     model.TaskStatus
		want         bool
	}{
		{"todo to in_progress", model.TaskStatusTodo, model.TaskStatusInProgress, true},
		{"in_progress to ai_review", model.TaskStatusInProgress, model.TaskStatusAIReview, true},
		{"in_progress to todo", model.TaskStatusInProgress, model.TaskStatusTodo, true},
		{"ai_review to human_review", model.TaskStatusAIReview, model.TaskStatusHumanReview, true},
		{"ai_review to in_progress", model.TaskStatusAIReview, model.TaskStatusInProgress, true},
		{"human_review to done", model.TaskStatusHumanReview, model.TaskStatusDone, true},
		{"human_review to in_progress", model.TaskStatusHumanReview, model.TaskStatusInProgress, true},
		{"done to todo", model.TaskStatusDone, model.TaskStatusTodo, true},
		{"todo to done", model.TaskStatusTodo, model.TaskStatusDone, false},
		{"todo to ai_review", model.TaskStatusTodo, model.TaskStatusAIReview, false},
		{"in_progress to done", model.TaskStatusInProgress, model.TaskStatusDone, false},
		{"ai_review to done", model.TaskStatusAIReview, model.TaskStatusDone, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isValidTransition(tc.currentState, tc.newState)
			assert.Equal(t, tc.want, result)
		})
	}
}
