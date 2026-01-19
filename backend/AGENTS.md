# OPENCODE BACKEND KNOWLEDGE BASE

## OVERVIEW
Go 1.24 API server using Gin (HTTP), GORM (PostgreSQL), and Keycloak (OIDC).

**Phase 1 Status**: ✅ OIDC Authentication Complete  
**Phase 2 Status**: ✅ Project Management Complete  
**Phase 3 Status**: ✅ Task Management Complete  
**Phase 4 Status**: 🔄 In Progress (4.1-4.5 Complete - File Browser Sidecar with Kubernetes Integration)

## STRUCTURE
```
.
├── cmd/api/           # Entry point (main.go) - wired with auth + projects + tasks + static serving
├── internal/
│   ├── api/           # HTTP Handlers - auth.go, projects.go, tasks.go ✅ Phase 3.4
│   ├── model/         # GORM structs (User, Project ✅ Phase 2, Task ✅ Phase 3.1)
│   ├── service/       # ✅ auth_service.go, project_service.go, task_service.go, kubernetes_service.go
│   ├── repository/    # ✅ user_repository.go, project_repository.go, task_repository.go
│   ├── middleware/    # ✅ auth.go - JWT validation, security.go - Security headers
│   ├── static/        # ✅ embed.go - Embedded frontend serving (production only)
│   ├── config/        # Environment & App configuration
│   └── db/            # Connection & Migration logic
└── go.mod             # Module path: github.com/npinot/vibe/backend
```

## PHASE 2 IMPLEMENTATION (COMPLETE)

### Project Management Stack
- **Project CRUD**: Full lifecycle management with K8s pod orchestration
- **Kubernetes Integration**: Pod creation, monitoring, lifecycle management
- **Real-time Updates**: WebSocket for pod status changes
- **Database**: PostgreSQL with soft deletes (DeletedAt)

