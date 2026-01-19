package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/npinot/vibe/backend/internal/model"
	"github.com/npinot/vibe/backend/internal/service"
)

type MockTaskService struct {
	mock.Mock
}

func (m *MockTaskService) CreateTask(ctx context.Context, projectID, userID uuid.UUID, title, description string, priority model.TaskPriority) (*model.Task, error) {
	args := m.Called(ctx, projectID, userID, title, description, priority)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Task), args.Error(1)
}

func (m *MockTaskService) GetTask(ctx context.Context, id, userID uuid.UUID) (*model.Task, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Task), args.Error(1)
}

func (m *MockTaskService) ListProjectTasks(ctx context.Context, projectID, userID uuid.UUID) ([]model.Task, error) {
	args := m.Called(ctx, projectID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Task), args.Error(1)
}

func (m *MockTaskService) UpdateTask(ctx context.Context, id, userID uuid.UUID, updates map[string]interface{}) (*model.Task, error) {
	args := m.Called(ctx, id, userID, updates)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Task), args.Error(1)
}

func (m *MockTaskService) MoveTask(ctx context.Context, id, userID uuid.UUID, newState model.TaskStatus, newPosition int) (*model.Task, error) {
	args := m.Called(ctx, id, userID, newState, newPosition)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Task), args.Error(1)
}

func (m *MockTaskService) DeleteTask(ctx context.Context, id, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockTaskService) ExecuteTask(ctx context.Context, id, userID uuid.UUID) (*model.Session, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Session), args.Error(1)
}

func (m *MockTaskService) StopTask(ctx context.Context, id, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

type MockProjectRepo struct {
	mock.Mock
}

func (m *MockProjectRepo) Create(ctx context.Context, project *model.Project) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}

func (m *MockProjectRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Project, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Project), args.Error(1)
}

func (m *MockProjectRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Project, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]model.Project), args.Error(1)
}

func (m *MockProjectRepo) Update(ctx context.Context, project *model.Project) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}

func (m *MockProjectRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockProjectRepo) UpdatePodStatus(ctx context.Context, id uuid.UUID, status string, podError string) error {
	args := m.Called(ctx, id, status, podError)
	return args.Error(0)
}

type MockK8sService struct {
	mock.Mock
}

func (m *MockK8sService) CreateProjectPod(ctx context.Context, project *model.Project) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}

func (m *MockK8sService) GetPodStatus(ctx context.Context, podName, namespace string) (string, error) {
	args := m.Called(ctx, podName, namespace)
	return args.String(0), args.Error(1)
}

func (m *MockK8sService) DeleteProjectPod(ctx context.Context, podName, namespace string) error {
	args := m.Called(ctx, podName, namespace)
	return args.Error(0)
}

func (m *MockK8sService) GetPodIP(ctx context.Context, podName, namespace string) (string, error) {
	args := m.Called(ctx, podName, namespace)
	return args.String(0), args.Error(1)
}

func (m *MockK8sService) WatchPodStatus(ctx context.Context, podName, namespace string) (<-chan string, error) {
	args := m.Called(ctx, podName, namespace)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(<-chan string), args.Error(1)
}

func setupTaskTestRouter(handler *TaskHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		testUser := &model.User{
			ID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			Email:       "test@example.com",
			Name:        "Test User",
			OIDCSubject: "test-subject",
		}
		c.Set("currentUser", testUser)
		c.Next()
	})

	return router
}

