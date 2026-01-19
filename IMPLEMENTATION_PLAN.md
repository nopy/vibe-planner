# OpenCode Project Manager - Comprehensive Implementation Plan

## Project Overview

A web application for managing software projects using OpenCode AI agent coding.

**Tech Stack:**
- Backend: Go 1.21+ (Gin framework)
- Frontend: React 18+ with TypeScript
- Database: PostgreSQL 15+
- Orchestration: Kubernetes (kind for local)
- Authentication: Keycloak (OIDC)
- Container Registry: registry.legal-suite.com
- AI Model: GPT-4o mini

**Team Size:** 3 developers
**MVP Scope:** Core features, can defer advanced optional features

---

## Architecture Summary

```
┌──────────────────────────────────────────────────────────────┐
│                  React SPA Frontend                          │
│              (Vite, TypeScript, Tailwind)                    │
└─────────────────────┬──────────────────────────────────────┘
                      │ HTTPS + JWT
┌─────────────────────▼──────────────────────────────────────┐
│           Kubernetes Cluster (kind)                         │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ Main Controller Pod                                  │  │
│  │ ├─ Go API Server (Gin) :8080                        │  │
│  │ ├─ WebSocket Handler                               │  │
│  │ └─ Pod Lifecycle Manager                           │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ PostgreSQL StatefulSet                              │  │
│  │ (persistent storage)                                │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ Per-Project Pod (on-demand)                         │  │
│  │ ├─ OpenCode Server                                 │  │
│  │ ├─ File Browser Sidecar (Go) :3001                │  │
│  │ └─ Session Proxy Sidecar (Go) :3002               │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│                  Keycloak (local dev)                        │
│            (OIDC Provider for authentication)                │
└──────────────────────────────────────────────────────────────┘
```

---

## Quick Start Commands

```bash
# Clone and navigate
git clone <repo-url> opencode-project-manager
cd opencode-project-manager

# Install dependencies
make setup

# Start local development environment
make dev

# Run tests
make test

# Build Docker images
make build-images

# Deploy to kind
make deploy-kind
```

---

## Phase 1: Foundation (Weeks 1-2)

### Objectives
- ✅ Go backend structure with Gin
- ✅ React frontend with Vite
- ✅ PostgreSQL local setup
- ✅ OIDC authentication flow (Keycloak)
- ✅ JWT middleware
- ✅ Basic UI shell

### Deliverables
1. Working auth flow (Keycloak → JWT → protected routes)
2. `/api/auth/me` endpoint
3. Login/logout pages in React
4. Docker-compose with all services
5. Database migration framework

### Key Files to Create
```
backend/
├── cmd/api/main.go
├── internal/config/config.go
├── internal/api/auth.go
├── internal/middleware/auth.go
├── internal/model/user.go
├── internal/repository/user.go
├── go.mod
└── Dockerfile

frontend/
├── src/pages/Login.tsx
├── src/pages/OidcCallback.tsx
├── src/contexts/AuthContext.tsx
├── src/hooks/useAuth.ts
├── src/services/api.ts
├── src/App.tsx
├── package.json
└── Dockerfile

db/
├── migrations/001_init.sql
└── docker/postgres-init.sh

docker-compose.yml
Makefile
```

---

## Phase 2: Project Management (Weeks 3-4)

### Objectives
- ✅ Project CRUD endpoints
- ✅ Kubernetes pod creation/deletion
- ✅ Project UI with create/delete
- ✅ Real-time status via WebSocket

### Key Endpoints
- `POST /api/projects` → creates project + spawns pod
- `GET /api/projects` → list projects
- `GET /api/projects/:id` → get project details
- `DELETE /api/projects/:id` → cleanup pod and archive project
- `WebSocket /ws/projects/:id/status` → real-time pod status

