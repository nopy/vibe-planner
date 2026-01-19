package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"

	"github.com/npinot/vibe/backend/internal/model"
)

// Mock for InteractionRepository specifically
type mockInteractionRepo struct {
	mock.Mock
}

func (m *mockInteractionRepo) Create(ctx context.Context, interaction *model.Interaction) error {
	args := m.Called(ctx, interaction)
	return args.Error(0)
}

func (m *mockInteractionRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Interaction, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Interaction), args.Error(1)
}

func (m *mockInteractionRepo) FindByTaskID(ctx context.Context, taskID uuid.UUID) ([]model.Interaction, error) {
	args := m.Called(ctx, taskID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Interaction), args.Error(1)
}

func (m *mockInteractionRepo) FindBySessionID(ctx context.Context, sessionID uuid.UUID) ([]model.Interaction, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Interaction), args.Error(1)
}

func (m *mockInteractionRepo) DeleteByTaskID(ctx context.Context, taskID uuid.UUID) error {
	args := m.Called(ctx, taskID)
	return args.Error(0)
}

// Mock for SessionRepository
type mockSessionRepo struct {
	mock.Mock
}

func (m *mockSessionRepo) Create(ctx context.Context, session *model.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *mockSessionRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Session, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Session), args.Error(1)
}

func (m *mockSessionRepo) FindByTaskID(ctx context.Context, taskID uuid.UUID) ([]model.Session, error) {
	args := m.Called(ctx, taskID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Session), args.Error(1)
}

func (m *mockSessionRepo) Update(ctx context.Context, session *model.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *mockSessionRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.SessionStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *mockSessionRepo) FindActiveSessionsForProject(ctx context.Context, projectID uuid.UUID) ([]model.Session, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Session), args.Error(1)
}

func (m *mockSessionRepo) UpdateOutput(ctx context.Context, id uuid.UUID, output string) error {
	args := m.Called(ctx, id, output)
	return args.Error(0)
}

func (m *mockSessionRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// Test setup helper
func setupTestInteractionService() (*interactionService, *mockInteractionRepo, *MockTaskRepository, *MockProjectRepository, *mockSessionRepo) {
	mockInteractionRepo := new(mockInteractionRepo)
	mockTaskRepo := new(MockTaskRepository)
	mockProjectRepo := new(MockProjectRepository)
	mockSessionRepo := new(mockSessionRepo)

	service := &interactionService{
		interactionRepo: mockInteractionRepo,
		taskRepo:        mockTaskRepo,
		projectRepo:     mockProjectRepo,
		sessionRepo:     mockSessionRepo,
	}

	return service, mockInteractionRepo, mockTaskRepo, mockProjectRepo, mockSessionRepo
}

// Test CreateUserMessage - Success
func TestCreateUserMessage_Success(t *testing.T) {
	service, mockInteractionRepo, mockTaskRepo, mockProjectRepo, _ := setupTestInteractionService()
	ctx := context.Background()

	taskID := uuid.New()
	userID := uuid.New()
	projectID := uuid.New()
	content := "Can you add error handling to this function?"

	// Setup mocks
	mockTaskRepo.On("FindByID", ctx, taskID).Return(&model.Task{
		ID:        taskID,
		ProjectID: projectID,
	}, nil)
	mockProjectRepo.On("FindByID", ctx, projectID).Return(&model.Project{
		ID:     projectID,
		UserID: userID,
	}, nil)
	mockInteractionRepo.On("Create", ctx, mock.MatchedBy(func(i *model.Interaction) bool {
		return i.TaskID == taskID &&
			i.UserID == userID &&
			i.MessageType == MessageTypeUser &&
			i.Content == content &&
			i.SessionID == nil
	})).Return(nil)

	// Execute
	interaction, err := service.CreateUserMessage(ctx, taskID, userID, content, nil)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, interaction)
	assert.Equal(t, taskID, interaction.TaskID)
	assert.Equal(t, userID, interaction.UserID)
	assert.Equal(t, MessageTypeUser, interaction.MessageType)
	assert.Equal(t, content, interaction.Content)
	mockInteractionRepo.AssertExpectations(t)
}

