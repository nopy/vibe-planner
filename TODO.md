# OpenCode Project Manager - TODO List

**Last Updated:** 2026-01-18 21:42 CET  
**Current Phase:** Phase 3 - Task Management & Kanban Board (Planning)  
**Branch:** main

---

## ✅ Phase 1: OIDC Authentication - COMPLETE

**Completion Date:** 2026-01-16 21:28 CET  
**Status:** All implementation complete, all E2E tests passing (7/7)

🎉 **Phase 1 archived to PHASE1.md** - Ready for Phase 2 development!

See [PHASE1.md](./PHASE1.md) for complete archive of Phase 1 tasks and resolution details.

---

## ✅ Phase 2: Project Management - COMPLETE

**Completion Date:** 2026-01-18 19:42 CET  
**Status:** Backend + Frontend + Infrastructure complete (2.1-2.12)

🎉 **Phase 2 archived to PHASE2.md** - Ready for Phase 3 development!

**Key Achievements:**
- ✅ Complete project CRUD operations with Kubernetes pod lifecycle
- ✅ 55 backend unit tests (repository, service, API layers) - all passing
- ✅ Integration tests for end-to-end project lifecycle
- ✅ Full project management UI with real-time WebSocket updates
- ✅ PostgreSQL deployment in Kubernetes
- ✅ RBAC configured with granular permissions
- ✅ `make kind-deploy` working end-to-end

See [PHASE2.md](./PHASE2.md) for complete archive of Phase 2 tasks and implementation details.

---

## 🔄 Phase 3: Task Management & Kanban Board (Weeks 5-6)

**Objective:** Implement task CRUD operations with state machine and drag-and-drop Kanban board UI.

**Status:** 📋 PLANNING

### Overview

Phase 3 introduces task management functionality:
- Tasks belong to projects (one-to-many relationship)
- State machine: TODO → IN_PROGRESS → AI_REVIEW → HUMAN_REVIEW → DONE
- Kanban board UI with drag-and-drop
- Real-time task updates via WebSocket
- Task detail panel for viewing/editing

---

### Task States & State Machine

```
┌──────┐     ┌─────────────┐     ┌───────────┐     ┌──────────────┐     ┌──────┐
│ TODO │────▶│ IN_PROGRESS │────▶│ AI_REVIEW │────▶│ HUMAN_REVIEW │────▶│ DONE │
└──────┘     └─────────────┘     └───────────┘     └──────────────┘     └──────┘
```

**Valid Transitions:**
- TODO → IN_PROGRESS (user starts work)
- IN_PROGRESS → AI_REVIEW (user requests AI execution)
- AI_REVIEW → HUMAN_REVIEW (AI completes, awaiting human review)
- AI_REVIEW → IN_PROGRESS (AI fails, user can retry)
- HUMAN_REVIEW → DONE (user approves)
- HUMAN_REVIEW → IN_PROGRESS (user requests changes)
- Any state → TODO (reset task)

---

### Backend Tasks (7 tasks)

#### 3.1 Database & Models
- [ ] **DB Migration**: Create `003_tasks.sql` migration
  - Add tasks table with all required fields
  - Foreign key to projects table (project_id)
  - State field with enum constraint
  - Index on (project_id, state) for filtering
  - Index on (project_id, position) for ordering
  - **Location:** `db/migrations/003_tasks.up.sql` + `003_tasks.down.sql`
  
- [ ] **Task Model**: Implement GORM model
  - UUID primary key
  - Belongs to Project (foreign key)
  - State field (TODO, IN_PROGRESS, AI_REVIEW, HUMAN_REVIEW, DONE)
  - Position field (integer, for ordering within columns)
  - Title, description, priority
  - Timestamps (created_at, updated_at)
  - Soft delete support (deleted_at)
  - **Location:** `backend/internal/model/task.go`

**Schema:**
```sql
CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    state VARCHAR(50) NOT NULL DEFAULT 'TODO',
    position INTEGER NOT NULL DEFAULT 0,
    priority VARCHAR(20) DEFAULT 'medium',
    assigned_to UUID REFERENCES users(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,
    CONSTRAINT tasks_state_check CHECK (state IN ('TODO', 'IN_PROGRESS', 'AI_REVIEW', 'HUMAN_REVIEW', 'DONE'))
);

CREATE INDEX idx_tasks_project_id ON tasks(project_id);
CREATE INDEX idx_tasks_project_state ON tasks(project_id, state);
CREATE INDEX idx_tasks_project_position ON tasks(project_id, position);
CREATE INDEX idx_tasks_deleted_at ON tasks(deleted_at);
```