### Key Files
```
backend/
├── internal/api/projects.go
├── internal/service/project.go
├── internal/service/kubernetes.go
├── internal/model/project.go
├── internal/repository/project.go
└── db/migrations/002_projects.sql

frontend/
├── src/pages/Projects.tsx
├── src/components/Projects/ProjectList.tsx
├── src/components/Projects/CreateProjectModal.tsx
├── src/components/Projects/ProjectCard.tsx
└── src/hooks/useProject.ts
```

---

## Phase 3: Task Management & Kanban (Weeks 5-6)

### Objectives
- ✅ Task CRUD with state machine
- ✅ Kanban board UI with drag-drop
- ✅ Task detail panel
- ✅ Real-time task updates

### Task States
```
TODO → IN_PROGRESS → AI_REVIEW → HUMAN_REVIEW → DONE
```

### Key Files
```
backend/
├── internal/api/tasks.go
├── internal/service/task.go
├── internal/model/task.go
├── internal/repository/task.go
└── db/migrations/003_tasks.sql

frontend/
├── src/components/Kanban/KanbanBoard.tsx
├── src/components/Kanban/KanbanColumn.tsx
├── src/components/Kanban/TaskCard.tsx
├── src/components/Kanban/TaskDetailPanel.tsx
└── src/hooks/useTasks.ts
```

---

## Phase 4: File Explorer (Weeks 7-8)

### Objectives
- ✅ File browser sidecar (Go)
- ✅ File tree component
- ✅ Monaco editor integration
- ✅ Multi-file support with tabs

### Sidecars
```
sidecars/file-browser/
├── cmd/main.go
├── internal/handler/files.go
├── internal/service/file.go
└── Dockerfile

sidecars/session-proxy/
├── cmd/main.go
├── internal/handler/session.go
├── internal/service/opencode.go
└── Dockerfile
```

### Frontend Components
```
frontend/
├── src/components/Explorer/FileExplorer.tsx
├── src/components/Explorer/FileTree.tsx
├── src/components/Explorer/TreeNode.tsx
├── src/components/Explorer/MonacoEditor.tsx
├── src/components/Explorer/EditorTabs.tsx
└── src/hooks/useFiles.ts
```

---

## Phase 5: OpenCode Integration (Weeks 9-10)

### Objectives
- Execute tasks via OpenCode AI agent
- Stream real-time output to frontend via SSE
- Task state transitions based on session lifecycle events
- Session management and error handling with retry logic

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│  Frontend (React)                                               │
│  ├─ TaskCard "Execute" button                                  │
│  ├─ ExecutionPanel (streaming output view)                     │
│  └─ ExecutionHistory (past runs)                               │
└─────────────────┬───────────────────────────────────────────────┘
                  │ HTTP/SSE
┌─────────────────▼───────────────────────────────────────────────┐
│  Backend API (Go)                                               │
│  ├─ POST /api/projects/:id/tasks/:taskId/execute               │
│  ├─ GET  /api/projects/:id/tasks/:taskId/output (SSE stream)   │
│  ├─ POST /api/projects/:id/tasks/:taskId/stop                  │
│  └─ GET  /api/projects/:id/sessions (list active sessions)     │
└─────────────────┬───────────────────────────────────────────────┘
                  │ HTTP (internal)
┌─────────────────▼───────────────────────────────────────────────┐
│  OpenCode Server Sidecar (:3003)                                │
│  ├─ POST /sessions (start new session)                         │
│  ├─ GET  /sessions/:id/stream (SSE output)                     │
│  ├─ POST /sessions/:id/stop (terminate session)                │
│  └─ GET  /sessions/:id/status (session health)                 │
└─────────────────┬───────────────────────────────────────────────┘
                  │ reads/writes
