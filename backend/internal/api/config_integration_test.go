//go:build integration
// +build integration

package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/npinot/vibe/backend/internal/model"
	"github.com/npinot/vibe/backend/internal/repository"
	"github.com/npinot/vibe/backend/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupConfigIntegrationTest prepares the database and services for config integration tests
func setupConfigIntegrationTest(t *testing.T) (*gorm.DB, *service.ConfigService, func()) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Get database URL from environment
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL or DATABASE_URL environment variable not set")
	}

	// Connect to database
	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err, "Failed to connect to database")

	// Run migrations for required models
	err = db.AutoMigrate(&model.User{}, &model.Project{}, &model.OpenCodeConfig{})
	require.NoError(t, err, "Failed to run AutoMigrate")

	// Generate encryption key for tests (32 bytes, base64-encoded)
	encryptionKey := generateEncryptionKey(t)

	// Initialize repository and service
	configRepo := repository.NewConfigRepository(db)
	configService, err := service.NewConfigService(configRepo, encryptionKey)
	require.NoError(t, err, "Failed to create ConfigService")

	// Cleanup function
	cleanup := func() {
		// Delete test configs (cascade will handle through FK)
		db.Exec("DELETE FROM opencode_configs WHERE project_id IN (SELECT id FROM projects WHERE name LIKE 'config-integration-test-%')")
		db.Exec("DELETE FROM projects WHERE name LIKE 'config-integration-test-%'")
		db.Exec("DELETE FROM users WHERE email LIKE 'config-test-%@integration.test'")

		// Close database connection
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	return db, configService, cleanup
}

// generateEncryptionKey generates a valid base64-encoded 32-byte encryption key
func generateEncryptionKey(t *testing.T) string {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err, "Failed to generate random encryption key")
	return base64.StdEncoding.EncodeToString(key)
}

// createTestUserForConfig creates a test user for config integration tests
func createTestUserForConfig(t *testing.T, db *gorm.DB) *model.User {
	user := &model.User{
		ID:          uuid.New(),
		Email:       fmt.Sprintf("config-test-%s@integration.test", uuid.New().String()[:8]),
		Name:        "Config Integration Test User",
		OIDCSubject: fmt.Sprintf("oidc-sub-%s", uuid.New().String()),
	}

	err := db.Create(user).Error
	require.NoError(t, err, "Failed to create test user")
	return user
}

// createTestProjectForConfig creates a test project for config integration tests
func createTestProjectForConfig(t *testing.T, db *gorm.DB, userID uuid.UUID) *model.Project {
	project := &model.Project{
		ID:          uuid.New(),
		Name:        fmt.Sprintf("config-integration-test-%s", uuid.New().String()[:8]),
		Description: "Config integration test project",
		RepoURL:     "https://github.com/test/config-repo.git",
		UserID:      userID,
	}

	err := db.Create(project).Error
	require.NoError(t, err, "Failed to create test project")
	return project
}

