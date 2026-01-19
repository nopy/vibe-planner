-- Add interactions table for two-way communication between users and AI agents
CREATE TABLE IF NOT EXISTS interactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    session_id UUID REFERENCES sessions(id) ON DELETE SET NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message_type VARCHAR(50) NOT NULL,
    content TEXT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT check_message_type CHECK (message_type IN ('user_message', 'agent_response', 'system_notification'))
);

-- Indexes for efficient queries
CREATE INDEX idx_interactions_task_id ON interactions(task_id);
CREATE INDEX idx_interactions_session_id ON interactions(session_id) WHERE session_id IS NOT NULL;
CREATE INDEX idx_interactions_created_at ON interactions(created_at DESC);
CREATE INDEX idx_interactions_task_created ON interactions(task_id, created_at DESC);

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_interactions_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_interactions_updated_at
    BEFORE UPDATE ON interactions
    FOR EACH ROW
    EXECUTE FUNCTION update_interactions_updated_at();
