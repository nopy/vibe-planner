package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/npinot/vibe/backend/internal/model"
)

// MockSessionRepository - local mock for session repository
type MockSessionRepository struct {
	mock.Mock
}

func (m *MockSessionRepository) Create(ctx context.Context, session *model.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockSessionRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Session, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Session), args.Error(1)
}

func (m *MockSessionRepository) FindByTaskID(ctx context.Context, taskID uuid.UUID) ([]model.Session, error) {
	args := m.Called(ctx, taskID)
	return args.Get(0).([]model.Session), args.Error(1)
}

func (m *MockSessionRepository) FindActiveSessionsForProject(ctx context.Context, projectID uuid.UUID) ([]model.Session, error) {
	args := m.Called(ctx, projectID)
	return args.Get(0).([]model.Session), args.Error(1)
}

func (m *MockSessionRepository) Update(ctx context.Context, session *model.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockSessionRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.SessionStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockSessionRepository) UpdateOutput(ctx context.Context, id uuid.UUID, output string) error {
	args := m.Called(ctx, id, output)
	return args.Error(0)
}

func (m *MockSessionRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func setupSessionServiceTest() (*sessionService, *MockSessionRepository) {
	sessionRepo := new(MockSessionRepository)
	taskRepo := new(MockTaskRepository)
	projectRepo := new(MockProjectRepository)
	k8sService := new(MockKubernetesService)

	service := &sessionService{
		sessionRepo: sessionRepo,
		taskRepo:    taskRepo,
		projectRepo: projectRepo,
		k8sService:  k8sService,
		httpClient:  &http.Client{}, // Initialize HTTP client for StopSession test
	}

	return service, sessionRepo
}

func TestSessionService_GetSession(t *testing.T) {
	service, sessionRepo := setupSessionServiceTest()
	ctx := context.Background()

	sessionID := uuid.New()
	expected := &model.Session{
		ID:        sessionID,
		TaskID:    uuid.New(),
		ProjectID: uuid.New(),
		Status:    model.SessionStatusRunning,
	}

	sessionRepo.On("FindByID", ctx, sessionID).Return(expected, nil)

	result, err := service.GetSession(ctx, sessionID)
	require.NoError(t, err)
	assert.Equal(t, expected.ID, result.ID)
	assert.Equal(t, expected.Status, result.Status)

	sessionRepo.AssertExpectations(t)
}

func TestSessionService_GetSession_NotFound(t *testing.T) {
	service, sessionRepo := setupSessionServiceTest()
	ctx := context.Background()

	sessionID := uuid.New()
	sessionRepo.On("FindByID", ctx, sessionID).Return(nil, gorm.ErrRecordNotFound)

	_, err := service.GetSession(ctx, sessionID)
	assert.ErrorIs(t, err, ErrSessionNotFound)

	sessionRepo.AssertExpectations(t)
}

func TestSessionService_GetSessionsByTaskID(t *testing.T) {
	service, sessionRepo := setupSessionServiceTest()
	ctx := context.Background()

	taskID := uuid.New()
	expected := []model.Session{
		{ID: uuid.New(), TaskID: taskID, Status: model.SessionStatusCompleted},
		{ID: uuid.New(), TaskID: taskID, Status: model.SessionStatusRunning},
	}

	sessionRepo.On("FindByTaskID", ctx, taskID).Return(expected, nil)

	result, err := service.GetSessionsByTaskID(ctx, taskID)
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, expected[0].ID, result[0].ID)

	sessionRepo.AssertExpectations(t)
}

func TestSessionService_GetActiveProjectSessions(t *testing.T) {
	service, sessionRepo := setupSessionServiceTest()
	ctx := context.Background()

	projectID := uuid.New()
	expected := []model.Session{
		{ID: uuid.New(), ProjectID: projectID, Status: model.SessionStatusPending},
		{ID: uuid.New(), ProjectID: projectID, Status: model.SessionStatusRunning},
	}

	sessionRepo.On("FindActiveSessionsForProject", ctx, projectID).Return(expected, nil)

	result, err := service.GetActiveProjectSessions(ctx, projectID)
	require.NoError(t, err)
	assert.Len(t, result, 2)

	sessionRepo.AssertExpectations(t)
}

func TestSessionService_UpdateSessionOutput(t *testing.T) {
	service, sessionRepo := setupSessionServiceTest()
	ctx := context.Background()

	sessionID := uuid.New()
	output := "Test output"

	sessionRepo.On("UpdateOutput", ctx, sessionID, output).Return(nil)

	err := service.UpdateSessionOutput(ctx, sessionID, output)
	require.NoError(t, err)

	sessionRepo.AssertExpectations(t)
}

