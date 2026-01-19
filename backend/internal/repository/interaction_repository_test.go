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

func setupInteractionTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	createTablesSQL := `
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			oidc_subject TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL,
			name TEXT,
			picture_url TEXT,
			last_login_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		);

		CREATE TABLE projects (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			slug TEXT NOT NULL UNIQUE,
			git_repository_url TEXT,
			opencode_config TEXT,
			pod_name TEXT,
			pod_status TEXT DEFAULT 'pending',
			user_id TEXT NOT NULL,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);

		CREATE TABLE tasks (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT,
			status TEXT NOT NULL DEFAULT 'todo',
			priority TEXT NOT NULL DEFAULT 'medium',
			position INTEGER NOT NULL DEFAULT 0,
			assigned_to TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);

		CREATE TABLE sessions (
			id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL,
			project_id TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			prompt TEXT,
			output TEXT,
			error TEXT,
			started_at DATETIME,
			completed_at DATETIME,
			duration_ms INTEGER DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			FOREIGN KEY (task_id) REFERENCES tasks(id),
			FOREIGN KEY (project_id) REFERENCES projects(id)
		);

		CREATE TABLE interactions (
			id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL,
			session_id TEXT,
			user_id TEXT NOT NULL,
			message_type TEXT NOT NULL,
			content TEXT NOT NULL,
			metadata TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			FOREIGN KEY (task_id) REFERENCES tasks(id),
			FOREIGN KEY (session_id) REFERENCES sessions(id),
			FOREIGN KEY (user_id) REFERENCES users(id)
		);

		CREATE INDEX idx_interactions_task_id ON interactions(task_id);
		CREATE INDEX idx_interactions_session_id ON interactions(session_id);
		CREATE INDEX idx_interactions_created_at ON interactions(created_at);
	`

	err = db.Exec(createTablesSQL).Error
	require.NoError(t, err)

	return db
}