// Test CreateUserMessage - Unauthorized
func TestCreateUserMessage_Unauthorized(t *testing.T) {
	service, _, mockTaskRepo, mockProjectRepo, _ := setupTestInteractionService()
	ctx := context.Background()

	taskID := uuid.New()
	userID := uuid.New()
	projectID := uuid.New()
	otherUserID := uuid.New()
	content := "Test message"

	// Setup mocks
	mockTaskRepo.On("FindByID", ctx, taskID).Return(&model.Task{
		ID:        taskID,
		ProjectID: projectID,
	}, nil)
	mockProjectRepo.On("FindByID", ctx, projectID).Return(&model.Project{
		ID:     projectID,
		UserID: otherUserID, // Different user owns the project
	}, nil)

	// Execute
	interaction, err := service.CreateUserMessage(ctx, taskID, userID, content, nil)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, interaction)
	assert.Equal(t, ErrTaskNotOwnedByUser, err)
}

// Test CreateUserMessage - Empty Content
func TestCreateUserMessage_EmptyContent(t *testing.T) {
	service, _, mockTaskRepo, mockProjectRepo, _ := setupTestInteractionService()
	ctx := context.Background()

	taskID := uuid.New()
	userID := uuid.New()
	projectID := uuid.New()

	// Setup mocks
	mockTaskRepo.On("FindByID", ctx, taskID).Return(&model.Task{
		ID:        taskID,
		ProjectID: projectID,
	}, nil)
	mockProjectRepo.On("FindByID", ctx, projectID).Return(&model.Project{
		ID:     projectID,
		UserID: userID,
	}, nil)

	// Execute
	interaction, err := service.CreateUserMessage(ctx, taskID, userID, "   ", nil)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, interaction)
	assert.ErrorIs(t, err, ErrInvalidMessageContent)
}

// Test CreateUserMessage - Task Not Found
func TestCreateUserMessage_TaskNotFound(t *testing.T) {
	service, _, mockTaskRepo, _, _ := setupTestInteractionService()
	ctx := context.Background()

	taskID := uuid.New()
	userID := uuid.New()

	// Setup mocks
	mockTaskRepo.On("FindByID", ctx, taskID).Return(nil, gorm.ErrRecordNotFound)

	// Execute
	interaction, err := service.CreateUserMessage(ctx, taskID, userID, "Test", nil)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, interaction)
	assert.Equal(t, ErrTaskNotFound, err)
}

// Test CreateAgentResponse - Success
func TestCreateAgentResponse_Success(t *testing.T) {
	service, mockInteractionRepo, mockTaskRepo, mockProjectRepo, mockSessionRepo := setupTestInteractionService()
	ctx := context.Background()

	taskID := uuid.New()
	userID := uuid.New()
	projectID := uuid.New()
	sessionID := uuid.New()
	content := "I've added try-catch blocks to handle potential errors..."

	// Setup mocks
	mockTaskRepo.On("FindByID", ctx, taskID).Return(&model.Task{
		ID:        taskID,
		ProjectID: projectID,
	}, nil)
	mockProjectRepo.On("FindByID", ctx, projectID).Return(&model.Project{
		ID:     projectID,
		UserID: userID,
	}, nil)
	mockSessionRepo.On("FindByID", ctx, sessionID).Return(&model.Session{
		ID:     sessionID,
		TaskID: taskID,
	}, nil)
	mockInteractionRepo.On("Create", ctx, mock.MatchedBy(func(i *model.Interaction) bool {
		return i.TaskID == taskID &&
			i.UserID == userID &&
			i.MessageType == MessageTypeAgent &&
			i.Content == content &&
			*i.SessionID == sessionID
	})).Return(nil)

	// Execute
	interaction, err := service.CreateAgentResponse(ctx, taskID, userID, sessionID, content, nil)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, interaction)
	assert.Equal(t, taskID, interaction.TaskID)
	assert.Equal(t, userID, interaction.UserID)
	assert.Equal(t, MessageTypeAgent, interaction.MessageType)
	assert.Equal(t, content, interaction.Content)
	assert.Equal(t, sessionID, *interaction.SessionID)
	mockInteractionRepo.AssertExpectations(t)
}

