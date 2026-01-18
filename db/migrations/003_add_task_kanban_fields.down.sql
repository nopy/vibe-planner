-- Rollback: Remove Kanban fields added in 003_add_task_kanban_fields.up.sql

-- Drop indexes
DROP INDEX IF EXISTS idx_tasks_deleted_at;
DROP INDEX IF EXISTS idx_tasks_project_position;

-- Remove columns (in reverse order)
ALTER TABLE tasks DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE tasks DROP COLUMN IF EXISTS assigned_to;
ALTER TABLE tasks DROP COLUMN IF EXISTS priority;
ALTER TABLE tasks DROP COLUMN IF EXISTS position;
