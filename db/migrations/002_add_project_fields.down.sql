-- Rollback: Remove fields added in 002_add_project_fields.up.sql

-- Drop index
DROP INDEX IF EXISTS idx_projects_deleted_at;

-- Remove columns (in reverse order)
ALTER TABLE projects DROP COLUMN IF EXISTS pod_error;
ALTER TABLE projects DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE projects DROP COLUMN IF EXISTS pod_created_at;
ALTER TABLE projects DROP COLUMN IF EXISTS repo_url;
