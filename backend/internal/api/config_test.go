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
	"gorm.io/gorm"

	"github.com/npinot/vibe/backend/internal/model"
)

type MockConfigService struct {
	mock.Mock
}

func (m *MockConfigService) GetActiveConfig(ctx context.Context, projectID uuid.UUID) (*model.OpenCodeConfig, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.OpenCodeConfig), args.Error(1)
}

func (m *MockConfigService) CreateOrUpdateConfig(ctx context.Context, config *model.OpenCodeConfig, apiKey string) error {
	args := m.Called(ctx, config, apiKey)
	return args.Error(0)
}

func (m *MockConfigService) RollbackToVersion(ctx context.Context, projectID uuid.UUID, version int) error {
	args := m.Called(ctx, projectID, version)
	return args.Error(0)
}

func (m *MockConfigService) GetConfigHistory(ctx context.Context, projectID uuid.UUID) ([]model.OpenCodeConfig, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.OpenCodeConfig), args.Error(1)
}

func (m *MockConfigService) GetDecryptedAPIKey(ctx context.Context, projectID uuid.UUID) (string, error) {
	args := m.Called(ctx, projectID)
	return args.Get(0).(string), args.Error(1)
}

func setupConfigTestRouter(handler *ConfigHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		testUser := &model.User{
			ID:    uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			Email: "test@example.com",
		}
		c.Set("currentUser", testUser)
		c.Next()
	})

	return router
}

func TestConfigHandler_GetActiveConfig(t *testing.T) {
	mockService := new(MockConfigService)
	handler := NewConfigHandler(mockService)
	router := setupConfigTestRouter(handler)

	router.GET("/projects/:id/config", handler.GetActiveConfig)

	t.Run("successful retrieval", func(t *testing.T) {
		projectID := uuid.New()
		config := &model.OpenCodeConfig{
			ID:            uuid.New(),
			ProjectID:     projectID,
			Version:       1,
			IsActive:      true,
			ModelProvider: "openai",
			ModelName:     "gpt-4o-mini",
			Temperature:   0.7,
			MaxTokens:     4096,
			EnabledTools:  []string{"file_ops", "web_search"},
		}

		mockService.On("GetActiveConfig", mock.Anything, projectID).Return(config, nil).Once()

		req, _ := http.NewRequest("GET", "/projects/"+projectID.String()+"/config", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp model.OpenCodeConfig
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "gpt-4o-mini", resp.ModelName)
		assert.Equal(t, 1, resp.Version)

		mockService.AssertExpectations(t)
	})

	t.Run("config not found", func(t *testing.T) {
		projectID := uuid.New()

		mockService.On("GetActiveConfig", mock.Anything, projectID).Return(nil, gorm.ErrRecordNotFound).Once()

		req, _ := http.NewRequest("GET", "/projects/"+projectID.String()+"/config", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp["error"], "config not found")

		mockService.AssertExpectations(t)
	})

	t.Run("invalid project ID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/projects/invalid-uuid/config", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp["error"], "invalid project ID")
	})

	t.Run("internal server error", func(t *testing.T) {
		projectID := uuid.New()

		mockService.On("GetActiveConfig", mock.Anything, projectID).Return(nil, errors.New("database error")).Once()

		req, _ := http.NewRequest("GET", "/projects/"+projectID.String()+"/config", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		mockService.AssertExpectations(t)
	})
}