func createTestUserForInteraction(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	err := db.Exec("INSERT INTO users (id, oidc_subject, email, name, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		userID.String(), "test-subject", "test@example.com", "Test User", time.Now(), time.Now()).Error
	require.NoError(t, err)
	return userID
}

func createTestProjectForInteraction(t *testing.T, db *gorm.DB, userID uuid.UUID) uuid.UUID {
	t.Helper()
	projectID := uuid.New()
	err := db.Exec("INSERT INTO projects (id, name, slug, user_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		projectID.String(), "Test Project", "test-project", userID.String(), time.Now(), time.Now()).Error
	require.NoError(t, err)
	return projectID
}

func createTestTaskForInteraction(t *testing.T, db *gorm.DB, projectID uuid.UUID) uuid.UUID {
	t.Helper()
	taskID := uuid.New()
	err := db.Exec("INSERT INTO tasks (id, project_id, title, status, priority, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		taskID.String(), projectID.String(), "Test Task", "todo", "medium", time.Now(), time.Now()).Error
	require.NoError(t, err)
	return taskID
}

func createTestSessionForInteraction(t *testing.T, db *gorm.DB, taskID, projectID uuid.UUID) uuid.UUID {
	t.Helper()
	sessionID := uuid.New()
	err := db.Exec("INSERT INTO sessions (id, task_id, project_id, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		sessionID.String(), taskID.String(), projectID.String(), "pending", time.Now(), time.Now()).Error
	require.NoError(t, err)
	return sessionID
}

func createTestInteraction(t *testing.T, db *gorm.DB, taskID, userID uuid.UUID, sessionID *uuid.UUID, messageType string) *model.Interaction {
	t.Helper()
	interaction := &model.Interaction{
		ID:          uuid.New(),
		TaskID:      taskID,
		SessionID:   sessionID,
		UserID:      userID,
		MessageType: messageType,
		Content:     "Test message content",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	sessionIDStr := ""
	if sessionID != nil {
		sessionIDStr = sessionID.String()
	}

	err := db.Exec("INSERT INTO interactions (id, task_id, session_id, user_id, message_type, content, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		interaction.ID.String(), taskID.String(), sessionIDStr, userID.String(), messageType, interaction.Content, interaction.CreatedAt, interaction.UpdatedAt).Error
	require.NoError(t, err)
	return interaction
}

func TestInteractionRepository_Create(t *testing.T) {
	db := setupInteractionTestDB(t)
	repo := NewInteractionRepository(db)
	ctx := context.Background()

	userID := createTestUserForInteraction(t, db)
	projectID := createTestProjectForInteraction(t, db, userID)
	taskID := createTestTaskForInteraction(t, db, projectID)

	interaction := &model.Interaction{
		TaskID:      taskID,
		UserID:      userID,
		MessageType: "user_message",
		Content:     "Hello, AI assistant!",
	}

	err := repo.Create(ctx, interaction)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, interaction.ID)
	assert.NotZero(t, interaction.CreatedAt)
}

func TestInteractionRepository_Create_GeneratesID(t *testing.T) {
	db := setupInteractionTestDB(t)
	repo := NewInteractionRepository(db)
	ctx := context.Background()

	userID := createTestUserForInteraction(t, db)
	projectID := createTestProjectForInteraction(t, db, userID)
	taskID := createTestTaskForInteraction(t, db, projectID)

	interaction := &model.Interaction{
		TaskID:      taskID,
		UserID:      userID,
		MessageType: "agent_response",
		Content:     "I can help with that!",
	}

	err := repo.Create(ctx, interaction)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, interaction.ID)
}

func TestInteractionRepository_Create_WithSessionID(t *testing.T) {
	db := setupInteractionTestDB(t)
	repo := NewInteractionRepository(db)
	ctx := context.Background()

	userID := createTestUserForInteraction(t, db)
	projectID := createTestProjectForInteraction(t, db, userID)
	taskID := createTestTaskForInteraction(t, db, projectID)
	sessionID := createTestSessionForInteraction(t, db, taskID, projectID)

	interaction := &model.Interaction{
		TaskID:      taskID,
		SessionID:   &sessionID,
		UserID:      userID,
		MessageType: "agent_response",
		Content:     "Processing your request...",
	}

	err := repo.Create(ctx, interaction)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, interaction.ID)
	assert.NotNil(t, interaction.SessionID)
	assert.Equal(t, sessionID, *interaction.SessionID)
}

func TestInteractionRepository_Create_WithMetadata(t *testing.T) {
	db := setupInteractionTestDB(t)
	repo := NewInteractionRepository(db)
	ctx := context.Background()

	userID := createTestUserForInteraction(t, db)
	projectID := createTestProjectForInteraction(t, db, userID)
	taskID := createTestTaskForInteraction(t, db, projectID)

	metadata := model.JSONB{
		"code_snippet": "console.log('hello');",
		"file_path":    "/src/index.js",
	}

	interaction := &model.Interaction{
		TaskID:      taskID,
		UserID:      userID,
		MessageType: "user_message",
		Content:     "Please review this code",
		Metadata:    metadata,
	}

	err := repo.Create(ctx, interaction)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, interaction.ID)
}

func TestInteractionRepository_FindByID(t *testing.T) {
	db := setupInteractionTestDB(t)
	repo := NewInteractionRepository(db)
	ctx := context.Background()

	userID := createTestUserForInteraction(t, db)
	projectID := createTestProjectForInteraction(t, db, userID)
	taskID := createTestTaskForInteraction(t, db, projectID)
	created := createTestInteraction(t, db, taskID, userID, nil, "user_message")

	found, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, created.TaskID, found.TaskID)
	assert.Equal(t, created.UserID, found.UserID)
	assert.Equal(t, "user_message", found.MessageType)
	assert.Equal(t, "Test message content", found.Content)
}

func TestInteractionRepository_FindByID_NotFound(t *testing.T) {
	db := setupInteractionTestDB(t)
	repo := NewInteractionRepository(db)
	ctx := context.Background()

	_, err := repo.FindByID(ctx, uuid.New())
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestInteractionRepository_FindByTaskID(t *testing.T) {
	db := setupInteractionTestDB(t)
	repo := NewInteractionRepository(db)
	ctx := context.Background()

	userID := createTestUserForInteraction(t, db)
	projectID := createTestProjectForInteraction(t, db, userID)
	taskID := createTestTaskForInteraction(t, db, projectID)

	interaction1 := createTestInteraction(t, db, taskID, userID, nil, "user_message")
	time.Sleep(time.Millisecond)
	interaction2 := createTestInteraction(t, db, taskID, userID, nil, "agent_response")
	time.Sleep(time.Millisecond)
	interaction3 := createTestInteraction(t, db, taskID, userID, nil, "user_message")

	otherTaskID := createTestTaskForInteraction(t, db, projectID)
	createTestInteraction(t, db, otherTaskID, userID, nil, "user_message")

	interactions, err := repo.FindByTaskID(ctx, taskID)
	require.NoError(t, err)
	assert.Len(t, interactions, 3)
	assert.Equal(t, interaction1.ID, interactions[0].ID)
	assert.Equal(t, interaction2.ID, interactions[1].ID)
	assert.Equal(t, interaction3.ID, interactions[2].ID)
}

func TestInteractionRepository_FindByTaskID_Empty(t *testing.T) {
	db := setupInteractionTestDB(t)
	repo := NewInteractionRepository(db)
	ctx := context.Background()

	interactions, err := repo.FindByTaskID(ctx, uuid.New())
	require.NoError(t, err)
	assert.Empty(t, interactions)
}

func TestInteractionRepository_FindByTaskID_OrderedByCreatedAt(t *testing.T) {
	db := setupInteractionTestDB(t)
	repo := NewInteractionRepository(db)
	ctx := context.Background()

	userID := createTestUserForInteraction(t, db)
	projectID := createTestProjectForInteraction(t, db, userID)
	taskID := createTestTaskForInteraction(t, db, projectID)

	now := time.Now()
	oldest := &model.Interaction{
		TaskID:      taskID,
		UserID:      userID,
		MessageType: "user_message",
		Content:     "First message",
		CreatedAt:   now.Add(-10 * time.Second),
		UpdatedAt:   now.Add(-10 * time.Second),
	}
	repo.Create(ctx, oldest)

	middle := &model.Interaction{
		TaskID:      taskID,
		UserID:      userID,
		MessageType: "agent_response",
		Content:     "Second message",
		CreatedAt:   now.Add(-5 * time.Second),
		UpdatedAt:   now.Add(-5 * time.Second),
	}
	repo.Create(ctx, middle)

	newest := &model.Interaction{
		TaskID:      taskID,
		UserID:      userID,
		MessageType: "user_message",
		Content:     "Third message",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	repo.Create(ctx, newest)

	interactions, err := repo.FindByTaskID(ctx, taskID)
	require.NoError(t, err)
	assert.Len(t, interactions, 3)
	assert.Equal(t, "First message", interactions[0].Content)
	assert.Equal(t, "Second message", interactions[1].Content)
	assert.Equal(t, "Third message", interactions[2].Content)
}

func TestInteractionRepository_FindBySessionID(t *testing.T) {
	db := setupInteractionTestDB(t)
	repo := NewInteractionRepository(db)
	ctx := context.Background()

	userID := createTestUserForInteraction(t, db)
	projectID := createTestProjectForInteraction(t, db, userID)
	taskID := createTestTaskForInteraction(t, db, projectID)
	sessionID := createTestSessionForInteraction(t, db, taskID, projectID)

	interaction1 := createTestInteraction(t, db, taskID, userID, &sessionID, "user_message")
	time.Sleep(time.Millisecond)
	interaction2 := createTestInteraction(t, db, taskID, userID, &sessionID, "agent_response")

	createTestInteraction(t, db, taskID, userID, nil, "system_notification")

	interactions, err := repo.FindBySessionID(ctx, sessionID)
	require.NoError(t, err)
	assert.Len(t, interactions, 2)
	assert.Equal(t, interaction1.ID, interactions[0].ID)
	assert.Equal(t, interaction2.ID, interactions[1].ID)
}

func TestInteractionRepository_FindBySessionID_Empty(t *testing.T) {
	db := setupInteractionTestDB(t)
	repo := NewInteractionRepository(db)
	ctx := context.Background()

	interactions, err := repo.FindBySessionID(ctx, uuid.New())
	require.NoError(t, err)
	assert.Empty(t, interactions)
}

func TestInteractionRepository_FindBySessionID_OrderedByCreatedAt(t *testing.T) {
	db := setupInteractionTestDB(t)
	repo := NewInteractionRepository(db)
	ctx := context.Background()

	userID := createTestUserForInteraction(t, db)
	projectID := createTestProjectForInteraction(t, db, userID)
	taskID := createTestTaskForInteraction(t, db, projectID)
	sessionID := createTestSessionForInteraction(t, db, taskID, projectID)

	now := time.Now()
	oldest := &model.Interaction{
		TaskID:      taskID,
		SessionID:   &sessionID,
		UserID:      userID,
		MessageType: "user_message",
		Content:     "First",
		CreatedAt:   now.Add(-10 * time.Second),
		UpdatedAt:   now.Add(-10 * time.Second),
	}
	repo.Create(ctx, oldest)

	newest := &model.Interaction{
		TaskID:      taskID,
		SessionID:   &sessionID,
		UserID:      userID,
		MessageType: "agent_response",
		Content:     "Second",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	repo.Create(ctx, newest)

	interactions, err := repo.FindBySessionID(ctx, sessionID)
	require.NoError(t, err)
	assert.Len(t, interactions, 2)
	assert.Equal(t, "First", interactions[0].Content)
	assert.Equal(t, "Second", interactions[1].Content)
}

func TestInteractionRepository_DeleteByTaskID(t *testing.T) {
	db := setupInteractionTestDB(t)
	repo := NewInteractionRepository(db)
	ctx := context.Background()

	userID := createTestUserForInteraction(t, db)
	projectID := createTestProjectForInteraction(t, db, userID)
	taskID := createTestTaskForInteraction(t, db, projectID)

	createTestInteraction(t, db, taskID, userID, nil, "user_message")
	createTestInteraction(t, db, taskID, userID, nil, "agent_response")

	err := repo.DeleteByTaskID(ctx, taskID)
	require.NoError(t, err)

	interactions, err := repo.FindByTaskID(ctx, taskID)
	require.NoError(t, err)
	assert.Empty(t, interactions)
}

func TestInteractionRepository_DeleteByTaskID_NoInteractions(t *testing.T) {
	db := setupInteractionTestDB(t)
	repo := NewInteractionRepository(db)
	ctx := context.Background()

	err := repo.DeleteByTaskID(ctx, uuid.New())
	require.NoError(t, err)
}

func TestInteractionRepository_DeleteByTaskID_OnlyDeletesForTask(t *testing.T) {
	db := setupInteractionTestDB(t)
	repo := NewInteractionRepository(db)
	ctx := context.Background()

	userID := createTestUserForInteraction(t, db)
	projectID := createTestProjectForInteraction(t, db, userID)
	taskID1 := createTestTaskForInteraction(t, db, projectID)
	taskID2 := createTestTaskForInteraction(t, db, projectID)

	createTestInteraction(t, db, taskID1, userID, nil, "user_message")
	createTestInteraction(t, db, taskID1, userID, nil, "agent_response")
	interaction := createTestInteraction(t, db, taskID2, userID, nil, "user_message")

	err := repo.DeleteByTaskID(ctx, taskID1)
	require.NoError(t, err)

	interactions1, err := repo.FindByTaskID(ctx, taskID1)
	require.NoError(t, err)
	assert.Empty(t, interactions1)

	interactions2, err := repo.FindByTaskID(ctx, taskID2)
	require.NoError(t, err)
	assert.Len(t, interactions2, 1)
	assert.Equal(t, interaction.ID, interactions2[0].ID)
}

func TestInteractionRepository_MessageTypeValidation(t *testing.T) {
	db := setupInteractionTestDB(t)
	repo := NewInteractionRepository(db)
	ctx := context.Background()

	userID := createTestUserForInteraction(t, db)
	projectID := createTestProjectForInteraction(t, db, userID)
	taskID := createTestTaskForInteraction(t, db, projectID)

	validTypes := []string{"user_message", "agent_response", "system_notification"}
	for _, msgType := range validTypes {
		interaction := &model.Interaction{
			TaskID:      taskID,
			UserID:      userID,
			MessageType: msgType,
			Content:     "Test content",
		}
		err := repo.Create(ctx, interaction)
		require.NoError(t, err, "Message type %s should be valid", msgType)
	}
}

func TestInteractionRepository_MultipleTasksConcurrent(t *testing.T) {
	db := setupInteractionTestDB(t)
	repo := NewInteractionRepository(db)
	ctx := context.Background()

	userID := createTestUserForInteraction(t, db)
	projectID := createTestProjectForInteraction(t, db, userID)

	task1ID := createTestTaskForInteraction(t, db, projectID)
	task2ID := createTestTaskForInteraction(t, db, projectID)

	createTestInteraction(t, db, task1ID, userID, nil, "user_message")
	createTestInteraction(t, db, task1ID, userID, nil, "agent_response")
	createTestInteraction(t, db, task2ID, userID, nil, "user_message")

	task1Interactions, err := repo.FindByTaskID(ctx, task1ID)
	require.NoError(t, err)
	assert.Len(t, task1Interactions, 2)

	task2Interactions, err := repo.FindByTaskID(ctx, task2ID)
	require.NoError(t, err)
	assert.Len(t, task2Interactions, 1)
}
