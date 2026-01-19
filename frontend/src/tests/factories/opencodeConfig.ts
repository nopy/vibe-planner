import type { OpenCodeConfig } from '@/types'

/**
 * Build a realistic OpenCodeConfig object for testing
 */
export function buildConfig(overrides?: Partial<OpenCodeConfig>): OpenCodeConfig {
  const baseConfig: OpenCodeConfig = {
    id: 'config-123',
    project_id: 'project-456',
    version: 1,
    is_active: true,

    // Model configuration
    model_provider: 'openai',
    model_name: 'gpt-4o-mini',
    model_version: undefined,

    // Provider configuration
    api_endpoint: undefined,
    api_key: undefined, // Never included in responses
    temperature: 0.7,
    max_tokens: 4000,

    // Tools configuration
    enabled_tools: ['file_ops', 'web_search', 'code_exec', 'terminal'],
    tools_config: {},

    // System configuration
    system_prompt: undefined,
    max_iterations: 10,
    timeout_seconds: 300,

    // Metadata
    created_by: 'test@example.com',
    created_at: '2024-01-15T10:00:00Z',
    updated_at: '2024-01-15T10:00:00Z',
  }

  return { ...baseConfig, ...overrides }
}

/**
 * Build a history of configuration versions
 */
export function buildConfigHistory(count = 3): OpenCodeConfig[] {
  const baseTime = new Date('2024-01-15T10:00:00Z').getTime()
  const configs: OpenCodeConfig[] = []

  for (let i = count; i > 0; i--) {
    const version = count - i + 1
    const timestamp = new Date(baseTime + i * 3600000).toISOString() // 1 hour apart

    configs.push(
      buildConfig({
        id: `config-${version}`,
        version,
        is_active: version === count, // Latest version is active
        created_at: timestamp,
        updated_at: timestamp,
        temperature: 0.5 + (version * 0.1), // Vary temperature
        max_tokens: 3000 + (version * 500), // Vary max tokens
      })
    )
  }

  return configs
}

/**
 * Build a config with custom provider
 */
export function buildCustomProviderConfig(
  overrides?: Partial<OpenCodeConfig>
): OpenCodeConfig {
  return buildConfig({
    model_provider: 'custom',
    model_name: 'llama-3-70b',
    api_endpoint: 'https://api.custom.ai/v1',
    ...overrides,
  })
}

/**
 * Build a config with Anthropic provider
 */
export function buildAnthropicConfig(overrides?: Partial<OpenCodeConfig>): OpenCodeConfig {
  return buildConfig({
    model_provider: 'anthropic',
    model_name: 'claude-3-5-sonnet-20240620',
    ...overrides,
  })
}

/**
 * Build a config with no tools enabled
 */
export function buildEmptyToolsConfig(overrides?: Partial<OpenCodeConfig>): OpenCodeConfig {
  return buildConfig({
    enabled_tools: [],
    ...overrides,
  })
}

/**
 * Build a config with minimal tools
 */
export function buildMinimalToolsConfig(overrides?: Partial<OpenCodeConfig>): OpenCodeConfig {
  return buildConfig({
    enabled_tools: ['file_ops'],
    ...overrides,
  })
}