### Implemented Endpoints
| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/healthz` | GET | None | Health check |
| `/ready` | GET | None | Readiness check |
| `/api/auth/oidc/login` | GET | None | Get Keycloak authorization URL |
| `/api/auth/oidc/callback` | GET | None | Exchange code for JWT |
| `/api/auth/me` | GET | JWT | Get current authenticated user |
| `/api/auth/logout` | POST | None | Client-side logout helper |
| `/api/projects` | GET | JWT | List user's projects |
| `/api/projects` | POST | JWT | Create new project |
| `/api/projects/:id` | GET | JWT | Get project details |
| `/api/projects/:id` | PATCH | JWT | Update project |
| `/api/projects/:id` | DELETE | JWT | Soft delete project |
| `/api/projects/:id/status` | GET (WS) | JWT | WebSocket for real-time pod status |
| `/api/projects/:id/tasks` | GET | JWT | List project's tasks ✅ Phase 3.4 |
| `/api/projects/:id/tasks` | POST | JWT | Create task ✅ Phase 3.4 |
| `/api/projects/:id/tasks/:taskId` | GET | JWT | Get task details ✅ Phase 3.4 |
| `/api/projects/:id/tasks/:taskId` | PATCH | JWT | Update task ✅ Phase 3.4 |
| `/api/projects/:id/tasks/:taskId/move` | PATCH | JWT | Move task (state + position) ✅ Phase 3.4 |
| `/api/projects/:id/tasks/:taskId` | DELETE | JWT | Delete task ✅ Phase 3.4 |
| `/api/projects/:id/tasks/:taskId/execute` | POST | JWT | Execute task (stub for Phase 5) ✅ Phase 3.4 |
| `/api/projects/:id/files/tree` | GET | JWT | Get directory tree (proxy to sidecar) ✅ Phase 4.3 |
| `/api/projects/:id/files/content` | GET | JWT | Get file content (proxy to sidecar) ✅ Phase 4.3 |
| `/api/projects/:id/files/info` | GET | JWT | Get file metadata (proxy to sidecar) ✅ Phase 4.3 |
| `/api/projects/:id/files/write` | POST | JWT | Write file (proxy to sidecar) ✅ Phase 4.3 |
| `/api/projects/:id/files` | DELETE | JWT | Delete file/dir (proxy to sidecar) ✅ Phase 4.3 |
| `/api/projects/:id/files/mkdir` | POST | JWT | Create directory (proxy to sidecar) ✅ Phase 4.3 |
| `/api/projects/:id/files/watch` | GET (WS) | JWT | WebSocket file changes (proxy to sidecar) ✅ Phase 4.3 |

### Key Components (Phase 2)

**ProjectService** (`internal/service/project_service.go`):
- CRUD operations with authorization checks
- Kubernetes pod lifecycle integration
- Slug generation and uniqueness validation

**ProjectRepository** (`internal/repository/project_repository.go`):
- Database CRUD operations
- Soft delete support
- Query methods with user filtering

**KubernetesService** (`internal/service/kubernetes_service.go`):
- Pod creation with 3-container spec (OpenCode + 2 sidecars)
- PVC creation for workspace persistence
- Pod status monitoring
- Cleanup on project deletion

## PHASE 3.1-3.4 IMPLEMENTATION (COMPLETE - 2026-01-18)

### Task Management Stack
- **Task CRUD**: Full lifecycle management with state machine validation
- **Repository Layer**: TaskRepository with 7 methods (30 tests)
- **Service Layer**: TaskService with 6 methods + state machine (35 tests)
- **API Layer**: TaskHandler with 7 REST endpoints (35 tests)
- **Database**: PostgreSQL with soft deletes and position ordering
- **Total Tests**: 100 task-related tests (all passing)

### Task Model Updates
- **Added Fields:**
  - `Position int` - Kanban column ordering (0-indexed)
  - `Priority TaskPriority` - Enum: low/medium/high
  - `AssignedTo *uuid.UUID` - Optional user assignment (Phase 7)
  - `DeletedAt gorm.DeletedAt` - Soft delete support
  - `Assignee *User` - Relationship pointer

- **Migration:** `003_add_task_kanban_fields.up.sql`
  - Adds 4 new columns to existing tasks table
  - Creates indexes on (project_id, position) and deleted_at
  - Includes column comment for position field

- **Model Location:** `backend/internal/model/task.go`
  - TaskStatus enum: todo, in_progress, ai_review, human_review, done
  - TaskPriority enum: low, medium, high
  - Full GORM tags with explicit column names
  - Soft delete via gorm.DeletedAt

### Key Components (Phase 3)

**TaskRepository** (`internal/repository/task_repository.go`):
- CRUD operations for Task model
- 7 methods: Create, FindByID, FindByProjectID, Update, UpdateStatus, UpdatePosition, SoftDelete
- Context-aware, returns errors for handling
- 30 unit tests (all passing)

**TaskService** (`internal/service/task_service.go`):
- Business logic layer with state machine validation
- 6 methods: CreateTask, GetTask, ListProjectTasks, UpdateTask, MoveTask, DeleteTask
- State machine enforces valid transitions between task states
- Authorization via project ownership check
- Input validation helpers (validateTaskTitle, validateTaskPriority)
- 35 unit tests (all passing)

**TaskHandler** (`internal/api/tasks.go`):
- HTTP handlers for task CRUD operations
- 7 endpoints: List, Create, Get, Update, Move, Delete, Execute (stub)
- Request/Response DTOs with JSON binding validation
- Error mapping to HTTP status codes (400, 401, 403, 404, 500)
- Authorization checks via middleware.GetCurrentUser
- 35 unit tests (all passing)

**State Machine:**
```go
var validTransitions = map[model.TaskStatus][]model.TaskStatus{
    TaskStatusTodo:        {TaskStatusInProgress},
    TaskStatusInProgress:  {TaskStatusAIReview, TaskStatusTodo},
    TaskStatusAIReview:    {TaskStatusHumanReview, TaskStatusInProgress},
    TaskStatusHumanReview: {TaskStatusDone, TaskStatusInProgress},
    TaskStatusDone:        {TaskStatusTodo}, // Allow reopening
}
```

**Routes (wired in cmd/api/main.go):**
- All routes under `/api/projects/:id/tasks` with JWT auth middleware
- TaskService initialized with TaskRepository + ProjectRepository
- TaskHandler created with dependency injection

## PHASE 4.1-4.3 IMPLEMENTATION (IN PROGRESS - 2026-01-19)

### File Browser Proxy Layer
- **Backend Proxy**: FileHandler in main API proxies file operations to file-browser sidecar
- **Pod IP Resolution**: Uses KubernetesService.GetPodIP() to discover sidecar pod dynamically
- **Authorization**: All endpoints verify project ownership before proxying
- **Sidecar Communication**: HTTP/WebSocket proxy to `http://<podIP>:3001`