func TestTaskHandler_CreateTask(t *testing.T) {
	mockService := new(MockTaskService)
	mockProjectRepo := new(MockProjectRepo)
	mockK8sService := new(MockK8sService)
	handler := NewTaskHandler(mockService, mockProjectRepo, mockK8sService)
	router := setupTaskTestRouter(handler)

	router.POST("/projects/:id/tasks", handler.CreateTask)

	t.Run("successful creation", func(t *testing.T) {
		projectID := uuid.New()
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

		reqBody := map[string]interface{}{
			"title":       "Test Task",
			"description": "Do something",
			"priority":    "medium",
		}

		created := &model.Task{
			ID:          uuid.New(),
			ProjectID:   projectID,
			Title:       "Test Task",
			Description: "Do something",
			Priority:    model.TaskPriorityMedium,
			Status:      model.TaskStatusTodo,
			CreatedBy:   userID,
		}

		mockService.On("CreateTask", mock.Anything, projectID, userID, "Test Task", "Do something", model.TaskPriorityMedium).
			Return(created, nil).Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/projects/"+projectID.String()+"/tasks", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp model.Task
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "Test Task", resp.Title)
		assert.Equal(t, model.TaskPriorityMedium, resp.Priority)

		mockService.AssertExpectations(t)
	})

	t.Run("successful creation with default priority", func(t *testing.T) {
		projectID := uuid.New()
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

		reqBody := map[string]interface{}{
			"title": "Task without priority",
		}

		created := &model.Task{
			ID:        uuid.New(),
			ProjectID: projectID,
			Title:     "Task without priority",
			Priority:  model.TaskPriorityMedium,
			Status:    model.TaskStatusTodo,
			CreatedBy: userID,
		}

		mockService.On("CreateTask", mock.Anything, projectID, userID, "Task without priority", "", model.TaskPriorityMedium).
			Return(created, nil).Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/projects/"+projectID.String()+"/tasks", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		projectID := uuid.New()
		req, _ := http.NewRequest("POST", "/projects/"+projectID.String()+"/tasks", bytes.NewBufferString("not json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid project ID", func(t *testing.T) {
		reqBody := map[string]interface{}{"title": "Test"}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/projects/invalid-uuid/tasks", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error - empty title", func(t *testing.T) {
		projectID := uuid.New()
		reqBody := map[string]interface{}{
			"title": "",
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/projects/"+projectID.String()+"/tasks", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("project not found", func(t *testing.T) {
		projectID := uuid.New()
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

		reqBody := map[string]interface{}{
			"title": "Task for missing project",
		}

		mockService.On("CreateTask", mock.Anything, projectID, userID, "Task for missing project", "", model.TaskPriorityMedium).
			Return(nil, service.ErrProjectNotFound).Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/projects/"+projectID.String()+"/tasks", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("unauthorized user", func(t *testing.T) {
		projectID := uuid.New()
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

		reqBody := map[string]interface{}{
			"title": "Should fail",
		}

		mockService.On("CreateTask", mock.Anything, projectID, userID, "Should fail", "", model.TaskPriorityMedium).
			Return(nil, service.ErrUnauthorized).Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/projects/"+projectID.String()+"/tasks", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("service validation error", func(t *testing.T) {
		projectID := uuid.New()
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

		reqBody := map[string]interface{}{
			"title": "x",
		}

		mockService.On("CreateTask", mock.Anything, projectID, userID, "x", "", model.TaskPriorityMedium).
			Return(nil, service.ErrInvalidTaskTitle).Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/projects/"+projectID.String()+"/tasks", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestTaskHandler_GetTask(t *testing.T) {
	mockService := new(MockTaskService)
	mockProjectRepo := new(MockProjectRepo)
	mockK8sService := new(MockK8sService)
	handler := NewTaskHandler(mockService, mockProjectRepo, mockK8sService)
	router := setupTaskTestRouter(handler)

	router.GET("/projects/:id/tasks/:taskId", handler.GetTask)

	t.Run("successful retrieval", func(t *testing.T) {
		projectID := uuid.New()
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		taskID := uuid.New()

		expected := &model.Task{
			ID:        taskID,
			ProjectID: projectID,
			Title:     "Existing Task",
			CreatedBy: userID,
			Status:    model.TaskStatusTodo,
		}

		mockService.On("GetTask", mock.Anything, taskID, userID).Return(expected, nil).Once()

		req, _ := http.NewRequest("GET", "/projects/"+projectID.String()+"/tasks/"+taskID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp model.Task
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "Existing Task", resp.Title)

		mockService.AssertExpectations(t)
	})

	t.Run("invalid task id", func(t *testing.T) {
		projectID := uuid.New()
		req, _ := http.NewRequest("GET", "/projects/"+projectID.String()+"/tasks/invalid-uuid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("task not found", func(t *testing.T) {
		projectID := uuid.New()
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		taskID := uuid.New()

		mockService.On("GetTask", mock.Anything, taskID, userID).Return(nil, service.ErrTaskNotFound).Once()

		req, _ := http.NewRequest("GET", "/projects/"+projectID.String()+"/tasks/"+taskID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("unauthorized", func(t *testing.T) {
		projectID := uuid.New()
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		taskID := uuid.New()

		mockService.On("GetTask", mock.Anything, taskID, userID).Return(nil, service.ErrUnauthorized).Once()

		req, _ := http.NewRequest("GET", "/projects/"+projectID.String()+"/tasks/"+taskID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestTaskHandler_ListProjectTasks(t *testing.T) {
	mockService := new(MockTaskService)
	mockProjectRepo := new(MockProjectRepo)
	mockK8sService := new(MockK8sService)
	handler := NewTaskHandler(mockService, mockProjectRepo, mockK8sService)
	router := setupTaskTestRouter(handler)

	router.GET("/projects/:id/tasks", handler.ListTasks)

	t.Run("list tasks success", func(t *testing.T) {
		projectID := uuid.New()
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

		tasks := []model.Task{
			{ID: uuid.New(), ProjectID: projectID, Title: "T1", CreatedBy: userID, Status: model.TaskStatusTodo},
			{ID: uuid.New(), ProjectID: projectID, Title: "T2", CreatedBy: userID, Status: model.TaskStatusInProgress},
		}

		mockService.On("ListProjectTasks", mock.Anything, projectID, userID).Return(tasks, nil).Once()

		req, _ := http.NewRequest("GET", "/projects/"+projectID.String()+"/tasks", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp []model.Task
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Len(t, resp, 2)
		mockService.AssertExpectations(t)
	})

	t.Run("empty list", func(t *testing.T) {
		projectID := uuid.New()
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

		mockService.On("ListProjectTasks", mock.Anything, projectID, userID).Return([]model.Task{}, nil).Once()

		req, _ := http.NewRequest("GET", "/projects/"+projectID.String()+"/tasks", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp []model.Task
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Len(t, resp, 0)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid project ID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/projects/invalid-uuid/tasks", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("project not found", func(t *testing.T) {
		projectID := uuid.New()
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

		mockService.On("ListProjectTasks", mock.Anything, projectID, userID).Return(nil, service.ErrProjectNotFound).Once()

		req, _ := http.NewRequest("GET", "/projects/"+projectID.String()+"/tasks", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("unauthorized", func(t *testing.T) {
		projectID := uuid.New()
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

		mockService.On("ListProjectTasks", mock.Anything, projectID, userID).Return(nil, service.ErrUnauthorized).Once()

		req, _ := http.NewRequest("GET", "/projects/"+projectID.String()+"/tasks", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		projectID := uuid.New()
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

		mockService.On("ListProjectTasks", mock.Anything, projectID, userID).Return(nil, errors.New("db error")).Once()

		req, _ := http.NewRequest("GET", "/projects/"+projectID.String()+"/tasks", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestTaskHandler_UpdateTask(t *testing.T) {
	mockService := new(MockTaskService)
	mockProjectRepo := new(MockProjectRepo)
	mockK8sService := new(MockK8sService)
	handler := NewTaskHandler(mockService, mockProjectRepo, mockK8sService)
	router := setupTaskTestRouter(handler)

	router.PATCH("/projects/:id/tasks/:taskId", handler.UpdateTask)

	t.Run("successful title update", func(t *testing.T) {
		projectID := uuid.New()
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		taskID := uuid.New()

		newTitle := "Updated Title"
		reqBody := map[string]interface{}{
			"title": newTitle,
		}

		updated := &model.Task{
			ID:        taskID,
			ProjectID: projectID,
			Title:     newTitle,
			CreatedBy: userID,
		}

		updates := map[string]interface{}{"title": newTitle}
		mockService.On("UpdateTask", mock.Anything, taskID, userID, updates).Return(updated, nil).Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PATCH", "/projects/"+projectID.String()+"/tasks/"+taskID.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp model.Task
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, newTitle, resp.Title)

		mockService.AssertExpectations(t)
	})

	t.Run("successful priority update", func(t *testing.T) {
		projectID := uuid.New()
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		taskID := uuid.New()

		reqBody := map[string]interface{}{
			"priority": "high",
		}

		updated := &model.Task{
			ID:        taskID,
			ProjectID: projectID,
			Title:     "Task",
			Priority:  model.TaskPriorityHigh,
			CreatedBy: userID,
		}

		updates := map[string]interface{}{"priority": model.TaskPriorityHigh}
		mockService.On("UpdateTask", mock.Anything, taskID, userID, updates).Return(updated, nil).Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PATCH", "/projects/"+projectID.String()+"/tasks/"+taskID.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("no fields to update", func(t *testing.T) {
		projectID := uuid.New()
		taskID := uuid.New()

		reqBody := map[string]interface{}{}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PATCH", "/projects/"+projectID.String()+"/tasks/"+taskID.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("task not found", func(t *testing.T) {
		projectID := uuid.New()
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		taskID := uuid.New()

		reqBody := map[string]interface{}{
			"title": "New Title",
		}

		updates := map[string]interface{}{"title": "New Title"}
		mockService.On("UpdateTask", mock.Anything, taskID, userID, updates).Return(nil, service.ErrTaskNotFound).Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PATCH", "/projects/"+projectID.String()+"/tasks/"+taskID.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestTaskHandler_MoveTask(t *testing.T) {
	mockService := new(MockTaskService)
	mockProjectRepo := new(MockProjectRepo)
	mockK8sService := new(MockK8sService)
	handler := NewTaskHandler(mockService, mockProjectRepo, mockK8sService)
	router := setupTaskTestRouter(handler)

	router.PATCH("/projects/:id/tasks/:taskId/move", handler.MoveTask)

	t.Run("successful state transition", func(t *testing.T) {
		projectID := uuid.New()
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		taskID := uuid.New()

		reqBody := map[string]interface{}{
			"status":   "in_progress",
			"position": 1,
		}

		moved := &model.Task{
			ID:        taskID,
			ProjectID: projectID,
			Title:     "Task",
			Status:    model.TaskStatusInProgress,
			Position:  1,
			CreatedBy: userID,
		}

		mockService.On("MoveTask", mock.Anything, taskID, userID, model.TaskStatusInProgress, 1).
			Return(moved, nil).Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PATCH", "/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/move", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp model.Task
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, model.TaskStatusInProgress, resp.Status)
		assert.Equal(t, 1, resp.Position)

		mockService.AssertExpectations(t)
	})

	t.Run("invalid state transition", func(t *testing.T) {
		projectID := uuid.New()
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		taskID := uuid.New()

		reqBody := map[string]interface{}{
			"status":   "done",
			"position": 0,
		}

		mockService.On("MoveTask", mock.Anything, taskID, userID, model.TaskStatusDone, 0).
			Return(nil, service.ErrInvalidStateTransition).Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PATCH", "/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/move", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("missing status field", func(t *testing.T) {
		projectID := uuid.New()
		taskID := uuid.New()

		reqBody := map[string]interface{}{
			"position": 1,
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PATCH", "/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/move", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestTaskHandler_DeleteTask(t *testing.T) {
	mockService := new(MockTaskService)
	mockProjectRepo := new(MockProjectRepo)
	mockK8sService := new(MockK8sService)
	handler := NewTaskHandler(mockService, mockProjectRepo, mockK8sService)
	router := setupTaskTestRouter(handler)

	router.DELETE("/projects/:id/tasks/:taskId", handler.DeleteTask)

	t.Run("successful deletion", func(t *testing.T) {
		projectID := uuid.New()
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		taskID := uuid.New()

		mockService.On("DeleteTask", mock.Anything, taskID, userID).Return(nil).Once()

		req, _ := http.NewRequest("DELETE", "/projects/"+projectID.String()+"/tasks/"+taskID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("task not found", func(t *testing.T) {
		projectID := uuid.New()
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		taskID := uuid.New()

		mockService.On("DeleteTask", mock.Anything, taskID, userID).Return(service.ErrTaskNotFound).Once()

		req, _ := http.NewRequest("DELETE", "/projects/"+projectID.String()+"/tasks/"+taskID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("unauthorized", func(t *testing.T) {
		projectID := uuid.New()
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		taskID := uuid.New()

		mockService.On("DeleteTask", mock.Anything, taskID, userID).Return(service.ErrUnauthorized).Once()

		req, _ := http.NewRequest("DELETE", "/projects/"+projectID.String()+"/tasks/"+taskID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid task ID", func(t *testing.T) {
		projectID := uuid.New()
		req, _ := http.NewRequest("DELETE", "/projects/"+projectID.String()+"/tasks/invalid-uuid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