#### 3.2 Repository Layer
- [ ] **Task Repository**: Implement data access layer
  - `Create(ctx, task *Task) error` - Create new task
  - `FindByID(ctx, id uuid.UUID) (*Task, error)` - Get task by ID
  - `FindByProjectID(ctx, projectID uuid.UUID) ([]Task, error)` - List project's tasks
  - `Update(ctx, task *Task) error` - Update task
  - `UpdateState(ctx, id uuid.UUID, newState string) error` - Update task state
  - `UpdatePosition(ctx, id uuid.UUID, newPosition int) error` - Update task position
  - `SoftDelete(ctx, id uuid.UUID) error` - Soft delete task
  - Interface-based design for testability
  - Context-aware methods
  - **Location:** `backend/internal/repository/task_repository.go`
  - **Tests:** `backend/internal/repository/task_repository_test.go` (target: 10+ tests)

#### 3.3 Business Logic Layer
- [ ] **Task Service**: Implement business logic
  - `CreateTask(projectID, userID uuid.UUID, title, description string, priority string) (*Task, error)`
    - Validate input (title required, max lengths)
    - Check user owns project
    - Set initial state to TODO
    - Set position (append to TODO column)
  - `GetTask(id, userID uuid.UUID) (*Task, error)` - Authorization check
  - `ListProjectTasks(projectID, userID uuid.UUID) ([]Task, error)` - Fetch project's tasks
  - `UpdateTask(id, userID uuid.UUID, updates map[string]interface{}) (*Task, error)` - Selective field updates
  - `MoveTask(id, userID uuid.UUID, newState string, newPosition int) (*Task, error)`
    - Validate state transition (use state machine)
    - Update position within new column
    - Reorder other tasks if needed
  - `DeleteTask(id, userID uuid.UUID) error` - Soft delete
  - **State Machine Validation** helper
  - Input validation helpers
  - **Location:** `backend/internal/service/task_service.go`
  - **Tests:** `backend/internal/service/task_service_test.go` (target: 20+ tests)

**State Machine Validation:**
```go
var validTransitions = map[string][]string{
    "TODO":         {"IN_PROGRESS"},
    "IN_PROGRESS":  {"AI_REVIEW", "TODO"},
    "AI_REVIEW":    {"HUMAN_REVIEW", "IN_PROGRESS"},
    "HUMAN_REVIEW": {"DONE", "IN_PROGRESS"},
    "DONE":         {"TODO"}, // Allow reopening
}

func isValidTransition(currentState, newState string) bool {
    allowed, exists := validTransitions[currentState]
    if !exists {
        return false
    }
    for _, s := range allowed {
        if s == newState {
            return true
        }
    }
    return false
}
```

#### 3.4 API Handlers
- [ ] **Task API Endpoints**: Implement HTTP handlers
  - `POST /api/projects/:projectId/tasks` - Create task (protected)
  - `GET /api/projects/:projectId/tasks` - List project's tasks (protected)
  - `GET /api/projects/:projectId/tasks/:id` - Get task details (protected)
  - `PATCH /api/projects/:projectId/tasks/:id` - Update task (protected)
  - `PATCH /api/projects/:projectId/tasks/:id/move` - Move task (state + position) (protected)
  - `DELETE /api/projects/:projectId/tasks/:id` - Delete task (protected)
  - Request validation (bind JSON + service-level validation)
  - Error handling with proper status codes
  - Authorization checks (user owns project)
  - **Location:** `backend/internal/api/tasks.go`
  - **Tests:** `backend/internal/api/tasks_test.go` (target: 15+ tests)

- [ ] **WebSocket Task Updates**: Real-time task state changes
  - `GET /api/projects/:projectId/tasks/stream` - WebSocket endpoint for task updates
  - Broadcast task create/update/delete events to all connected clients
  - Authorization check (user owns project)
  - **Location:** `backend/internal/api/tasks.go` (extend)

#### 3.5 Integration
- [ ] **Register Routes**: Wire up task endpoints
  - Add task routes to Gin router (nested under projects)
  - Apply auth middleware to all task routes
  - Initialize TaskService with TaskRepository
  - Create TaskHandler with dependency injection
  - **Location:** `backend/cmd/api/main.go` (modify)

#### 3.6 Testing
- [ ] **Unit Tests**: Test core logic
  - TaskRepository CRUD operations (10+ tests)
  - TaskService business logic (20+ tests)
  - TaskHandler API endpoints (15+ tests)
  - Mock-based testing for clean isolation
  - **Target:** 45+ total task-related tests

- [ ] **Integration Test**: End-to-end task management
  - Create task via API
  - Move task through states (TODO → IN_PROGRESS → AI_REVIEW → HUMAN_REVIEW → DONE)
  - Verify state machine validation (reject invalid transitions)
  - Delete task
  - **Location:** `backend/internal/api/tasks_integration_test.go`

