# OPENCODE PROJECT MANAGER - AGENT KNOWLEDGE BASE

**Generated:** 2026-01-19 08:50:00  
**Branch:** main  
**Project:** Go backend + React frontend + K8s orchestration
**Status:** ✅ Phase 1 Complete + ✅ Phase 2 Complete + ✅ Phase 3 Complete → 🚧 Phase 4.1-4.2 Complete (File Browser Sidecar with Real-time Watching)

---

## OVERVIEW

Multi-module monorepo: Go API server, React SPA, 2 Go sidecars (file-browser, session-proxy), K8s manifests. Project management system with AI-powered coding via OpenCode agents. All critical issues resolved. **Phase 1 + Phase 2 + Phase 3 COMPLETE** - Full project management with Kubernetes pod lifecycle, real-time WebSocket updates, task management with Kanban board. **Phase 4 IN PROGRESS** - File Explorer (4.1-4.2 Complete: File Browser Sidecar Setup with Real-time File Watching - 72 tests passing).

---

## STRUCTURE

```
.
├── backend/              # Go API (Gin, GORM, PostgreSQL)
├── frontend/             # React SPA (Vite, TypeScript, Tailwind)
├── sidecars/             # 2 Go services (file ops, session proxy)
├── k8s/                  # Kubernetes manifests (base + overlays)
├── db/migrations/        # SQL migrations (postgres)
├── scripts/              # Shell utilities (Keycloak, image build, kind deploy)
├── docker-compose.yml    # Local services (postgres, keycloak, redis)
└── Makefile              # Dev commands
```

---

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Start backend | `backend/cmd/api/main.go` | Entry point, port 8090 |
| Start frontend | `frontend/src/main.tsx` | Vite SPA entry, port 5173 |
| Auth handlers | `backend/internal/api/auth.go` | ✅ Fully implemented |
| Auth service | `backend/internal/service/auth_service.go` | ✅ OIDC + JWT |
| Auth middleware | `backend/internal/middleware/auth.go` | ✅ JWT validation |
| User repository | `backend/internal/repository/user_repository.go` | ✅ CRUD + upsert |
| Project handlers | `backend/internal/api/projects.go` | ✅ CRUD + WebSocket |
| Project service | `backend/internal/service/project_service.go` | ✅ Business logic |
| Project repository | `backend/internal/repository/project_repository.go` | ✅ CRUD operations |
| K8s service | `backend/internal/service/kubernetes_service.go` | ✅ Pod lifecycle |
| Task handlers | `backend/internal/api/tasks.go` | ✅ CRUD + move (Phase 3.4) |
| Task service | `backend/internal/service/task_service.go` | ✅ State machine + validation (Phase 3.3) |
| Task repository | `backend/internal/repository/task_repository.go` | ✅ CRUD + position (Phase 3.2) |
| Integration tests | `backend/internal/api/projects_integration_test.go` | ✅ E2E project lifecycle |
| Integration docs | `backend/INTEGRATION_TESTING.md` | ✅ Test setup guide |
| Models | `backend/internal/model/` | GORM structs (User, Project, Task) |
| Task model | `backend/internal/model/task.go` | ✅ Updated with Kanban fields (Phase 3.1) |
| DB schema | `db/migrations/001_init.up.sql` | All tables defined |
| DB migrations | `db/migrations/002_add_project_fields.up.sql` | Project fields |
| Task migration | `db/migrations/003_add_task_kanban_fields.up.sql` | ✅ Kanban fields (Phase 3.1) |
| React auth | `frontend/src/contexts/AuthContext.tsx` | ✅ Global auth state |
| React routes | `frontend/src/App.tsx` | ✅ Protected routes + Project pages |
| Login page | `frontend/src/pages/LoginPage.tsx` | ✅ OIDC flow |
| Callback page | `frontend/src/pages/OidcCallbackPage.tsx` | ✅ Token exchange |
| Project detail page | `frontend/src/pages/ProjectDetailPage.tsx` | ✅ Full metadata + real-time status |
| Project list | `frontend/src/components/Projects/ProjectList.tsx` | ✅ Grid layout with CRUD |
| Project card | `frontend/src/components/Projects/ProjectCard.tsx` | ✅ Status badges + delete |
| Create modal | `frontend/src/components/Projects/CreateProjectModal.tsx` | ✅ Form validation |
| Kanban board | `frontend/src/components/Kanban/KanbanBoard.tsx` | ✅ Drag-drop + optimistic updates (Phase 3.8) |
| Kanban column | `frontend/src/components/Kanban/KanbanColumn.tsx` | ✅ Droppable zones (Phase 3.8) |
| Task card | `frontend/src/components/Kanban/TaskCard.tsx` | ✅ Draggable with priority colors (Phase 3.8) |
| WebSocket hook | `frontend/src/hooks/useProjectStatus.ts` | ✅ Real-time pod status updates |
| App layout | `frontend/src/components/AppLayout.tsx` | ✅ Navigation header + menu |
| Types | `frontend/src/types/index.ts` | ✅ TS interfaces (User, Project, Task, etc.) |
| API client | `frontend/src/services/api.ts` | ✅ Axios client with JWT + Project & Task APIs |
| File browser | `sidecars/file-browser/cmd/main.go` | ✅ Port 3001 (Phase 4.1-4.2 Complete) |
| File service | `sidecars/file-browser/internal/service/file.go` | ✅ CRUD + path validation (Phase 4.1) |
| File watcher | `sidecars/file-browser/internal/service/watcher.go` | ✅ fsnotify + WebSocket broadcast (Phase 4.2) |
| File handlers | `sidecars/file-browser/internal/handler/files.go` | ✅ 6 HTTP endpoints (Phase 4.1) |
| Watch handler | `sidecars/file-browser/internal/handler/watch.go` | ✅ WebSocket /files/watch endpoint (Phase 4.2) |
| File service tests | `sidecars/file-browser/internal/service/file_test.go` | ✅ 24 tests passing (Phase 4.1) |
| Watcher tests | `sidecars/file-browser/internal/service/watcher_test.go` | ✅ 11 tests passing (Phase 4.2) |
| File handler tests | `sidecars/file-browser/internal/handler/files_test.go` | ✅ 34 tests passing (Phase 4.1) |
| Watch handler tests | `sidecars/file-browser/internal/handler/watch_test.go` | ✅ 5 tests passing (Phase 4.2) |
| Session proxy | `sidecars/session-proxy/cmd/main.go` | Port 3002 (Phase 5) |
| K8s base | `k8s/base/` | Namespace, ConfigMap, RBAC |
| K8s RBAC | `k8s/base/rbac.yaml` | ✅ ServiceAccount + Role |
| K8s PostgreSQL | `k8s/base/postgres.yaml` | ✅ StatefulSet + PVC |
| K8s dev | `k8s/overlays/dev/` | Dev environment patches |

