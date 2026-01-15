# OpenCode Project Manager - Architecture Document

## System Architecture

### High-Level Design

```
┌─────────────────────────────────────────────────────────────────────┐
│                                                                       │
│                   Frontend (React SPA)                               │
│        - Vite for fast development and production builds             │
│        - TypeScript for type safety                                  │
│        - Tailwind CSS for styling                                    │
│        - Monaco Editor for code viewing                              │
│        - dnd-kit for drag-and-drop kanban                            │
│                                                                       │
└─────────────────────┬───────────────────────────────────────────────┘
                      │ HTTPS + JWT Authentication
                      │
┌─────────────────────▼───────────────────────────────────────────────┐
│                                                                       │
│                   API Gateway / Load Balancer                        │
│                   (Kubernetes Ingress)                               │
│                                                                       │
└─────────────────────┬───────────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────────┐
│                                                                       │
│              Main Controller Service (Go/Gin)                        │
│                                                                       │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  API Routes                                                  │  │
│  │  - Authentication (OIDC, JWT)                              │  │
│  │  - Projects CRUD                                           │  │
│  │  - Tasks CRUD + State Machine                             │  │
│  │  - File Browser Proxy                                     │  │
│  │  - Config Management                                      │  │
│  │  - Session Management                                    │  │
│  │  - Interactions (two-way feedback)                        │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                       │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  WebSocket Handler                                           │  │
│  │  - Real-time task updates                                  │  │
│  │  - Pod status streaming                                    │  │
│  │  - OpenCode output streaming                               │  │
│  │  - Event distribution to clients                           │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                       │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  Service Layer                                               │  │
│  │  - ProjectService (pod lifecycle management)              │  │
│  │  - TaskService (state transitions)                        │  │
│  │  - OpenCodeService (session spawning)                     │  │
│  │  - FileService (proxy to sidecars)                        │  │
│  │  - ConfigService (credential management)                 │  │
│  │  - AuthService (OIDC + JWT)                               │  │
│  │  - KubernetesService (pod/pvc creation)                  │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                       │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  Repository Layer (Database Access)                          │  │
│  │  - GORM models + queries                                   │  │
│  │  - Connection pooling                                     │  │
│  │  - Transaction management                                │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                       │
└─────────────────────┬───────────────────────────────────────────────┘
                      │
        ┌─────────────┴──────────────┬──────────────┬──────────────┐
        │                            │              │              │
┌───────▼──────────┐  ┌──────────────▼──┐  ┌───────▼─────┐  ┌──────▼──────┐
│                  │  │                  │  │             │  │             │
│  PostgreSQL      │  │  Per-Project Pod │  │  Keycloak  │  │   Redis     │
│  (Persistent)    │  │  (On-Demand)     │  │   (OIDC)   │  │  (Session   │
│                  │  │                  │  │            │  │   Cache)    │
│  - Users         │  │ OpenCode Server  │  │            │  │             │
│  - Projects      │  │ File Browser     │  │            │  │             │
│  - Tasks         │  │ Session Proxy    │  │            │  │             │
│  - Configs       │  │ Shared Volume    │  │            │  │             │
│  - Sessions      │  │                  │  │            │  │             │
│  - Audit Log     │  │                  │  │            │  │             │
│                  │  │                  │  │            │  │             │
└──────────────────┘  └──────────────────┘  └────────────┘  └─────────────┘
```

---

## Component Breakdown

### 1. Frontend (React)

**Framework Stack:**
- React 18+ with TypeScript
- Vite (build tool)
- React Router v6 (routing)
- SWR or React Query (data fetching)
- Zustand or Context API (state management)
- Tailwind CSS (styling)
- shadcn/ui (component library)

**Key Components:**
```
App.tsx
├── AuthContext (global auth state)
├── Router
│   ├── LoginPage
│   ├── OidcCallback
│   └── ProtectedRoutes
│       ├── Dashboard
│       ├── ProjectPage
│       │   ├── KanbanBoard
│       │   │   ├── KanbanColumn (TODO, IN_PROGRESS, etc.)
│       │   │   ├── TaskCard
│       │   │   └── TaskDetailPanel
│       │   ├── FileExplorer
│       │   │   ├── FileTree
│       │   │   └── MonacoEditor
│       │   └── ConfigPanel
│       └── ProjectSettings
```

