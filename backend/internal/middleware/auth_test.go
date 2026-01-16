package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/npinot/vibe/backend/internal/config"
	"github.com/npinot/vibe/backend/internal/model"
)

// MockUserRepository is a mock implementation of UserRepository for middleware tests
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) FindByOIDCSubject(ctx context.Context, subject string) (*model.User, error) {
	args := m.Called(ctx, subject)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) Create(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Update(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) CreateOrUpdateFromOIDC(ctx context.Context, subject, email, name, picture string) (*model.User, error) {
	args := m.Called(ctx, subject, email, name, picture)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func generateTestJWT(userID uuid.UUID, secret string, expiry time.Duration) string {
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"sub":     "test-subject",
		"email":   "test@example.com",
		"name":    "Test User",
		"exp":     time.Now().Add(expiry).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

func setupTestMiddleware(cfg *config.Config, userRepo *MockUserRepository) (*gin.Engine, *AuthMiddleware) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	middleware := NewAuthMiddleware(cfg, userRepo)
	return router, middleware
}

func TestNewAuthMiddleware(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret-key-min-32-chars-long",
	}
	mockRepo := new(MockUserRepository)

	middleware := NewAuthMiddleware(cfg, mockRepo)

	assert.NotNil(t, middleware)
	assert.Equal(t, cfg, middleware.cfg)
	assert.Equal(t, mockRepo, middleware.userRepo)
}

func TestAuthMiddleware_JWTAuth_Success(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret-key-min-32-chars-long",
	}
	mockRepo := new(MockUserRepository)
	router, middleware := setupTestMiddleware(cfg, mockRepo)

	testUser := &model.User{
		ID:          uuid.New(),
		OIDCSubject: "test-subject",
		Email:       "test@example.com",
		Name:        "Test User",
	}

	validToken := generateTestJWT(testUser.ID, cfg.JWTSecret, time.Hour)

	mockRepo.On("FindByID", mock.Anything, testUser.ID.String()).Return(testUser, nil).Once()

	router.GET("/protected", middleware.JWTAuth(), func(c *gin.Context) {
		user, err := GetCurrentUser(c)
		assert.NoError(t, err)
		assert.Equal(t, testUser.Email, user.Email)
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestAuthMiddleware_JWTAuth_MissingHeader(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret-key-min-32-chars-long",
	}
	mockRepo := new(MockUserRepository)
	router, middleware := setupTestMiddleware(cfg, mockRepo)

	router.GET("/protected", middleware.JWTAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Missing authorization header")
}

func TestAuthMiddleware_JWTAuth_InvalidHeaderFormat(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret-key-min-32-chars-long",
	}
	mockRepo := new(MockUserRepository)
	router, middleware := setupTestMiddleware(cfg, mockRepo)

	router.GET("/protected", middleware.JWTAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	testCases := []struct {
		name   string
		header string
	}{
		{"No Bearer prefix", "token-without-bearer"},
		{"Wrong prefix", "Basic token123"},
		{"Missing token", "Bearer"},
		{"Empty", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/protected", nil)
			req.Header.Set("Authorization", tc.header)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

func TestAuthMiddleware_JWTAuth_InvalidToken(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret-key-min-32-chars-long",
	}
	mockRepo := new(MockUserRepository)
	router, middleware := setupTestMiddleware(cfg, mockRepo)

	router.GET("/protected", middleware.JWTAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	testCases := []struct {
		name  string
		token string
	}{
		{"Malformed token", "malformed.token.here"},
		{"Random string", "not-a-jwt-token"},
		{"Empty token", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+tc.token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)

			var response map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["error"], "Invalid token")
		})
	}
}

func TestAuthMiddleware_JWTAuth_WrongSecret(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret-key-min-32-chars-long",
	}
	mockRepo := new(MockUserRepository)
	router, middleware := setupTestMiddleware(cfg, mockRepo)

	router.GET("/protected", middleware.JWTAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	userID := uuid.New()
	wrongSecretToken := generateTestJWT(userID, "wrong-secret-key-min-32-chars-long", time.Hour)

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+wrongSecretToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Invalid token")
}

func TestAuthMiddleware_JWTAuth_ExpiredToken(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret-key-min-32-chars-long",
	}
	mockRepo := new(MockUserRepository)
	router, middleware := setupTestMiddleware(cfg, mockRepo)

	router.GET("/protected", middleware.JWTAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	userID := uuid.New()
	expiredToken := generateTestJWT(userID, cfg.JWTSecret, -time.Hour) // Expired 1 hour ago

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+expiredToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Invalid token")
}

func TestAuthMiddleware_JWTAuth_UserNotFound(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret-key-min-32-chars-long",
	}
	mockRepo := new(MockUserRepository)
	router, middleware := setupTestMiddleware(cfg, mockRepo)

	userID := uuid.New()
	validToken := generateTestJWT(userID, cfg.JWTSecret, time.Hour)

	mockRepo.On("FindByID", mock.Anything, userID.String()).Return(nil, fmt.Errorf("user not found")).Once()

	router.GET("/protected", middleware.JWTAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "User not found")

	mockRepo.AssertExpectations(t)
}

func TestGetCurrentUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("user exists in context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		testUser := &model.User{
			ID:    uuid.New(),
			Email: "test@example.com",
			Name:  "Test User",
		}
		c.Set("currentUser", testUser)

		user, err := GetCurrentUser(c)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, testUser.Email, user.Email)
		assert.Equal(t, testUser.Name, user.Name)
	})

	t.Run("user not in context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())

		user, err := GetCurrentUser(c)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "user not found in context")
	})

	t.Run("invalid user type in context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("currentUser", "not-a-user-object")

		user, err := GetCurrentUser(c)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "invalid user type in context")
	})
}