---

## CRITICAL ISSUES ~~(Fix Before Development)~~ **[RESOLVED 2026-01-15]**

**All critical issues have been resolved. Project is ready for Phase 2 development.**

**1. Committed Binaries** ✅ FIXED
- ~~`backend/opencode-api`, `sidecars/*/file-browser`, `sidecars/*/session-proxy` are checked in~~
- **Resolution:** Deleted binaries + updated `.gitignore` to prevent future commits

**2. Multi-Module Without Workspace** ✅ FIXED
- ~~3 separate `go.mod` files (backend + 2 sidecars)~~
- **Resolution:** Created `go.work` at root with all 3 modules

**3. Missing Service/Repository Layers** ✅ FIXED
- ~~No `internal/service/` or `internal/repository/` in backend~~
- **Resolution:** Implemented for auth (AuthService, UserRepository) - pattern established for Phase 2

**4. Frontend Structure Mismatch** ✅ FIXED
- ~~README claims `src/components/`, `src/hooks/`, `src/contexts/` but they don't exist~~
- **Resolution:** Created all directories and populated with Phase 1 components

**5. Placeholder Module Path** ✅ FIXED
- ~~`github.com/yourusername/opencode-project-manager` in go.mod~~
- **Resolution:** Updated to `github.com/npinot/vibe/backend` and sidecars paths, all imports updated

**6. Keycloak DB Mismatch** ✅ FIXED
- ~~docker-compose: `POSTGRES_DB=opencode_dev` but Keycloak expects `keycloak` DB~~
- **Resolution:** Added `init-multiple-dbs.sh` script to create both databases

**7. Broken `make docker-push`** ✅ FIXED
- ~~Runs `docker-compose push` but compose only has postgres/keycloak/redis~~
- **Resolution:** Updated Makefile to call `build-images.sh` then push each image individually

**8. No CI Pipeline** ⚠️ DEFERRED
- No `.github/workflows/` found
- **Status:** Deferred to Phase 9 (Testing & Documentation)

---

## CONVENTIONS

### Go Backend

**Imports (3 groups, blank-separated):**
```go
import (
    "context"
    "log"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"

    "github.com/npinot/vibe/backend/internal/service"
)
```

**Error Handling:**
- Always explicit (no `_` discard)
- Wrap with context: `fmt.Errorf("failed to X: %w", err)`
- Log at top level (handlers), return up the stack

**Types:**
- IDs: `uuid.UUID`
- Timestamps: `time.Time`
- Optional fields: pointers
- Struct tags: `json:"field_name" gorm:"column:field_name"`