┌─────────────────▼───────────────────────────────────────────────┐
│  Project Workspace (PVC /workspace)                             │
│  - Source code files (managed by file-browser)                  │
│  - OpenCode configuration (.opencode/config.json)               │
│  - Session history and logs (.opencode/sessions/)               │
└─────────────────────────────────────────────────────────────────┘
```

### Backend Implementation

#### 5.1 Session Management Service
**Status:** 📋 Planned

**Tasks:**
- Session Model (`internal/model/session.go`)
  - Fields: ID, TaskID, ProjectID, Status, StartedAt, CompletedAt, Error
  - Status enum: PENDING, RUNNING, COMPLETED, FAILED, CANCELLED
  - GORM relationships to Task and Project

- Session Repository (`internal/repository/session_repository.go`)
  - CreateSession(session *Session) error
  - GetSessionByID(id uuid.UUID) (*Session, error)
  - GetActiveSessionsForProject(projectID uuid.UUID) ([]*Session, error)
  - UpdateSessionStatus(id uuid.UUID, status SessionStatus) error

- Session Service (`internal/service/session_service.go`)
  - StartSession(taskID uuid.UUID, prompt string) (*Session, error)
  - StopSession(sessionID uuid.UUID) error
  - GetSessionStatus(sessionID uuid.UUID) (*Session, error)
  - CallOpenCodeAPI(podIP string, endpoint string) (response, error)

**Success Criteria:**
- Session CRUD operations working
- Can communicate with OpenCode sidecar via HTTP
- Session lifecycle tracked in database
- At least 20 unit tests passing

#### 5.2 Task Execution API
**Status:** 📋 Planned

**Tasks:**
- Execute Endpoint (`POST /api/projects/:id/tasks/:taskId/execute`)
  - Extract project pod IP from Kubernetes API
  - Create session via SessionService
  - Start OpenCode session on sidecar
  - Update task status to IN_PROGRESS
  - Return session ID to client

- Output Stream Endpoint (`GET /api/projects/:id/tasks/:taskId/output`)
  - Server-Sent Events (SSE) endpoint
  - Proxy SSE stream from OpenCode sidecar
  - Forward events to frontend in real-time
  - Handle connection cleanup on close

- Stop Execution (`POST /api/projects/:id/tasks/:taskId/stop`)
  - Call OpenCode sidecar stop endpoint
  - Update session status to CANCELLED
  - Update task status back to TODO

**Success Criteria:**
- Can start OpenCode session from API call
- SSE stream proxies output in real-time
- Can stop running sessions
- Task state transitions working (TODO → IN_PROGRESS)
- At least 15 integration tests passing

#### 5.3 OpenCode Sidecar Integration
**Status:** 📋 Planned

**Tasks:**
- Pod Template Update (`internal/service/pod_template.go`)
  - Add fourth container (opencode-server)
  - Mount workspace PVC to /workspace
  - Set environment variables (WORKSPACE_DIR, PORT=3003)
  - Configure resource limits (CPU: 200m-500m, Memory: 256Mi-512Mi)
  - Add liveness/readiness probes

- Health Check Configuration
  - Liveness: HTTP GET /health on port 3003
  - Readiness: HTTP GET /ready on port 3003
  - Initial delay: 15s (OpenCode server startup time)

**Success Criteria:**
- Project pods spawn with 4 containers (main + file-browser + session-proxy + opencode-server)
- OpenCode sidecar starts successfully and responds to health checks
- Workspace volume accessible to all containers
- All backend tests still passing (no regressions)

### Frontend Implementation

#### 5.4 Execute Task UI
**Status:** 📋 Planned

**Tasks:**
- TaskCard Updates (`components/Kanban/TaskCard.tsx`)
  - Add "Execute" button (lightning bolt icon)
  - Show execution status badge (running/completed/failed)
  - Disable button when task is already running

- Task Detail Panel (`components/Kanban/TaskDetailPanel.tsx`)
  - Add "Execute Task" button in header
  - Show execution history section
  - Display current session status

- API Client (`services/api.ts`)
  - executeTask(projectId, taskId) → Promise<{ sessionId: string }>
  - stopTaskExecution(projectId, taskId, sessionId) → Promise<void>

**Success Criteria:**
- "Execute" button visible on all task cards
- Button disabled when execution in progress
- Visual feedback for execution state changes
- API client methods implemented and typed

#### 5.5 Real-time Output Streaming
**Status:** 📋 Planned

**Tasks:**
- Execution Output Panel (`components/Execution/ExecutionOutputPanel.tsx`)
  - Terminal-like UI with dark theme
  - Auto-scroll to bottom on new output
  - Syntax highlighting for code blocks
  - Show timestamps for each message

- SSE Hook (`hooks/useTaskExecution.ts`)
  - useTaskExecution(projectId, taskId, sessionId)
  - Connect to /api/projects/:id/tasks/:taskId/output SSE endpoint
  - Handle connection errors with retry logic
  - Parse SSE events and update state
  - Clean up EventSource on unmount

- Event Types
  - `output`: Regular console output
  - `error`: Error messages (red text)
  - `status`: Session status changes (pending→running→completed)
  - `done`: Session completed successfully

**Success Criteria:**
- SSE connection established successfully
- Output streams in real-time
- Auto-scroll works smoothly
- Connection cleanup on component unmount
- Graceful error handling with retry

#### 5.6 Execution History
**Status:** 📋 Planned

**Tasks:**
- Execution History List (`components/Execution/ExecutionHistory.tsx`)
  - List all sessions for a task (newest first)
  - Show: timestamp, duration, status badge, output preview (first 100 chars)
  - Expand/collapse full output logs

- API Endpoint (Backend)
  - GET /api/projects/:id/tasks/:taskId/sessions
  - Returns: Array of sessions with metadata and output summaries

- API Client (Frontend)
  - getTaskExecutionHistory(projectId, taskId) → Promise<Session[]>

**Success Criteria:**
- Can view past execution history
- Session metadata displayed correctly
- Can expand/collapse full logs
- Sorted by most recent first

### Testing & Verification

#### 5.7 Integration Testing
**Status:** 📋 Planned

**Tasks:**
- Backend Integration Tests (`internal/api/tasks_execution_integration_test.go`)
  - Test: Create project → create task → execute task → verify session created
  - Test: Stop running session → verify session cancelled → task reset to TODO
  - Test: OpenCode sidecar unavailable → verify graceful error handling
  - Test: Concurrent execution attempts → verify second request rejected

- Manual E2E Testing Checklist:
  - Create project and wait for pod to be Running
  - Create task with description "Add a README file"
  - Click "Execute" button on task card
  - Verify task status changes to IN_PROGRESS
  - Verify execution output streams in real-time
  - Wait for session completion
  - Verify task state transitions to AI_REVIEW
  - Check execution history shows completed session
  - Verify README file created in workspace (via File Explorer)

**Success Criteria:**
- At least 10 integration tests passing
- E2E workflow verified manually
- Error handling tested and working

### Key Files to Create/Modify

**Backend:**
```
backend/
├── internal/model/session.go                              # NEW
├── internal/repository/session_repository.go              # NEW
├── internal/repository/session_repository_test.go         # NEW
├── internal/service/session_service.go                    # NEW
├── internal/service/session_service_test.go               # NEW
├── internal/service/pod_template.go                       # MODIFY (add 4th container)
├── internal/api/tasks.go                                  # MODIFY (add 3 endpoints)
├── internal/api/tasks_execution_test.go                   # NEW
└── internal/api/tasks_execution_integration_test.go       # NEW
```

**Frontend:**
```
frontend/
├── src/components/Kanban/TaskCard.tsx                     # MODIFY (add Execute button)
├── src/components/Kanban/TaskDetailPanel.tsx              # MODIFY (add Execute section)
├── src/components/Execution/ExecutionOutputPanel.tsx      # NEW
├── src/components/Execution/ExecutionHistory.tsx          # NEW
├── src/hooks/useTaskExecution.ts                          # NEW
├── src/services/api.ts                                    # MODIFY (add execution methods)
└── src/types/index.ts                                     # MODIFY (add Session types)
```

### Success Metrics

**Phase 5 is complete when:**

1. **Backend:**
   - Session model, repository, service implemented (20+ tests)
   - Task execution API endpoints working (15+ tests)
   - OpenCode sidecar added to pod template
   - All 4 containers starting successfully in project pods
   - SSE streaming functional

2. **Frontend:**
   - "Execute" button on task cards
   - Real-time output streaming with SSE
   - Execution history display
   - All TypeScript types defined
   - No console errors

3. **Integration:**
   - Can execute task end-to-end
   - Output streams in real-time
   - Task state transitions working
   - Can view execution history
   - OpenCode session logs persisted

4. **Testing:**
   - 35+ new unit tests passing (backend)
   - 10+ integration tests passing
   - Manual E2E checklist completed
   - All existing tests still passing (no regressions)

### Dependencies

**Required Before Starting:**
- ✅ Phase 4 complete (file explorer needed to view OpenCode output files)
- ✅ Phase 3 complete (task management and state machine)
- ✅ Phase 2 complete (Kubernetes pod lifecycle)

**External Dependencies:**
- OpenCode server Docker image (verify availability in registry)
- SSE support in Gin framework (use `gin.Context.Stream()`)
- EventSource API (browser native, no additional libraries)

### Deferred Items (Phase 5+)

Items not critical for MVP but valuable for future:

1. **Session Persistence:**
   - Store full session output logs in database
   - Compress old logs after 30 days
   - Add pagination for execution history

2. **Execution Queueing:**
   - Queue tasks when OpenCode server is busy
   - Show queue position to user
   - Automatic retry on transient failures

3. **Multi-session Support:**
   - Allow multiple OpenCode sessions per project
   - Resource limits to prevent overload
   - Priority queueing for tasks

4. **Advanced Monitoring:**
   - Grafana dashboards for session metrics
   - Alert on failed sessions
   - Track token usage per session

### Technical Notes

**OpenCode Sidecar Configuration:**
- Port: 3003 (internal to pod)
- Resource Limits: 200m-500m CPU, 256Mi-512Mi memory
- Workspace: /workspace (shared PVC with main container and file-browser)
- Health Check: HTTP GET /health every 10s

**SSE vs WebSocket:**
- Using SSE (Server-Sent Events) for output streaming
- Simpler than WebSocket for one-way server→client data flow
- Native browser support via EventSource API
- Automatic reconnection on disconnect

**Task State Transitions:**
- Execute task: TODO → IN_PROGRESS
- Session completes: IN_PROGRESS → AI_REVIEW
- Session fails: IN_PROGRESS → TODO (with error logged)
- Human reviews: AI_REVIEW → HUMAN_REVIEW or DONE

---

## Phase 6: OpenCode Config (Weeks 11-12)

### Objectives
- ✅ Config CRUD with versioning
- ✅ Advanced config UI (model, provider, tools)
- ✅ Config history and rollback

### Key Files
```
backend/
├── internal/api/config.go
├── internal/service/config.go
├── internal/model/opencode_config.go
├── internal/repository/config.go
└── db/migrations/004_opencode_configs.sql

