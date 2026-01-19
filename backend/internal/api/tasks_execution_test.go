package api

import (
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

type MockTaskServiceExecution struct {
	mock.Mock
}

func (m *MockTaskServiceExecution) CreateTask(ctx context.Context, projectID, userID uuid.UUID, title, description string, priority model.TaskPriority) (*model.Task, error) {
	args := m.Called(ctx, projectID, userID, title, description, priority)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Task), args.Error(1)
}

func (m *MockTaskServiceExecution) GetTask(ctx context.Context, id, userID uuid.UUID) (*model.Task, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Task), args.Error(1)
}

func (m *MockTaskServiceExecution) ListProjectTasks(ctx context.Context, projectID, userID uuid.UUID) ([]model.Task, error) {
	args := m.Called(ctx, projectID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Task), args.Error(1)
}

func (m *MockTaskServiceExecution) UpdateTask(ctx context.Context, id, userID uuid.UUID, updates map[string]interface{}) (*model.Task, error) {
	args := m.Called(ctx, id, userID, updates)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Task), args.Error(1)
}

func (m *MockTaskServiceExecution) MoveTask(ctx context.Context, id, userID uuid.UUID, newState model.TaskStatus, newPosition int) (*model.Task, error) {
	args := m.Called(ctx, id, userID, newState, newPosition)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Task), args.Error(1)
}

func (m *MockTaskServiceExecution) DeleteTask(ctx context.Context, id, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockTaskServiceExecution) ExecuteTask(ctx context.Context, id, userID uuid.UUID) (*model.Session, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Session), args.Error(1)
}

func (m *MockTaskServiceExecution) StopTask(ctx context.Context, id, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

type MockProjectRepositoryExecution struct {
	mock.Mock
}

func (m *MockProjectRepositoryExecution) FindByID(ctx context.Context, id uuid.UUID) (*model.Project, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Project), args.Error(1)
}

func (m *MockProjectRepositoryExecution) Create(ctx context.Context, project *model.Project) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}

func (m *MockProjectRepositoryExecution) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Project, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]model.Project), args.Error(1)
}

func (m *MockProjectRepositoryExecution) Update(ctx context.Context, project *model.Project) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}

func (m *MockProjectRepositoryExecution) SoftDelete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockProjectRepositoryExecution) UpdatePodStatus(ctx context.Context, id uuid.UUID, status string, podError string) error {
	args := m.Called(ctx, id, status, podError)
	return args.Error(0)
}

type MockKubernetesServiceExecution struct {
	mock.Mock
}

func (m *MockKubernetesServiceExecution) CreateProjectPod(ctx context.Context, project *model.Project) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}

func (m *MockKubernetesServiceExecution) DeleteProjectPod(ctx context.Context, podName, namespace string) error {
	args := m.Called(ctx, podName, namespace)
	return args.Error(0)
}

func (m *MockKubernetesServiceExecution) GetPodStatus(ctx context.Context, podName, namespace string) (string, error) {
	args := m.Called(ctx, podName, namespace)
	return args.String(0), args.Error(1)
}

func (m *MockKubernetesServiceExecution) WatchPodStatus(ctx context.Context, podName, namespace string) (<-chan string, error) {
	args := m.Called(ctx, podName, namespace)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(<-chan string), args.Error(1)
}

func (m *MockKubernetesServiceExecution) GetPodIP(ctx context.Context, podName, namespace string) (string, error) {
	args := m.Called(ctx, podName, namespace)
	return args.String(0), args.Error(1)
}

func TestExecuteTask_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockTaskService := new(MockTaskServiceExecution)
	mockProjectRepo := new(MockProjectRepositoryExecution)
	mockK8sService := new(MockKubernetesServiceExecution)

	handler := NewTaskHandler(mockTaskService, mockProjectRepo, mockK8sService)

	taskID := uuid.New()
	userID := uuid.New()
	sessionID := uuid.New()

	mockTaskService.On("ExecuteTask", mock.Anything, taskID, userID).Return(&model.Session{
		ID:     sessionID,
		TaskID: taskID,
		Status: model.SessionStatusPending,
	}, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("currentUser", &model.User{ID: userID})
		c.Next()
	})
	router.POST("/tasks/:taskId/execute", handler.ExecuteTask)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/tasks/"+taskID.String()+"/execute", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, sessionID.String(), response["session_id"])
	assert.Equal(t, string(model.SessionStatusPending), response["status"])

	mockTaskService.AssertExpectations(t)
}