// Test CreateAgentResponse - Session Not Belong To Task
func TestCreateAgentResponse_SessionNotBelongToTask(t *testing.T) {
	service, _, mockTaskRepo, mockProjectRepo, mockSessionRepo := setupTestInteractionService()
	ctx := context.Background()

	taskID := uuid.New()
	userID := uuid.New()
	projectID := uuid.New()
	sessionID := uuid.New()
	otherTaskID := uuid.New()

	// Setup mocks
	mockTaskRepo.On("FindByID", ctx, taskID).Return(&model.Task{
		ID:        taskID,
		ProjectID: projectID,
	}, nil)
	mockProjectRepo.On("FindByID", ctx, projectID).Return(&model.Project{
		ID:     projectID,
		UserID: userID,
	}, nil)
	mockSessionRepo.On("FindByID", ctx, sessionID).Return(&model.Session{
		ID:     sessionID,
		TaskID: otherTaskID, // Different task
	}, nil)

	// Execute
	interaction, err := service.CreateAgentResponse(ctx, taskID, userID, sessionID, "Test", nil)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, interaction)
	assert.Contains(t, err.Error(), "session does not belong to task")
}

// Test CreateSystemNotification - Success
func TestCreateSystemNotification_Success(t *testing.T) {
	service, mockInteractionRepo, mockTaskRepo, mockProjectRepo, _ := setupTestInteractionService()
	ctx := context.Background()

	taskID := uuid.New()
	userID := uuid.New()
	projectID := uuid.New()
	content := "Task execution started"

	// Setup mocks
	mockTaskRepo.On("FindByID", ctx, taskID).Return(&model.Task{
		ID:        taskID,
		ProjectID: projectID,
	}, nil)
	mockProjectRepo.On("FindByID", ctx, projectID).Return(&model.Project{
		ID:     projectID,
		UserID: userID,
	}, nil)
	mockInteractionRepo.On("Create", ctx, mock.MatchedBy(func(i *model.Interaction) bool {
		return i.TaskID == taskID &&
			i.UserID == userID &&
			i.MessageType == MessageTypeSystem &&
			i.Content == content &&
			i.SessionID == nil
	})).Return(nil)

	// Execute
	interaction, err := service.CreateSystemNotification(ctx, taskID, userID, nil, content, nil)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, interaction)
	assert.Equal(t, MessageTypeSystem, interaction.MessageType)
}

// Test CreateSystemNotification - With Session
func TestCreateSystemNotification_WithSession(t *testing.T) {
	service, mockInteractionRepo, mockTaskRepo, mockProjectRepo, mockSessionRepo := setupTestInteractionService()
	ctx := context.Background()

	taskID := uuid.New()
	userID := uuid.New()
	projectID := uuid.New()
	sessionID := uuid.New()
	content := "Agent is analyzing code..."

	// Setup mocks
	mockTaskRepo.On("FindByID", ctx, taskID).Return(&model.Task{
		ID:        taskID,
		ProjectID: projectID,
	}, nil)
	mockProjectRepo.On("FindByID", ctx, projectID).Return(&model.Project{
		ID:     projectID,
		UserID: userID,
	}, nil)
	mockSessionRepo.On("FindByID", ctx, sessionID).Return(&model.Session{
		ID:     sessionID,
		TaskID: taskID,
	}, nil)
	mockInteractionRepo.On("Create", ctx, mock.MatchedBy(func(i *model.Interaction) bool {
		return i.TaskID == taskID &&
			i.MessageType == MessageTypeSystem &&
			*i.SessionID == sessionID
	})).Return(nil)

	// Execute
	interaction, err := service.CreateSystemNotification(ctx, taskID, userID, &sessionID, content, nil)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, interaction)
	assert.Equal(t, sessionID, *interaction.SessionID)
}

