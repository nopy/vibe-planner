-- Rollback session persistence fields

DROP INDEX IF EXISTS idx_sessions_remote_session_id;
DROP INDEX IF EXISTS idx_sessions_prompt_request_id;

ALTER TABLE sessions
DROP COLUMN IF EXISTS remote_session_id,
DROP COLUMN IF EXISTS last_event_id,
DROP COLUMN IF EXISTS prompt_request_id;