func TestExecuteTask_TaskNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockTaskService := new(MockTaskServiceExecution)
	mockProjectRepo := new(MockProjectRepositoryExecution)
	mockK8sService := new(MockKubernetesServiceExecution)

	handler := NewTaskHandler(mockTaskService, mockProjectRepo, mockK8sService)

	taskID := uuid.New()
	userID := uuid.New()

	mockTaskService.On("ExecuteTask", mock.Anything, taskID, userID).Return(nil, service.ErrTaskNotFound)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("currentUser", &model.User{ID: userID})
		c.Next()
	})
	router.POST("/tasks/:taskId/execute", handler.ExecuteTask)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/tasks/"+taskID.String()+"/execute", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	mockTaskService.AssertExpectations(t)
}

func TestExecuteTask_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockTaskService := new(MockTaskServiceExecution)
	mockProjectRepo := new(MockProjectRepositoryExecution)
	mockK8sService := new(MockKubernetesServiceExecution)

	handler := NewTaskHandler(mockTaskService, mockProjectRepo, mockK8sService)

	taskID := uuid.New()
	userID := uuid.New()

	mockTaskService.On("ExecuteTask", mock.Anything, taskID, userID).Return(nil, service.ErrUnauthorized)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("currentUser", &model.User{ID: userID})
		c.Next()
	})
	router.POST("/tasks/:taskId/execute", handler.ExecuteTask)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/tasks/"+taskID.String()+"/execute", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	mockTaskService.AssertExpectations(t)
}

func TestExecuteTask_InvalidStateTransition(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockTaskService := new(MockTaskServiceExecution)
	mockProjectRepo := new(MockProjectRepositoryExecution)
	mockK8sService := new(MockKubernetesServiceExecution)

	handler := NewTaskHandler(mockTaskService, mockProjectRepo, mockK8sService)

	taskID := uuid.New()
	userID := uuid.New()

	mockTaskService.On("ExecuteTask", mock.Anything, taskID, userID).Return(nil, service.ErrInvalidStateTransition)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("currentUser", &model.User{ID: userID})
		c.Next()
	})
	router.POST("/tasks/:taskId/execute", handler.ExecuteTask)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/tasks/"+taskID.String()+"/execute", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	mockTaskService.AssertExpectations(t)
}

func TestExecuteTask_SessionAlreadyActive(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockTaskService := new(MockTaskServiceExecution)
	mockProjectRepo := new(MockProjectRepositoryExecution)
	mockK8sService := new(MockKubernetesServiceExecution)

	handler := NewTaskHandler(mockTaskService, mockProjectRepo, mockK8sService)

	taskID := uuid.New()
	userID := uuid.New()

	mockTaskService.On("ExecuteTask", mock.Anything, taskID, userID).Return(nil, service.ErrSessionAlreadyActive)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("currentUser", &model.User{ID: userID})
		c.Next()
	})
	router.POST("/tasks/:taskId/execute", handler.ExecuteTask)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/tasks/"+taskID.String()+"/execute", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	mockTaskService.AssertExpectations(t)
}

func TestExecuteTask_InvalidTaskID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockTaskService := new(MockTaskServiceExecution)
	mockProjectRepo := new(MockProjectRepositoryExecution)
	mockK8sService := new(MockKubernetesServiceExecution)

	handler := NewTaskHandler(mockTaskService, mockProjectRepo, mockK8sService)

	userID := uuid.New()

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("currentUser", &model.User{ID: userID})
		c.Next()
	})
	router.POST("/tasks/:taskId/execute", handler.ExecuteTask)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/tasks/invalid-uuid/execute", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExecuteTask_InternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockTaskService := new(MockTaskServiceExecution)
	mockProjectRepo := new(MockProjectRepositoryExecution)
	mockK8sService := new(MockKubernetesServiceExecution)

	handler := NewTaskHandler(mockTaskService, mockProjectRepo, mockK8sService)

	taskID := uuid.New()
	userID := uuid.New()

	mockTaskService.On("ExecuteTask", mock.Anything, taskID, userID).Return(nil, errors.New("database error"))

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("currentUser", &model.User{ID: userID})
		c.Next()
	})
	router.POST("/tasks/:taskId/execute", handler.ExecuteTask)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/tasks/"+taskID.String()+"/execute", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	mockTaskService.AssertExpectations(t)
}