**Data Flow:**
```
User Action → Component State → Hook (useAuth, useTasks, etc.)
           → API Call via SWR/React Query
           → Backend API
           → Cache update
           → Component re-render
```

**State Management:**
- Auth: Context + sessionStorage (tokens)
- Projects: Query (SWR/React Query)
- Tasks: Query (cached, real-time via WebSocket)
- UI: Local component state

---

### 2. Backend (Go/Gin)

**Architecture Pattern:** Layered architecture (Clean Architecture)

```
HTTP Request
    ↓
Router (Gin)
    ↓
Middleware (Auth, CORS, Logging)
    ↓
Handler (API layer) - validate input
    ↓
Service (Business logic layer) - orchestrate operations
    ↓
Repository (Data access layer) - database queries
    ↓
Database (PostgreSQL)
```

**Project Structure:**
```
backend/
├── cmd/api/
│   └── main.go                    # Entry point, router setup
├── internal/
│   ├── api/                       # HTTP handlers
│   │   ├── auth.go
│   │   ├── projects.go
│   │   ├── tasks.go
│   │   ├── files.go
│   │   ├── config.go
│   │   ├── interactions.go
│   │   └── ws.go
│   ├── service/                   # Business logic
│   │   ├── auth.go
│   │   ├── project.go
│   │   ├── task.go
│   │   ├── opencode.go
│   │   ├── kubernetes.go
│   │   ├── file.go
│   │   ├── config.go
│   │   └── interaction.go
│   ├── repository/                # Database access
│   │   ├── user.go
│   │   ├── project.go
│   │   ├── task.go
│   │   ├── config.go
│   │   ├── session.go
│   │   └── interaction.go
│   ├── model/                     # Domain models + GORM models
│   │   ├── user.go
│   │   ├── project.go
│   │   ├── task.go
│   │   ├── opencode_config.go
│   │   ├── session.go
│   │   ├── interaction.go
│   │   └── types.go
│   ├── middleware/
│   │   ├── auth.go                # JWT validation
│   │   ├── cors.go
│   │   ├── logging.go
│   │   ├── error_handler.go
│   │   └── request_id.go
│   ├── config/
│   │   ├── config.go              # App config from env
│   │   ├── database.go
│   │   └── oidc.go
│   ├── util/
│   │   ├── crypto.go              # Encrypt/decrypt credentials
│   │   ├── jwt.go
│   │   ├── errors.go
│   │   └── helpers.go
│   ├── db/
│   │   ├── migrations/            # SQL migrations
│   │   │   ├── 001_init.sql
│   │   │   ├── 002_projects.sql
│   │   │   └── ...
│   │   └── postgres.go            # Connection setup
│   └── logger/
│       └── logger.go
├── go.mod
├── go.sum
└── Dockerfile
```

**Key Design Patterns:**
- Interface-based design for testability
- Dependency injection via function parameters
- Error wrapping with context
- Middleware chain
- Repository pattern for data access

---

### 3. Sidecar Services

#### File Browser Sidecar (Go)

**Responsibilities:**
- Serve directory tree structure
- Read file contents (with safe path validation)
- Write files to workspace
- Create/delete directories
- File metadata (size, type, modified time)

**Architecture:**
```
HTTP Request to :3001
    ↓
Router (Gin)
    ↓
Handler (validates workspace path)
    ↓
Service (file operations)
    ↓
Filesystem (workspace PVC)
```

**Endpoints:**
```
GET  /files/tree?path=/&max_depth=5
GET  /files/content?path=/file.go
POST /files/write (create/update)
DELETE /files?path=/file.go
POST /files/mkdir
```

**Security Measures:**
- Path traversal validation (reject .. in paths)
- Workspace boundary enforcement (all paths must be within /workspace)
- Symlink follow restrictions
- No executable permissions needed

#### Session Proxy Sidecar (Go)

**Responsibilities:**
- Proxy HTTP requests to local OpenCode server
- Stream events via SSE
- WebSocket tunneling for PTY interaction
- Session lifecycle management

**Architecture:**
```
HTTP Request to :3002
    ↓
Router (Gin)
    ↓
Handler (request routing)
    ↓
OpenCode SDK / HTTP Proxy
    ↓
OpenCode Server (:3000)
```

**Endpoints:**
```
GET  /session
POST /session (create)
GET  /session/:id
GET  /session/:id/events (SSE)
GET  /session/:id/pty/connect (WebSocket)
POST /session/:id/prompt
```

