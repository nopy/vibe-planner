-- Add missing fields to projects table for Phase 2.1

-- Add repo_url for git repository URL
ALTER TABLE projects ADD COLUMN IF NOT EXISTS repo_url TEXT;

-- Add pod_created_at to track when K8s pod was spawned
ALTER TABLE projects ADD COLUMN IF NOT EXISTS pod_created_at TIMESTAMP;

-- Add deleted_at for soft deletes (GORM soft delete support)
ALTER TABLE projects ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

-- Add pod_error to store pod creation/runtime errors
ALTER TABLE projects ADD COLUMN IF NOT EXISTS pod_error TEXT;

-- Create index on deleted_at for soft delete queries
CREATE INDEX IF NOT EXISTS idx_projects_deleted_at ON projects(deleted_at);
