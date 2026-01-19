-- Rollback: Remove sessions table added in 004_add_sessions.up.sql

-- Drop indexes
DROP INDEX IF EXISTS idx_sessions_project_status;
DROP INDEX IF EXISTS idx_sessions_deleted_at;
DROP INDEX IF EXISTS idx_sessions_status;
DROP INDEX IF EXISTS idx_sessions_project_id;
DROP INDEX IF EXISTS idx_sessions_task_id;

-- Drop table
DROP TABLE IF EXISTS sessions;
