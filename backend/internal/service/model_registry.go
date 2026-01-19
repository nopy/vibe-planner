package service

// ModelInfo represents detailed information about a supported AI model
type ModelInfo struct {
	Provider     string
	Name         string
	MaxTokens    int                // Maximum tokens the model can generate
	ContextSize  int                // Maximum context window size
	Pricing      map[string]float64 // Pricing per 1M tokens (input/output)
	Description  string             // Human-readable description
	Capabilities []string           // List of capabilities (e.g., "function-calling", "vision")
}

// SupportedModels is the registry of all supported AI models
var SupportedModels = []ModelInfo{
	// OpenAI Models
	{
		Provider:     "openai",
		Name:         "gpt-4o",
		MaxTokens:    128000,
		ContextSize:  128000,
		Pricing:      map[string]float64{"input": 2.50, "output": 10.00},
		Description:  "OpenAI GPT-4o - Most capable multimodal model with 128k context",
		Capabilities: []string{"function-calling", "vision", "code"},
	},
	{
		Provider:     "openai",
		Name:         "gpt-4o-mini",
		MaxTokens:    128000,
		ContextSize:  128000,
		Pricing:      map[string]float64{"input": 0.15, "output": 0.60},
		Description:  "OpenAI GPT-4o Mini - Fast and affordable with 128k context (Recommended)",
		Capabilities: []string{"function-calling", "vision", "code"},
	},
	{
		Provider:     "openai",
		Name:         "gpt-4",
		MaxTokens:    8192,
		ContextSize:  8192,
		Pricing:      map[string]float64{"input": 30.00, "output": 60.00},
		Description:  "OpenAI GPT-4 - Previous generation flagship model",
		Capabilities: []string{"function-calling", "code"},
	},
	{
		Provider:     "openai",
		Name:         "gpt-4-turbo",
		MaxTokens:    4096,
		ContextSize:  128000,
		Pricing:      map[string]float64{"input": 10.00, "output": 30.00},
		Description:  "OpenAI GPT-4 Turbo - Optimized GPT-4 with larger context",
		Capabilities: []string{"function-calling", "vision", "code"},
	},
	{
		Provider:     "openai",
		Name:         "gpt-3.5-turbo",
		MaxTokens:    4096,
		ContextSize:  16385,
		Pricing:      map[string]float64{"input": 0.50, "output": 1.50},
		Description:  "OpenAI GPT-3.5 Turbo - Fast and economical general-purpose model",
		Capabilities: []string{"function-calling", "code"},
	},

	// Anthropic Models
	{
		Provider:     "anthropic",
		Name:         "claude-3-opus-20240229",
		MaxTokens:    4096,
		ContextSize:  200000,
		Pricing:      map[string]float64{"input": 15.00, "output": 75.00},
		Description:  "Anthropic Claude 3 Opus - Most capable Claude model with 200k context",
		Capabilities: []string{"function-calling", "vision", "code"},
	},
	{
		Provider:     "anthropic",
		Name:         "claude-3-sonnet-20240229",
		MaxTokens:    4096,
		ContextSize:  200000,
		Pricing:      map[string]float64{"input": 3.00, "output": 15.00},
		Description:  "Anthropic Claude 3 Sonnet - Balanced performance and cost with 200k context",
		Capabilities: []string{"function-calling", "vision", "code"},
	},
	{
		Provider:     "anthropic",
		Name:         "claude-3-haiku-20240307",
		MaxTokens:    4096,
		ContextSize:  200000,
		Pricing:      map[string]float64{"input": 0.25, "output": 1.25},
		Description:  "Anthropic Claude 3 Haiku - Fastest and most affordable Claude model",
		Capabilities: []string{"function-calling", "vision", "code"},
	},
	{
		Provider:     "anthropic",
		Name:         "claude-3.5-sonnet-20240620",
		MaxTokens:    8192,
		ContextSize:  200000,
		Pricing:      map[string]float64{"input": 3.00, "output": 15.00},
		Description:  "Anthropic Claude 3.5 Sonnet - Latest generation with enhanced coding",
		Capabilities: []string{"function-calling", "vision", "code", "artifacts"},
	},
}

// modelRegistry is an internal map for quick lookups
var modelRegistry = buildModelRegistry()

func buildModelRegistry() map[string]map[string]*ModelInfo {
	registry := make(map[string]map[string]*ModelInfo)
	for i := range SupportedModels {
		model := &SupportedModels[i]
		if registry[model.Provider] == nil {
			registry[model.Provider] = make(map[string]*ModelInfo)
		}
		registry[model.Provider][model.Name] = model
	}
	return registry
}

// IsValidModel checks if a provider and model name combination is supported
func IsValidModel(provider, name string) bool {
	if providerModels, ok := modelRegistry[provider]; ok {
		_, exists := providerModels[name]
		return exists
	}
	return false
}

// GetModelInfo retrieves detailed information about a specific model
func GetModelInfo(provider, name string) *ModelInfo {
	if providerModels, ok := modelRegistry[provider]; ok {
		if model, exists := providerModels[name]; exists {
			return model
		}
	}
	return nil
}

// GetModelMaxTokens returns the maximum tokens for a specific model
func GetModelMaxTokens(provider, name string) int {
	if model := GetModelInfo(provider, name); model != nil {
		return model.MaxTokens
	}
	return 0
}

// GetProviderModels returns all models for a specific provider
func GetProviderModels(provider string) []*ModelInfo {
	models := []*ModelInfo{}
	if providerModels, ok := modelRegistry[provider]; ok {
		for _, model := range providerModels {
			models = append(models, model)
		}
	}
	return models
}

// GetAllProviders returns a list of all supported providers
func GetAllProviders() []string {
	providers := []string{}
	seen := make(map[string]bool)
	for _, model := range SupportedModels {
		if !seen[model.Provider] {
			providers = append(providers, model.Provider)
			seen[model.Provider] = true
		}
	}
	return providers
}