---

### Frontend Tasks (6 tasks)

#### 3.7 Types & API Client
- [ ] **Task Types**: Define TypeScript interfaces
  - `TaskState` type: `'TODO' | 'IN_PROGRESS' | 'AI_REVIEW' | 'HUMAN_REVIEW' | 'DONE'`
  - `TaskPriority` type: `'low' | 'medium' | 'high'`
  - `CreateTaskRequest` interface (title, description?, priority?)
  - `UpdateTaskRequest` interface (all fields optional)
  - `MoveTaskRequest` interface (state, position)
  - `Task` interface (id, project_id, title, description, state, position, priority, created_at, updated_at)
  - **Location:** `frontend/src/types/index.ts` (extend)

- [ ] **Task API Client**: Implement API methods
  - `createTask(projectId: string, data: CreateTaskRequest): Promise<Task>`
  - `getTasks(projectId: string): Promise<Task[]>`
  - `getTask(projectId: string, taskId: string): Promise<Task>`
  - `updateTask(projectId: string, taskId: string, data: UpdateTaskRequest): Promise<Task>`
  - `moveTask(projectId: string, taskId: string, data: MoveTaskRequest): Promise<Task>`
  - `deleteTask(projectId: string, taskId: string): Promise<void>`
  - **Location:** `frontend/src/services/api.ts` (extend)

#### 3.8 Kanban Board Components
- [ ] **KanbanBoard Component**: Main board container
  - Fetch tasks on mount using `getTasks()` API
  - Group tasks by state (5 columns: TODO, IN_PROGRESS, AI_REVIEW, HUMAN_REVIEW, DONE)
  - Drag-and-drop context provider (`@dnd-kit/core`)
  - Handle drag end → call `moveTask()` API
  - Optimistic updates
  - Loading and error states
  - **Location:** `frontend/src/components/Kanban/KanbanBoard.tsx`

- [ ] **KanbanColumn Component**: Single column (e.g., "TODO")
  - Display column title and task count
  - Droppable zone for tasks
  - Vertical scrolling for many tasks
  - "Add Task" button (opens CreateTaskModal)
  - **Location:** `frontend/src/components/Kanban/KanbanColumn.tsx`

- [ ] **TaskCard Component**: Single task display
  - Draggable card with task title
  - Priority indicator (color-coded: high=red, medium=yellow, low=green)
  - Click card → open TaskDetailPanel
  - Compact design for board view
  - **Location:** `frontend/src/components/Kanban/TaskCard.tsx`

#### 3.9 Task Detail & Forms
- [ ] **TaskDetailPanel Component**: Sliding panel for task details
  - Display full task metadata (title, description, state, priority, timestamps)
  - Edit mode (inline or modal)
  - Delete task button with confirmation
  - Close button (slide out)
  - **Location:** `frontend/src/components/Kanban/TaskDetailPanel.tsx`

- [ ] **CreateTaskModal Component**: Task creation form
  - Form fields: title (required), description (optional), priority (dropdown)
  - Client-side validation (title required, max 255 chars)
  - Submit → call API → close modal → refresh board
  - Cancel button
  - **Location:** `frontend/src/components/Kanban/CreateTaskModal.tsx`

#### 3.10 Real-time Updates
- [ ] **WebSocket Hook**: Task update subscription
  - `useTaskUpdates(projectId: string)` hook
  - Connect to `ws://localhost:8090/api/projects/:projectId/tasks/stream`
  - Listen for task create/update/delete events
  - Update local state on message
  - Cleanup on unmount
  - Auto-reconnect logic
  - **Location:** `frontend/src/hooks/useTaskUpdates.ts`

- [ ] **Integrate WebSocket in KanbanBoard**
  - Use `useTaskUpdates` hook
  - Merge WebSocket updates with local task state
  - Real-time updates across browser tabs/users
  - **Location:** `frontend/src/components/Kanban/KanbanBoard.tsx` (modify)

#### 3.11 Routes & Navigation
- [ ] **Update ProjectDetailPage**: Add tasks section
  - Replace "Tasks" placeholder with link to Kanban board
  - Navigate to `/projects/:id/tasks` on click
  - **Location:** `frontend/src/pages/ProjectDetailPage.tsx` (modify)

- [ ] **Add Task Routes**: Update router
  - `/projects/:id/tasks` → KanbanBoard page
  - Protected route wrapped in AppLayout
  - **Location:** `frontend/src/App.tsx` (modify)

---

## Success Criteria (Phase 3 Complete When...)

