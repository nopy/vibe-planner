package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/npinot/vibe/backend/internal/model"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	createSQLiteUsersTableSQL := `
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			oidc_subject TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL,
			name TEXT,
			picture_url TEXT,
			last_login_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		)
	`
	err = db.Exec(createSQLiteUsersTableSQL).Error
	require.NoError(t, err)

	return db
}

func TestNewUserRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)

	assert.NotNil(t, repo)
}

func TestUserRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	tests := []struct {
		name    string
		user    *model.User
		wantErr bool
	}{
		{
			name: "valid user",
			user: &model.User{
				OIDCSubject: "test-subject-1",
				Email:       "test@example.com",
				Name:        "Test User",
				PictureURL:  "https://example.com/pic.jpg",
			},
			wantErr: false,
		},
		{
			name: "duplicate OIDC subject",
			user: &model.User{
				OIDCSubject: "test-subject-1",
				Email:       "another@example.com",
				Name:        "Another User",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Create(ctx, tt.user)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, uuid.Nil, tt.user.ID)
				assert.False(t, tt.user.CreatedAt.IsZero())
			}
		})
	}
}

func TestUserRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create test user
	testUser := &model.User{
		OIDCSubject: "test-subject",
		Email:       "test@example.com",
		Name:        "Test User",
	}
	err := repo.Create(ctx, testUser)
	require.NoError(t, err)

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "existing user",
			id:      testUser.ID.String(),
			wantErr: false,
		},
		{
			name:    "non-existent user",
			id:      uuid.New().String(),
			wantErr: true,
		},
		{
			name:    "invalid UUID",
			id:      "invalid-uuid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := repo.FindByID(ctx, tt.id)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, testUser.Email, user.Email)
				assert.Equal(t, testUser.OIDCSubject, user.OIDCSubject)
			}
		})
	}
}

func TestUserRepository_FindByOIDCSubject(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create test user
	testUser := &model.User{
		OIDCSubject: "test-subject",
		Email:       "test@example.com",
		Name:        "Test User",
	}
	err := repo.Create(ctx, testUser)
	require.NoError(t, err)

	tests := []struct {
		name                string
		subject             string
		wantErr             bool
		isErrRecordNotFound bool
	}{
		{
			name:    "existing user",
			subject: "test-subject",
			wantErr: false,
		},
		{
			name:                "non-existent user",
			subject:             "non-existent-subject",
			wantErr:             true,
			isErrRecordNotFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := repo.FindByOIDCSubject(ctx, tt.subject)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, user)
				if tt.isErrRecordNotFound {
					assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, testUser.Email, user.Email)
				assert.Equal(t, testUser.OIDCSubject, user.OIDCSubject)
			}
		})
	}
}

func TestUserRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create test user
	testUser := &model.User{
		OIDCSubject: "test-subject",
		Email:       "test@example.com",
		Name:        "Test User",
	}
	err := repo.Create(ctx, testUser)
	require.NoError(t, err)

	// Update user
	testUser.Email = "updated@example.com"
	testUser.Name = "Updated User"

	err = repo.Update(ctx, testUser)
	assert.NoError(t, err)

	// Verify update
	updatedUser, err := repo.FindByID(ctx, testUser.ID.String())
	require.NoError(t, err)
	assert.Equal(t, "updated@example.com", updatedUser.Email)
	assert.Equal(t, "Updated User", updatedUser.Name)
}

func TestUserRepository_CreateOrUpdateFromOIDC(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	// Mock time for consistent testing
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	originalTimeNow := model.TimeNow
	model.TimeNow = func() time.Time { return now }
	defer func() { model.TimeNow = originalTimeNow }()

	t.Run("create new user", func(t *testing.T) {
		user, err := repo.CreateOrUpdateFromOIDC(
			ctx,
			"new-subject",
			"new@example.com",
			"New User",
			"https://example.com/new.jpg",
		)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.NotEqual(t, uuid.Nil, user.ID)
		assert.Equal(t, "new-subject", user.OIDCSubject)
		assert.Equal(t, "new@example.com", user.Email)
		assert.Equal(t, "New User", user.Name)
		assert.Equal(t, "https://example.com/new.jpg", user.PictureURL)
		assert.NotNil(t, user.LastLoginAt)
		assert.Equal(t, now, *user.LastLoginAt)
	})

	t.Run("update existing user", func(t *testing.T) {
		// Create user
		user1, err := repo.CreateOrUpdateFromOIDC(
			ctx,
			"existing-subject",
			"old@example.com",
			"Old Name",
			"https://example.com/old.jpg",
		)
		require.NoError(t, err)
		originalID := user1.ID

		// Update after 1 hour
		now = now.Add(time.Hour)

		// Call again with updated info
		user2, err := repo.CreateOrUpdateFromOIDC(
			ctx,
			"existing-subject",
			"updated@example.com",
			"Updated Name",
			"https://example.com/updated.jpg",
		)

		assert.NoError(t, err)
		assert.NotNil(t, user2)
		assert.Equal(t, originalID, user2.ID) // Same user ID
		assert.Equal(t, "existing-subject", user2.OIDCSubject)
		assert.Equal(t, "updated@example.com", user2.Email)
		assert.Equal(t, "Updated Name", user2.Name)
		assert.Equal(t, "https://example.com/updated.jpg", user2.PictureURL)
		assert.NotNil(t, user2.LastLoginAt)
		assert.Equal(t, now, *user2.LastLoginAt)
	})

	t.Run("update without picture", func(t *testing.T) {
		user, err := repo.CreateOrUpdateFromOIDC(
			ctx,
			"no-picture-subject",
			"test@example.com",
			"Test User",
			"",
		)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "", user.PictureURL)
	})
}
