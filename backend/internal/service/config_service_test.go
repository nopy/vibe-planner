package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/npinot/vibe/backend/internal/model"
	"github.com/npinot/vibe/backend/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// MockConfigRepository is a mock implementation of ConfigRepository
type MockConfigRepository struct {
	mock.Mock
}

func (m *MockConfigRepository) GetActiveConfig(ctx context.Context, projectID uuid.UUID) (*model.OpenCodeConfig, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.OpenCodeConfig), args.Error(1)
}

func (m *MockConfigRepository) CreateConfig(ctx context.Context, config *model.OpenCodeConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockConfigRepository) GetConfigVersions(ctx context.Context, projectID uuid.UUID) ([]model.OpenCodeConfig, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.OpenCodeConfig), args.Error(1)
}

func (m *MockConfigRepository) GetConfigByVersion(ctx context.Context, projectID uuid.UUID, version int) (*model.OpenCodeConfig, error) {
	args := m.Called(ctx, projectID, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.OpenCodeConfig), args.Error(1)
}

func (m *MockConfigRepository) DeleteConfig(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// Helper function to generate a valid encryption key
func generateEncryptionKey() string {
	key := make([]byte, 32)
	rand.Read(key)
	return base64.StdEncoding.EncodeToString(key)
}

// Helper function to create a valid config
func createValidConfig() *model.OpenCodeConfig {
	return &model.OpenCodeConfig{
		ID:             uuid.New(),
		ProjectID:      uuid.New(),
		Version:        1,
		IsActive:       true,
		ModelProvider:  "openai",
		ModelName:      "gpt-4o-mini",
		Temperature:    0.7,
		MaxTokens:      4096,
		EnabledTools:   model.ToolsList{"file_ops", "web_search"},
		MaxIterations:  10,
		TimeoutSeconds: 300,
		CreatedBy:      uuid.New(),
	}
}

// Test NewConfigService

func TestNewConfigService_ValidKey(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()

	service, err := NewConfigService(mockRepo, key)

	assert.NoError(t, err)
	assert.NotNil(t, service)
	assert.Equal(t, 32, len(service.encryptionKey))
}

func TestNewConfigService_InvalidKey_NotBase64(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := "not-base64!!!!"

	service, err := NewConfigService(mockRepo, key)

	assert.Error(t, err)
	assert.Nil(t, service)
	assert.Contains(t, err.Error(), "encryption key must be base64-encoded 32 bytes")
}

func TestNewConfigService_InvalidKey_WrongLength(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	// 16 bytes instead of 32
	shortKey := make([]byte, 16)
	rand.Read(shortKey)
	key := base64.StdEncoding.EncodeToString(shortKey)

	service, err := NewConfigService(mockRepo, key)

	assert.Error(t, err)
	assert.Nil(t, service)
}

// Test GetActiveConfig

func TestGetActiveConfig_Success(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	ctx := context.Background()
	projectID := uuid.New()
	expectedConfig := createValidConfig()
	expectedConfig.APIKeyEncrypted = []byte("encrypted-key")

	mockRepo.On("GetActiveConfig", ctx, projectID).Return(expectedConfig, nil)

	config, err := service.GetActiveConfig(ctx, projectID)

	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Nil(t, config.APIKeyEncrypted) // Should be sanitized
	mockRepo.AssertExpectations(t)
}

func TestGetActiveConfig_NotFound(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	ctx := context.Background()
	projectID := uuid.New()

	mockRepo.On("GetActiveConfig", ctx, projectID).Return(nil, gorm.ErrRecordNotFound)

	config, err := service.GetActiveConfig(ctx, projectID)

	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to get active config")
	mockRepo.AssertExpectations(t)
}

// Test CreateOrUpdateConfig

func TestCreateOrUpdateConfig_Success_NoAPIKey(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	ctx := context.Background()
	config := createValidConfig()

	mockRepo.On("CreateConfig", ctx, config).Return(nil)

	err := service.CreateOrUpdateConfig(ctx, config, "")

	assert.NoError(t, err)
	assert.Nil(t, config.APIKeyEncrypted)
	mockRepo.AssertExpectations(t)
}

func TestCreateOrUpdateConfig_Success_WithAPIKey(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	ctx := context.Background()
	config := createValidConfig()
	apiKey := "sk-test-key-12345"

	mockRepo.On("CreateConfig", ctx, mock.MatchedBy(func(c *model.OpenCodeConfig) bool {
		return c.ProjectID == config.ProjectID && len(c.APIKeyEncrypted) > 0
	})).Return(nil)

	err := service.CreateOrUpdateConfig(ctx, config, apiKey)

	assert.NoError(t, err)
	assert.NotNil(t, config.APIKeyEncrypted)
	assert.NotEmpty(t, config.APIKeyEncrypted)
	mockRepo.AssertExpectations(t)
}

func TestCreateOrUpdateConfig_ValidationFails_InvalidProvider(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	ctx := context.Background()
	config := createValidConfig()
	config.ModelProvider = "invalid-provider"

	err := service.CreateOrUpdateConfig(ctx, config, "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid model provider")
	mockRepo.AssertNotCalled(t, "CreateConfig")
}

func TestCreateOrUpdateConfig_ValidationFails_InvalidTemperature(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	ctx := context.Background()
	config := createValidConfig()
	config.Temperature = 3.0 // Invalid: max is 2.0

	err := service.CreateOrUpdateConfig(ctx, config, "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "temperature must be between 0 and 2")
	mockRepo.AssertNotCalled(t, "CreateConfig")
}

func TestCreateOrUpdateConfig_RepositoryError(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	ctx := context.Background()
	config := createValidConfig()

	mockRepo.On("CreateConfig", ctx, config).Return(errors.New("database error"))

	err := service.CreateOrUpdateConfig(ctx, config, "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create config")
	mockRepo.AssertExpectations(t)
}

// Test RollbackToVersion

func TestRollbackToVersion_Success(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	ctx := context.Background()
	projectID := uuid.New()
	version := 2
	oldConfig := createValidConfig()
	oldConfig.Version = version

	mockRepo.On("GetConfigByVersion", ctx, projectID, version).Return(oldConfig, nil)
	mockRepo.On("CreateConfig", ctx, mock.MatchedBy(func(c *model.OpenCodeConfig) bool {
		return c.ID == uuid.Nil && c.ProjectID == oldConfig.ProjectID
	})).Return(nil)

	err := service.RollbackToVersion(ctx, projectID, version)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestRollbackToVersion_VersionNotFound(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	ctx := context.Background()
	projectID := uuid.New()
	version := 999

	mockRepo.On("GetConfigByVersion", ctx, projectID, version).Return(nil, gorm.ErrRecordNotFound)

	err := service.RollbackToVersion(ctx, projectID, version)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config version 999 not found")
	mockRepo.AssertNotCalled(t, "CreateConfig")
	mockRepo.AssertExpectations(t)
}

// Test GetConfigHistory

func TestGetConfigHistory_Success(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	ctx := context.Background()
	projectID := uuid.New()

	config1 := createValidConfig()
	config1.Version = 1
	config1.APIKeyEncrypted = []byte("encrypted1")

	config2 := createValidConfig()
	config2.Version = 2
	config2.APIKeyEncrypted = []byte("encrypted2")

	expectedConfigs := []model.OpenCodeConfig{*config1, *config2}

	mockRepo.On("GetConfigVersions", ctx, projectID).Return(expectedConfigs, nil)

	configs, err := service.GetConfigHistory(ctx, projectID)

	assert.NoError(t, err)
	assert.Len(t, configs, 2)
	// Verify API keys are sanitized
	for _, config := range configs {
		assert.Nil(t, config.APIKeyEncrypted)
	}
	mockRepo.AssertExpectations(t)
}

func TestGetConfigHistory_Empty(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	ctx := context.Background()
	projectID := uuid.New()

	mockRepo.On("GetConfigVersions", ctx, projectID).Return([]model.OpenCodeConfig{}, nil)

	configs, err := service.GetConfigHistory(ctx, projectID)

	assert.NoError(t, err)
	assert.Empty(t, configs)
	mockRepo.AssertExpectations(t)
}

// Test GetDecryptedAPIKey

func TestGetDecryptedAPIKey_Success(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	ctx := context.Background()
	projectID := uuid.New()

	originalKey := "sk-test-key-12345"
	encryptedKey, _ := service.encryptAPIKey(originalKey)

	config := createValidConfig()
	config.APIKeyEncrypted = encryptedKey

	mockRepo.On("GetActiveConfig", ctx, projectID).Return(config, nil)

	decryptedKey, err := service.GetDecryptedAPIKey(ctx, projectID)

	assert.NoError(t, err)
	assert.Equal(t, originalKey, decryptedKey)
	mockRepo.AssertExpectations(t)
}

func TestGetDecryptedAPIKey_NoAPIKey(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	ctx := context.Background()
	projectID := uuid.New()

	config := createValidConfig()
	config.APIKeyEncrypted = nil

	mockRepo.On("GetActiveConfig", ctx, projectID).Return(config, nil)

	decryptedKey, err := service.GetDecryptedAPIKey(ctx, projectID)

	assert.Error(t, err)
	assert.Empty(t, decryptedKey)
	assert.Contains(t, err.Error(), "no API key configured")
	mockRepo.AssertExpectations(t)
}

// Test validateConfig

func TestValidateConfig_ValidOpenAI(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	config := createValidConfig()
	config.ModelProvider = "openai"
	config.ModelName = "gpt-4o"

	err := service.validateConfig(config)

	assert.NoError(t, err)
}

func TestValidateConfig_ValidAnthropic(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	config := createValidConfig()
	config.ModelProvider = "anthropic"
	config.ModelName = "claude-3-opus-20240229"

	err := service.validateConfig(config)

	assert.NoError(t, err)
}

func TestValidateConfig_ValidCustom(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	endpoint := "https://api.custom.com/v1"
	config := createValidConfig()
	config.ModelProvider = "custom"
	config.ModelName = "llama-3-70b"
	config.APIEndpoint = &endpoint

	err := service.validateConfig(config)

	assert.NoError(t, err)
}

func TestValidateConfig_InvalidProvider(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	config := createValidConfig()
	config.ModelProvider = "invalid"

	err := service.validateConfig(config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid model provider")
}

func TestValidateConfig_InvalidOpenAIModel(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	config := createValidConfig()
	config.ModelProvider = "openai"
	config.ModelName = "gpt-5"

	err := service.validateConfig(config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid model")
}

func TestValidateConfig_InvalidAnthropicModel(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	config := createValidConfig()
	config.ModelProvider = "anthropic"
	config.ModelName = "claude-4"

	err := service.validateConfig(config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid model")
}

func TestValidateConfig_TemperatureTooLow(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	config := createValidConfig()
	config.Temperature = -0.1

	err := service.validateConfig(config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "temperature must be between 0 and 2")
}

func TestValidateConfig_TemperatureTooHigh(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	config := createValidConfig()
	config.Temperature = 2.1

	err := service.validateConfig(config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "temperature must be between 0 and 2")
}

func TestValidateConfig_MaxTokensTooLow(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	config := createValidConfig()
	config.MaxTokens = 0

	err := service.validateConfig(config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_tokens must be between 1 and 128000")
}

func TestValidateConfig_MaxTokensTooHigh(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	httpsEndpoint := "https://api.custom.com/v1"
	config := createValidConfig()
	config.ModelProvider = "custom"
	config.ModelName = "custom-model"
	config.APIEndpoint = &httpsEndpoint
	config.MaxTokens = 200000

	err := service.validateConfig(config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_tokens must be between 1 and 128000")
}

func TestValidateConfig_MaxIterationsTooLow(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	config := createValidConfig()
	config.MaxIterations = 0

	err := service.validateConfig(config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_iterations must be between 1 and 50")
}

func TestValidateConfig_MaxIterationsTooHigh(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	config := createValidConfig()
	config.MaxIterations = 100

	err := service.validateConfig(config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_iterations must be between 1 and 50")
}

func TestValidateConfig_TimeoutTooLow(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	config := createValidConfig()
	config.TimeoutSeconds = 30

	err := service.validateConfig(config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout_seconds must be between 60 and 3600")
}

func TestValidateConfig_TimeoutTooHigh(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	config := createValidConfig()
	config.TimeoutSeconds = 5000

	err := service.validateConfig(config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout_seconds must be between 60 and 3600")
}

func TestValidateConfig_InvalidTool(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	config := createValidConfig()
	config.EnabledTools = model.ToolsList{"file_ops", "invalid_tool"}

	err := service.validateConfig(config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid tool")
}

func TestValidateConfig_CustomProvider_MissingEndpoint(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	config := createValidConfig()
	config.ModelProvider = "custom"
	config.APIEndpoint = nil

	err := service.validateConfig(config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "api_endpoint is required for custom provider")
}

func TestValidateConfig_CustomProvider_EmptyEndpoint(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	emptyEndpoint := ""
	config := createValidConfig()
	config.ModelProvider = "custom"
	config.APIEndpoint = &emptyEndpoint

	err := service.validateConfig(config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "api_endpoint is required for custom provider")
}

func TestValidateConfig_CustomProvider_NotHTTPS(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	httpEndpoint := "http://api.custom.com/v1"
	config := createValidConfig()
	config.ModelProvider = "custom"
	config.APIEndpoint = &httpEndpoint

	err := service.validateConfig(config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "api_endpoint must use HTTPS")
}

// Test Encryption/Decryption

func TestEncryptDecrypt_Success(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	originalText := "sk-test-api-key-12345"

	encrypted, err := service.encryptAPIKey(originalText)
	assert.NoError(t, err)
	assert.NotEmpty(t, encrypted)

	decrypted, err := service.decryptAPIKey(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, originalText, decrypted)
}

func TestEncryptDecrypt_EmptyString(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	originalText := ""

	encrypted, err := service.encryptAPIKey(originalText)
	assert.NoError(t, err)
	assert.NotEmpty(t, encrypted)

	decrypted, err := service.decryptAPIKey(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, originalText, decrypted)
}

func TestDecrypt_InvalidCiphertext(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	invalidCiphertext := []byte("too-short")

	decrypted, err := service.decryptAPIKey(invalidCiphertext)

	assert.Error(t, err)
	assert.Empty(t, decrypted)
	assert.Contains(t, err.Error(), "ciphertext too short")
}

func TestDecrypt_CorruptedCiphertext(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	// Encrypt first
	encrypted, _ := service.encryptAPIKey("test-key")

	// Corrupt the ciphertext
	encrypted[len(encrypted)-1] ^= 0xff

	decrypted, err := service.decryptAPIKey(encrypted)

	assert.Error(t, err)
	assert.Empty(t, decrypted)
}

func TestEncryptDecrypt_LongString(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	// Generate a long API key
	longKey := "sk-proj-" + string(make([]byte, 500))
	for i := range longKey {
		longKey = longKey[:6] + string(rune('a'+i%26)) + longKey[7:]
	}

	encrypted, err := service.encryptAPIKey(longKey)
	assert.NoError(t, err)

	decrypted, err := service.decryptAPIKey(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, longKey, decrypted)
}

// Test provider-specific max_tokens validation

func TestValidateConfig_MaxTokens_ExceedsModelLimit_GPT4(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	config := createValidConfig()
	config.ModelProvider = "openai"
	config.ModelName = "gpt-4"
	config.MaxTokens = 10000 // Exceeds gpt-4 limit of 8192

	err := service.validateConfig(config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_tokens")
	assert.Contains(t, err.Error(), "exceeds model limit")
}

func TestValidateConfig_MaxTokens_WithinModelLimit_GPT4(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	config := createValidConfig()
	config.ModelProvider = "openai"
	config.ModelName = "gpt-4"
	config.MaxTokens = 8192 // Exactly at the limit

	err := service.validateConfig(config)

	assert.NoError(t, err)
}

func TestValidateConfig_MaxTokens_ExceedsModelLimit_GPT4oMini(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	config := createValidConfig()
	config.ModelProvider = "openai"
	config.ModelName = "gpt-4o-mini"
	config.MaxTokens = 129000 // Exceeds gpt-4o-mini limit of 128000

	err := service.validateConfig(config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_tokens")
	assert.Contains(t, err.Error(), "exceeds model limit")
}

func TestValidateConfig_MaxTokens_WithinModelLimit_GPT4oMini(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	config := createValidConfig()
	config.ModelProvider = "openai"
	config.ModelName = "gpt-4o-mini"
	config.MaxTokens = 128000 // Exactly at the limit

	err := service.validateConfig(config)

	assert.NoError(t, err)
}

func TestValidateConfig_MaxTokens_ExceedsModelLimit_Claude3Opus(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	config := createValidConfig()
	config.ModelProvider = "anthropic"
	config.ModelName = "claude-3-opus-20240229"
	config.MaxTokens = 5000 // Exceeds claude-3-opus limit of 4096

	err := service.validateConfig(config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_tokens")
	assert.Contains(t, err.Error(), "exceeds model limit")
}

func TestValidateConfig_MaxTokens_WithinModelLimit_Claude3Opus(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	config := createValidConfig()
	config.ModelProvider = "anthropic"
	config.ModelName = "claude-3-opus-20240229"
	config.MaxTokens = 4096 // Exactly at the limit

	err := service.validateConfig(config)

	assert.NoError(t, err)
}

func TestValidateConfig_MaxTokens_CustomProvider_NoModelLimit(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	key := generateEncryptionKey()
	service, _ := NewConfigService(repository.ConfigRepository(mockRepo), key)

	httpsEndpoint := "https://api.custom.com/v1"
	config := createValidConfig()
	config.ModelProvider = "custom"
	config.ModelName = "llama-3-70b"
	config.APIEndpoint = &httpsEndpoint
	config.MaxTokens = 128000 // Should only check general bounds, not model-specific

	err := service.validateConfig(config)

	assert.NoError(t, err)
}