frontend/
├── src/components/Config/ConfigPanel.tsx
├── src/components/Config/ModelSelector.tsx
├── src/components/Config/ProviderConfig.tsx
├── src/components/Config/ToolsManagement.tsx
└── src/hooks/useConfig.ts
```

---

## Phase 7: Two-Way Interaction (Weeks 13-14)

### Objectives
- ✅ User feedback during execution
- ✅ Agent response handling
- ✅ Interaction history

### Key Files
```
backend/
├── internal/model/interaction.go
├── internal/repository/interaction.go
├── internal/api/interactions.go
└── db/migrations/005_interactions.sql

frontend/
├── src/components/Kanban/InteractionForm.tsx
└── src/hooks/useInteractions.ts
```

---

## Phase 8: Kubernetes & Deployment (Weeks 15-16)

### Objectives
- ✅ Production-ready manifests
- ✅ Kind cluster setup
- ✅ Health checks
- ✅ Scaling considerations

### Key Files
```
k8s/
├── base/
│   ├── namespace.yaml
│   ├── configmap.yaml
│   ├── secrets.yaml
│   ├── postgres-statefulset.yaml
│   ├── controller-deployment.yaml
│   ├── service.yaml
│   ├── ingress.yaml
│   └── rbac.yaml
├── overlays/
│   ├── dev/kustomization.yaml
│   └── prod/kustomization.yaml
└── kind-config.yaml
```

---

## Phase 9: Testing & Docs (Weeks 17-18)

### Objectives
- ✅ Unit tests (>70% coverage)
- ✅ Integration tests
- ✅ E2E tests
- ✅ API documentation
- ✅ Deployment guide

### Key Tests
```
backend/
├── internal/service/project_test.go
├── internal/api/auth_test.go
└── ...

