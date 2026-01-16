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
)

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) GetAuthorizationURL(state string) (string, error) {
	args := m.Called(state)
	return args.String(0), args.Error(1)
}

func (m *MockAuthService) ExchangeCodeForToken(ctx context.Context, code string) (*model.User, string, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, "", args.Error(2)
	}
	return args.Get(0).(*model.User), args.String(1), args.Error(2)
}

func (m *MockAuthService) GenerateJWT(user *model.User) (string, error) {
	args := m.Called(user)
	return args.String(0), args.Error(1)
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestAuthHandler_OIDCLogin(t *testing.T) {
	router := setupTestRouter()
	mockService := new(MockAuthService)
	handler := NewAuthHandler(mockService)

	router.GET("/auth/oidc/login", handler.OIDCLogin)

	t.Run("successful authorization URL generation", func(t *testing.T) {
		expectedURL := "https://auth.example.com/authorize?client_id=test&state=abc123"
		mockService.On("GetAuthorizationURL", "abc123").Return(expectedURL, nil).Once()

		req, _ := http.NewRequest("GET", "/auth/oidc/login?state=abc123", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, expectedURL, response["authorization_url"])

		mockService.AssertExpectations(t)
	})

	t.Run("authorization URL generation without state", func(t *testing.T) {
		expectedURL := "https://auth.example.com/authorize?client_id=test&state=generated"
		mockService.On("GetAuthorizationURL", "").Return(expectedURL, nil).Once()

		req, _ := http.NewRequest("GET", "/auth/oidc/login", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, expectedURL, response["authorization_url"])

		mockService.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		mockService.On("GetAuthorizationURL", "").Return("", errors.New("service error")).Once()

		req, _ := http.NewRequest("GET", "/auth/oidc/login", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "Failed to generate authorization URL")

		mockService.AssertExpectations(t)
	})
}

func TestAuthHandler_OIDCCallback(t *testing.T) {
	router := setupTestRouter()
	mockService := new(MockAuthService)
	handler := NewAuthHandler(mockService)

	router.GET("/auth/oidc/callback", handler.OIDCCallback)

	testUser := &model.User{
		ID:          uuid.New(),
		OIDCSubject: "test-subject",
		Email:       "test@example.com",
		Name:        "Test User",
	}
	testToken := "jwt.token.here"

	t.Run("successful callback", func(t *testing.T) {
		mockService.On("ExchangeCodeForToken", mock.Anything, "auth-code-123").
			Return(testUser, testToken, nil).Once()

		req, _ := http.NewRequest("GET", "/auth/oidc/callback?code=auth-code-123", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, testToken, response["token"])

		user := response["user"].(map[string]interface{})
		assert.Equal(t, testUser.Email, user["email"])
		assert.Equal(t, testUser.Name, user["name"])

		mockService.AssertExpectations(t)
	})

	t.Run("missing code", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/auth/oidc/callback", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "Missing authorization code")
	})

	t.Run("exchange error", func(t *testing.T) {
		mockService.On("ExchangeCodeForToken", mock.Anything, "invalid-code").
			Return(nil, "", errors.New("exchange failed")).Once()

		req, _ := http.NewRequest("GET", "/auth/oidc/callback?code=invalid-code", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "Failed to exchange code for token")

		mockService.AssertExpectations(t)
	})
}

func TestAuthHandler_GetCurrentUser(t *testing.T) {
	router := setupTestRouter()
	mockService := new(MockAuthService)
	handler := NewAuthHandler(mockService)

	testUser := &model.User{
		ID:          uuid.New(),
		OIDCSubject: "test-subject",
		Email:       "test@example.com",
		Name:        "Test User",
	}

	t.Run("authenticated user", func(t *testing.T) {
		router.GET("/auth/me", func(c *gin.Context) {
			c.Set("currentUser", testUser)
			handler.GetCurrentUser(c)
		})

		req, _ := http.NewRequest("GET", "/auth/me", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.User
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, testUser.Email, response.Email)
		assert.Equal(t, testUser.Name, response.Name)
	})

	t.Run("unauthenticated user", func(t *testing.T) {
		router2 := setupTestRouter()
		router2.GET("/auth/me", handler.GetCurrentUser)

		req, _ := http.NewRequest("GET", "/auth/me", nil)
		w := httptest.NewRecorder()
		router2.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "User not found")
	})
}

func TestAuthHandler_Logout(t *testing.T) {
	router := setupTestRouter()
	mockService := new(MockAuthService)
	handler := NewAuthHandler(mockService)

	router.POST("/auth/logout", handler.Logout)

	t.Run("successful logout", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/auth/logout", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Logged out successfully", response["message"])
	})
}