// TestConfigLifecycle_Integration tests the complete configuration lifecycle
func TestConfigLifecycle_Integration(t *testing.T) {
	db, configService, cleanup := setupConfigIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()

	// Step 1: Create test user and project
	user := createTestUserForConfig(t, db)
	project := createTestProjectForConfig(t, db, user.ID)

	// Step 2: Create initial configuration
	config1 := &model.OpenCodeConfig{
		ProjectID:      project.ID,
		ModelProvider:  "openai",
		ModelName:      "gpt-4o-mini",
		Temperature:    0.7,
		MaxTokens:      4096,
		EnabledTools:   model.ToolsList{"file_ops", "web_search"},
		MaxIterations:  10,
		TimeoutSeconds: 300,
		CreatedBy:      user.ID,
	}

	err := configService.CreateOrUpdateConfig(ctx, config1, "sk-test-key-initial")
	require.NoError(t, err, "Failed to create initial config")

	// Step 3: Verify config saved with version=1 and is active
	activeConfig, err := configService.GetActiveConfig(ctx, project.ID)
	require.NoError(t, err, "Failed to get active config")
	assert.Equal(t, 1, activeConfig.Version, "Initial version should be 1")
	assert.True(t, activeConfig.IsActive, "Config should be active")
	assert.Equal(t, "openai", activeConfig.ModelProvider)
	assert.Equal(t, "gpt-4o-mini", activeConfig.ModelName)
	assert.Equal(t, 0.7, activeConfig.Temperature)
	assert.Equal(t, 4096, activeConfig.MaxTokens)
	assert.Nil(t, activeConfig.APIKeyEncrypted, "API key should be sanitized in GetActiveConfig")

	// Verify encryption stored in DB (query directly)
	var dbConfig model.OpenCodeConfig
	err = db.Where("project_id = ? AND version = ?", project.ID, 1).First(&dbConfig).Error
	require.NoError(t, err, "Failed to query config from DB")
	assert.NotNil(t, dbConfig.APIKeyEncrypted, "API key should be encrypted in DB")
	assert.Greater(t, len(dbConfig.APIKeyEncrypted), 0, "Encrypted key should not be empty")

	// Step 4: Update config (change model to gpt-4o)
	config2 := &model.OpenCodeConfig{
		ProjectID:      project.ID,
		ModelProvider:  "openai",
		ModelName:      "gpt-4o",
		Temperature:    0.8,
		MaxTokens:      8192,
		EnabledTools:   model.ToolsList{"file_ops", "web_search", "code_exec"},
		MaxIterations:  15,
		TimeoutSeconds: 600,
		CreatedBy:      user.ID,
	}

	err = configService.CreateOrUpdateConfig(ctx, config2, "sk-test-key-updated")
	require.NoError(t, err, "Failed to update config")

	// Step 5: Verify new version=2 created and is active, old version=1 deactivated
	activeConfig2, err := configService.GetActiveConfig(ctx, project.ID)
	require.NoError(t, err, "Failed to get active config after update")
	assert.Equal(t, 2, activeConfig2.Version, "Updated version should be 2")
	assert.True(t, activeConfig2.IsActive, "Updated config should be active")
	assert.Equal(t, "gpt-4o", activeConfig2.ModelName)
	assert.Equal(t, 0.8, activeConfig2.Temperature)

	// Verify old version is deactivated
	var oldConfig model.OpenCodeConfig
	err = db.Where("project_id = ? AND version = ?", project.ID, 1).First(&oldConfig).Error
	require.NoError(t, err, "Failed to query old config version")
	assert.False(t, oldConfig.IsActive, "Old config version should be deactivated")

	// Step 6: Get config history (should return 2 versions in reverse order)
	history, err := configService.GetConfigHistory(ctx, project.ID)
	require.NoError(t, err, "Failed to get config history")
	assert.Len(t, history, 2, "Should have 2 config versions")
	assert.Equal(t, 2, history[0].Version, "Latest version should be first")
	assert.Equal(t, 1, history[1].Version, "Older version should be second")

	// Verify API keys sanitized in history
	for i, cfg := range history {
		assert.Nil(t, cfg.APIKeyEncrypted, "API key should be sanitized in history at index %d", i)
	}

	// Step 7: Rollback to version 1
	err = configService.RollbackToVersion(ctx, project.ID, 1)
	require.NoError(t, err, "Failed to rollback to version 1")

	// Step 8: Verify rollback created version=3 with version=1 data
	activeConfig3, err := configService.GetActiveConfig(ctx, project.ID)
	require.NoError(t, err, "Failed to get active config after rollback")
	assert.Equal(t, 3, activeConfig3.Version, "Rollback should create version 3")
	assert.True(t, activeConfig3.IsActive, "Rolled back config should be active")
	assert.Equal(t, "gpt-4o-mini", activeConfig3.ModelName, "Rollback should restore old model name")
	assert.Equal(t, 0.7, activeConfig3.Temperature, "Rollback should restore old temperature")

	// Verify version 2 is now deactivated
	var version2Config model.OpenCodeConfig
	err = db.Where("project_id = ? AND version = ?", project.ID, 2).First(&version2Config).Error
	require.NoError(t, err, "Failed to query version 2 config")
	assert.False(t, version2Config.IsActive, "Version 2 should be deactivated after rollback")

	// Step 9: Delete project and verify configs cascade deleted
	err = db.Delete(project).Error
	require.NoError(t, err, "Failed to delete project")

	// Verify all configs for this project are deleted (cascade)
	var configCount int64
	err = db.Model(&model.OpenCodeConfig{}).Where("project_id = ?", project.ID).Count(&configCount).Error
	require.NoError(t, err, "Failed to count configs")
	assert.Equal(t, int64(0), configCount, "All configs should be cascade deleted when project is deleted")
}