func TestSessionService_StopSession(t *testing.T) {
	service, sessionRepo := setupSessionServiceTest()
	ctx := context.Background()

	sessionID := uuid.New()
	projectID := uuid.New()
	startedAt := time.Now().Add(-5 * time.Minute)

	session := &model.Session{
		ID:        sessionID,
		TaskID:    uuid.New(),
		ProjectID: projectID,
		Status:    model.SessionStatusRunning,
		StartedAt: &startedAt,
	}

	project := &model.Project{
		ID:           projectID,
		PodName:      "test-pod",
		PodNamespace: "opencode",
	}

	mockAPIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/sessions/"+sessionID.String()+"/stop")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("OpenCode API error"))
	}))
	defer mockAPIServer.Close()

	podIP := mockAPIServer.URL[7:]

	sessionRepo.On("FindByID", ctx, sessionID).Return(session, nil)
	service.projectRepo.(*MockProjectRepository).On("FindByID", ctx, projectID).Return(project, nil)
	service.k8sService.(*MockKubernetesService).On("GetPodIP", ctx, "test-pod", "opencode").Return(podIP, nil)

	err := service.StopSession(ctx, sessionID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrOpenCodeAPICall)

	sessionRepo.AssertExpectations(t)
	service.projectRepo.(*MockProjectRepository).AssertExpectations(t)
	service.k8sService.(*MockKubernetesService).AssertExpectations(t)
}

func TestSessionService_StopSession_NotFound(t *testing.T) {
	service, sessionRepo := setupSessionServiceTest()
	ctx := context.Background()

	sessionID := uuid.New()
	sessionRepo.On("FindByID", ctx, sessionID).Return(nil, gorm.ErrRecordNotFound)

	err := service.StopSession(ctx, sessionID)
	assert.ErrorIs(t, err, ErrSessionNotFound)

	sessionRepo.AssertExpectations(t)
}

func TestSessionService_StopSession_InvalidStatus(t *testing.T) {
	service, sessionRepo := setupSessionServiceTest()
	ctx := context.Background()

	sessionID := uuid.New()
	session := &model.Session{
		ID:        sessionID,
		TaskID:    uuid.New(),
		ProjectID: uuid.New(),
		Status:    model.SessionStatusCompleted,
	}

	sessionRepo.On("FindByID", ctx, sessionID).Return(session, nil)

	err := service.StopSession(ctx, sessionID)
	assert.ErrorIs(t, err, ErrInvalidSessionStatus)

	sessionRepo.AssertExpectations(t)
}

func TestSessionService_StartSession_TaskNotFound(t *testing.T) {
	service, _ := setupSessionServiceTest()
	ctx := context.Background()

	taskID := uuid.New()
	service.taskRepo.(*MockTaskRepository).On("FindByID", ctx, taskID).Return(nil, gorm.ErrRecordNotFound)

	_, err := service.StartSession(ctx, taskID, "Test prompt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task not found")

	service.taskRepo.(*MockTaskRepository).AssertExpectations(t)
}

func TestSessionService_StartSession_ProjectNotFound(t *testing.T) {
	service, _ := setupSessionServiceTest()
	ctx := context.Background()

	taskID := uuid.New()
	projectID := uuid.New()

	task := &model.Task{
		ID:        taskID,
		ProjectID: projectID,
	}

	service.taskRepo.(*MockTaskRepository).On("FindByID", ctx, taskID).Return(task, nil)
	service.projectRepo.(*MockProjectRepository).On("FindByID", ctx, projectID).Return(nil, gorm.ErrRecordNotFound)

	_, err := service.StartSession(ctx, taskID, "Test prompt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project not found")

	service.taskRepo.(*MockTaskRepository).AssertExpectations(t)
	service.projectRepo.(*MockProjectRepository).AssertExpectations(t)
}

func TestSessionService_StartSession_AlreadyActive(t *testing.T) {
	service, sessionRepo := setupSessionServiceTest()
	ctx := context.Background()

	taskID := uuid.New()
	projectID := uuid.New()

	task := &model.Task{
		ID:        taskID,
		ProjectID: projectID,
	}

	project := &model.Project{
		ID:           projectID,
		PodName:      "test-pod",
		PodNamespace: "opencode",
	}

	activeSessions := []model.Session{
		{ID: uuid.New(), TaskID: taskID, ProjectID: projectID, Status: model.SessionStatusRunning},
	}

	service.taskRepo.(*MockTaskRepository).On("FindByID", ctx, taskID).Return(task, nil)
	service.projectRepo.(*MockProjectRepository).On("FindByID", ctx, projectID).Return(project, nil)
	sessionRepo.On("FindActiveSessionsForProject", ctx, projectID).Return(activeSessions, nil)

	_, err := service.StartSession(ctx, taskID, "Test prompt")
	assert.ErrorIs(t, err, ErrSessionAlreadyActive)

	service.taskRepo.(*MockTaskRepository).AssertExpectations(t)
	service.projectRepo.(*MockProjectRepository).AssertExpectations(t)
	sessionRepo.AssertExpectations(t)
}

func TestSessionService_callOpenCodeStart_ErrorHandling(t *testing.T) {
	service := &sessionService{
		httpClient: nil, // Will cause error
	}

	err := service.callOpenCodeStart(context.Background(), "10.0.0.1", uuid.New(), "test")
	assert.Error(t, err)
}

func TestSessionService_callOpenCodeStop_ErrorHandling(t *testing.T) {
	service := &sessionService{
		httpClient: nil, // Will cause error
	}

	err := service.callOpenCodeStop(context.Background(), "10.0.0.1", uuid.New())
	assert.Error(t, err)
}