**Naming:**
- Interfaces: `UserRepository`, `Authenticator`
- Functions: `CreateUser`, `ValidateToken`
- Receivers: `u *User`, `ur *UserRepository`

### Frontend

**Imports (3 groups):**
```typescript
import { useState, useEffect } from 'react'

import axios from 'axios'
import { useParams } from 'react-router-dom'

import { ProjectCard } from '@/components/ProjectCard'
import { useAuth } from '@/hooks/useAuth'
import type { Project } from '@/types'
```

**Types:**
- Interfaces in `src/types/index.ts`
- Always type props/state/returns
- Optional: `description?: string`
- Use `interface` for objects, `type` for unions

**Components:**
- Functional only
- Hooks for state/effects
- Single responsibility

**Styling:**
- Tailwind utilities only
- Responsive: `sm:`, `md:`, `lg:`

---

## ANTI-PATTERNS (FORBIDDEN)

**Never:**
- `as any`, `@ts-ignore`, `@ts-expect-error`
- Commit secrets, API keys, credentials
- Mix business logic in handlers
- Suppress type errors

**Always:**
- Validate user input (frontend + backend)
- Use parameterized queries (GORM does this)
- Handle errors explicitly
- Check DB connection before server start

---

## COMMANDS

**Development:**
```bash
make dev                    # All services
make dev-services           # Postgres, Keycloak, Redis
make backend-dev            # Go server :8080
make frontend-dev           # Vite dev server :5173
```

**Testing:**
```bash
# Backend
cd backend && go test ./...
cd backend && go test -v -run TestFunctionName ./path/to/package

# Frontend
cd frontend && npm test
cd frontend && npm test -- path/to/test.spec.ts
cd frontend && npm test -- --watch
```

**Linting:**
```bash
# Backend
cd backend && go fmt ./...
cd backend && go vet ./...

# Frontend
cd frontend && npm run lint
cd frontend && npm run format
```

**Database:**
```bash
make db-migrate-up          # Run migrations
make db-migrate-down        # Rollback 1
make db-reset               # Drop all + rerun
```

**Build:**
```bash
# Local binaries
cd backend && go build -o opencode-api cmd/api/main.go
cd frontend && npm run build

# Docker images
make docker-build-prod      # Production (unified 29MB image)
make docker-build-dev       # Development (separate images)
make docker-push-prod       # Build and push production
make docker-push-dev        # Build and push development

# Or use script directly
./scripts/build-images.sh --mode prod --version v1.0.0
./scripts/build-images.sh --mode dev --push
```

---

## TEST SETUP

### Unit Tests
- **Backend:** Standard Go tests (`*_test.go`)
  - Repository layer: 9 tests (all passing)
  - Service layer: 26 tests (all passing)
  - API layer: 20 tests (all passing)
  - **Total:** 55 unit tests, all passing
  - **Run:** `cd backend && go test ./...`

### Integration Tests
- **Backend:** End-to-end project lifecycle tests
  - **Location:** `backend/internal/api/projects_integration_test.go`
  - **Documentation:** `backend/INTEGRATION_TESTING.md`
  - **Requirements:** PostgreSQL + Kubernetes cluster
  - **Build tag:** `-tags=integration` (isolated from regular tests)
  - **Run:** `cd backend && go test -tags=integration -v ./internal/api`
  - **Tests:**
    - Complete project lifecycle (create → verify → delete)
    - Pod failure graceful handling
  - **Environment vars:** `TEST_DATABASE_URL`, `K8S_NAMESPACE`, `KUBECONFIG`

### Frontend Tests
- **Vitest:** No config file - uses defaults
- **Coverage:** `go test -coverprofile=coverage.out ./...`

### E2E Tests
- **Status:** Cypress referenced in docs but not present (Phase 9)

---

## BUILD PATTERNS

### Docker Build Modes

**Production (Unified):**
- Single image with embedded frontend (29MB)
- Built from root `Dockerfile`
- Frontend served via `embed.FS` from Go binary
- Image: `registry.legal-suite.com/opencode/app:VERSION`
- Command: `./scripts/build-images.sh --mode prod`

**Development (Separate):**
- Separate backend and frontend images
- Backend from `backend/Dockerfile`
- Frontend + nginx from `frontend/Dockerfile`
- Images: `backend:VERSION`, `frontend:VERSION`
- Command: `./scripts/build-images.sh --mode dev`

**Build Script Features:**
- Supports `--mode prod|dev`
- Custom `--version` tag (default: `latest`)
- Optional `--push` to registry
- Custom `--registry` URL
- Color-coded output with status indicators
- Builds all 3 images: app/backend+frontend, file-browser-sidecar, session-proxy-sidecar

