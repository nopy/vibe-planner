package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test IsValidModel function

func TestIsValidModel_OpenAI_GPT4o(t *testing.T) {
	assert.True(t, IsValidModel("openai", "gpt-4o"))
}

func TestIsValidModel_OpenAI_GPT4oMini(t *testing.T) {
	assert.True(t, IsValidModel("openai", "gpt-4o-mini"))
}

func TestIsValidModel_OpenAI_GPT4(t *testing.T) {
	assert.True(t, IsValidModel("openai", "gpt-4"))
}

func TestIsValidModel_OpenAI_GPT4Turbo(t *testing.T) {
	assert.True(t, IsValidModel("openai", "gpt-4-turbo"))
}

func TestIsValidModel_OpenAI_GPT35Turbo(t *testing.T) {
	assert.True(t, IsValidModel("openai", "gpt-3.5-turbo"))
}

func TestIsValidModel_Anthropic_Claude3Opus(t *testing.T) {
	assert.True(t, IsValidModel("anthropic", "claude-3-opus-20240229"))
}

func TestIsValidModel_Anthropic_Claude3Sonnet(t *testing.T) {
	assert.True(t, IsValidModel("anthropic", "claude-3-sonnet-20240229"))
}

func TestIsValidModel_Anthropic_Claude3Haiku(t *testing.T) {
	assert.True(t, IsValidModel("anthropic", "claude-3-haiku-20240307"))
}

func TestIsValidModel_Anthropic_Claude35Sonnet(t *testing.T) {
	assert.True(t, IsValidModel("anthropic", "claude-3.5-sonnet-20240620"))
}

func TestIsValidModel_InvalidProvider(t *testing.T) {
	assert.False(t, IsValidModel("invalid-provider", "gpt-4o"))
}

func TestIsValidModel_InvalidModel(t *testing.T) {
	assert.False(t, IsValidModel("openai", "gpt-5"))
}

func TestIsValidModel_InvalidBoth(t *testing.T) {
	assert.False(t, IsValidModel("invalid-provider", "invalid-model"))
}

func TestIsValidModel_CaseSensitive(t *testing.T) {
	// Should be case-sensitive
	assert.False(t, IsValidModel("OpenAI", "GPT-4o"))
	assert.False(t, IsValidModel("openai", "GPT-4O"))
}

// Test GetModelInfo function

func TestGetModelInfo_OpenAI_GPT4o(t *testing.T) {
	model := GetModelInfo("openai", "gpt-4o")

	assert.NotNil(t, model)
	assert.Equal(t, "openai", model.Provider)
	assert.Equal(t, "gpt-4o", model.Name)
	assert.Equal(t, 128000, model.MaxTokens)
	assert.Equal(t, 128000, model.ContextSize)
	assert.Equal(t, 2.50, model.Pricing["input"])
	assert.Equal(t, 10.00, model.Pricing["output"])
	assert.Contains(t, model.Description, "GPT-4o")
	assert.Contains(t, model.Capabilities, "function-calling")
	assert.Contains(t, model.Capabilities, "vision")
	assert.Contains(t, model.Capabilities, "code")
}

func TestGetModelInfo_OpenAI_GPT4oMini(t *testing.T) {
	model := GetModelInfo("openai", "gpt-4o-mini")

	assert.NotNil(t, model)
	assert.Equal(t, "gpt-4o-mini", model.Name)
	assert.Equal(t, 128000, model.MaxTokens)
	assert.Equal(t, 0.15, model.Pricing["input"])
	assert.Equal(t, 0.60, model.Pricing["output"])
	assert.Contains(t, model.Description, "Recommended")
}

func TestGetModelInfo_Anthropic_Claude3Opus(t *testing.T) {
	model := GetModelInfo("anthropic", "claude-3-opus-20240229")

	assert.NotNil(t, model)
	assert.Equal(t, "anthropic", model.Provider)
	assert.Equal(t, "claude-3-opus-20240229", model.Name)
	assert.Equal(t, 4096, model.MaxTokens)
	assert.Equal(t, 200000, model.ContextSize)
	assert.Equal(t, 15.00, model.Pricing["input"])
	assert.Equal(t, 75.00, model.Pricing["output"])
	assert.Contains(t, model.Description, "Claude 3 Opus")
}

func TestGetModelInfo_InvalidProvider(t *testing.T) {
	model := GetModelInfo("invalid-provider", "gpt-4o")
	assert.Nil(t, model)
}

func TestGetModelInfo_InvalidModel(t *testing.T) {
	model := GetModelInfo("openai", "gpt-5")
	assert.Nil(t, model)
}

// Test GetModelMaxTokens function

func TestGetModelMaxTokens_OpenAI_GPT4o(t *testing.T) {
	maxTokens := GetModelMaxTokens("openai", "gpt-4o")
	assert.Equal(t, 128000, maxTokens)
}