### Key Components (Phase 4)

**FileHandler** (`internal/api/files.go`):
- HTTP proxy layer for 6 file operations (425 lines)
- WebSocket proxy for real-time file watching
- Pod IP resolution via KubernetesService
- Authorization via project ownership validation
- Error mapping: 400 (bad input), 401 (unauthorized), 500 (internal), 502 (sidecar error)

**KubernetesService Extension** (`internal/service/kubernetes_service.go`):
- Added `GetPodIP(ctx, podName, namespace) (string, error)` method
- Fetches pod resource and returns `pod.Status.PodIP`
- Handles pod not found and IP not yet assigned errors

**AuthMiddleware Extension** (`internal/middleware/auth.go`):
- Added `GetCurrentUserID(c *gin.Context) uuid.UUID` helper
- Extracts user ID from context (set by JWTAuth middleware)
- Returns `uuid.Nil` on error (safer than panic)

**Routes (wired in cmd/api/main.go):**
- 6 HTTP routes + 1 WebSocket route under `/api/projects/:id/files/...`
- JWT authentication middleware applied
- FileHandler initialized with ProjectRepository and KubernetesService

**Proxy Pattern:**
```go
// HTTP Proxy
sidecarURL := fmt.Sprintf("http://%s:3001/files/tree?path=%s", podIP, path)
req, _ := http.NewRequestWithContext(ctx, "GET", sidecarURL, nil)
resp, _ := httpClient.Do(req)
io.Copy(ginContext.Writer, resp.Body)

// WebSocket Proxy (bidirectional)
sidecarConn, _ := websocket.Dial(sidecarURL)
clientConn, _ := upgrader.Upgrade(ginContext.Writer, ginContext.Request)
// goroutine 1: pump client → sidecar
// goroutine 2: pump sidecar → client
```

**Test Coverage:**
- 22 unit tests for FileHandler (all passing)
- Tests use httptest.Server with dynamic port resolution
- Mock project repository and Kubernetes service
- Total backend tests: 84 (up from 62)

### Phase 4.5: Kubernetes Integration (COMPLETE - 2026-01-19)

**Pod Template Enhancement** (`internal/service/pod_template.go`):
- Added file-browser sidecar container to project pod spec (lines 94-150)
- Independent resource limits: 50Mi/100Mi memory, 50m/100m CPU (optimized for sidecar)
- Liveness probe: HTTP GET /healthz:3001 (initialDelay: 5s, period: 10s, timeout: 3s)
- Readiness probe: HTTP GET /healthz:3001 (initialDelay: 3s, period: 5s, timeout: 3s)
- Shared workspace volume mount: /workspace (PVC backed)
- Environment variables: WORKSPACE_DIR=/workspace, PORT=3001

**Container Spec:**
```go
{
    Name:  "file-browser",
    Image: config.FileBrowserImage, // registry.legal-suite.com/opencode/file-browser-sidecar:latest
    Ports: []corev1.ContainerPort{{ContainerPort: 3001, Protocol: TCP}},
    VolumeMounts: []corev1.VolumeMount{{Name: "workspace", MountPath: "/workspace"}},
    Resources: corev1.ResourceRequirements{
        Requests: {CPU: "50m", Memory: "50Mi"},
        Limits:   {CPU: "100m", Memory: "100Mi"},
    },
    LivenessProbe: &corev1.Probe{HTTPGet: {Path: "/healthz", Port: 3001}, ...},
    ReadinessProbe: &corev1.Probe{HTTPGet: {Path: "/healthz", Port: 3001}, ...},
}
```