func TestStopTask_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockTaskService := new(MockTaskServiceExecution)
	mockProjectRepo := new(MockProjectRepositoryExecution)
	mockK8sService := new(MockKubernetesServiceExecution)

	handler := NewTaskHandler(mockTaskService, mockProjectRepo, mockK8sService)

	taskID := uuid.New()
	userID := uuid.New()

	mockTaskService.On("StopTask", mock.Anything, taskID, userID).Return(nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("currentUser", &model.User{ID: userID})
		c.Next()
	})
	router.POST("/tasks/:taskId/stop", handler.StopTask)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/tasks/"+taskID.String()+"/stop", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	mockTaskService.AssertExpectations(t)
}

func TestStopTask_TaskNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockTaskService := new(MockTaskServiceExecution)
	mockProjectRepo := new(MockProjectRepositoryExecution)
	mockK8sService := new(MockKubernetesServiceExecution)

	handler := NewTaskHandler(mockTaskService, mockProjectRepo, mockK8sService)

	taskID := uuid.New()
	userID := uuid.New()

	mockTaskService.On("StopTask", mock.Anything, taskID, userID).Return(service.ErrTaskNotFound)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("currentUser", &model.User{ID: userID})
		c.Next()
	})
	router.POST("/tasks/:taskId/stop", handler.StopTask)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/tasks/"+taskID.String()+"/stop", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	mockTaskService.AssertExpectations(t)
}

func TestStopTask_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockTaskService := new(MockTaskServiceExecution)
	mockProjectRepo := new(MockProjectRepositoryExecution)
	mockK8sService := new(MockKubernetesServiceExecution)

	handler := NewTaskHandler(mockTaskService, mockProjectRepo, mockK8sService)

	taskID := uuid.New()
	userID := uuid.New()

	mockTaskService.On("StopTask", mock.Anything, taskID, userID).Return(service.ErrUnauthorized)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("currentUser", &model.User{ID: userID})
		c.Next()
	})
	router.POST("/tasks/:taskId/stop", handler.StopTask)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/tasks/"+taskID.String()+"/stop", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	mockTaskService.AssertExpectations(t)
}

func TestStopTask_InvalidStateTransition(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockTaskService := new(MockTaskServiceExecution)
	mockProjectRepo := new(MockProjectRepositoryExecution)
	mockK8sService := new(MockKubernetesServiceExecution)

	handler := NewTaskHandler(mockTaskService, mockProjectRepo, mockK8sService)

	taskID := uuid.New()
	userID := uuid.New()

	mockTaskService.On("StopTask", mock.Anything, taskID, userID).Return(service.ErrInvalidStateTransition)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("currentUser", &model.User{ID: userID})
		c.Next()
	})
	router.POST("/tasks/:taskId/stop", handler.StopTask)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/tasks/"+taskID.String()+"/stop", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	mockTaskService.AssertExpectations(t)
}

func TestStopTask_InvalidTaskID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockTaskService := new(MockTaskServiceExecution)
	mockProjectRepo := new(MockProjectRepositoryExecution)
	mockK8sService := new(MockKubernetesServiceExecution)

	handler := NewTaskHandler(mockTaskService, mockProjectRepo, mockK8sService)

	userID := uuid.New()

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("currentUser", &model.User{ID: userID})
		c.Next()
	})
	router.POST("/tasks/:taskId/stop", handler.StopTask)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/tasks/invalid-uuid/stop", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStopTask_InternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockTaskService := new(MockTaskServiceExecution)
	mockProjectRepo := new(MockProjectRepositoryExecution)
	mockK8sService := new(MockKubernetesServiceExecution)

	handler := NewTaskHandler(mockTaskService, mockProjectRepo, mockK8sService)

	taskID := uuid.New()
	userID := uuid.New()

	mockTaskService.On("StopTask", mock.Anything, taskID, userID).Return(errors.New("database error"))

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("currentUser", &model.User{ID: userID})
		c.Next()
	})
	router.POST("/tasks/:taskId/stop", handler.StopTask)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/tasks/"+taskID.String()+"/stop", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	mockTaskService.AssertExpectations(t)
}