// TestConfigAPIKeyEncryption_Integration tests API key encryption and security
func TestConfigAPIKeyEncryption_Integration(t *testing.T) {
	db, configService, cleanup := setupConfigIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()

	// Step 1: Create test user and project
	user := createTestUserForConfig(t, db)
	project := createTestProjectForConfig(t, db, user.ID)

	// Step 2: Create config with API key
	originalAPIKey := "sk-proj-test1234567890abcdefghijklmnopqrstuvwxyz"
	config := &model.OpenCodeConfig{
		ProjectID:      project.ID,
		ModelProvider:  "openai",
		ModelName:      "gpt-4o-mini",
		Temperature:    0.7,
		MaxTokens:      4096,
		EnabledTools:   model.ToolsList{"file_ops", "web_search"},
		MaxIterations:  10,
		TimeoutSeconds: 300,
		CreatedBy:      user.ID,
	}

	err := configService.CreateOrUpdateConfig(ctx, config, originalAPIKey)
	require.NoError(t, err, "Failed to create config with API key")

	// Step 3: Verify API key encrypted in database (not plaintext)
	var dbConfig model.OpenCodeConfig
	err = db.Where("project_id = ?", project.ID).First(&dbConfig).Error
	require.NoError(t, err, "Failed to query config from database")

	assert.NotNil(t, dbConfig.APIKeyEncrypted, "Encrypted API key should exist in database")
	assert.Greater(t, len(dbConfig.APIKeyEncrypted), 0, "Encrypted key should not be empty")

	// Verify it's NOT plaintext (should not contain the original key)
	encryptedString := string(dbConfig.APIKeyEncrypted)
	assert.NotContains(t, encryptedString, originalAPIKey, "Database should not contain plaintext API key")
	assert.NotContains(t, encryptedString, "sk-proj-", "Database should not contain plaintext key prefix")

	// Step 4: Retrieve config via API (GetActiveConfig) - should NOT expose API key
	activeConfig, err := configService.GetActiveConfig(ctx, project.ID)
	require.NoError(t, err, "Failed to get active config")
	assert.Nil(t, activeConfig.APIKeyEncrypted, "GetActiveConfig should sanitize API key")

	// Step 5: Retrieve config history - should NOT expose API keys
	history, err := configService.GetConfigHistory(ctx, project.ID)
	require.NoError(t, err, "Failed to get config history")
	require.Len(t, history, 1)
	assert.Nil(t, history[0].APIKeyEncrypted, "GetConfigHistory should sanitize API keys")

	// Step 6: Use internal service method to decrypt key (for internal use only)
	decryptedKey, err := configService.GetDecryptedAPIKey(ctx, project.ID)
	require.NoError(t, err, "Failed to decrypt API key")
	assert.Equal(t, originalAPIKey, decryptedKey, "Decrypted key should match original")

	// Step 7: Test decryption with empty API key (should fail gracefully)
	project2 := createTestProjectForConfig(t, db, user.ID)
	config2 := &model.OpenCodeConfig{
		ProjectID:      project2.ID,
		ModelProvider:  "openai",
		ModelName:      "gpt-4o-mini",
		Temperature:    0.7,
		MaxTokens:      4096,
		EnabledTools:   model.ToolsList{"file_ops"},
		MaxIterations:  10,
		TimeoutSeconds: 300,
		CreatedBy:      user.ID,
	}

	err = configService.CreateOrUpdateConfig(ctx, config2, "") // No API key
	require.NoError(t, err, "Should allow config without API key")

	// Try to get decrypted key when none exists
	_, err = configService.GetDecryptedAPIKey(ctx, project2.ID)
	assert.Error(t, err, "Should return error when no API key configured")
	assert.Contains(t, err.Error(), "no API key configured", "Error should indicate no key")

	// Step 8: Test encryption round-trip with special characters
	specialKey := "sk-test-!@#$%^&*()_+-=[]{}|;':\",./<>?"
	config3 := &model.OpenCodeConfig{
		ProjectID:      project2.ID,
		ModelProvider:  "anthropic",
		ModelName:      "claude-3-haiku-20240307",
		Temperature:    1.0,
		MaxTokens:      2048,
		EnabledTools:   model.ToolsList{"file_ops", "terminal"},
		MaxIterations:  5,
		TimeoutSeconds: 180,
		CreatedBy:      user.ID,
	}

	err = configService.CreateOrUpdateConfig(ctx, config3, specialKey)
	require.NoError(t, err, "Should encrypt special characters correctly")

	decryptedSpecialKey, err := configService.GetDecryptedAPIKey(ctx, project2.ID)
	require.NoError(t, err, "Should decrypt special characters correctly")
	assert.Equal(t, specialKey, decryptedSpecialKey, "Special characters should survive encryption round-trip")

	// Step 9: Verify ciphertext is non-deterministic (encrypt same key twice, get different ciphertext)
	project3 := createTestProjectForConfig(t, db, user.ID)
	project4 := createTestProjectForConfig(t, db, user.ID)

	sameKey := "sk-identical-key-12345"

	config4 := &model.OpenCodeConfig{
		ProjectID:      project3.ID,
		ModelProvider:  "openai",
		ModelName:      "gpt-4o-mini",
		Temperature:    0.7,
		MaxTokens:      4096,
		EnabledTools:   model.ToolsList{"file_ops"},
		MaxIterations:  10,
		TimeoutSeconds: 300,
		CreatedBy:      user.ID,
	}

	config5 := &model.OpenCodeConfig{
		ProjectID:      project4.ID,
		ModelProvider:  "openai",
		ModelName:      "gpt-4o-mini",
		Temperature:    0.7,
		MaxTokens:      4096,
		EnabledTools:   model.ToolsList{"file_ops"},
		MaxIterations:  10,
		TimeoutSeconds: 300,
		CreatedBy:      user.ID,
	}

	err = configService.CreateOrUpdateConfig(ctx, config4, sameKey)
	require.NoError(t, err)
	err = configService.CreateOrUpdateConfig(ctx, config5, sameKey)
	require.NoError(t, err)

	var dbConfig4, dbConfig5 model.OpenCodeConfig
	err = db.Where("project_id = ?", project3.ID).First(&dbConfig4).Error
	require.NoError(t, err)
	err = db.Where("project_id = ?", project4.ID).First(&dbConfig5).Error
	require.NoError(t, err)

	assert.NotEqual(t, dbConfig4.APIKeyEncrypted, dbConfig5.APIKeyEncrypted,
		"Encrypting same key twice should produce different ciphertext (due to random nonce)")

	// But both should decrypt to the same plaintext
	decrypted4, err := configService.GetDecryptedAPIKey(ctx, project3.ID)
	require.NoError(t, err)
	decrypted5, err := configService.GetDecryptedAPIKey(ctx, project4.ID)
	require.NoError(t, err)
	assert.Equal(t, sameKey, decrypted4)
	assert.Equal(t, sameKey, decrypted5)
}
