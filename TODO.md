# OpenCode Project Manager - TODO List

**Last Updated:** 2026-01-17 11:46 CET  
**Current Phase:** Phase 2 - Project Management (2.1 & 2.2 Complete)  
**Branch:** main

---

## ✅ Phase 1: OIDC Authentication - COMPLETE

**Completion Date:** 2026-01-16 21:28 CET  
**Status:** All implementation complete, all E2E tests passing (7/7)

🎉 **Phase 1 archived to PHASE1.md** - Ready for Phase 2 development!

**Key Achievements:**
- ✅ Complete OIDC authentication flow (Keycloak + JWT)
- ✅ Backend auth service with middleware
- ✅ Frontend auth context and protected routes
- ✅ All E2E tests passing (no code replay errors)
- ✅ User creation in PostgreSQL verified

See [PHASE1.md](./PHASE1.md) for complete archive of Phase 1 tasks and resolution details.

---

## 🔄 Phase 2: Project Management (Weeks 3-4)

**Objective:** Implement project CRUD operations with Kubernetes pod lifecycle management.

**Status:** 🔄 IN PROGRESS (2.1, 2.2, 2.3 Complete - Ready for 2.4 Business Logic)

### Overview

Phase 2 introduces the core project management functionality:
- Projects are stored in PostgreSQL
- Each project spawns a dedicated Kubernetes pod with:
  - OpenCode server container
  - File browser sidecar (port 3001)
  - Session proxy sidecar (port 3002)
  - Shared PVC for workspace persistence
- Real-time pod status updates via WebSocket

---

### Backend Tasks (11 tasks)

#### 2.1 Database & Models ✅ COMPLETE
- [x] **DB Migration**: Create `002_add_project_fields.sql` migration
  - Added repo_url, pod_created_at, deleted_at, pod_error fields
  - Index on deleted_at for soft delete queries
  - Works with existing projects table from 001_init.sql
  - **Location:** `db/migrations/002_add_project_fields.up.sql` + `002_add_project_fields.down.sql`
  - **Status:** ✅ Migrated and verified in database

- [x] **Project Model**: Implement GORM model
  - UUID primary key with all required fields
  - Explicit GORM column tags for all fields
  - Soft delete support (`gorm.DeletedAt`)
  - User association (belongs to User)
  - K8s metadata fields (pod_name, pod_namespace, workspace_pvc_name, pod_status, pod_created_at, pod_error)
  - ProjectStatus enum constants (initializing, ready, error, archived)
  - **Location:** `backend/internal/model/project.go`
  - **Status:** ✅ Implemented and compiles successfully

#### 2.2 Repository Layer ✅ COMPLETE
- [x] **Project Repository**: Implement data access layer
  - `Create(ctx, project *Project) error` - Creates new projects with auto-UUID generation
  - `FindByID(ctx, id uuid.UUID) (*Project, error)` - Retrieves project by ID
  - `FindByUserID(ctx, userID uuid.UUID) ([]Project, error)` - Lists user's projects (ordered by created_at DESC)
  - `Update(ctx, project *Project) error` - Updates existing project
  - `SoftDelete(ctx, id uuid.UUID) error` - Soft deletes project (sets DeletedAt)
  - `UpdatePodStatus(ctx, id uuid.UUID, status string, podError string) error` - Updates pod status and error
  - Interface-based design for testability
  - Context-aware methods for cancellation/timeout support
  - **Location:** `backend/internal/repository/project_repository.go`
  - **Tests:** `backend/internal/repository/project_repository_test.go` (9 tests, all passing)
  - **Status:** ✅ Implemented with comprehensive unit tests, all tests passing

#### 2.3 Kubernetes Service Layer ✅ COMPLETE
- [x] **Kubernetes Client Wrapper**: Implement K8s operations
  - ✅ Initialize in-cluster or kubeconfig-based client
  - ✅ `CreateProjectPod(ctx, project *Project) error` - spawn pod with 3 containers + PVC
  - ✅ `DeleteProjectPod(ctx, podName, namespace string) error` - cleanup pod and PVC
  - ✅ `GetPodStatus(ctx, podName, namespace string) (string, error)` - query pod phase
  - ✅ `WatchPodStatus(ctx, podName, namespace string) (<-chan string, error)` - watch for status changes
  - ✅ KubernetesService interface for testability
  - ✅ Configurable images, resources, namespace
  - ✅ k8s.io/client-go@v0.32.0 integrated
  - **Location:** `backend/internal/service/kubernetes_service.go` (265 lines)
  - **Tests:** `backend/internal/service/kubernetes_service_test.go` (8 tests, all passing)

