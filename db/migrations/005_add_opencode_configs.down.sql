-- Rollback opencode_configs table for Phase 6.1

-- Drop indexes first
DROP INDEX IF EXISTS idx_opencode_configs_version;
DROP INDEX IF EXISTS idx_opencode_configs_active;
DROP INDEX IF EXISTS idx_opencode_configs_project_id;

-- Drop the table
DROP TABLE IF EXISTS opencode_configs;
