-- Add session persistence fields for crash recovery
-- These fields enable the sidecar to resume sessions after restart

ALTER TABLE sessions
ADD COLUMN IF NOT EXISTS remote_session_id VARCHAR(255),
ADD COLUMN IF NOT EXISTS last_event_id VARCHAR(255),
ADD COLUMN IF NOT EXISTS prompt_request_id VARCHAR(255);

-- Create index on remote_session_id for lookups during recovery
CREATE INDEX IF NOT EXISTS idx_sessions_remote_session_id ON sessions(remote_session_id);

-- Create index on prompt_request_id for idempotency checks
CREATE INDEX IF NOT EXISTS idx_sessions_prompt_request_id ON sessions(prompt_request_id);

-- Add comments for clarity
COMMENT ON COLUMN sessions.remote_session_id IS 'OpenCode SDK session ID for sidecar reconnection';
COMMENT ON COLUMN sessions.last_event_id IS 'Last processed SSE event ID for replay';
COMMENT ON COLUMN sessions.prompt_request_id IS 'Idempotency key to prevent duplicate prompt execution';