---

### 4. Database (PostgreSQL)

**Schema Design:**

```
users
  id (PK)
  oidc_subject (UNIQUE)
  email
  name
  picture_url
  last_login_at
  created_at
  updated_at

projects
  id (PK)
  user_id (FK)
  name
  slug
  description
  pod_name
  pod_namespace
  pod_status
  workspace_pvc_name
  status (initializing, ready, error, archived)
  created_at
  updated_at

tasks
  id (PK)
  project_id (FK)
  title
  description
  status (todo, in_progress, ai_review, human_review, done)
  current_session_id
  opencode_output
  execution_duration_ms
  file_references (JSONB)
  created_by (FK)
  created_at
  updated_at

opencode_configs
  id (PK)
  project_id (FK)
  model
  provider
  provider_api_key_encrypted
  tools (JSONB)
  instructions
  temperature
  max_tokens
  is_active
  version
  created_at

opencode_sessions
  id (PK)
  project_id (FK)
  task_id (FK)
  remote_session_id
  status
  prompt
  final_output
  exit_code
  execution_start_at
  execution_end_at

interactions
  id (PK)
  task_id (FK)
  session_id (FK)
  sequence_number
  user_prompt
  user_id (FK)
  agent_response
  user_prompt_at
  agent_response_at

task_events (audit trail)
  id (PK)
  task_id (FK)
  event_type
  actor_user_id (FK)
  metadata (JSONB)
  created_at

audit_log
  id (PK)
  user_id (FK)
  action
  resource_type
  resource_id
  details (JSONB)
  ip_address
  user_agent
  created_at
```

**Indexing Strategy:**
- Foreign keys are indexed
- Frequently queried fields (status, user_id, created_at) are indexed
- Composite indexes for queries like (project_id, status)
- JSONB fields have GIN indexes where needed

**Constraints:**
- NOT NULL on required fields
- UNIQUE constraints on natural identifiers (slug per user)
- CHECK constraints on enums (validated at DB level)
- ON DELETE CASCADE for dependent records

---

### 5. Kubernetes Architecture

**Deployment Model:**

```
┌─────────────────────────────────────────────────────────────┐
│ Kubernetes Cluster (kind)                                   │
│                                                              │
│ ┌─────────────────────────────────────────────────────┐    │
│ │ Namespace: opencode                                  │    │
│ │                                                      │    │
│ │ ┌────────────────────────────────────────────────┐  │    │
│ │ │ Deployment: opencode-controller (2+ replicas) │  │    │
│ │ │ ├─ Service: opencode-controller                │  │    │
│ │ │ └─ Ingress: opencode.local                     │  │    │
│ │ └────────────────────────────────────────────────┘  │    │
│ │                                                      │    │
│ │ ┌────────────────────────────────────────────────┐  │    │
│ │ │ StatefulSet: postgres (1 replica in MVP)       │  │    │
│ │ │ ├─ Service: postgres-headless                  │  │    │
│ │ │ └─ PVC: postgres-data                          │  │    │
│ │ └────────────────────────────────────────────────┘  │    │
│ │                                                      │    │
│ │ ┌────────────────────────────────────────────────┐  │    │
│ │ │ ConfigMaps:                                    │  │    │
│ │ │ ├─ app-config (environment variables)          │  │    │
│ │ │ └─ postgres-init (init scripts)                │  │    │
│ │ └────────────────────────────────────────────────┘  │    │
│ │                                                      │    │
│ │ ┌────────────────────────────────────────────────┐  │    │
│ │ │ Secrets:                                       │  │    │
│ │ │ ├─ app-secrets (API keys, credentials)        │  │    │
│ │ │ └─ docker-registry (private registry auth)    │  │    │
│ │ └────────────────────────────────────────────────┘  │    │
│ │                                                      │    │
│ │ ┌────────────────────────────────────────────────┐  │    │
│ │ │ RBAC:                                          │  │    │
│ │ │ ├─ ServiceAccount: opencode-controller        │  │    │
│ │ │ ├─ Role: opencode-pod-manager                 │  │    │
│ │ │ └─ RoleBinding: opencode-pod-manager          │  │    │
│ │ └────────────────────────────────────────────────┘  │    │
│ │                                                      │    │
│ │ ┌────────────────────────────────────────────────┐  │    │
│ │ │ Per-Project Pod (created dynamically)         │  │    │
│ │ │ ├─ Container: opencode (official image)       │  │    │
│ │ │ ├─ Container: file-browser-sidecar            │  │    │
│ │ │ ├─ Container: session-proxy-sidecar           │  │    │
│ │ │ ├─ PVC: project-workspace                     │  │    │
│ │ │ └─ ConfigMap: project-config                  │  │    │
│ │ └────────────────────────────────────────────────┘  │    │
│ │                                                      │    │
│ └─────────────────────────────────────────────────────┘    │
│                                                              │
│ ┌─────────────────────────────────────────────────────┐    │
│ │ External Services (outside cluster)                  │    │
│ │ ├─ Keycloak (OIDC provider)                         │    │
│ │ └─ Private Docker Registry                          │    │
│ └─────────────────────────────────────────────────────┘    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**Lifecycle Management:**

1. **Pod Creation (Project Creation)**
   - User creates project via API
   - ProjectService calls KubernetesService.CreateProject
   - KubernetesService:
     - Creates PVC for workspace
     - Creates ConfigMap with OpenCode config
     - Creates Pod with 3 containers (OpenCode + 2 sidecars)
   - Pod is scheduled by Kubernetes
   - Wait for all containers ready
   - Update project.pod_name, project.status = "ready"

2. **Pod Cleanup (Project Deletion)**
   - User deletes project
   - ProjectService calls KubernetesService.DeleteProject
   - KubernetesService:
     - Deletes Pod (grace period for graceful shutdown)
     - Deletes PVC
     - Deletes ConfigMap
   - Archive project record in DB (soft delete)

3. **Health Monitoring**
   - Controller polls pod status regularly
   - Updates project.pod_status based on K8s pod phase
   - If pod fails, alert user and provide troubleshooting info

---

### 6. Authentication & Authorization

**OIDC Flow:**

```
1. User navigates to app
2. Frontend checks for JWT in storage
3. No JWT → redirect to /login
4. User clicks "Login with Keycloak"
5. Frontend initiates OIDC authorization code flow
   - POST /api/auth/oidc/login
   - Backend returns authorization_url (Keycloak login)