// Test GetTaskHistory - Success
func TestGetTaskHistory_Success(t *testing.T) {
	service, mockInteractionRepo, mockTaskRepo, mockProjectRepo, _ := setupTestInteractionService()
	ctx := context.Background()

	taskID := uuid.New()
	userID := uuid.New()
	projectID := uuid.New()

	expectedInteractions := []model.Interaction{
		{
			ID:          uuid.New(),
			TaskID:      taskID,
			UserID:      userID,
			MessageType: MessageTypeUser,
			Content:     "First message",
			CreatedAt:   time.Now().Add(-2 * time.Hour),
		},
		{
			ID:          uuid.New(),
			TaskID:      taskID,
			UserID:      userID,
			MessageType: MessageTypeAgent,
			Content:     "Agent response",
			CreatedAt:   time.Now().Add(-1 * time.Hour),
		},
	}

	// Setup mocks
	mockTaskRepo.On("FindByID", ctx, taskID).Return(&model.Task{
		ID:        taskID,
		ProjectID: projectID,
	}, nil)
	mockProjectRepo.On("FindByID", ctx, projectID).Return(&model.Project{
		ID:     projectID,
		UserID: userID,
	}, nil)
	mockInteractionRepo.On("FindByTaskID", ctx, taskID).Return(expectedInteractions, nil)

	// Execute
	interactions, err := service.GetTaskHistory(ctx, taskID, userID)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, interactions, 2)
	assert.Equal(t, "First message", interactions[0].Content)
	assert.Equal(t, "Agent response", interactions[1].Content)
}

// Test GetTaskHistory - Unauthorized
func TestGetTaskHistory_Unauthorized(t *testing.T) {
	service, _, mockTaskRepo, mockProjectRepo, _ := setupTestInteractionService()
	ctx := context.Background()

	taskID := uuid.New()
	userID := uuid.New()
	projectID := uuid.New()
	otherUserID := uuid.New()

	// Setup mocks
	mockTaskRepo.On("FindByID", ctx, taskID).Return(&model.Task{
		ID:        taskID,
		ProjectID: projectID,
	}, nil)
	mockProjectRepo.On("FindByID", ctx, projectID).Return(&model.Project{
		ID:     projectID,
		UserID: otherUserID,
	}, nil)

	// Execute
	interactions, err := service.GetTaskHistory(ctx, taskID, userID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, interactions)
	assert.Equal(t, ErrTaskNotOwnedByUser, err)
}

// Test GetSessionHistory - Success
func TestGetSessionHistory_Success(t *testing.T) {
	service, mockInteractionRepo, mockTaskRepo, mockProjectRepo, mockSessionRepo := setupTestInteractionService()
	ctx := context.Background()

	sessionID := uuid.New()
	taskID := uuid.New()
	userID := uuid.New()
	projectID := uuid.New()

	expectedInteractions := []model.Interaction{
		{
			ID:          uuid.New(),
			TaskID:      taskID,
			SessionID:   &sessionID,
			UserID:      userID,
			MessageType: MessageTypeAgent,
			Content:     "Session message",
			CreatedAt:   time.Now(),
		},
	}

	// Setup mocks
	mockSessionRepo.On("FindByID", ctx, sessionID).Return(&model.Session{
		ID:     sessionID,
		TaskID: taskID,
	}, nil)
	mockTaskRepo.On("FindByID", ctx, taskID).Return(&model.Task{
		ID:        taskID,
		ProjectID: projectID,
	}, nil)
	mockProjectRepo.On("FindByID", ctx, projectID).Return(&model.Project{
		ID:     projectID,
		UserID: userID,
	}, nil)
	mockInteractionRepo.On("FindBySessionID", ctx, sessionID).Return(expectedInteractions, nil)

	// Execute
	interactions, err := service.GetSessionHistory(ctx, sessionID, userID)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, interactions, 1)
	assert.Equal(t, "Session message", interactions[0].Content)
}

