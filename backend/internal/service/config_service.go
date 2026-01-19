package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/npinot/vibe/backend/internal/model"
	"github.com/npinot/vibe/backend/internal/repository"
)

type ConfigService struct {
	configRepo    repository.ConfigRepository
	encryptionKey []byte // 32-byte AES-256 key
}

func NewConfigService(configRepo repository.ConfigRepository, encryptionKey string) (*ConfigService, error) {
	// Decode base64 encryption key
	key, err := base64.StdEncoding.DecodeString(encryptionKey)
	if err != nil || len(key) != 32 {
		return nil, errors.New("encryption key must be base64-encoded 32 bytes")
	}

	return &ConfigService{
		configRepo:    configRepo,
		encryptionKey: key,
	}, nil
}

// GetActiveConfig retrieves the active configuration for a project
func (s *ConfigService) GetActiveConfig(ctx context.Context, projectID uuid.UUID) (*model.OpenCodeConfig, error) {
	config, err := s.configRepo.GetActiveConfig(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active config: %w", err)
	}

	// Don't expose encrypted key in API response
	config.APIKeyEncrypted = nil

	return config, nil
}

// CreateOrUpdateConfig creates a new configuration version
func (s *ConfigService) CreateOrUpdateConfig(ctx context.Context, config *model.OpenCodeConfig, apiKey string) error {
	// Validate configuration
	if err := s.validateConfig(config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// Encrypt API key if provided
	if apiKey != "" {
		encrypted, err := s.encryptAPIKey(apiKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt API key: %w", err)
		}
		config.APIKeyEncrypted = encrypted
	}

	// Create config (repository handles versioning)
	if err := s.configRepo.CreateConfig(ctx, config); err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	return nil
}

// RollbackToVersion activates a previous configuration version
func (s *ConfigService) RollbackToVersion(ctx context.Context, projectID uuid.UUID, version int) error {
	// Get the old version
	oldConfig, err := s.configRepo.GetConfigByVersion(ctx, projectID, version)
	if err != nil {
		return fmt.Errorf("config version %d not found: %w", version, err)
	}

	// Create a new version with the old config data
	newConfig := *oldConfig
	newConfig.ID = uuid.Nil // Will be auto-generated

	return s.configRepo.CreateConfig(ctx, &newConfig)
}

// GetConfigHistory retrieves all configuration versions
func (s *ConfigService) GetConfigHistory(ctx context.Context, projectID uuid.UUID) ([]model.OpenCodeConfig, error) {
	configs, err := s.configRepo.GetConfigVersions(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get config history: %w", err)
	}

	// Sanitize: Remove encrypted API keys from response
	for i := range configs {
		configs[i].APIKeyEncrypted = nil
	}

	return configs, nil
}

// GetDecryptedAPIKey retrieves and decrypts the API key for internal use
func (s *ConfigService) GetDecryptedAPIKey(ctx context.Context, projectID uuid.UUID) (string, error) {
	config, err := s.configRepo.GetActiveConfig(ctx, projectID)
	if err != nil {
		return "", err
	}

	if len(config.APIKeyEncrypted) == 0 {
		return "", errors.New("no API key configured")
	}

	return s.decryptAPIKey(config.APIKeyEncrypted)
}

// validateConfig validates configuration fields
func (s *ConfigService) validateConfig(config *model.OpenCodeConfig) error {
	// Validate model provider
	validProviders := map[string]bool{
		"openai":    true,
		"anthropic": true,
		"custom":    true,
	}
	if !validProviders[config.ModelProvider] {
		return fmt.Errorf("invalid model provider: %s", config.ModelProvider)
	}

	// Validate model name based on provider
	if config.ModelProvider == "openai" {
		validModels := map[string]bool{
			"gpt-4o":        true,
			"gpt-4o-mini":   true,
			"gpt-4":         true,
			"gpt-3.5-turbo": true,
		}
		if !validModels[config.ModelName] {
			return fmt.Errorf("invalid OpenAI model: %s", config.ModelName)
		}
	}

	if config.ModelProvider == "anthropic" {
		validModels := map[string]bool{
			"claude-3-opus-20240229":   true,
			"claude-3-sonnet-20240229": true,
			"claude-3-haiku-20240307":  true,
		}
		if !validModels[config.ModelName] {
			return fmt.Errorf("invalid Anthropic model: %s", config.ModelName)
		}
	}

	// Validate temperature range
	if config.Temperature < 0 || config.Temperature > 2 {
		return errors.New("temperature must be between 0 and 2")
	}

	// Validate max_tokens
	if config.MaxTokens <= 0 || config.MaxTokens > 128000 {
		return errors.New("max_tokens must be between 1 and 128000")
	}

	// Validate max_iterations
	if config.MaxIterations <= 0 || config.MaxIterations > 50 {
		return errors.New("max_iterations must be between 1 and 50")
	}

	// Validate timeout_seconds
	if config.TimeoutSeconds < 60 || config.TimeoutSeconds > 3600 {
		return errors.New("timeout_seconds must be between 60 and 3600")
	}

	// Validate enabled_tools
	validTools := map[string]bool{
		"file_ops":   true,
		"web_search": true,
		"code_exec":  true,
		"terminal":   true,
	}
	for _, tool := range config.EnabledTools {
		if !validTools[tool] {
			return fmt.Errorf("invalid tool: %s", tool)
		}
	}

	// Validate API endpoint for custom provider
	if config.ModelProvider == "custom" {
		if config.APIEndpoint == nil || *config.APIEndpoint == "" {
			return errors.New("api_endpoint is required for custom provider")
		}
		// Validate HTTPS requirement
		if !strings.HasPrefix(*config.APIEndpoint, "https://") {
			return errors.New("api_endpoint must use HTTPS")
		}
	}

	return nil
}

// encryptAPIKey encrypts an API key using AES-256-GCM
func (s *ConfigService) encryptAPIKey(plaintext string) ([]byte, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return ciphertext, nil
}

// decryptAPIKey decrypts an encrypted API key
func (s *ConfigService) decryptAPIKey(ciphertext []byte) (string, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
