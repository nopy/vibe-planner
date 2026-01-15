DROP TRIGGER IF EXISTS update_opencode_sessions_updated_at ON opencode_sessions;
DROP TRIGGER IF EXISTS update_tasks_updated_at ON tasks;
DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS task_events;
DROP TABLE IF EXISTS interactions;
DROP TABLE IF EXISTS opencode_sessions;
DROP TABLE IF EXISTS opencode_configs;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS users;

DROP EXTENSION IF EXISTS "pgcrypto";
DROP EXTENSION IF EXISTS "uuid-ossp";