func TestConfigHandler_CreateOrUpdateConfig(t *testing.T) {
	mockService := new(MockConfigService)
	handler := NewConfigHandler(mockService)
	router := setupConfigTestRouter(handler)

	router.POST("/projects/:id/config", handler.CreateOrUpdateConfig)

	t.Run("successful creation", func(t *testing.T) {
		projectID := uuid.New()

		reqBody := CreateConfigRequest{
			ModelProvider:  "openai",
			ModelName:      "gpt-4o-mini",
			Temperature:    0.7,
			MaxTokens:      4096,
			EnabledTools:   []string{"file_ops", "web_search"},
			MaxIterations:  10,
			TimeoutSeconds: 300,
		}

		mockService.On("CreateOrUpdateConfig", mock.Anything, mock.AnythingOfType("*model.OpenCodeConfig"), "").Return(nil).Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/projects/"+projectID.String()+"/config", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp model.OpenCodeConfig
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "gpt-4o-mini", resp.ModelName)

		mockService.AssertExpectations(t)
	})

	t.Run("successful creation with API key", func(t *testing.T) {
		projectID := uuid.New()

		reqBody := CreateConfigRequest{
			ModelProvider:  "openai",
			ModelName:      "gpt-4o-mini",
			APIKey:         "sk-test-key-123",
			Temperature:    0.7,
			MaxTokens:      4096,
			EnabledTools:   []string{"file_ops"},
			MaxIterations:  10,
			TimeoutSeconds: 300,
		}

		mockService.On("CreateOrUpdateConfig", mock.Anything, mock.AnythingOfType("*model.OpenCodeConfig"), "sk-test-key-123").Return(nil).Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/projects/"+projectID.String()+"/config", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		mockService.AssertExpectations(t)
	})

	t.Run("invalid project ID", func(t *testing.T) {
		reqBody := CreateConfigRequest{
			ModelProvider:  "openai",
			ModelName:      "gpt-4o-mini",
			Temperature:    0.7,
			MaxTokens:      4096,
			EnabledTools:   []string{"file_ops"},
			MaxIterations:  10,
			TimeoutSeconds: 300,
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/projects/invalid-uuid/config", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		projectID := uuid.New()

		req, _ := http.NewRequest("POST", "/projects/"+projectID.String()+"/config", bytes.NewBufferString("not json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid model_provider", func(t *testing.T) {
		projectID := uuid.New()

		reqBody := map[string]interface{}{
			"model_provider":  "invalid-provider",
			"model_name":      "gpt-4o-mini",
			"temperature":     0.7,
			"max_tokens":      4096,
			"enabled_tools":   []string{"file_ops"},
			"max_iterations":  10,
			"timeout_seconds": 300,
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/projects/"+projectID.String()+"/config", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing required field - model_name", func(t *testing.T) {
		projectID := uuid.New()

		reqBody := map[string]interface{}{
			"model_provider":  "openai",
			"temperature":     0.7,
			"max_tokens":      4096,
			"enabled_tools":   []string{"file_ops"},
			"max_iterations":  10,
			"timeout_seconds": 300,
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/projects/"+projectID.String()+"/config", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("temperature out of range - too high", func(t *testing.T) {
		projectID := uuid.New()

		reqBody := map[string]interface{}{
			"model_provider":  "openai",
			"model_name":      "gpt-4o-mini",
			"temperature":     2.5,
			"max_tokens":      4096,
			"enabled_tools":   []string{"file_ops"},
			"max_iterations":  10,
			"timeout_seconds": 300,
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/projects/"+projectID.String()+"/config", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("temperature out of range - negative", func(t *testing.T) {
		projectID := uuid.New()

		reqBody := map[string]interface{}{
			"model_provider":  "openai",
			"model_name":      "gpt-4o-mini",
			"temperature":     -0.1,
			"max_tokens":      4096,
			"enabled_tools":   []string{"file_ops"},
			"max_iterations":  10,
			"timeout_seconds": 300,
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/projects/"+projectID.String()+"/config", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("max_tokens too high", func(t *testing.T) {
		projectID := uuid.New()

		reqBody := map[string]interface{}{
			"model_provider":  "openai",
			"model_name":      "gpt-4o-mini",
			"temperature":     0.7,
			"max_tokens":      200000,
			"enabled_tools":   []string{"file_ops"},
			"max_iterations":  10,
			"timeout_seconds": 300,
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/projects/"+projectID.String()+"/config", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("max_tokens zero", func(t *testing.T) {
		projectID := uuid.New()

		reqBody := map[string]interface{}{
			"model_provider":  "openai",
			"model_name":      "gpt-4o-mini",
			"temperature":     0.7,
			"max_tokens":      0,
			"enabled_tools":   []string{"file_ops"},
			"max_iterations":  10,
			"timeout_seconds": 300,
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/projects/"+projectID.String()+"/config", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("max_iterations too high", func(t *testing.T) {
		projectID := uuid.New()

		reqBody := map[string]interface{}{
			"model_provider":  "openai",
			"model_name":      "gpt-4o-mini",
			"temperature":     0.7,
			"max_tokens":      4096,
			"enabled_tools":   []string{"file_ops"},
			"max_iterations":  100,
			"timeout_seconds": 300,
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/projects/"+projectID.String()+"/config", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("timeout_seconds too low", func(t *testing.T) {
		projectID := uuid.New()

		reqBody := map[string]interface{}{
			"model_provider":  "openai",
			"model_name":      "gpt-4o-mini",
			"temperature":     0.7,
			"max_tokens":      4096,
			"enabled_tools":   []string{"file_ops"},
			"max_iterations":  10,
			"timeout_seconds": 30,
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/projects/"+projectID.String()+"/config", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service validation error", func(t *testing.T) {
		projectID := uuid.New()

		reqBody := CreateConfigRequest{
			ModelProvider:  "openai",
			ModelName:      "invalid-model",
			Temperature:    0.7,
			MaxTokens:      4096,
			EnabledTools:   []string{"file_ops"},
			MaxIterations:  10,
			TimeoutSeconds: 300,
		}

		mockService.On("CreateOrUpdateConfig", mock.Anything, mock.AnythingOfType("*model.OpenCodeConfig"), "").
			Return(errors.New("invalid OpenAI model: invalid-model")).Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/projects/"+projectID.String()+"/config", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		mockService.AssertExpectations(t)
	})
}

func TestConfigHandler_GetConfigHistory(t *testing.T) {
	mockService := new(MockConfigService)
	handler := NewConfigHandler(mockService)
	router := setupConfigTestRouter(handler)

	router.GET("/projects/:id/config/versions", handler.GetConfigHistory)

	t.Run("successful retrieval with multiple versions", func(t *testing.T) {
		projectID := uuid.New()
		configs := []model.OpenCodeConfig{
			{
				ID:            uuid.New(),
				ProjectID:     projectID,
				Version:       2,
				IsActive:      true,
				ModelProvider: "openai",
				ModelName:     "gpt-4o",
			},
			{
				ID:            uuid.New(),
				ProjectID:     projectID,
				Version:       1,
				IsActive:      false,
				ModelProvider: "openai",
				ModelName:     "gpt-4o-mini",
			},
		}

		mockService.On("GetConfigHistory", mock.Anything, projectID).Return(configs, nil).Once()

		req, _ := http.NewRequest("GET", "/projects/"+projectID.String()+"/config/versions", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp []model.OpenCodeConfig
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Len(t, resp, 2)
		assert.Equal(t, 2, resp[0].Version)

		mockService.AssertExpectations(t)
	})

	t.Run("empty history", func(t *testing.T) {
		projectID := uuid.New()
		configs := []model.OpenCodeConfig{}

		mockService.On("GetConfigHistory", mock.Anything, projectID).Return(configs, nil).Once()

		req, _ := http.NewRequest("GET", "/projects/"+projectID.String()+"/config/versions", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp []model.OpenCodeConfig
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Len(t, resp, 0)

		mockService.AssertExpectations(t)
	})

	t.Run("API keys sanitized in response", func(t *testing.T) {
		projectID := uuid.New()
		configs := []model.OpenCodeConfig{
			{
				ID:              uuid.New(),
				ProjectID:       projectID,
				Version:         1,
				IsActive:        true,
				ModelProvider:   "openai",
				ModelName:       "gpt-4o-mini",
				APIKeyEncrypted: nil,
			},
		}

		mockService.On("GetConfigHistory", mock.Anything, projectID).Return(configs, nil).Once()

		req, _ := http.NewRequest("GET", "/projects/"+projectID.String()+"/config/versions", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp []model.OpenCodeConfig
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Nil(t, resp[0].APIKeyEncrypted)

		mockService.AssertExpectations(t)
	})

	t.Run("invalid project ID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/projects/invalid-uuid/config/versions", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("internal server error", func(t *testing.T) {
		projectID := uuid.New()

		mockService.On("GetConfigHistory", mock.Anything, projectID).Return(nil, errors.New("database error")).Once()

		req, _ := http.NewRequest("GET", "/projects/"+projectID.String()+"/config/versions", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		mockService.AssertExpectations(t)
	})
}

func TestConfigHandler_RollbackConfig(t *testing.T) {
	mockService := new(MockConfigService)
	handler := NewConfigHandler(mockService)
	router := setupConfigTestRouter(handler)

	router.POST("/projects/:id/config/rollback/:version", handler.RollbackConfig)

	t.Run("successful rollback", func(t *testing.T) {
		projectID := uuid.New()
		version := 1

		mockService.On("RollbackToVersion", mock.Anything, projectID, version).Return(nil).Once()

		req, _ := http.NewRequest("POST", "/projects/"+projectID.String()+"/config/rollback/1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp["message"], "rolled back successfully")

		mockService.AssertExpectations(t)
	})

	t.Run("version not found", func(t *testing.T) {
		projectID := uuid.New()
		version := 999

		mockService.On("RollbackToVersion", mock.Anything, projectID, version).Return(gorm.ErrRecordNotFound).Once()

		req, _ := http.NewRequest("POST", "/projects/"+projectID.String()+"/config/rollback/999", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		mockService.AssertExpectations(t)
	})

	t.Run("invalid project ID", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/projects/invalid-uuid/config/rollback/1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid version - not a number", func(t *testing.T) {
		projectID := uuid.New()

		req, _ := http.NewRequest("POST", "/projects/"+projectID.String()+"/config/rollback/abc", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid version - zero", func(t *testing.T) {
		projectID := uuid.New()

		req, _ := http.NewRequest("POST", "/projects/"+projectID.String()+"/config/rollback/0", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid version - negative", func(t *testing.T) {
		projectID := uuid.New()

		req, _ := http.NewRequest("POST", "/projects/"+projectID.String()+"/config/rollback/-1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("internal server error", func(t *testing.T) {
		projectID := uuid.New()
		version := 1

		mockService.On("RollbackToVersion", mock.Anything, projectID, version).Return(errors.New("database error")).Once()

		req, _ := http.NewRequest("POST", "/projects/"+projectID.String()+"/config/rollback/1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		mockService.AssertExpectations(t)
	})
}

func TestConfigHandler_AuthenticationRequired(t *testing.T) {
	mockService := new(MockConfigService)
	handler := NewConfigHandler(mockService)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	projectID := uuid.New()

	router.GET("/projects/:id/config", handler.GetActiveConfig)
	router.POST("/projects/:id/config", handler.CreateOrUpdateConfig)
	router.GET("/projects/:id/config/versions", handler.GetConfigHistory)
	router.POST("/projects/:id/config/rollback/:version", handler.RollbackConfig)

	t.Run("CreateOrUpdateConfig - unauthenticated", func(t *testing.T) {
		reqBody := CreateConfigRequest{
			ModelProvider:  "openai",
			ModelName:      "gpt-4o-mini",
			Temperature:    0.7,
			MaxTokens:      4096,
			EnabledTools:   []string{"file_ops"},
			MaxIterations:  10,
			TimeoutSeconds: 300,
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/projects/"+projectID.String()+"/config", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
