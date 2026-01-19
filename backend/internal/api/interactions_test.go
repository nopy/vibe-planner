package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/npinot/vibe/backend/internal/model"
	"github.com/npinot/vibe/backend/internal/service"
)

type MockInteractionService struct {
	mock.Mock
}

func (m *MockInteractionService) CreateUserMessage(ctx context.Context, taskID, userID uuid.UUID, content string, metadata model.JSONB) (*model.Interaction, error) {
	args := m.Called(ctx, taskID, userID, content, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Interaction), args.Error(1)
}

func (m *MockInteractionService) CreateAgentResponse(ctx context.Context, taskID, userID, sessionID uuid.UUID, content string, metadata model.JSONB) (*model.Interaction, error) {
	args := m.Called(ctx, taskID, userID, sessionID, content, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Interaction), args.Error(1)
}

func (m *MockInteractionService) CreateSystemNotification(ctx context.Context, taskID, userID uuid.UUID, sessionID *uuid.UUID, content string, metadata model.JSONB) (*model.Interaction, error) {
	args := m.Called(ctx, taskID, userID, sessionID, content, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Interaction), args.Error(1)
}

func (m *MockInteractionService) GetTaskHistory(ctx context.Context, taskID, userID uuid.UUID) ([]model.Interaction, error) {
	args := m.Called(ctx, taskID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Interaction), args.Error(1)
}

func (m *MockInteractionService) GetSessionHistory(ctx context.Context, sessionID, userID uuid.UUID) ([]model.Interaction, error) {
	args := m.Called(ctx, sessionID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Interaction), args.Error(1)
}

func (m *MockInteractionService) DeleteTaskHistory(ctx context.Context, taskID, userID uuid.UUID) error {
	args := m.Called(ctx, taskID, userID)
	return args.Error(0)
}

func (m *MockInteractionService) ValidateTaskOwnership(ctx context.Context, taskID, userID uuid.UUID) error {
	args := m.Called(ctx, taskID, userID)
	return args.Error(0)
}

func setupTestInteractionHandler() (*InteractionHandler, *MockInteractionService) {
	gin.SetMode(gin.TestMode)
	mockService := new(MockInteractionService)
	handler := NewInteractionHandler(mockService)
	return handler, mockService
}

func TestNewInteractionHandler(t *testing.T) {
	handler, _ := setupTestInteractionHandler()

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.interactionService)
	assert.NotNil(t, handler.sessions)
	assert.NotNil(t, handler.sessions.connections)
	assert.True(t, handler.upgrader.CheckOrigin(nil))
}

func TestGetTaskHistory_Success(t *testing.T) {
	handler, mockService := setupTestInteractionHandler()

	taskID := uuid.New()
	userID := uuid.New()
	sessionID := uuid.New()

	interactions := []model.Interaction{
		{
			ID:          uuid.New(),
			TaskID:      taskID,
			SessionID:   &sessionID,
			UserID:      userID,
			MessageType: "user_message",
			Content:     "Hello",
			Metadata:    model.JSONB{"key": "value"},
			CreatedAt:   time.Now(),
		},
		{
			ID:          uuid.New(),
			TaskID:      taskID,
			SessionID:   &sessionID,
			UserID:      userID,
			MessageType: "agent_response",
			Content:     "Hi there",
			Metadata:    model.JSONB{},
			CreatedAt:   time.Now(),
		},
	}

	mockService.On("GetTaskHistory", mock.Anything, taskID, userID).Return(interactions, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("currentUser", &model.User{ID: userID})
	c.Params = gin.Params{{Key: "id", Value: taskID.String()}}
	c.Request = httptest.NewRequest("GET", "/api/tasks/"+taskID.String()+"/interactions", nil)

	handler.GetTaskHistory(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string][]map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response["interactions"], 2)
	assert.Equal(t, "user_message", response["interactions"][0]["message_type"])
	assert.Equal(t, "Hello", response["interactions"][0]["content"])
}

func TestGetTaskHistory_InvalidTaskID(t *testing.T) {
	handler, _ := setupTestInteractionHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("currentUser", &model.User{ID: uuid.New()})
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	c.Request = httptest.NewRequest("GET", "/api/tasks/invalid-uuid/interactions", nil)

	handler.GetTaskHistory(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid task ID")
}

func TestGetTaskHistory_UserNotAuthenticated(t *testing.T) {
	handler, _ := setupTestInteractionHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}
	c.Request = httptest.NewRequest("GET", "/api/tasks/"+uuid.New().String()+"/interactions", nil)

	handler.GetTaskHistory(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "User not found")
}

