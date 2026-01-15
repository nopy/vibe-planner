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
- ✅ Execute tasks via OpenCode
- ✅ Stream output to frontend
- ✅ Task state transitions based on session events
- ✅ Error handling

### Key Endpoints
- `POST /api/projects/:id/tasks/:taskId/execute` → spawn session
- `GET /api/projects/:id/tasks/:taskId/output` → SSE stream
- `WebSocket /ws/tasks/:id/output` → real-time output

### Key Files
```
backend/
├── internal/service/opencode.go
├── internal/api/tasks.go (extend)
└── internal/model/session.go
```

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