**Docker Image:**
- Multi-stage build: golang:1.24-alpine (builder) → alpine:latest (runtime)
- Image size: 21.1MB (includes Alpine + ca-certificates + wget for health checks)
- HEALTHCHECK: `wget --spider http://localhost:3001/healthz` (30s interval, 3s timeout, 5s start period)
- Binary: Statically linked (`CGO_ENABLED=0`, `-ldflags="-s -w"`)
- Location: `sidecars/file-browser/Dockerfile`

**Verification:**
- ✅ Backend service package compiles successfully
- ✅ All backend tests pass (no regressions)
- ✅ Docker image builds and verified with `docker inspect`
- ✅ HEALTHCHECK configuration confirmed
- ✅ Pod spec includes all required fields per TODO.md spec

## PHASE 1 IMPLEMENTATION (COMPLETE)

### Authentication Stack
- **OIDC Provider**: Keycloak (go-oidc v3.17.0)
- **JWT**: golang-jwt/jwt v5 (HS256 signing)
- **User Storage**: PostgreSQL via GORM with auto-upsert

### Implemented Endpoints
| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/healthz` | GET | None | Health check |
| `/ready` | GET | None | Readiness check |
| `/api/auth/oidc/login` | GET | None | Get Keycloak authorization URL |
| `/api/auth/oidc/callback` | GET | None | Exchange code for JWT |
| `/api/auth/me` | GET | JWT | Get current authenticated user |
| `/api/auth/logout` | POST | None | Client-side logout helper |

### Key Components

**AuthService** (`internal/service/auth_service.go`):
- Initializes OIDC provider with Keycloak issuer
- Generates OAuth2 authorization URLs with state
- Exchanges authorization codes for tokens
- Verifies ID token signatures via JWKS
- Generates application JWTs (HS256)

**UserRepository** (`internal/repository/user_repository.go`):
- CRUD operations for User model
- `CreateOrUpdateFromOIDC()` - upserts users from OIDC claims

**AuthMiddleware** (`internal/middleware/auth.go`):
- Validates JWT signatures and claims
- Loads authenticated user from DB
- Injects user into Gin context

**SecurityHeaders** (`internal/middleware/security.go`):
- Sets X-Content-Type-Options, X-Frame-Options, X-XSS-Protection
- Adds HSTS for HTTPS connections
- Configures Permissions-Policy

**StaticServing** (`internal/static/embed.go`):
- Embeds frontend dist/ via Go embed.FS (production only)
- Serves static assets with appropriate cache headers
- SPA fallback for client-side routing (React Router)
- Smart caching: long cache for hashed assets, no-cache for index.html

### Configuration (Environment Variables)
```bash
OIDC_ISSUER=http://localhost:8081/realms/opencode
OIDC_CLIENT_ID=opencode-app
OIDC_CLIENT_SECRET=opencode-secret
JWT_SECRET=your-secret-key-min-32-chars
JWT_EXPIRY=3600
DATABASE_URL=postgres://opencode:password@localhost:5432/opencode_dev
PORT=8090
```

## CONVENTIONS

### Import Ordering
Group imports into three blocks separated by blank lines:
1. Standard library
2. Third-party packages (Gin, GORM, etc.)
3. Internal project packages

### Error Handling
- **Explicit**: Always check `err != nil`.
- **Wrapped**: Use `fmt.Errorf("context: %w", err)` to preserve stack.
- **Top-level Logging**: Log errors in Handlers/main; return errors up from internal layers.

### GORM Struct Tags
- Use `json` and `gorm` tags consistently.
- Primary keys: `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
- Timestamps: `CreatedAt`, `UpdatedAt`, `DeletedAt` (for soft deletes).

### Handler Responsibilities
- Auth handlers (`internal/api/auth.go`) fully implemented with service/repository pattern
- Future handlers should follow same pattern: parse input → call service → return JSON
- No direct DB access in handlers (use repositories)

### Testing
- Filename pattern: `*_test.go` in the same package as code.
- Mocking: Use interfaces for Services/Repositories to enable unit testing handlers.

## COMMANDS
```bash
# Run server
go run cmd/api/main.go

# Run all tests
go test ./...

# Run specific test
go test -v -run TestName ./path/to/package

# Build binary
go build -o opencode-api cmd/api/main.go
```