6. Frontend redirects to Keycloak
7. User enters credentials on Keycloak
8. Keycloak redirects to /auth/callback with authorization code
9. Frontend calls POST /api/auth/oidc/callback with code
10. Backend:
    - Validates authorization code with Keycloak
    - Exchanges code for ID token
    - Verifies ID token signature
    - Extracts user claims (sub, email, name)
    - Creates/updates user in DB
    - Generates JWT (signed with JWT_SECRET)
    - Returns JWT + user info
11. Frontend stores JWT in httpOnly cookie + sessionStorage
12. Subsequent requests include JWT in Authorization header
13. Backend middleware validates JWT on each request
```

**JWT Structure:**
```json
{
  "sub": "user-id-uuid",
  "email": "user@example.com",
  "name": "User Name",
  "iat": 1234567890,
  "exp": 1234571490
}
```

**Token Refresh:**
```
- JWT expires after 1 hour (configurable)
- Frontend detects expiration on API 401 response
- Frontend calls POST /api/auth/refresh
- Backend issues new JWT
- Frontend retries original request
```

**Authorization:**
```
Middleware checks JWT claims:
- Validates signature
- Checks expiration
- Extracts user_id from claims
- Attaches user context to request
- Subsequent handlers access user info via context
```

**Project Access Control:**
```
- Users can only access their own projects
- Check: project.user_id == auth_context.user_id
- Enforced at service layer (not relying on API caller)
```

---

### 7. Real-Time Communication

**WebSocket Connections:**

```
Client → WebSocket → Backend Handler → Service → Message Queue
                       ↓
                    Broadcast to subscribed clients
```

**Event Types:**

```
Project Pod Status:
  {type: "pod_status", projectId, status: "running|pending|failed"}

Task Updates:
  {type: "task_status_changed", taskId, newStatus, timestamp}

OpenCode Output:
  {type: "session_output", taskId, sessionId, output: "text", timestamp}

File Changes:
  {type: "file_changed", path, operation: "create|modify|delete"}
