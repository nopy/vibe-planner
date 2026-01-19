-- Create opencode_configs table for Phase 6.1 - OpenCode configuration management

CREATE TABLE IF NOT EXISTS opencode_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    version INT NOT NULL DEFAULT 1,
    is_active BOOLEAN NOT NULL DEFAULT true,
    
    -- Model configuration
    model_provider VARCHAR(50) NOT NULL,
    model_name VARCHAR(100) NOT NULL,
    model_version VARCHAR(50),
    
    -- Provider configuration
    api_endpoint TEXT,
    api_key_encrypted BYTEA,
    temperature DECIMAL(3,2) NOT NULL DEFAULT 0.7,
    max_tokens INT NOT NULL DEFAULT 4096,
    
    -- Tools configuration (JSON)
    enabled_tools JSONB NOT NULL DEFAULT '["file_ops", "web_search", "code_exec"]'::jsonb,
    tools_config JSONB,
    
    -- System configuration
    system_prompt TEXT,
    max_iterations INT NOT NULL DEFAULT 10,
    timeout_seconds INT NOT NULL DEFAULT 300,
    
    -- Metadata
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT unique_project_version UNIQUE(project_id, version),
    CONSTRAINT check_version_positive CHECK (version > 0),
    CONSTRAINT check_temperature_range CHECK (temperature >= 0 AND temperature <= 2),
    CONSTRAINT check_max_tokens_positive CHECK (max_tokens > 0),
    CONSTRAINT check_max_iterations_positive CHECK (max_iterations > 0),
    CONSTRAINT check_timeout_positive CHECK (timeout_seconds > 0)
);

-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_opencode_configs_project_id ON opencode_configs(project_id);
CREATE INDEX IF NOT EXISTS idx_opencode_configs_active ON opencode_configs(project_id, is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_opencode_configs_version ON opencode_configs(project_id, version);

-- Add comments for documentation
COMMENT ON TABLE opencode_configs IS 'OpenCode agent configuration with versioning (Phase 6.1)';
COMMENT ON COLUMN opencode_configs.project_id IS 'Reference to the project this config belongs to';
COMMENT ON COLUMN opencode_configs.version IS 'Configuration version number (incremental, starts at 1)';
COMMENT ON COLUMN opencode_configs.is_active IS 'Whether this is the currently active configuration';
COMMENT ON COLUMN opencode_configs.model_provider IS 'AI provider: openai, anthropic, custom';
COMMENT ON COLUMN opencode_configs.model_name IS 'Model name: gpt-4o, claude-3-opus, etc.';
COMMENT ON COLUMN opencode_configs.model_version IS 'Optional specific model version';
COMMENT ON COLUMN opencode_configs.api_endpoint IS 'Custom API endpoint (for custom provider)';
COMMENT ON COLUMN opencode_configs.api_key_encrypted IS 'AES-256-GCM encrypted API key';
COMMENT ON COLUMN opencode_configs.temperature IS 'Model temperature (0-2, controls creativity)';
COMMENT ON COLUMN opencode_configs.max_tokens IS 'Maximum tokens to generate';
COMMENT ON COLUMN opencode_configs.enabled_tools IS 'JSON array of enabled tool names';
COMMENT ON COLUMN opencode_configs.tools_config IS 'JSON object with tool-specific configuration';
COMMENT ON COLUMN opencode_configs.system_prompt IS 'Optional custom system prompt';
COMMENT ON COLUMN opencode_configs.max_iterations IS 'Maximum agent iterations per session';
COMMENT ON COLUMN opencode_configs.timeout_seconds IS 'Session timeout in seconds';
