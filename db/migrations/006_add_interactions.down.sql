-- Rollback interactions table for Phase 7.1

-- Drop trigger first
DROP TRIGGER IF EXISTS trigger_update_interactions_updated_at ON interactions;

-- Drop trigger function
DROP FUNCTION IF EXISTS update_interactions_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_interactions_task_created;
DROP INDEX IF EXISTS idx_interactions_created_at;
DROP INDEX IF EXISTS idx_interactions_session_id;
DROP INDEX IF EXISTS idx_interactions_task_id;

-- Drop the table
DROP TABLE IF EXISTS interactions;