// Test GetSessionHistory - Session Not Found
func TestGetSessionHistory_SessionNotFound(t *testing.T) {
	service, _, _, _, mockSessionRepo := setupTestInteractionService()
	ctx := context.Background()

	sessionID := uuid.New()
	userID := uuid.New()

	// Setup mocks
	mockSessionRepo.On("FindByID", ctx, sessionID).Return(nil, gorm.ErrRecordNotFound)

	// Execute
	interactions, err := service.GetSessionHistory(ctx, sessionID, userID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, interactions)
	assert.Contains(t, err.Error(), "session not found")
}

// Test DeleteTaskHistory - Success
func TestDeleteTaskHistory_Success(t *testing.T) {
	service, mockInteractionRepo, mockTaskRepo, mockProjectRepo, _ := setupTestInteractionService()
	ctx := context.Background()

	taskID := uuid.New()
	userID := uuid.New()
	projectID := uuid.New()

	// Setup mocks
	mockTaskRepo.On("FindByID", ctx, taskID).Return(&model.Task{
		ID:        taskID,
		ProjectID: projectID,
	}, nil)
	mockProjectRepo.On("FindByID", ctx, projectID).Return(&model.Project{
		ID:     projectID,
		UserID: userID,
	}, nil)
	mockInteractionRepo.On("DeleteByTaskID", ctx, taskID).Return(nil)

	// Execute
	err := service.DeleteTaskHistory(ctx, taskID, userID)

	// Assert
	assert.NoError(t, err)
	mockInteractionRepo.AssertExpectations(t)
}

// Test DeleteTaskHistory - Unauthorized
func TestDeleteTaskHistory_Unauthorized(t *testing.T) {
	service, _, mockTaskRepo, mockProjectRepo, _ := setupTestInteractionService()
	ctx := context.Background()

	taskID := uuid.New()
	userID := uuid.New()
	projectID := uuid.New()
	otherUserID := uuid.New()

	// Setup mocks
	mockTaskRepo.On("FindByID", ctx, taskID).Return(&model.Task{
		ID:        taskID,
		ProjectID: projectID,
	}, nil)
	mockProjectRepo.On("FindByID", ctx, projectID).Return(&model.Project{
		ID:     projectID,
		UserID: otherUserID,
	}, nil)

	// Execute
	err := service.DeleteTaskHistory(ctx, taskID, userID)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, ErrTaskNotOwnedByUser, err)
}

// Test ValidateTaskOwnership - Success
func TestValidateTaskOwnership_Success(t *testing.T) {
	service, _, mockTaskRepo, mockProjectRepo, _ := setupTestInteractionService()
	ctx := context.Background()

	taskID := uuid.New()
	userID := uuid.New()
	projectID := uuid.New()

	// Setup mocks
	mockTaskRepo.On("FindByID", ctx, taskID).Return(&model.Task{
		ID:        taskID,
		ProjectID: projectID,
	}, nil)
	mockProjectRepo.On("FindByID", ctx, projectID).Return(&model.Project{
		ID:     projectID,
		UserID: userID,
	}, nil)

	// Execute
	err := service.ValidateTaskOwnership(ctx, taskID, userID)

	// Assert
	assert.NoError(t, err)
}

// Test ValidateTaskOwnership - Task Not Found
func TestValidateTaskOwnership_TaskNotFound(t *testing.T) {
	service, _, mockTaskRepo, _, _ := setupTestInteractionService()
	ctx := context.Background()

	taskID := uuid.New()
	userID := uuid.New()

	// Setup mocks
	mockTaskRepo.On("FindByID", ctx, taskID).Return(nil, gorm.ErrRecordNotFound)

	// Execute
	err := service.ValidateTaskOwnership(ctx, taskID, userID)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, ErrTaskNotFound, err)
}

