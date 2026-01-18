-- Add Kanban board fields to tasks table for Phase 3

-- Add position field for ordering tasks within Kanban columns
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS position INTEGER NOT NULL DEFAULT 0;

-- Add priority field for task prioritization
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS priority VARCHAR(20) DEFAULT 'medium';

-- Add assigned_to field for future task assignment (Phase 7)
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS assigned_to UUID REFERENCES users(id);

-- Add deleted_at for soft deletes (GORM soft delete support)
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

-- Create index on (project_id, position) for efficient Kanban column ordering
CREATE INDEX IF NOT EXISTS idx_tasks_project_position ON tasks(project_id, position);

-- Create index on deleted_at for soft delete queries
CREATE INDEX IF NOT EXISTS idx_tasks_deleted_at ON tasks(deleted_at);

-- Add comment for position field to clarify its purpose
COMMENT ON COLUMN tasks.position IS 'Integer position for ordering within Kanban columns (0-indexed)';