- [x] **Pod Manifest Template**: Define pod specification
  - ✅ 3-container pod:
    1. OpenCode server (port 3000) with health probes
    2. File browser sidecar (port 3001)
    3. Session proxy sidecar (port 3002)
  - ✅ Shared PVC mounted to all containers at `/workspace`
  - ✅ Configurable resource limits (CPU: 1000m, Memory: 1Gi)
  - ✅ Configurable resource requests (CPU: 100m, Memory: 256Mi)
  - ✅ Labels for project_id tracking
  - ✅ PVC with ReadWriteOnce, configurable size (default 1Gi)
  - **Location:** `backend/internal/service/pod_template.go` (184 lines)
  - **Status:** ✅ Implemented with comprehensive builder functions

#### 2.4 Business Logic Layer
- [ ] **Project Service**: Implement business logic
  - `CreateProject(userID uuid.UUID, name, description, repoUrl string) (*Project, error)`
    - Validate input
    - Create project in DB
    - Spawn K8s pod via KubernetesService
    - Update project with pod metadata
    - Return project
  - `GetProject(id, userID uuid.UUID) (*Project, error)` - authorization check
  - `ListProjects(userID uuid.UUID) ([]Project, error)`
  - `UpdateProject(id, userID uuid.UUID, updates map[string]interface{}) error`
  - `DeleteProject(id, userID uuid.UUID) error`
    - Delete pod from K8s
    - Soft delete in DB
  - **Location:** `backend/internal/service/project_service.go`

#### 2.5 API Handlers
- [ ] **Project API Endpoints**: Implement HTTP handlers
  - `POST /api/projects` - Create project (protected)
  - `GET /api/projects` - List user's projects (protected)
  - `GET /api/projects/:id` - Get project details (protected)
  - `PATCH /api/projects/:id` - Update project (protected)
  - `DELETE /api/projects/:id` - Delete project (protected)
  - Request validation (bind JSON)
  - Error handling with proper status codes
  - Authorization checks (user owns project)
  - **Location:** `backend/internal/api/projects.go`

- [ ] **WebSocket Status Endpoint**: Real-time pod status
  - `WebSocket /ws/projects/:id/status` - Stream pod status changes
  - Upgrade HTTP to WebSocket
  - Watch K8s pod status via KubernetesService
  - Send status updates to client as JSON
  - Cleanup on disconnect
  - **Location:** `backend/internal/api/projects.go` (or separate `websocket.go`)

#### 2.6 Integration
- [ ] **Register Routes**: Wire up project endpoints
  - Add project routes to Gin router
  - Apply auth middleware to all project routes
  - **Location:** `backend/cmd/api/main.go`

- [ ] **Kubernetes RBAC**: Configure service account permissions
  - Create ServiceAccount for backend pod
  - Create Role with permissions: `pods`, `persistentvolumeclaims` (create, delete, get, list, watch)
  - Create RoleBinding
  - Update backend deployment to use ServiceAccount
  - **Location:** `k8s/base/rbac.yaml` + `k8s/base/deployment.yaml`

#### 2.7 Testing & Verification
- [ ] **Unit Tests**: Test core logic
  - ProjectRepository CRUD operations (use testcontainers or in-memory DB)
  - ProjectService business logic (mock repository and K8s service)
  - **Location:** `backend/internal/repository/project_repository_test.go`, `backend/internal/service/project_service_test.go`

- [ ] **Integration Test**: End-to-end project creation
  - POST /api/projects → verify pod created in K8s
  - Verify PVC created
  - GET /api/projects/:id → verify project returned
  - DELETE /api/projects/:id → verify pod deleted
  - **Location:** `backend/internal/api/projects_test.go`

---

### Frontend Tasks (8 tasks)

#### 2.8 Types & API Client
- [ ] **Project Types**: Define TypeScript interfaces
  - `Project` interface (id, userId, name, description, repoUrl, createdAt, updatedAt, podStatus)
  - `CreateProjectRequest` interface
  - `UpdateProjectRequest` interface
  - Pod status enum: `Pending | Running | Succeeded | Failed | Unknown`
  - **Location:** `frontend/src/types/index.ts`

- [ ] **Project API Client**: Implement API methods
  - `createProject(data: CreateProjectRequest): Promise<Project>`
  - `getProjects(): Promise<Project[]>`
  - `getProject(id: string): Promise<Project>`
  - `updateProject(id: string, data: UpdateProjectRequest): Promise<Project>`
  - `deleteProject(id: string): Promise<void>`
  - Use axios instance from `services/api.ts` (JWT already configured)
  - **Location:** `frontend/src/services/api.ts` (extend existing)