frontend/
├── src/components/Kanban/__tests__/KanbanBoard.test.tsx
└── ...

e2e/
├── tests/auth.cy.ts
├── tests/projects.cy.ts
└── tests/tasks.cy.ts
```

---

## Phase 10: Polish & Optimization (Weeks 19-20)

### Objectives
- ✅ Performance tuning
- ✅ Security hardening
- ✅ Error handling improvements
- ✅ UX polish

---

## Development Workflow

### Local Development

```bash
# Terminal 1: Start services (postgres, keycloak, redis)
make dev-services

# Terminal 2: Start Go backend
make backend-dev

# Terminal 3: Start React frontend
make frontend-dev

# Terminal 4: Run tests
make test-watch
```

### Git Workflow

```bash
# Create feature branch
git checkout -b feature/kanban-board

# Make changes, commit regularly
git add .
git commit -m "feat: add drag-drop to kanban board"

# Push and create PR
git push origin feature/kanban-board

# PR review, merge when approved
# CI/CD runs tests automatically
```

### Code Organization

**Backend (Go):**
- API layer (handlers) → Service layer (business logic) → Repository layer (DB access)
- Clear separation of concerns
- Interface-based design for testability

**Frontend (React):**
- Component-based architecture
- Context for global state (auth, project)
- Hooks for data fetching (SWR)
- Tailwind CSS for styling

**Database:**
- SQL migrations for schema versioning
- Liquibase or golang-migrate for management
- Connection pooling configured

---

## Key Configuration Values

### Environment Variables

**Backend (.env)**
```
# Database
DATABASE_URL=postgres://user:pass@localhost:5432/opencode_dev

