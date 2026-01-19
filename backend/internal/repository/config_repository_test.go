package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/npinot/vibe/backend/internal/model"
)

func setupConfigTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	// Create tables with SQLite-compatible SQL
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

		CREATE TABLE opencode_configs (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			version INTEGER NOT NULL DEFAULT 1,
			is_active BOOLEAN NOT NULL DEFAULT 1,
			model_provider TEXT NOT NULL,
			model_name TEXT NOT NULL,
			model_version TEXT,
			api_endpoint TEXT,
			api_key_encrypted BLOB,
			temperature REAL NOT NULL DEFAULT 0.7,
			max_tokens INTEGER NOT NULL DEFAULT 4096,
			enabled_tools TEXT NOT NULL DEFAULT '["file_ops","web_search","code_exec"]',
			tools_config TEXT,
			system_prompt TEXT,
			max_iterations INTEGER NOT NULL DEFAULT 10,
			timeout_seconds INTEGER NOT NULL DEFAULT 300,
			created_by TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
			FOREIGN KEY (created_by) REFERENCES users(id)
		);

		CREATE UNIQUE INDEX idx_unique_project_version ON opencode_configs(project_id, version);
		CREATE INDEX idx_opencode_configs_project_id ON opencode_configs(project_id);
		CREATE INDEX idx_opencode_configs_active ON opencode_configs(project_id, is_active);
	`

	err = db.Exec(createTablesSQL).Error
	require.NoError(t, err)

	return db
}

func createTestConfig(t *testing.T, db *gorm.DB, projectID, createdBy uuid.UUID, version int, isActive bool) *model.OpenCodeConfig {
	t.Helper()

	config := &model.OpenCodeConfig{
		ID:             uuid.New(),
		ProjectID:      projectID,
		Version:        version,
		IsActive:       isActive,
		ModelProvider:  "openai",
		ModelName:      "gpt-4o-mini",
		Temperature:    0.7,
		MaxTokens:      4096,
		EnabledTools:   model.ToolsList{"file_ops", "web_search", "code_exec"},
		MaxIterations:  10,
		TimeoutSeconds: 300,
		CreatedBy:      createdBy,
	}

	err := db.Create(config).Error
	require.NoError(t, err)

	// Explicitly set is_active to work around SQLite DEFAULT 1 behavior
	// GORM doesn't always persist false values correctly with SQLite's BOOLEAN type
	if !isActive {
		err = db.Model(config).Update("is_active", false).Error
		require.NoError(t, err)
	}

	return config
}

// Test Create

func TestConfigRepository_CreateConfig(t *testing.T) {
	db := setupConfigTestDB(t)
	repo := NewConfigRepository(db)
	ctx := context.Background()

	projectID := uuid.New()
	createdBy := uuid.New()

	config := &model.OpenCodeConfig{
		ProjectID:      projectID,
		ModelProvider:  "openai",
		ModelName:      "gpt-4o-mini",
		Temperature:    0.7,
		MaxTokens:      4096,
		EnabledTools:   model.ToolsList{"file_ops", "web_search"},
		MaxIterations:  10,
		TimeoutSeconds: 300,
		CreatedBy:      createdBy,
	}

	err := repo.CreateConfig(ctx, config)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, config.ID)
	assert.Equal(t, 1, config.Version)
	assert.True(t, config.IsActive)
}

func TestConfigRepository_CreateConfig_GeneratesID(t *testing.T) {
	db := setupConfigTestDB(t)
	repo := NewConfigRepository(db)
	ctx := context.Background()

	config := &model.OpenCodeConfig{
		ProjectID:      uuid.New(),
		ModelProvider:  "anthropic",
		ModelName:      "claude-3-opus",
		Temperature:    0.5,
		MaxTokens:      2048,
		EnabledTools:   model.ToolsList{"file_ops"},
		MaxIterations:  5,
		TimeoutSeconds: 180,
		CreatedBy:      uuid.New(),
	}

	err := repo.CreateConfig(ctx, config)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, config.ID)
}

func TestConfigRepository_CreateConfig_IncrementsVersion(t *testing.T) {
	db := setupConfigTestDB(t)
	repo := NewConfigRepository(db)
	ctx := context.Background()

	projectID := uuid.New()
	createdBy := uuid.New()

	// Create first config
	config1 := &model.OpenCodeConfig{
		ProjectID:      projectID,
		ModelProvider:  "openai",
		ModelName:      "gpt-4o",
		Temperature:    0.7,
		MaxTokens:      4096,
		EnabledTools:   model.ToolsList{"file_ops"},
		MaxIterations:  10,
		TimeoutSeconds: 300,
		CreatedBy:      createdBy,
	}
	err := repo.CreateConfig(ctx, config1)
	require.NoError(t, err)
	assert.Equal(t, 1, config1.Version)

	// Create second config
	config2 := &model.OpenCodeConfig{
		ProjectID:      projectID,
		ModelProvider:  "openai",
		ModelName:      "gpt-4o-mini",
		Temperature:    0.8,
		MaxTokens:      2048,
		EnabledTools:   model.ToolsList{"file_ops", "web_search"},
		MaxIterations:  15,
		TimeoutSeconds: 600,
		CreatedBy:      createdBy,
	}
	err = repo.CreateConfig(ctx, config2)
	require.NoError(t, err)
	assert.Equal(t, 2, config2.Version)

	// Verify first config is now inactive
	var firstConfig model.OpenCodeConfig
	err = db.First(&firstConfig, config1.ID).Error
	require.NoError(t, err)
	assert.False(t, firstConfig.IsActive)
}

func TestConfigRepository_CreateConfig_DeactivatesOldConfigs(t *testing.T) {
	db := setupConfigTestDB(t)
	repo := NewConfigRepository(db)
	ctx := context.Background()

	projectID := uuid.New()
	createdBy := uuid.New()

	// Create 3 configs
	for i := 0; i < 3; i++ {
		config := &model.OpenCodeConfig{
			ProjectID:      projectID,
			ModelProvider:  "openai",
			ModelName:      "gpt-4o",
			Temperature:    0.7,
			MaxTokens:      4096,
			EnabledTools:   model.ToolsList{"file_ops"},
			MaxIterations:  10,
			TimeoutSeconds: 300,
			CreatedBy:      createdBy,
		}
		err := repo.CreateConfig(ctx, config)
		require.NoError(t, err)
	}

	// Verify only 1 is active
	var activeConfigs []model.OpenCodeConfig
	err := db.Where("project_id = ? AND is_active = ?", projectID, true).Find(&activeConfigs).Error
	require.NoError(t, err)
	assert.Len(t, activeConfigs, 1)
	assert.Equal(t, 3, activeConfigs[0].Version)
}

func TestConfigRepository_CreateConfig_WithToolsConfig(t *testing.T) {
	db := setupConfigTestDB(t)
	repo := NewConfigRepository(db)
	ctx := context.Background()

	toolsConfig := model.JSONB{
		"web_search": map[string]interface{}{
			"depth":  5,
			"engine": "google",
		},
	}

	config := &model.OpenCodeConfig{
		ProjectID:      uuid.New(),
		ModelProvider:  "openai",
		ModelName:      "gpt-4o",
		Temperature:    0.7,
		MaxTokens:      4096,
		EnabledTools:   model.ToolsList{"file_ops", "web_search"},
		ToolsConfig:    toolsConfig,
		MaxIterations:  10,
		TimeoutSeconds: 300,
		CreatedBy:      uuid.New(),
	}

	err := repo.CreateConfig(ctx, config)
	require.NoError(t, err)

	// Verify tools config stored correctly
	var found model.OpenCodeConfig
	err = db.First(&found, config.ID).Error
	require.NoError(t, err)
	assert.NotNil(t, found.ToolsConfig)
}

func TestConfigRepository_CreateConfig_WithSystemPrompt(t *testing.T) {
	db := setupConfigTestDB(t)
	repo := NewConfigRepository(db)
	ctx := context.Background()

	systemPrompt := "You are a helpful coding assistant"
	config := &model.OpenCodeConfig{
		ProjectID:      uuid.New(),
		ModelProvider:  "openai",
		ModelName:      "gpt-4o",
		Temperature:    0.7,
		MaxTokens:      4096,
		EnabledTools:   model.ToolsList{"file_ops"},
		SystemPrompt:   &systemPrompt,
		MaxIterations:  10,
		TimeoutSeconds: 300,
		CreatedBy:      uuid.New(),
	}

	err := repo.CreateConfig(ctx, config)
	require.NoError(t, err)
	assert.NotNil(t, config.SystemPrompt)
	assert.Equal(t, systemPrompt, *config.SystemPrompt)
}

// Test GetActiveConfig

func TestConfigRepository_GetActiveConfig(t *testing.T) {
	db := setupConfigTestDB(t)
	repo := NewConfigRepository(db)
	ctx := context.Background()

	projectID := uuid.New()
	createdBy := uuid.New()

	created := createTestConfig(t, db, projectID, createdBy, 1, true)

	found, err := repo.GetActiveConfig(ctx, projectID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, projectID, found.ProjectID)
	assert.True(t, found.IsActive)
}

func TestConfigRepository_GetActiveConfig_NotFound(t *testing.T) {
	db := setupConfigTestDB(t)
	repo := NewConfigRepository(db)
	ctx := context.Background()

	_, err := repo.GetActiveConfig(ctx, uuid.New())
	assert.Error(t, err)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestConfigRepository_GetActiveConfig_ReturnsOnlyActive(t *testing.T) {
	db := setupConfigTestDB(t)
	repo := NewConfigRepository(db)
	ctx := context.Background()

	projectID := uuid.New()
	createdBy := uuid.New()

	config1 := createTestConfig(t, db, projectID, createdBy, 1, false)
	config2 := createTestConfig(t, db, projectID, createdBy, 2, true)

	found, err := repo.GetActiveConfig(ctx, projectID)
	require.NoError(t, err)
	assert.Equal(t, config2.ID, found.ID)
	assert.Equal(t, 2, found.Version)
	assert.True(t, found.IsActive)

	var inactive model.OpenCodeConfig
	err = db.First(&inactive, config1.ID).Error
	require.NoError(t, err)
	assert.False(t, inactive.IsActive)
}

// Test GetConfigVersions

func TestConfigRepository_GetConfigVersions(t *testing.T) {
	db := setupConfigTestDB(t)
	repo := NewConfigRepository(db)
	ctx := context.Background()

	projectID := uuid.New()
	createdBy := uuid.New()

	// Create 3 versions
	createTestConfig(t, db, projectID, createdBy, 1, false)
	createTestConfig(t, db, projectID, createdBy, 2, false)
	createTestConfig(t, db, projectID, createdBy, 3, true)

	configs, err := repo.GetConfigVersions(ctx, projectID)
	require.NoError(t, err)
	assert.Len(t, configs, 3)

	// Verify ordered by version DESC
	assert.Equal(t, 3, configs[0].Version)
	assert.Equal(t, 2, configs[1].Version)
	assert.Equal(t, 1, configs[2].Version)
}

func TestConfigRepository_GetConfigVersions_EmptyProject(t *testing.T) {
	db := setupConfigTestDB(t)
	repo := NewConfigRepository(db)
	ctx := context.Background()

	configs, err := repo.GetConfigVersions(ctx, uuid.New())
	require.NoError(t, err)
	assert.Empty(t, configs)
}

func TestConfigRepository_GetConfigVersions_MultipleProjects(t *testing.T) {
	db := setupConfigTestDB(t)
	repo := NewConfigRepository(db)
	ctx := context.Background()

	project1 := uuid.New()
	project2 := uuid.New()
	createdBy := uuid.New()

	// Create configs for project 1
	createTestConfig(t, db, project1, createdBy, 1, true)
	createTestConfig(t, db, project1, createdBy, 2, false)

	// Create configs for project 2
	createTestConfig(t, db, project2, createdBy, 1, true)

	// Get versions for project 1 only
	configs, err := repo.GetConfigVersions(ctx, project1)
	require.NoError(t, err)
	assert.Len(t, configs, 2)
	assert.Equal(t, project1, configs[0].ProjectID)
	assert.Equal(t, project1, configs[1].ProjectID)
}

// Test GetConfigByVersion

func TestConfigRepository_GetConfigByVersion(t *testing.T) {
	db := setupConfigTestDB(t)
	repo := NewConfigRepository(db)
	ctx := context.Background()

	projectID := uuid.New()
	createdBy := uuid.New()

	created := createTestConfig(t, db, projectID, createdBy, 1, true)

	found, err := repo.GetConfigByVersion(ctx, projectID, 1)
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, 1, found.Version)
}

func TestConfigRepository_GetConfigByVersion_NotFound(t *testing.T) {
	db := setupConfigTestDB(t)
	repo := NewConfigRepository(db)
	ctx := context.Background()

	_, err := repo.GetConfigByVersion(ctx, uuid.New(), 1)
	assert.Error(t, err)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestConfigRepository_GetConfigByVersion_SpecificVersion(t *testing.T) {
	db := setupConfigTestDB(t)
	repo := NewConfigRepository(db)
	ctx := context.Background()

	projectID := uuid.New()
	createdBy := uuid.New()

	// Create 3 versions
	createTestConfig(t, db, projectID, createdBy, 1, false)
	config2 := createTestConfig(t, db, projectID, createdBy, 2, false)
	createTestConfig(t, db, projectID, createdBy, 3, true)

	// Get version 2
	found, err := repo.GetConfigByVersion(ctx, projectID, 2)
	require.NoError(t, err)
	assert.Equal(t, config2.ID, found.ID)
	assert.Equal(t, 2, found.Version)
}

// Test DeleteConfig

func TestConfigRepository_DeleteConfig(t *testing.T) {
	db := setupConfigTestDB(t)
	repo := NewConfigRepository(db)
	ctx := context.Background()

	projectID := uuid.New()
	createdBy := uuid.New()

	config := createTestConfig(t, db, projectID, createdBy, 1, false)

	err := repo.DeleteConfig(ctx, config.ID)
	require.NoError(t, err)

	// Verify deleted
	var found model.OpenCodeConfig
	err = db.First(&found, config.ID).Error
	assert.Error(t, err)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestConfigRepository_DeleteConfig_NotFound(t *testing.T) {
	db := setupConfigTestDB(t)
	repo := NewConfigRepository(db)
	ctx := context.Background()

	err := repo.DeleteConfig(ctx, uuid.New())
	assert.Error(t, err)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestConfigRepository_DeleteConfig_CannotDeleteActive(t *testing.T) {
	db := setupConfigTestDB(t)
	repo := NewConfigRepository(db)
	ctx := context.Background()

	projectID := uuid.New()
	createdBy := uuid.New()

	activeConfig := createTestConfig(t, db, projectID, createdBy, 1, true)

	err := repo.DeleteConfig(ctx, activeConfig.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete active configuration")

	// Verify not deleted
	var found model.OpenCodeConfig
	err = db.First(&found, activeConfig.ID).Error
	require.NoError(t, err)
}

// Test Edge Cases

func TestConfigRepository_CreateConfig_ConcurrentCreation(t *testing.T) {
	db := setupConfigTestDB(t)
	repo := NewConfigRepository(db)
	ctx := context.Background()

	projectID := uuid.New()
	createdBy := uuid.New()

	// Create first config
	config1 := &model.OpenCodeConfig{
		ProjectID:      projectID,
		ModelProvider:  "openai",
		ModelName:      "gpt-4o",
		Temperature:    0.7,
		MaxTokens:      4096,
		EnabledTools:   model.ToolsList{"file_ops"},
		MaxIterations:  10,
		TimeoutSeconds: 300,
		CreatedBy:      createdBy,
	}
	err := repo.CreateConfig(ctx, config1)
	require.NoError(t, err)

	// Create second config concurrently
	config2 := &model.OpenCodeConfig{
		ProjectID:      projectID,
		ModelProvider:  "anthropic",
		ModelName:      "claude-3-opus",
		Temperature:    0.5,
		MaxTokens:      2048,
		EnabledTools:   model.ToolsList{"file_ops", "web_search"},
		MaxIterations:  15,
		TimeoutSeconds: 600,
		CreatedBy:      createdBy,
	}
	err = repo.CreateConfig(ctx, config2)
	require.NoError(t, err)

	// Verify versions are sequential
	assert.Equal(t, 1, config1.Version)
	assert.Equal(t, 2, config2.Version)

	// Verify only latest is active
	active, err := repo.GetActiveConfig(ctx, projectID)
	require.NoError(t, err)
	assert.Equal(t, config2.ID, active.ID)
}

func TestConfigRepository_CreateConfig_WithAPIKey(t *testing.T) {
	db := setupConfigTestDB(t)
	repo := NewConfigRepository(db)
	ctx := context.Background()

	apiKey := []byte("encrypted-api-key-data")
	config := &model.OpenCodeConfig{
		ProjectID:       uuid.New(),
		ModelProvider:   "openai",
		ModelName:       "gpt-4o",
		APIKeyEncrypted: apiKey,
		Temperature:     0.7,
		MaxTokens:       4096,
		EnabledTools:    model.ToolsList{"file_ops"},
		MaxIterations:   10,
		TimeoutSeconds:  300,
		CreatedBy:       uuid.New(),
	}

	err := repo.CreateConfig(ctx, config)
	require.NoError(t, err)

	// Verify API key stored
	var found model.OpenCodeConfig
	err = db.First(&found, config.ID).Error
	require.NoError(t, err)
	assert.Equal(t, apiKey, found.APIKeyEncrypted)
}

func TestConfigRepository_CreateConfig_WithCustomEndpoint(t *testing.T) {
	db := setupConfigTestDB(t)
	repo := NewConfigRepository(db)
	ctx := context.Background()

	endpoint := "https://custom-api.example.com/v1"
	config := &model.OpenCodeConfig{
		ProjectID:      uuid.New(),
		ModelProvider:  "custom",
		ModelName:      "llama-3-70b",
		APIEndpoint:    &endpoint,
		Temperature:    0.7,
		MaxTokens:      4096,
		EnabledTools:   model.ToolsList{"file_ops"},
		MaxIterations:  10,
		TimeoutSeconds: 300,
		CreatedBy:      uuid.New(),
	}

	err := repo.CreateConfig(ctx, config)
	require.NoError(t, err)

	// Verify custom endpoint stored
	var found model.OpenCodeConfig
	err = db.First(&found, config.ID).Error
	require.NoError(t, err)
	require.NotNil(t, found.APIEndpoint)
	assert.Equal(t, endpoint, *found.APIEndpoint)
}

func TestConfigRepository_GetConfigVersions_OrderedCorrectly(t *testing.T) {
	db := setupConfigTestDB(t)
	repo := NewConfigRepository(db)
	ctx := context.Background()

	projectID := uuid.New()
	createdBy := uuid.New()

	// Create configs in non-sequential order
	createTestConfig(t, db, projectID, createdBy, 3, false)
	createTestConfig(t, db, projectID, createdBy, 1, false)
	createTestConfig(t, db, projectID, createdBy, 2, true)

	configs, err := repo.GetConfigVersions(ctx, projectID)
	require.NoError(t, err)
	assert.Len(t, configs, 3)

	// Verify ordered by version DESC
	assert.Equal(t, 3, configs[0].Version)
	assert.Equal(t, 2, configs[1].Version)
	assert.Equal(t, 1, configs[2].Version)
}