**Registry:** `registry.legal-suite.com/opencode`

**Other Scripts:**
- **Keycloak setup:** `scripts/setup-keycloak.sh` (creates realm `opencode` and client `opencode-app`)
- **Kind deploy:** `scripts/deploy-kind.sh`

---

## DEPENDENCIES

**Go:**
- Gin (HTTP), GORM (ORM), UUID, godotenv
- go-oidc v3 (OIDC provider), golang-jwt/jwt v5 (JWT)
- Go 1.24 across all modules

**TypeScript/React:**
- React 18, React Router, Axios, Zustand
- Vite, Tailwind CSS, Monaco Editor, dnd-kit
- ESLint (extends recommended + TS + React hooks + prettier)
- Prettier (configured in `.prettierrc`)

---

## GOTCHAS

1. **Three `go.mod` files** - backend and each sidecar are separate modules (unified via `go.work`)
2. **Strict lint policy** - ESLint `--max-warnings 0` fails on any warning
3. **Prettier config** - `.prettierrc` configured with project defaults
4. **ESLint + Prettier** - `eslint-config-prettier` installed to prevent conflicts
5. **Path alias `@/`** - maps to `./src/` in tsconfig
6. **Migration tool** - Uses golang-migrate CLI (not GORM auto-migrate in prod)
7. **Phase 1 complete** - OIDC authentication fully implemented (backend + frontend)
8. **Backend port** - Runs on 8090 (not 8080 due to port conflict with SearXNG)
9. **Unified production image** - Single 29MB Docker image serves both API and SPA (embedded with `go:embed`)
10. **Build modes** - Use `--mode prod` for production (unified), `--mode dev` for development (separate)
11. **Phase 2 backend complete (2.1-2.7)** - Project CRUD, K8s pod lifecycle, RBAC configured, integration tests
12. **Phase 2.8 frontend complete** - Project types and API client implemented (TypeScript interfaces + axios methods)
13. **Phase 2.9 frontend complete** - React UI components implemented (ProjectList, ProjectCard, CreateProjectModal, ProjectDetailPage)
14. **Phase 2.10 frontend complete** - WebSocket hook for real-time pod status updates (useProjectStatus)
15. **Phase 2.11 frontend complete** - Navigation menu with AppLayout component (Projects link, user email, logout)
16. **Phase 2.12 infrastructure complete** - Kind cluster deployment working (`make kind-deploy` functional)
17. **Integration tests** - Use `-tags=integration` flag to run, requires PostgreSQL + Kubernetes cluster
18. **Phase 2 archived** - Complete implementation summary in PHASE2.md (2026-01-18)
19. **Current phase:** Phase 3 - Task Management & Kanban Board (3.1-3.8 Complete)
20. **Test coverage (Phase 3):** 100 backend unit tests (repository: 30, service: 35, handlers: 35) - all passing
21. **Phase 3 complete (2026-01-19 00:45)** - Full task management backend + Kanban UI:
    - Backend: Task model, repository, service, API handlers (100 tests)
    - Frontend: Task types, API client, Kanban board with drag-drop (@dnd-kit)
    - Components: KanbanBoard (183 lines), KanbanColumn (59 lines), TaskCard (58 lines)
    - Features: Optimistic updates, error rollback, priority colors, responsive layout
    - Real-time: WebSocket streaming with exponential backoff, task detail panel, create modal
22. **Phase 4.1 complete (2026-01-19 08:37)** - File Browser Sidecar Setup:
    - Sidecar: Production-ready Go 1.24 service with structured logging (slog)
    - Service layer: Complete CRUD operations with path traversal prevention
    - API handlers: 6 HTTP endpoints with centralized error handling
    - Docker: 20.8MB image with HEALTHCHECK (30s interval, wget-based)
    - Tests: 58 comprehensive unit tests (24 service + 34 handler) - all passing
    - Security: Path validation (rejects ..), 10MB file size limits enforced
    - Health: 3 endpoints (/healthz, /health, /ready), graceful shutdown (10s timeout)
23. **Phase 4.2 complete (2026-01-19 08:50)** - File Watcher with Real-time Broadcasting:
    - Watcher: fsnotify-based recursive directory watching with WebSocket broadcast
    - Features: Event debouncing (100ms), monotonic versioning, thread-safe client registry
    - WebSocket: /files/watch endpoint with ping/pong keep-alive (30s interval)
    - Events: created, modified, deleted, renamed (mapped from fsnotify ops)
    - Pattern: Follows backend TaskBroadcaster design (Phase 3.10)
    - Tests: 16 unit tests (11 service + 5 handler) - all passing (2 skipped for integration)
    - Total: 74 tests passing across file-browser sidecar (58 Phase 4.1 + 16 Phase 4.2)
