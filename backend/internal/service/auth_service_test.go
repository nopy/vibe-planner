package service

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/npinot/vibe/backend/internal/config"
	"github.com/npinot/vibe/backend/internal/model"
	"github.com/npinot/vibe/backend/internal/repository"
)

// MockUserRepository is a mock implementation of UserRepository
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

var _ repository.UserRepository = (*MockUserRepository)(nil)

func TestAuthService_GetAuthorizationURL(t *testing.T) {
	cfg := &config.Config{
		OIDCIssuer:       "http://localhost:8081/realms/test",
		OIDCClientID:     "test-client",
		OIDCClientSecret: "test-secret",
		JWTSecret:        "test-jwt-secret-key-min-32-chars",
		JWTExpiry:        3600,
	}

	mockRepo := new(MockUserRepository)

	t.Run("generate URL with provided state", func(t *testing.T) {
		// Skip this test in CI as it requires actual OIDC provider
		t.Skip("Requires live OIDC provider")
	})

	t.Run("generate URL with auto state", func(t *testing.T) {
		// Skip this test in CI as it requires actual OIDC provider
		t.Skip("Requires live OIDC provider")
	})

	_ = cfg
	_ = mockRepo
}

func TestAuthService_GenerateJWT(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-jwt-secret-key-min-32-chars",
		JWTExpiry: 3600,
	}

	mockRepo := new(MockUserRepository)

	// Create a mock auth service (without OIDC provider)
	svc := &authService{
		cfg:      cfg,
		userRepo: mockRepo,
	}

	testUser := &model.User{
		ID:          uuid.New(),
		OIDCSubject: "test-subject",
		Email:       "test@example.com",
		Name:        "Test User",
	}

	t.Run("valid JWT generation", func(t *testing.T) {
		tokenString, err := svc.GenerateJWT(testUser)

		assert.NoError(t, err)
		assert.NotEmpty(t, tokenString)

		// Parse and validate the token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWTSecret), nil
		})

		require.NoError(t, err)
		assert.True(t, token.Valid)

		// Check claims
		claims, ok := token.Claims.(jwt.MapClaims)
		require.True(t, ok)

		assert.Equal(t, testUser.ID.String(), claims["user_id"])
		assert.Equal(t, testUser.OIDCSubject, claims["sub"])
		assert.Equal(t, testUser.Email, claims["email"])
		assert.Equal(t, testUser.Name, claims["name"])

		// Check expiry
		exp, ok := claims["exp"].(float64)
		require.True(t, ok)
		expTime := time.Unix(int64(exp), 0)
		assert.True(t, expTime.After(time.Now()))
		assert.True(t, expTime.Before(time.Now().Add(time.Hour*2)))

		// Check issued at
		iat, ok := claims["iat"].(float64)
		require.True(t, ok)
		iatTime := time.Unix(int64(iat), 0)
		assert.True(t, iatTime.Before(time.Now().Add(time.Minute)))
	})

	t.Run("JWT contains correct claims", func(t *testing.T) {
		tokenString, err := svc.GenerateJWT(testUser)
		require.NoError(t, err)

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWTSecret), nil
		})
		require.NoError(t, err)

		claims, ok := token.Claims.(jwt.MapClaims)
		require.True(t, ok)

		// Verify all required claims exist
		requiredClaims := []string{"user_id", "sub", "email", "name", "exp", "iat"}
		for _, claim := range requiredClaims {
			_, exists := claims[claim]
			assert.True(t, exists, "Missing claim: %s", claim)
		}
	})
}

func TestAuthService_JWTExpiry(t *testing.T) {
	testCases := []struct {
		name   string
		expiry int
	}{
		{"1 hour", 3600},
		{"30 minutes", 1800},
		{"24 hours", 86400},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{
				JWTSecret: "test-jwt-secret-key-min-32-chars",
				JWTExpiry: tc.expiry,
			}

			mockRepo := new(MockUserRepository)
			svc := &authService{
				cfg:      cfg,
				userRepo: mockRepo,
			}

			testUser := &model.User{
				ID:          uuid.New(),
				OIDCSubject: "test-subject",
				Email:       "test@example.com",
				Name:        "Test User",
			}

			tokenString, err := svc.GenerateJWT(testUser)
			require.NoError(t, err)

			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				return []byte(cfg.JWTSecret), nil
			})
			require.NoError(t, err)

			claims, ok := token.Claims.(jwt.MapClaims)
			require.True(t, ok)

			exp, ok := claims["exp"].(float64)
			require.True(t, ok)

			iat, ok := claims["iat"].(float64)
			require.True(t, ok)

			actualExpiry := int(exp - iat)
			assert.Equal(t, tc.expiry, actualExpiry)
		})
	}
}

func TestAuthService_JWTWithDifferentUsers(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-jwt-secret-key-min-32-chars",
		JWTExpiry: 3600,
	}

	mockRepo := new(MockUserRepository)
	svc := &authService{
		cfg:      cfg,
		userRepo: mockRepo,
	}

	user1 := &model.User{
		ID:          uuid.New(),
		OIDCSubject: "subject-1",
		Email:       "user1@example.com",
		Name:        "User One",
	}

	user2 := &model.User{
		ID:          uuid.New(),
		OIDCSubject: "subject-2",
		Email:       "user2@example.com",
		Name:        "User Two",
	}

	token1, err := svc.GenerateJWT(user1)
	require.NoError(t, err)

	token2, err := svc.GenerateJWT(user2)
	require.NoError(t, err)

	// Tokens should be different
	assert.NotEqual(t, token1, token2)

	// Parse both tokens
	parsedToken1, err := jwt.Parse(token1, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.JWTSecret), nil
	})
	require.NoError(t, err)

	parsedToken2, err := jwt.Parse(token2, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.JWTSecret), nil
	})
	require.NoError(t, err)

	claims1 := parsedToken1.Claims.(jwt.MapClaims)
	claims2 := parsedToken2.Claims.(jwt.MapClaims)

	assert.Equal(t, user1.ID.String(), claims1["user_id"])
	assert.Equal(t, user2.ID.String(), claims2["user_id"])
	assert.NotEqual(t, claims1["user_id"], claims2["user_id"])
}