# OIDC (Keycloak)
OIDC_ISSUER=http://localhost:8081/realms/opencode
OIDC_CLIENT_ID=opencode-app
OIDC_CLIENT_SECRET=secret

# JWT
JWT_SECRET=local-dev-secret
JWT_EXPIRY=3600

# Kubernetes (for pod creation)
KUBECONFIG=/path/to/kubeconfig
K8S_NAMESPACE=opencode

# OpenCode
OPENCODE_INSTALL_PATH=/usr/local/bin/opencode

# Server
PORT=8080
LOG_LEVEL=debug
ENVIRONMENT=development
```

**Frontend (.env)**
```
VITE_API_URL=http://localhost:8080
VITE_OIDC_AUTHORITY=http://localhost:8081/realms/opencode
VITE_OIDC_CLIENT_ID=opencode-app
VITE_OIDC_REDIRECT_URI=http://localhost:5173/auth/callback
```

---

## Database Migrations

Location: `db/migrations/`

**Structure:**
- `001_init.sql` - Users, projects, tasks tables
- `002_opencode_configs.sql` - Config versioning
- `003_sessions.sql` - OpenCode sessions
- `004_interactions.sql` - Two-way interaction
- `005_audit_log.sql` - Audit trail

Each migration is idempotent and reversible.

---

## API Documentation

Base URL: `http://localhost:8080/api`