#### 2.9 UI Components
- [ ] **ProjectList Component**: Display all projects
  - Fetch projects on mount
  - Display project cards in grid
  - Show pod status badge (color-coded: Pending=yellow, Running=green, Failed=red)
  - "Create Project" button → opens modal
  - Loading state while fetching
  - Empty state (no projects)
  - **Location:** `frontend/src/components/Projects/ProjectList.tsx`

- [ ] **ProjectCard Component**: Single project display
  - Project name, description
  - Pod status indicator (live badge)
  - Created date
  - Click → navigate to `/projects/:id`
  - Delete button with confirmation
  - **Location:** `frontend/src/components/Projects/ProjectCard.tsx`

- [ ] **CreateProjectModal Component**: Project creation form
  - Form fields: name (required), description (optional), repoUrl (optional)
  - Form validation (name length, URL format)
  - Submit → call API → close modal → refresh list
  - Cancel button
  - Loading state during creation
  - Error display
  - **Location:** `frontend/src/components/Projects/CreateProjectModal.tsx`

- [ ] **ProjectDetailPage**: Single project view
  - Display project metadata
  - Show real-time pod status (via WebSocket)
  - Placeholder for future tabs (Tasks, Files, Config)
  - Edit project button
  - Delete project button
  - Breadcrumb navigation
  - **Location:** `frontend/src/pages/ProjectDetailPage.tsx`

#### 2.10 Real-time Updates
- [ ] **WebSocket Hook**: Pod status subscription
  - `useProjectStatus(projectId: string)` hook
  - Connect to `ws://localhost:8090/ws/projects/:id/status`
  - Listen for status updates
  - Update local state on message
  - Cleanup on unmount
  - Reconnect logic on disconnect
  - **Location:** `frontend/src/hooks/useProjectStatus.ts`

#### 2.11 Routes & Navigation
- [ ] **Add Project Routes**: Update router
  - `/projects` → ProjectList page (protected)
  - `/projects/:id` → ProjectDetailPage (protected)
  - Update navigation menu (add "Projects" link)
  - **Location:** `frontend/src/App.tsx`

---

### Infrastructure Tasks (3 tasks)

#### 2.12 Kubernetes Setup
- [ ] **Update Base Manifests**: Add project pod template
  - Define PVC template for project workspaces
  - ConfigMap for OpenCode server config (if needed)
  - **Location:** `k8s/base/` (new files or updates)

- [ ] **Local Testing**: Verify in kind cluster
  - Deploy updated manifests to kind
  - Test project creation via API
  - Verify pod spawns correctly
  - Verify PVC mounts
  - Check logs of all 3 containers
  - **Command:** `make kind-deploy` then manual API testing

#### 2.13 Documentation
- [ ] **Update Documentation**: Reflect Phase 2 changes
  - Update AGENTS.md with Phase 2 status
  - Update README.md with project management features
  - Add API examples to DEVELOPMENT.md
  - **Location:** `AGENTS.md`, `README.md`, `DEVELOPMENT.md`

---

## Success Criteria (Phase 2 Complete When...)

- [x] **2.1 Database & Models Complete**
  - [x] Database migration adding project fields (repo_url, pod_created_at, deleted_at, pod_error)
  - [x] Project GORM model with all fields and soft delete support
  - [x] Migration verified in PostgreSQL
- [x] **2.2 Repository Layer Complete**
  - [x] ProjectRepository interface with all CRUD operations
  - [x] Comprehensive unit tests (9 tests, all passing)
  - [x] Context-aware methods for cancellation/timeout
  - [x] Soft delete functionality verified
- [x] **2.3 Kubernetes Service Layer Complete**
  - [x] KubernetesService with pod lifecycle management
  - [x] Pod template with 3 containers + shared PVC
  - [x] In-cluster and kubeconfig client support
  - [x] Real-time pod status watching via channels
  - [x] Comprehensive unit tests (8 tests, all passing)
  - [x] Configurable images, resources, and namespace
- [ ] User can create a project via UI
- [ ] Project creation spawns a K8s pod with 3 containers
- [ ] Project list shows all user's projects with live pod status
- [ ] Project detail page displays real-time status updates
- [ ] User can delete a project (pod cleanup verified)
- [ ] All backend unit tests passing
- [ ] Integration test: full project lifecycle (create → verify pod → delete → verify cleanup)
- [ ] No TypeScript errors in frontend
- [ ] All ESLint warnings resolved

