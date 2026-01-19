# OpenCode Server Sidecar

This container runs the OpenCode AI agent runtime within project pods.

## Purpose
- Executes AI-powered coding tasks
- Provides REST/WebSocket API for task execution
- Manages workspace files and git operations

## Configuration
- **Port:** 3000
- **Workspace:** /workspace (shared PVC with other containers)
- **Health Check:** HTTP GET /health

## Environment Variables
- `WORKSPACE_DIR` - Workspace directory path (default: /workspace)
- `PORT` - Server port (default: 3000)
- `PROJECT_ID` - UUID of the associated project

## TODO
This is currently a placeholder. Needs:
1. Actual OpenCode CLI/server installation
2. Proper health check endpoint implementation
3. WebSocket support for real-time task execution
4. Integration with session-proxy sidecar