- [ ] **3.1 Database & Models Complete**
  - [ ] Migration `003_tasks.sql` created and applied
  - [ ] Task GORM model with state machine validation
  - [ ] Indexes on project_id, state, position, deleted_at

- [ ] **3.2 Repository Layer Complete**
  - [ ] TaskRepository interface with 7 methods
  - [ ] 10+ unit tests (all passing)
  - [ ] Context-aware methods for cancellation/timeout

- [ ] **3.3 Business Logic Layer Complete**
  - [ ] TaskService with 6 core methods
  - [ ] State machine validation implemented
  - [ ] Input validation helpers
  - [ ] 20+ unit tests (all passing)

- [ ] **3.4 API Handlers Complete**
  - [ ] 6 CRUD endpoints + WebSocket endpoint
  - [ ] Request/Response DTOs with validation
  - [ ] 15+ unit tests (all passing)

- [ ] **3.5 Integration Complete**
  - [ ] Routes wired up in main.go
  - [ ] TaskService initialized with dependencies

- [ ] **3.6 Testing Complete**
  - [ ] 45+ task-related unit tests (all passing)
  - [ ] Integration test for complete task lifecycle

- [ ] **3.7 Types & API Client Complete**
  - [ ] TypeScript interfaces for tasks
  - [ ] 6 API client methods implemented

- [ ] **3.8 Kanban Board Components Complete**
  - [ ] KanbanBoard with drag-and-drop
  - [ ] KanbanColumn component
  - [ ] TaskCard component

- [ ] **3.9 Task Detail & Forms Complete**
  - [ ] TaskDetailPanel for viewing/editing
  - [ ] CreateTaskModal with validation

- [ ] **3.10 Real-time Updates Complete**
  - [ ] WebSocket hook for task updates
  - [ ] Integration in KanbanBoard

- [ ] **3.11 Routes & Navigation Complete**
  - [ ] ProjectDetailPage updated with tasks link
  - [ ] Task routes added to App.tsx

- [ ] **Manual E2E Testing**
  - [ ] User can create tasks
  - [ ] User can drag tasks between columns
  - [ ] State machine validation working (invalid transitions rejected)
  - [ ] Task detail panel shows/edits task
  - [ ] Real-time updates work across browser tabs
  - [ ] User can delete tasks

---

## Phase 3 Dependencies

**Required Before Starting:**
- ✅ Phase 2 complete (project management working)
- ✅ PostgreSQL running
- ✅ Kubernetes cluster accessible (kind or other)

**External Dependencies:**
- React drag-and-drop library: `@dnd-kit/core` + `@dnd-kit/sortable`
- No additional Go dependencies needed (uses existing Gin, GORM, WebSocket)

---

## Deferred to Later Phases

**Not in Phase 3 scope:**
- Task assignment to specific users (add in Phase 7 if needed)
- Task dependencies / subtasks (future enhancement)
- Task comments / activity log (future enhancement)
- File attachments to tasks (Phase 4 integration)
- AI execution from tasks (Phase 5)

---

## Notes & Considerations

### Drag-and-Drop Library
- **Recommended:** `@dnd-kit/core` (modern, accessible, TypeScript support)
- **Alternative:** `react-beautiful-dnd` (older, widely used)
- **Features needed:** Vertical reordering within columns, moving between columns

### Position Management
- **Strategy:** Integer position field (0, 1, 2, ...)
- **On drag:** Update dragged task's position, reorder other tasks in affected columns
- **Optimization:** Batch position updates if needed (future)

### State Machine Enforcement
- **Backend validation:** Reject invalid state transitions in TaskService
- **Frontend UX:** Disable invalid drag targets (e.g., can't drag from TODO to DONE)
- **Error handling:** Show user-friendly message if transition rejected

### Real-time Updates
- **WebSocket events:** `task.created`, `task.updated`, `task.deleted`, `task.moved`
- **Payload:** Full task object (id, project_id, title, state, position, etc.)
- **Merge strategy:** Replace existing task in local state by ID

### Performance
- **Task count:** Assume <100 tasks per project for MVP (no pagination needed)
- **WebSocket:** Single connection per project, broadcast to all connected clients
- **Optimistic updates:** Update UI immediately, rollback on API error

---

## Next Phase Preview

**Phase 4: File Explorer (Weeks 7-8)**

### Objectives
- File browser sidecar integration
- File tree component
- Monaco editor for code editing
- Multi-file support with tabs

### Key Features
- Browse project workspace files
- Edit files with syntax highlighting
- Save changes to workspace
- Real-time file sync across tabs

---

**Phase 3 Start Date:** TBD  
**Target Completion:** TBD (flexible, 3-developer team)  
**Author:** Sisyphus (OpenCode AI Agent)

---

**Last Updated:** 2026-01-18 21:42 CET
