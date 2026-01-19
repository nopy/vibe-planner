-- Create sessions table for Phase 5.1 - OpenCode session tracking

CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    prompt TEXT,
    output TEXT,
    error TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    duration_ms BIGINT DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_sessions_task_id ON sessions(task_id);
CREATE INDEX IF NOT EXISTS idx_sessions_project_id ON sessions(project_id);
CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
CREATE INDEX IF NOT EXISTS idx_sessions_deleted_at ON sessions(deleted_at);

-- Create compound index for active sessions query (project + status)
CREATE INDEX IF NOT EXISTS idx_sessions_project_status ON sessions(project_id, status) WHERE deleted_at IS NULL;

-- Add comments for documentation
COMMENT ON TABLE sessions IS 'OpenCode execution sessions for tasks (Phase 5.1)';
COMMENT ON COLUMN sessions.task_id IS 'Reference to the task being executed';
COMMENT ON COLUMN sessions.project_id IS 'Reference to the project (for quick filtering)';
COMMENT ON COLUMN sessions.status IS 'Session status: pending, running, completed, failed, cancelled';
COMMENT ON COLUMN sessions.prompt IS 'User prompt sent to OpenCode';
COMMENT ON COLUMN sessions.output IS 'Accumulated output from OpenCode execution';
COMMENT ON COLUMN sessions.error IS 'Error message if session failed';
COMMENT ON COLUMN sessions.duration_ms IS 'Execution duration in milliseconds';