### Authentication Endpoints

```
POST   /auth/oidc/login         - Get OIDC login URL
POST   /auth/oidc/callback      - Handle OIDC callback
GET    /auth/me                 - Get current user
POST   /auth/logout             - Logout user
POST   /auth/refresh            - Refresh JWT token
```

### Project Endpoints

```
GET    /projects                - List projects
POST   /projects                - Create project
GET    /projects/:id            - Get project details
PATCH  /projects/:id            - Update project
DELETE /projects/:id            - Delete project
```

### Task Endpoints

```
GET    /projects/:id/tasks                  - List tasks
POST   /projects/:id/tasks                  - Create task
GET    /projects/:id/tasks/:taskId          - Get task details
PATCH  /projects/:id/tasks/:taskId          - Update task
DELETE /projects/:id/tasks/:taskId          - Delete task
POST   /projects/:id/tasks/:taskId/execute  - Execute task (spawn OpenCode)
GET    /projects/:id/tasks/:taskId/output   - Stream task output (SSE)
```

### File Endpoints

```
GET    /projects/:id/files/tree             - Get directory tree
GET    /projects/:id/files/content          - Get file content
POST   /projects/:id/files/write            - Write file
DELETE /projects/:id/files                  - Delete file
POST   /projects/:id/files/mkdir            - Create directory
```

### Config Endpoints

```
GET    /projects/:id/config                 - Get active config
POST   /projects/:id/config                 - Create/update config
GET    /projects/:id/config/versions        - List config versions
POST   /projects/:id/config/rollback        - Rollback to version
```

### WebSocket Connections

```
WS     /ws/projects/:id/status              - Project pod status
WS     /ws/tasks/:id/output                 - Task output stream
```

---

## Docker Images

### Images to Build and Push

```
registry.legal-suite.com/opencode/backend:latest
registry.legal-suite.com/opencode/frontend:latest
registry.legal-suite.com/opencode/file-browser-sidecar:latest
registry.legal-suite.com/opencode/session-proxy-sidecar:latest
```

### Build Script

```bash
#!/bin/bash
REGISTRY=registry.legal-suite.com/opencode
VERSION=${1:-latest}

# Build backend
docker build -t $REGISTRY/backend:$VERSION ./backend
docker push $REGISTRY/backend:$VERSION

# Build frontend
docker build -t $REGISTRY/frontend:$VERSION ./frontend
docker push $REGISTRY/frontend:$VERSION

# Build sidecars
docker build -t $REGISTRY/file-browser-sidecar:$VERSION ./sidecars/file-browser
docker push $REGISTRY/file-browser-sidecar:$VERSION

docker build -t $REGISTRY/session-proxy-sidecar:$VERSION ./sidecars/session-proxy
docker push $REGISTRY/session-proxy-sidecar:$VERSION
```

---

## Kubernetes Deployment (Kind)

### Setup Kind Cluster

```bash
# Create cluster with config
kind create cluster --config k8s/kind-config.yaml --name opencode-dev

# Verify
kubectl cluster-info
kubectl get nodes
```

### Deploy to Kind

```bash
# Create namespace
kubectl create namespace opencode

# Apply secrets
kubectl apply -f k8s/base/secrets.yaml

# Apply base manifests
kubectl apply -k k8s/base/

# Check status
kubectl get pods -n opencode
kubectl get svc -n opencode

# Port forward to access locally
kubectl port-forward -n opencode svc/opencode-controller 8080:80
```

### Local Database Access

```bash
# Get postgres pod
kubectl get pods -n opencode -l app=postgres

# Port forward
kubectl port-forward -n opencode pod/postgres-0 5432:5432

# Connect
psql postgres://user:password@localhost:5432/opencode_prod
```