func TestGetModelMaxTokens_OpenAI_GPT4(t *testing.T) {
	maxTokens := GetModelMaxTokens("openai", "gpt-4")
	assert.Equal(t, 8192, maxTokens)
}

func TestGetModelMaxTokens_Anthropic_Claude3Opus(t *testing.T) {
	maxTokens := GetModelMaxTokens("anthropic", "claude-3-opus-20240229")
	assert.Equal(t, 4096, maxTokens)
}

func TestGetModelMaxTokens_InvalidModel(t *testing.T) {
	maxTokens := GetModelMaxTokens("openai", "invalid-model")
	assert.Equal(t, 0, maxTokens)
}

// Test GetProviderModels function

func TestGetProviderModels_OpenAI(t *testing.T) {
	models := GetProviderModels("openai")

	assert.NotNil(t, models)
	assert.GreaterOrEqual(t, len(models), 5) // At least 5 OpenAI models

	// Verify all returned models are OpenAI models
	for _, model := range models {
		assert.Equal(t, "openai", model.Provider)
	}
}

func TestGetProviderModels_Anthropic(t *testing.T) {
	models := GetProviderModels("anthropic")

	assert.NotNil(t, models)
	assert.GreaterOrEqual(t, len(models), 4) // At least 4 Anthropic models

	// Verify all returned models are Anthropic models
	for _, model := range models {
		assert.Equal(t, "anthropic", model.Provider)
	}
}

func TestGetProviderModels_InvalidProvider(t *testing.T) {
	models := GetProviderModels("invalid-provider")

	assert.NotNil(t, models)
	assert.Empty(t, models)
}

// Test GetAllProviders function

func TestGetAllProviders_ReturnsAllProviders(t *testing.T) {
	providers := GetAllProviders()

	assert.NotNil(t, providers)
	assert.Contains(t, providers, "openai")
	assert.Contains(t, providers, "anthropic")
	assert.GreaterOrEqual(t, len(providers), 2)
}

func TestGetAllProviders_NoDuplicates(t *testing.T) {
	providers := GetAllProviders()

	seen := make(map[string]bool)
	for _, provider := range providers {
		assert.False(t, seen[provider], "Provider %s appears multiple times", provider)
		seen[provider] = true
	}
}

// Test SupportedModels registry

func TestSupportedModels_AllHaveRequiredFields(t *testing.T) {
	for _, model := range SupportedModels {
		assert.NotEmpty(t, model.Provider, "Model missing Provider")
		assert.NotEmpty(t, model.Name, "Model missing Name")
		assert.Greater(t, model.MaxTokens, 0, "Model %s has invalid MaxTokens", model.Name)
		assert.Greater(t, model.ContextSize, 0, "Model %s has invalid ContextSize", model.Name)
		assert.NotNil(t, model.Pricing, "Model %s missing Pricing", model.Name)
		assert.Contains(t, model.Pricing, "input", "Model %s missing input pricing", model.Name)
		assert.Contains(t, model.Pricing, "output", "Model %s missing output pricing", model.Name)
		assert.NotEmpty(t, model.Description, "Model %s missing Description", model.Name)
		assert.NotEmpty(t, model.Capabilities, "Model %s missing Capabilities", model.Name)
	}
}

func TestSupportedModels_AllModelsUnique(t *testing.T) {
	seen := make(map[string]bool)
	for _, model := range SupportedModels {
		key := model.Provider + "/" + model.Name
		assert.False(t, seen[key], "Duplicate model: %s", key)
		seen[key] = true
	}
}

func TestSupportedModels_ContextSizeGreaterOrEqualMaxTokens(t *testing.T) {
	for _, model := range SupportedModels {
		assert.GreaterOrEqual(t, model.ContextSize, model.MaxTokens,
			"Model %s has ContextSize (%d) less than MaxTokens (%d)",
			model.Name, model.ContextSize, model.MaxTokens)
	}
}

// Test model registry internal structure

func TestModelRegistry_InitializedCorrectly(t *testing.T) {
	assert.NotNil(t, modelRegistry)
	assert.NotEmpty(t, modelRegistry)

	// Verify registry contains expected providers
	assert.Contains(t, modelRegistry, "openai")
	assert.Contains(t, modelRegistry, "anthropic")
}

func TestModelRegistry_ContainsAllModels(t *testing.T) {
	for _, model := range SupportedModels {
		assert.Contains(t, modelRegistry, model.Provider)
		assert.Contains(t, modelRegistry[model.Provider], model.Name)

		// Verify the stored model matches the original
		storedModel := modelRegistry[model.Provider][model.Name]
		assert.Equal(t, model.Provider, storedModel.Provider)
		assert.Equal(t, model.Name, storedModel.Name)
		assert.Equal(t, model.MaxTokens, storedModel.MaxTokens)
	}
}
