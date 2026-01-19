package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/npinot/vibe/backend/internal/model"
)

func setupSessionTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&model.Session{}, &model.Task{}, &model.Project{}, &model.User{})
	require.NoError(t, err)

	return db
}

func createTestSession(t *testing.T, db *gorm.DB, taskID, projectID uuid.UUID, status model.SessionStatus) *model.Session {
	session := &model.Session{
		TaskID:    taskID,
		ProjectID: projectID,
		Status:    status,
		Prompt:    "Test prompt",
	}
	err := db.Create(session).Error
	require.NoError(t, err)
	return session
}

func TestSessionRepository_Create(t *testing.T) {
	db := setupSessionTestDB(t)
	repo := NewSessionRepository(db)
	ctx := context.Background()

	taskID := uuid.New()
	projectID := uuid.New()

	session := &model.Session{
		TaskID:    taskID,
		ProjectID: projectID,
		Status:    model.SessionStatusPending,
		Prompt:    "Create a README file",
	}

	err := repo.Create(ctx, session)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, session.ID)
	assert.NotZero(t, session.CreatedAt)
}

func TestSessionRepository_Create_GeneratesID(t *testing.T) {
	db := setupSessionTestDB(t)
	repo := NewSessionRepository(db)
	ctx := context.Background()

	session := &model.Session{
		TaskID:    uuid.New(),
		ProjectID: uuid.New(),
		Status:    model.SessionStatusPending,
	}

	err := repo.Create(ctx, session)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, session.ID)
}

func TestSessionRepository_FindByID(t *testing.T) {
	db := setupSessionTestDB(t)
	repo := NewSessionRepository(db)
	ctx := context.Background()

	created := createTestSession(t, db, uuid.New(), uuid.New(), model.SessionStatusRunning)

	found, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, created.TaskID, found.TaskID)
	assert.Equal(t, created.ProjectID, found.ProjectID)
	assert.Equal(t, model.SessionStatusRunning, found.Status)
}

func TestSessionRepository_FindByID_NotFound(t *testing.T) {
	db := setupSessionTestDB(t)
	repo := NewSessionRepository(db)
	ctx := context.Background()

	_, err := repo.FindByID(ctx, uuid.New())
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestSessionRepository_FindByTaskID(t *testing.T) {
	db := setupSessionTestDB(t)
	repo := NewSessionRepository(db)
	ctx := context.Background()

	taskID := uuid.New()
	projectID := uuid.New()

	// Create multiple sessions for same task
	session1 := createTestSession(t, db, taskID, projectID, model.SessionStatusCompleted)
	time.Sleep(time.Millisecond) // Ensure different timestamps
	session2 := createTestSession(t, db, taskID, projectID, model.SessionStatusRunning)

	// Create session for different task
	createTestSession(t, db, uuid.New(), projectID, model.SessionStatusPending)

	sessions, err := repo.FindByTaskID(ctx, taskID)
	require.NoError(t, err)
	assert.Len(t, sessions, 2)
	// Should be ordered by created_at DESC (newest first)
	assert.Equal(t, session2.ID, sessions[0].ID)
	assert.Equal(t, session1.ID, sessions[1].ID)
}

func TestSessionRepository_FindByTaskID_Empty(t *testing.T) {
	db := setupSessionTestDB(t)
	repo := NewSessionRepository(db)
	ctx := context.Background()

	sessions, err := repo.FindByTaskID(ctx, uuid.New())
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestSessionRepository_FindActiveSessionsForProject(t *testing.T) {
	db := setupSessionTestDB(t)
	repo := NewSessionRepository(db)
	ctx := context.Background()

	projectID := uuid.New()

	// Create active sessions (pending, running)
	pending := createTestSession(t, db, uuid.New(), projectID, model.SessionStatusPending)
	running := createTestSession(t, db, uuid.New(), projectID, model.SessionStatusRunning)

	// Create inactive sessions
	createTestSession(t, db, uuid.New(), projectID, model.SessionStatusCompleted)
	createTestSession(t, db, uuid.New(), projectID, model.SessionStatusFailed)
	createTestSession(t, db, uuid.New(), projectID, model.SessionStatusCancelled)

	// Create session for different project
	createTestSession(t, db, uuid.New(), uuid.New(), model.SessionStatusRunning)

	active, err := repo.FindActiveSessionsForProject(ctx, projectID)
	require.NoError(t, err)
	assert.Len(t, active, 2)

	ids := []uuid.UUID{active[0].ID, active[1].ID}
	assert.Contains(t, ids, pending.ID)
	assert.Contains(t, ids, running.ID)
}

func TestSessionRepository_FindActiveSessionsForProject_Empty(t *testing.T) {
	db := setupSessionTestDB(t)
	repo := NewSessionRepository(db)
	ctx := context.Background()

	projectID := uuid.New()

	// Only create completed sessions
	createTestSession(t, db, uuid.New(), projectID, model.SessionStatusCompleted)
	createTestSession(t, db, uuid.New(), projectID, model.SessionStatusFailed)

	active, err := repo.FindActiveSessionsForProject(ctx, projectID)
	require.NoError(t, err)
	assert.Empty(t, active)
}

func TestSessionRepository_Update(t *testing.T) {
	db := setupSessionTestDB(t)
	repo := NewSessionRepository(db)
	ctx := context.Background()

	session := createTestSession(t, db, uuid.New(), uuid.New(), model.SessionStatusRunning)

	session.Output = "Updated output"
	session.Status = model.SessionStatusCompleted
	now := time.Now()
	session.CompletedAt = &now

	err := repo.Update(ctx, session)
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated output", found.Output)
	assert.Equal(t, model.SessionStatusCompleted, found.Status)
	assert.NotNil(t, found.CompletedAt)
}

func TestSessionRepository_UpdateStatus(t *testing.T) {
	db := setupSessionTestDB(t)
	repo := NewSessionRepository(db)
	ctx := context.Background()

	session := createTestSession(t, db, uuid.New(), uuid.New(), model.SessionStatusPending)

	err := repo.UpdateStatus(ctx, session.ID, model.SessionStatusRunning)
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, model.SessionStatusRunning, found.Status)
}

func TestSessionRepository_UpdateOutput(t *testing.T) {
	db := setupSessionTestDB(t)
	repo := NewSessionRepository(db)
	ctx := context.Background()

	session := createTestSession(t, db, uuid.New(), uuid.New(), model.SessionStatusRunning)

	newOutput := "Execution output line 1\nExecution output line 2"
	err := repo.UpdateOutput(ctx, session.ID, newOutput)
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, newOutput, found.Output)
}

func TestSessionRepository_SoftDelete(t *testing.T) {
	db := setupSessionTestDB(t)
	repo := NewSessionRepository(db)
	ctx := context.Background()

	session := createTestSession(t, db, uuid.New(), uuid.New(), model.SessionStatusCompleted)

	err := repo.SoftDelete(ctx, session.ID)
	require.NoError(t, err)

	// Should not find deleted session
	_, err = repo.FindByID(ctx, session.ID)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// Verify soft delete (should exist with DeletedAt set)
	var deleted model.Session
	err = db.Unscoped().Where("id = ?", session.ID).First(&deleted).Error
	require.NoError(t, err)
	assert.NotNil(t, deleted.DeletedAt)
}