---

## Security Considerations

### MVP (Acceptable)
- [x] HTTPS in production (via Ingress)
- [x] OIDC authentication
- [x] JWT token validation
- [x] Database credentials in Secrets
- [x] RBAC for K8s access
- [x] Encrypted config storage (credentials)

### Future Hardening
- [ ] Network policies
- [ ] Pod security policies
- [ ] Audit logging
- [ ] Rate limiting
- [ ] CSRF protection
- [ ] Input validation & sanitization
- [ ] SQL injection prevention (using parameterized queries)

---

## Testing Strategy

### Unit Tests
- Service layer logic
- Model validation
- Utility functions
- Target: 70%+ coverage

### Integration Tests
- API endpoints
- Database operations
- File system operations
- OIDC flow

### E2E Tests
- User login workflow
- Create project workflow
- Execute task workflow
- File browsing workflow

### Running Tests

```bash
# Backend
cd backend
go test ./... -v -cover

# Frontend
cd frontend
npm test

# E2E (after services are running)
npx cypress run
```

---

## Monitoring & Logging

### Logs

- Backend: stdout (captured by K8s)
- Frontend: browser console, error tracking (Sentry optional)
- Containers: docker logs

### Metrics (Future)

- Prometheus endpoints at `/metrics`
- Track:
  - API response times
  - Task execution times
  - Pod creation/deletion times
  - Database query times

### Health Checks

```
GET /healthz        - Simple health check
GET /ready          - Ready to accept traffic
GET /live           - Pod alive check
```

---

## Troubleshooting

### Common Issues

**Pod Fails to Start**
```bash
kubectl describe pod <pod-name> -n opencode
kubectl logs <pod-name> -n opencode
```

**Database Connection Error**
```bash
kubectl port-forward -n opencode pod/postgres-0 5432:5432
psql postgres://user:pass@localhost:5432/opencode_prod
```

**OIDC Token Validation Fails**
- Check OIDC_ISSUER URL matches Keycloak configuration
- Verify OIDC_CLIENT_ID and OIDC_CLIENT_SECRET
- Check JWT_SECRET is consistent

**File Browser Sidecar Not Accessible**
- Check sidecar pod is running: `kubectl get pods`
- Verify service discovery: `kubectl get svc`
- Check logs: `kubectl logs <sidecar-pod>`

---

## Team Collaboration

### Code Review Checklist
- [ ] Tests pass
- [ ] Code follows style guide
- [ ] No security issues
- [ ] Documentation updated
- [ ] PR description clear

### Commit Message Format
```
feat: add kanban board drag-drop
fix: correct file path traversal
docs: update API documentation
test: add task execution tests
refactor: extract service layer
chore: update dependencies
```

### Documentation Updates
- Update DEVELOPMENT.md for new setup steps
- Update API.md for new endpoints
- Update ARCHITECTURE.md for structural changes
- Add code comments for complex logic

---

## Next Steps

1. **Review this plan** with the team
2. **Setup development environment** (docker, kind, etc.)
3. **Create git repository** with structure
4. **Begin Phase 1** implementation
5. **Weekly sync** to review progress and adjust

---

## Success Metrics

✅ Week 2:  Auth flow working, basic UI shell
✅ Week 4:  Projects created/deleted, pods spawn
✅ Week 6:  Kanban board fully functional
✅ Week 8:  File explorer with Monaco editor
✅ Week 10: OpenCode integration working
✅ Week 12: Config management UI complete
✅ Week 14: Two-way interactions working
✅ Week 16: K8s deployment working
✅ Week 18: Tests and docs complete
✅ Week 20: Production-ready

---

## References

- OpenCode: https://github.com/anomalyco/opencode
- Keycloak: https://www.keycloak.org/
- Kind: https://kind.sigs.k8s.io/
- Gin: https://gin-gonic.com/
- React: https://react.dev/
- Kubernetes: https://kubernetes.io/