func TestTaskOutputStream_MissingSessionID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockTaskService := new(MockTaskServiceExecution)
	mockProjectRepo := new(MockProjectRepositoryExecution)
	mockK8sService := new(MockKubernetesServiceExecution)

	handler := NewTaskHandler(mockTaskService, mockProjectRepo, mockK8sService)

	projectID := uuid.New()
	taskID := uuid.New()
	userID := uuid.New()

	project := &model.Project{
		ID:           projectID,
		UserID:       userID,
		PodName:      "project-12345678",
		PodNamespace: "opencode",
	}

	mockProjectRepo.On("FindByID", mock.Anything, projectID).Return(project, nil)
	mockTaskService.On("GetTask", mock.Anything, taskID, userID).Return(&model.Task{
		ID:        taskID,
		ProjectID: projectID,
		Status:    model.TaskStatusInProgress,
	}, nil)
	mockK8sService.On("GetPodIP", mock.Anything, "project-12345678", "opencode").Return("10.0.0.1", nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("currentUser", &model.User{ID: userID})
		c.Next()
	})
	router.GET("/projects/:id/tasks/:taskId/output", handler.TaskOutputStream)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/output", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	mockProjectRepo.AssertExpectations(t)
	mockTaskService.AssertExpectations(t)
}

func TestTaskOutputStream_InvalidSessionID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockTaskService := new(MockTaskServiceExecution)
	mockProjectRepo := new(MockProjectRepositoryExecution)
	mockK8sService := new(MockKubernetesServiceExecution)

	handler := NewTaskHandler(mockTaskService, mockProjectRepo, mockK8sService)

	projectID := uuid.New()
	taskID := uuid.New()
	userID := uuid.New()

	project := &model.Project{
		ID:           projectID,
		UserID:       userID,
		PodName:      "project-12345678",
		PodNamespace: "opencode",
	}

	mockProjectRepo.On("FindByID", mock.Anything, projectID).Return(project, nil)
	mockTaskService.On("GetTask", mock.Anything, taskID, userID).Return(&model.Task{
		ID:        taskID,
		ProjectID: projectID,
		Status:    model.TaskStatusInProgress,
	}, nil)
	mockK8sService.On("GetPodIP", mock.Anything, "project-12345678", "opencode").Return("10.0.0.1", nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("currentUser", &model.User{ID: userID})
		c.Next()
	})
	router.GET("/projects/:id/tasks/:taskId/output", handler.TaskOutputStream)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/output?session_id=invalid-uuid", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	mockProjectRepo.AssertExpectations(t)
	mockTaskService.AssertExpectations(t)
}

func TestTaskOutputStream_ProjectNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockTaskService := new(MockTaskServiceExecution)
	mockProjectRepo := new(MockProjectRepositoryExecution)
	mockK8sService := new(MockKubernetesServiceExecution)

	handler := NewTaskHandler(mockTaskService, mockProjectRepo, mockK8sService)

	projectID := uuid.New()
	taskID := uuid.New()
	userID := uuid.New()

	mockProjectRepo.On("FindByID", mock.Anything, projectID).Return(nil, errors.New("not found"))

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("currentUser", &model.User{ID: userID})
		c.Next()
	})
	router.GET("/projects/:id/tasks/:taskId/output", handler.TaskOutputStream)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/output", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	mockProjectRepo.AssertExpectations(t)
}

func TestTaskOutputStream_TaskBelongsToDifferentProject(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockTaskService := new(MockTaskServiceExecution)
	mockProjectRepo := new(MockProjectRepositoryExecution)
	mockK8sService := new(MockKubernetesServiceExecution)

	handler := NewTaskHandler(mockTaskService, mockProjectRepo, mockK8sService)

	projectID := uuid.New()
	otherProjectID := uuid.New()
	taskID := uuid.New()
	userID := uuid.New()

	project := &model.Project{
		ID:           projectID,
		UserID:       userID,
		PodName:      "project-12345678",
		PodNamespace: "opencode",
	}

	mockProjectRepo.On("FindByID", mock.Anything, projectID).Return(project, nil)
	mockTaskService.On("GetTask", mock.Anything, taskID, userID).Return(&model.Task{
		ID:        taskID,
		ProjectID: otherProjectID,
		Status:    model.TaskStatusInProgress,
	}, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("currentUser", &model.User{ID: userID})
		c.Next()
	})
	router.GET("/projects/:id/tasks/:taskId/output", handler.TaskOutputStream)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/output", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	mockProjectRepo.AssertExpectations(t)
	mockTaskService.AssertExpectations(t)
}