func TestGetTaskHistory_TaskNotFound(t *testing.T) {
	handler, mockService := setupTestInteractionHandler()

	taskID := uuid.New()
	userID := uuid.New()

	mockService.On("GetTaskHistory", mock.Anything, taskID, userID).Return(nil, service.ErrTaskNotFound)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("currentUser", &model.User{ID: userID})
	c.Params = gin.Params{{Key: "id", Value: taskID.String()}}
	c.Request = httptest.NewRequest("GET", "/api/tasks/"+taskID.String()+"/interactions", nil)

	handler.GetTaskHistory(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "Task not found")
}

func TestGetTaskHistory_Unauthorized(t *testing.T) {
	handler, mockService := setupTestInteractionHandler()

	taskID := uuid.New()
	userID := uuid.New()

	mockService.On("GetTaskHistory", mock.Anything, taskID, userID).Return(nil, service.ErrTaskNotOwnedByUser)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("currentUser", &model.User{ID: userID})
	c.Params = gin.Params{{Key: "id", Value: taskID.String()}}
	c.Request = httptest.NewRequest("GET", "/api/tasks/"+taskID.String()+"/interactions", nil)

	handler.GetTaskHistory(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "Access denied")
}

func TestGetTaskHistory_InternalError(t *testing.T) {
	handler, mockService := setupTestInteractionHandler()

	taskID := uuid.New()
	userID := uuid.New()

	mockService.On("GetTaskHistory", mock.Anything, taskID, userID).Return(nil, fmt.Errorf("database error"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("currentUser", &model.User{ID: userID})
	c.Params = gin.Params{{Key: "id", Value: taskID.String()}}
	c.Request = httptest.NewRequest("GET", "/api/tasks/"+taskID.String()+"/interactions", nil)

	handler.GetTaskHistory(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to fetch interaction history")
}

func TestSessionManager_RegisterUnregister(t *testing.T) {
	sm := &sessionManager{
		connections: make(map[uuid.UUID]map[*websocket.Conn]bool),
	}

	taskID := uuid.New()
	conn1 := &websocket.Conn{}
	conn2 := &websocket.Conn{}

	sm.register(taskID, conn1)
	assert.Len(t, sm.connections[taskID], 1)

	sm.register(taskID, conn2)
	assert.Len(t, sm.connections[taskID], 2)

	sm.unregister(taskID, conn1)
	assert.Len(t, sm.connections[taskID], 1)

	sm.unregister(taskID, conn2)
	_, exists := sm.connections[taskID]
	assert.False(t, exists)
}

func TestBroadcastAgentResponse(t *testing.T) {
	handler, _ := setupTestInteractionHandler()

	taskID := uuid.New()

	conn := &websocket.Conn{}
	handler.sessions.register(taskID, conn)

	assert.Len(t, handler.sessions.connections[taskID], 1)

	handler.sessions.unregister(taskID, conn)
	_, exists := handler.sessions.connections[taskID]
	assert.False(t, exists)
}

func TestBroadcastSystemNotification(t *testing.T) {
	handler, _ := setupTestInteractionHandler()

	taskID := uuid.New()

	conn1 := &websocket.Conn{}
	conn2 := &websocket.Conn{}

	handler.sessions.register(taskID, conn1)
	handler.sessions.register(taskID, conn2)

	assert.Len(t, handler.sessions.connections[taskID], 2)

	handler.sessions.unregister(taskID, conn1)
	assert.Len(t, handler.sessions.connections[taskID], 1)

	handler.sessions.unregister(taskID, conn2)
	_, exists := handler.sessions.connections[taskID]
	assert.False(t, exists)
}

func TestTaskInteractionWebSocket_InvalidTaskID(t *testing.T) {
	handler, _ := setupTestInteractionHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("currentUser", &model.User{ID: uuid.New()})
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	c.Request = httptest.NewRequest("GET", "/api/tasks/invalid-uuid/interact", nil)

	handler.TaskInteractionWebSocket(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid task ID")
}

func TestTaskInteractionWebSocket_UserNotAuthenticated(t *testing.T) {
	handler, _ := setupTestInteractionHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}
	c.Request = httptest.NewRequest("GET", "/api/tasks/"+uuid.New().String()+"/interact", nil)

	handler.TaskInteractionWebSocket(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "User not found")
}

func TestTaskInteractionWebSocket_TaskNotFound(t *testing.T) {
	handler, mockService := setupTestInteractionHandler()

	taskID := uuid.New()
	userID := uuid.New()

	mockService.On("ValidateTaskOwnership", mock.Anything, taskID, userID).Return(service.ErrTaskNotFound)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("currentUser", &model.User{ID: userID})
	c.Params = gin.Params{{Key: "id", Value: taskID.String()}}
	c.Request = httptest.NewRequest("GET", "/api/tasks/"+taskID.String()+"/interact", nil)

	handler.TaskInteractionWebSocket(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "Task not found")
}

func TestTaskInteractionWebSocket_Unauthorized(t *testing.T) {
	handler, mockService := setupTestInteractionHandler()

	taskID := uuid.New()
	userID := uuid.New()

	mockService.On("ValidateTaskOwnership", mock.Anything, taskID, userID).Return(service.ErrTaskNotOwnedByUser)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("currentUser", &model.User{ID: userID})
	c.Params = gin.Params{{Key: "id", Value: taskID.String()}}
	c.Request = httptest.NewRequest("GET", "/api/tasks/"+taskID.String()+"/interact", nil)

	handler.TaskInteractionWebSocket(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "Access denied")
}

func TestTaskInteractionWebSocket_OwnershipValidationError(t *testing.T) {
	handler, mockService := setupTestInteractionHandler()

	taskID := uuid.New()
	userID := uuid.New()

	mockService.On("ValidateTaskOwnership", mock.Anything, taskID, userID).Return(fmt.Errorf("database error"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("currentUser", &model.User{ID: userID})
	c.Params = gin.Params{{Key: "id", Value: taskID.String()}}
	c.Request = httptest.NewRequest("GET", "/api/tasks/"+taskID.String()+"/interact", nil)

	handler.TaskInteractionWebSocket(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to validate task ownership")
}

func TestHandleUserMessage_Success(t *testing.T) {
	handler, mockService := setupTestInteractionHandler()

	ctx := context.Background()
	taskID := uuid.New()
	userID := uuid.New()
	content := "User message content"
	metadata := model.JSONB{"key": "value"}

	expectedInteraction := &model.Interaction{
		ID:          uuid.New(),
		TaskID:      taskID,
		UserID:      userID,
		MessageType: "user_message",
		Content:     content,
		Metadata:    metadata,
		CreatedAt:   time.Now(),
	}

	mockService.On("CreateUserMessage", ctx, taskID, userID, content, metadata).Return(expectedInteraction, nil)

	msg := WebSocketMessage{
		Type:     "user_message",
		Content:  content,
		Metadata: map[string]interface{}{"key": "value"},
	}

	err := handler.handleUserMessage(ctx, taskID, userID, msg)

	assert.NoError(t, err)
	mockService.AssertExpectations(t)
}

func TestHandleUserMessage_ServiceError(t *testing.T) {
	handler, mockService := setupTestInteractionHandler()

	ctx := context.Background()
	taskID := uuid.New()
	userID := uuid.New()
	content := "User message content"
	metadata := model.JSONB{"key": "value"}

	mockService.On("CreateUserMessage", ctx, taskID, userID, content, metadata).Return(nil, fmt.Errorf("database error"))

	msg := WebSocketMessage{
		Type:     "user_message",
		Content:  content,
		Metadata: map[string]interface{}{"key": "value"},
	}

	err := handler.handleUserMessage(ctx, taskID, userID, msg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store user message")
}

func TestWebSocketMessage_JSONSerialization(t *testing.T) {
	msg := WebSocketMessage{
		Type:      "user_message",
		Content:   "Test content",
		Metadata:  map[string]interface{}{"key": "value"},
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(msg)
	assert.NoError(t, err)

	var decoded WebSocketMessage
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, msg.Type, decoded.Type)
	assert.Equal(t, msg.Content, decoded.Content)
	assert.Equal(t, msg.Metadata["key"], decoded.Metadata["key"])
}