---

## Phase 2 Dependencies

**Required Before Starting:**
- ✅ Phase 1 complete (auth working)
- ✅ PostgreSQL running
- ✅ Kubernetes cluster accessible (kind or other)
- ✅ Service account with RBAC permissions configured

**External Dependencies:**
- Kubernetes cluster (kind for local dev)
- Docker registry for sidecar images (file-browser, session-proxy)
- OpenCode server image (TBD - may use existing or build custom)

---

## Deferred to Later Phases

**Not in Phase 2 scope:**
- Task management (Phase 3)
- File explorer UI (Phase 4)
- OpenCode execution (Phase 5)
- Configuration management (Phase 6)
- Two-way interactions (Phase 7)

---

## Notes & Considerations

### Pod Naming Convention
- Format: `project-<project-id>-<random-suffix>`
- Namespace: `opencode` (consistent with base manifests)
- Labels: `app=opencode-project`, `project-id=<uuid>`

### PVC Naming Convention
- Format: `workspace-<project-id>`
- Storage class: Use cluster default (kind uses `standard`)
- Size: Start with 1Gi, make configurable later

### Pod Status Mapping
- K8s Pod Phase → Project Status
  - `Pending` → "Pending"
  - `Running` → "Running"
  - `Succeeded` → "Completed" (not expected for long-running pods)
  - `Failed` → "Failed"
  - `Unknown` → "Unknown"

### Error Handling
- Pod creation failures should NOT block project creation in DB
- Store pod creation errors in project metadata (add `pod_error` column if needed)
- Retry logic for transient K8s errors
- User-friendly error messages in UI

### Security
- Ensure user can only access their own projects (authorization checks)
- Validate project name (no special chars for K8s compatibility)
- Limit number of projects per user (add quota later if needed)

### Performance
- Paginate project list if >100 projects
- Cache pod status for 5-10 seconds to reduce K8s API calls
- Use WebSocket for real-time updates (don't poll)

---

## Next Phase Preview

**Phase 3: Task Management & Kanban Board (Weeks 5-6)**
- Task CRUD operations
- State machine: TODO → IN_PROGRESS → AI_REVIEW → HUMAN_REVIEW → DONE
- Kanban board UI with drag-and-drop
- Task detail panel

---

**Phase 2 Start Date:** 2026-01-16 23:44 CET  
**Phase 2.3 Completion:** 2026-01-17 12:17 CET  
**Target Completion:** TBD (flexible, 3-developer team)  
**Author:** Sisyphus (OpenCode AI Agent)

---

## Phase 2.3 Implementation Summary

**Completed:** 2026-01-17 12:17 CET

### What Was Implemented:

1. **KubernetesService Interface** (`kubernetes_service.go`)
   - Factory function: `NewKubernetesService(kubeconfig, namespace, config)`
   - Auto-detects in-cluster vs local kubeconfig
   - 4 core methods: CreateProjectPod, DeleteProjectPod, GetPodStatus, WatchPodStatus
   - Configurable via `KubernetesConfig` struct (images, resources, storage)

2. **Pod Template Builder** (`pod_template.go`)
   - `buildProjectPodSpec()` - Creates complete pod with 3 containers
   - `buildPVCSpec()` - Creates PersistentVolumeClaim
   - Health probes on OpenCode server (liveness + readiness)
   - Shared `/workspace` volume across all containers

3. **Comprehensive Testing** (`kubernetes_service_test.go`)
   - 8 unit tests using fake Kubernetes clientset
   - Tests cover: pod creation, deletion, status query, watch mechanism
   - All tests passing (8/8) ✅

### Key Features:
- ✅ Interface-based design for testability
- ✅ Context-aware for cancellation/timeout
- ✅ Graceful cleanup (deletes both pod and PVC)
- ✅ Real-time status updates via Go channels
- ✅ Configurable resource limits and requests
- ✅ Project-ID labeling for tracking

### Dependencies Added:
- `k8s.io/client-go@v0.32.0`
- `k8s.io/apimachinery@v0.32.0`
- `k8s.io/api@v0.32.0`

### Files Created:
- `backend/internal/service/kubernetes_service.go` (265 lines)
- `backend/internal/service/pod_template.go` (184 lines)
- `backend/internal/service/kubernetes_service_test.go` (434 lines)

### Next: Phase 2.4 - Business Logic Layer
Ready to implement ProjectService that orchestrates:
- Project creation in DB (via ProjectRepository)
- Pod spawning in K8s (via KubernetesService)
- Project updates with pod metadata
- Delete operations with cleanup