// Test ValidateTaskOwnership - Project Not Found
func TestValidateTaskOwnership_ProjectNotFound(t *testing.T) {
	service, _, mockTaskRepo, mockProjectRepo, _ := setupTestInteractionService()
	ctx := context.Background()

	taskID := uuid.New()
	userID := uuid.New()
	projectID := uuid.New()

	// Setup mocks
	mockTaskRepo.On("FindByID", ctx, taskID).Return(&model.Task{
		ID:        taskID,
		ProjectID: projectID,
	}, nil)
	mockProjectRepo.On("FindByID", ctx, projectID).Return(nil, gorm.ErrRecordNotFound)

	// Execute
	err := service.ValidateTaskOwnership(ctx, taskID, userID)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, ErrProjectNotFound, err)
}

// Test ValidateTaskOwnership - Unauthorized
func TestValidateTaskOwnership_Unauthorized(t *testing.T) {
	service, _, mockTaskRepo, mockProjectRepo, _ := setupTestInteractionService()
	ctx := context.Background()

	taskID := uuid.New()
	userID := uuid.New()
	projectID := uuid.New()
	otherUserID := uuid.New()

	// Setup mocks
	mockTaskRepo.On("FindByID", ctx, taskID).Return(&model.Task{
		ID:        taskID,
		ProjectID: projectID,
	}, nil)
	mockProjectRepo.On("FindByID", ctx, projectID).Return(&model.Project{
		ID:     projectID,
		UserID: otherUserID,
	}, nil)

	// Execute
	err := service.ValidateTaskOwnership(ctx, taskID, userID)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, ErrTaskNotOwnedByUser, err)
}

// Test validateMessageContent - Empty
func TestValidateMessageContent_Empty(t *testing.T) {
	err := validateMessageContent("", MessageTypeUser)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidMessageContent)
}

// Test validateMessageContent - Whitespace Only
func TestValidateMessageContent_WhitespaceOnly(t *testing.T) {
	err := validateMessageContent("   \n\t  ", MessageTypeUser)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidMessageContent)
}

// Test validateMessageContent - Too Long User Message
func TestValidateMessageContent_TooLongUserMessage(t *testing.T) {
	longContent := string(make([]byte, 2001))
	err := validateMessageContent(longContent, MessageTypeUser)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidMessageContent)
	assert.Contains(t, err.Error(), "2000")
}

// Test validateMessageContent - Agent Message Under Limit
func TestValidateMessageContent_AgentMessageUnderLimit(t *testing.T) {
	content := string(make([]byte, 10000))
	err := validateMessageContent(content, MessageTypeAgent)
	assert.NoError(t, err)
}

// Test validateMessageContent - System Message Too Long
func TestValidateMessageContent_SystemMessageTooLong(t *testing.T) {
	longContent := string(make([]byte, 50001))
	err := validateMessageContent(longContent, MessageTypeSystem)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidMessageContent)
}

// Test validateMessageContent - Valid
func TestValidateMessageContent_Valid(t *testing.T) {
	err := validateMessageContent("This is a valid message", MessageTypeUser)
	assert.NoError(t, err)
}

// Test CreateUserMessage - Repository Error
func TestCreateUserMessage_RepositoryError(t *testing.T) {
	service, mockInteractionRepo, mockTaskRepo, mockProjectRepo, _ := setupTestInteractionService()
	ctx := context.Background()

	taskID := uuid.New()
	userID := uuid.New()
	projectID := uuid.New()

	// Setup mocks
	mockTaskRepo.On("FindByID", ctx, taskID).Return(&model.Task{
		ID:        taskID,
		ProjectID: projectID,
	}, nil)
	mockProjectRepo.On("FindByID", ctx, projectID).Return(&model.Project{
		ID:     projectID,
		UserID: userID,
	}, nil)
	mockInteractionRepo.On("Create", ctx, mock.Anything).Return(errors.New("database error"))

	// Execute
	interaction, err := service.CreateUserMessage(ctx, taskID, userID, "Test", nil)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, interaction)
	assert.Contains(t, err.Error(), "database error")
}