```

**Subscription Management:**
```
- Store active WebSocket connections in memory map
- Map key: (userId, projectId, resourceId)
- Value: channel for message delivery
- On connection: register in map
- On disconnect: unregister from map
- On event: broadcast to all subscribers
```

**Benefits:**
- Real-time task updates for all connected clients
- Live pod status monitoring
- Live stream of OpenCode output
- Reduced polling overhead

---

## Data Flow Diagrams

### Creating a Project

```
User clicks "New Project"
    ↓
CreateProjectModal (React)
    ↓
POST /api/projects {name, description}
    ↓
Auth Middleware validates JWT
    ↓
ProjectHandler.Create
    ↓
ProjectService.Create
  ├─ Validate input
  ├─ Create project record (status: initializing)
  ├─ Generate project slug
  ├─ KubernetesService.CreateProject
  │   ├─ Create PVC
  │   ├─ Create ConfigMap with opencode config
  │   └─ Create Pod spec
  ├─ Poll pod readiness
  └─ Update project.status = "ready"
    ↓
Return project details
    ↓
Frontend updates project list
    ↓
WebSocket notifies all clients
    ↓
User sees new project in list
```

### Executing a Task

```
User clicks "Execute" on task
    ↓
POST /api/projects/:id/tasks/:taskId/execute
    ↓
TaskHandler.Execute
    ↓
TaskService.Execute
  ├─ Validate task state
  ├─ Get project and config
  ├─ OpenCodeService.CreateSession
  │   ├─ Call session-proxy sidecar
  │   ├─ POST /session with prompt
  │   └─ Get remote_session_id from OpenCode
  ├─ Create OpenCodeSession record
  └─ Update task.status = "in_progress"
    ↓
Frontend GET /projects/:id/tasks/:taskId/output (SSE)
    ↓
Backend subscribes to session events
    ↓
SSE stream sends events as they arrive
  {type: "output", data: "..."}
  {type: "status", data: "running"}
  {type: "completion", data: {...}}
    ↓
Frontend updates TaskDetailPanel
    ↓
On completion:
  ├─ Update task.status = "ai_review"
  ├─ Store final output
  └─ WebSocket notifies kanban board
```

---

## Error Handling Strategy

**Error Types:**

```go
type ErrorType string

const (
  ErrNotFound       ErrorType = "NOT_FOUND"
  ErrUnauthorized   ErrorType = "UNAUTHORIZED"
  ErrForbidden      ErrorType = "FORBIDDEN"
  ErrValidation     ErrorType = "VALIDATION_ERROR"
  ErrConflict       ErrorType = "CONFLICT"
  ErrInternal       ErrorType = "INTERNAL_ERROR"
  ErrUnavailable    ErrorType = "SERVICE_UNAVAILABLE"
  ErrKubernetesError ErrorType = "KUBERNETES_ERROR"
)

type APIError struct {
  Type    ErrorType
  Message string
  Status  int
  Details interface{}
}
```

**HTTP Status Mapping:**
```
400 - Validation errors
401 - Missing/invalid JWT
403 - Forbidden (user doesn't own resource)
404 - Resource not found
409 - Conflict (e.g., slug already exists)
500 - Internal server error
503 - Service unavailable (pod not ready)
```

**Error Response Format:**
```json
{
  "error": {
    "type": "VALIDATION_ERROR",
    "message": "Project name is required",
    "status": 400,
    "details": {
      "field": "name",
      "reason": "required"
    }
  }
}
```

---

## Performance Considerations

### Database
- Connection pooling (25-50 connections)
- Query indexes on frequently accessed columns
- N+1 query prevention via eager loading
- Pagination for list endpoints

### Caching
- Frontend: React Query cache invalidation
- Backend: Redis for session state (future)
- ETag headers for browser caching

### File Operations
- Stream large files instead of loading into memory
- Limit directory depth for tree operations
- Compress API responses with gzip

### Pod Management
- Lazy creation of pods (on-demand)
- Graceful shutdown with 30s timeout
- Resource limits to prevent runaway pods

---

## Security Hardening

### MVP (Current)
✅ HTTPS (via Ingress)
✅ OIDC + JWT authentication
✅ Database password encryption
✅ Credential storage encryption
✅ RBAC for K8s API access
✅ Path traversal validation

### Future Enhancements
- [ ] Rate limiting per user
- [ ] API request signing
- [ ] Audit log encryption
- [ ] Network policies in K8s
- [ ] Pod security policies
- [ ] Secrets encryption at rest
- [ ] Log redaction (no sensitive data)

