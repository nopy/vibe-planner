# FRONTEND AGENT KNOWLEDGE BASE (React/TS)

## OVERVIEW
React 18 + TypeScript SPA powered by Vite, providing the project management dashboard.

**Phase 1 Status**: ✅ OIDC Authentication Complete  
**Phase 2.8 Status**: ✅ Project Types & API Client Complete  
**Phase 2.9 Status**: ✅ UI Components Complete  
**Phase 2.10 Status**: ✅ Real-time Updates Complete  
**Phase 2.11 Status**: ✅ Routes & Navigation Complete  
**Phase 3.7 Status**: ✅ Task Types & API Client Complete

## STRUCTURE
- `src/App.tsx`: Central router with AuthProvider and protected routes
- `src/services/api.ts`: ✅ Axios client with JWT + Project & Task API methods
- `src/types/index.ts`: ✅ TypeScript interfaces (User, Project, Task with Kanban fields, PodStatus, etc.)
- `src/contexts/AuthContext.tsx`: ✅ Global auth state (login, callback, logout)
- `src/hooks/useAuth.ts`: ✅ Auth hook for components
- `src/pages/LoginPage.tsx`: ✅ Login UI with Keycloak integration
- `src/pages/OidcCallbackPage.tsx`: ✅ OIDC redirect handler
- `src/pages/ProjectDetailPage.tsx`: ✅ Full project detail view with metadata
- `src/components/ProtectedRoute.tsx`: ✅ Route protection wrapper
- `src/components/Projects/ProjectList.tsx`: ✅ Grid layout with project cards
- `src/components/Projects/ProjectCard.tsx`: ✅ Project card with status badges
- `src/components/Projects/CreateProjectModal.tsx`: ✅ Project creation form
- `src/components/AppLayout.tsx`: ✅ Navigation header with menu
- `src/hooks/useProjectStatus.ts`: ✅ WebSocket hook for real-time pod status updates
- `src/styles/index.css`: Tailwind CSS entry point
- `src/main.tsx`: Application entry point
- `src/vite-env.d.ts`: ✅ TypeScript environment definitions

## PHASE 1 IMPLEMENTATION (COMPLETE)

### Authentication Flow
1. User clicks "Login with Keycloak" → `AuthContext.login()` called
2. Backend `/api/auth/oidc/login` returns Keycloak authorization URL
3. Redirect to Keycloak login page
4. User authenticates with Keycloak
5. Keycloak redirects to `/auth/callback?code=...`
6. `OidcCallbackPage` extracts code and calls `AuthContext.handleCallback(code)`
7. Backend `/api/auth/oidc/callback` exchanges code for JWT
8. JWT stored in localStorage, user state updated
9. Redirect to `/projects` (protected route)

### Key Components

**AuthContext** (`src/contexts/AuthContext.tsx`):
- Manages: `user`, `token`, `isAuthenticated`, `isLoading`
- Methods: `login()`, `handleCallback(code)`, `logout()`
- Token stored in localStorage
- Auto-validates token on app mount via `/api/auth/me`

**ProtectedRoute** (`src/components/ProtectedRoute.tsx`):
- Wraps protected routes
- Shows loading spinner while checking auth
- Redirects to `/login` if not authenticated
- Preserves original location for post-login redirect

**API Client** (`src/services/api.ts`):
- Axios instance with base URL from env
- Request interceptor: Adds `Authorization: Bearer <token>`
- Response interceptor: 401 → clear token + redirect to login
- ✅ **Project API Methods** (Phase 2.8):
  - `createProject(data: CreateProjectRequest): Promise<Project>`
  - `getProjects(): Promise<Project[]>`
  - `getProject(id: string): Promise<Project>`
  - `updateProject(id: string, data: UpdateProjectRequest): Promise<Project>`
  - `deleteProject(id: string): Promise<void>`
- ✅ **Task API Methods** (Phase 3.7):
  - `listTasks(projectId: string): Promise<Task[]>`
  - `createTask(projectId: string, data: CreateTaskRequest): Promise<Task>`
  - `getTask(projectId: string, taskId: string): Promise<Task>`
  - `updateTask(projectId: string, taskId: string, data: UpdateTaskRequest): Promise<Task>`
  - `moveTask(projectId: string, taskId: string, data: MoveTaskRequest): Promise<Task>`
  - `deleteTask(projectId: string, taskId: string): Promise<void>`

## PHASE 2.9 IMPLEMENTATION (COMPLETE)

### UI Components

**ProjectList** (`src/components/Projects/ProjectList.tsx`):
- Fetches all user projects on mount
- Responsive grid layout (1/2/3 columns)
- Loading spinner and error states
- Empty state with call-to-action
- Integrates CreateProjectModal
- Optimistic updates on create/delete

**ProjectCard** (`src/components/Projects/ProjectCard.tsx`):
- Displays project name, description, status badge
- Color-coded status (Ready=green, Initializing=yellow, Error=red, Archived=gray)
- Formatted creation date
- Click to navigate to project detail
- Two-step delete confirmation

**CreateProjectModal** (`src/components/Projects/CreateProjectModal.tsx`):
- Modal form for project creation
- Fields: name (required), description, repo_url
- Client-side validation (matches backend rules)
- Real-time field errors
- Loading state during API call
- Auto-refresh project list on success

**ProjectDetailPage** (`src/pages/ProjectDetailPage.tsx`):
- Full project metadata display
- Kubernetes pod information (pod name, namespace, PVC)
- Status badge matching ProjectCard
- Real-time pod status updates via WebSocket (useProjectStatus hook)
- Connection indicator (green/red dot) and "Live" badge
- WebSocket error banner with reconnect button
- Breadcrumb navigation
- Delete with warning message
- Placeholder sections for future features (Tasks, Files, Config)

**useProjectStatus Hook** (`src/hooks/useProjectStatus.ts`):
- WebSocket connection to `/api/projects/:id/status`
- Auto-connect on mount, cleanup on unmount
- Automatic reconnection (max 5 attempts, 3-second delay)
- Connection state tracking and error handling
- Manual reconnect function
- Environment-configurable URL (`VITE_WS_URL`)

### Environment Variables (VITE_* prefix required)
```bash
VITE_API_URL=http://localhost:8090
VITE_WS_URL=ws://localhost:8090/api/projects
VITE_OIDC_AUTHORITY=http://localhost:8081/realms/opencode
VITE_OIDC_CLIENT_ID=opencode-app
VITE_OIDC_REDIRECT_URI=http://localhost:5173/auth/callback
```

## CONVENTIONS

### Import Ordering
1. React and standard hooks (e.g., `useState`, `useEffect`)
2. Third-party libraries (e.g., `axios`, `zustand`, `@dnd-kit`)
3. Local modules/components using `@/` path alias
4. CSS and type definitions

### TypeScript
- **Strict Mode**: Fully enabled in `tsconfig.json`. No `any` allowed.
- **Types**: Prefer `interface` for object definitions.
- **Location**: All domain models/shared types must reside in `src/types/index.ts`.

### Components & State
- **Functional Only**: Use functional components with hooks (no class components)
- **Hooks**: Custom hooks in `src/hooks/` (e.g., `useAuth`)
- **State**: React Context for auth state (AuthContext), zustand for future global state

### Linting & Styling
- **ESLint**: Strict zero-warning policy (`--max-warnings 0`) enforced in CI.
- **Styling**: Tailwind CSS utility classes only. 
- **Responsive**: Use `sm:`, `md:`, `lg:` prefixes for mobile-first design.

## COMMANDS
```bash
npm run dev           # Start Vite dev server on :5173
npm run lint          # Run ESLint (strict mode)
npm run format        # Run Prettier formatting
npm run build         # Production build (runs tsc + vite build)
npm test              # Run Vitest suite
npm test -- <path>    # Run specific test file
npm test -- --watch   # Run tests in watch mode
```

## GOTCHAS
- **Path Alias**: `@/` maps to `src/`. Ensure synchronization with `tsconfig.json`
- **Prettier**: `.prettierrc` configured with project defaults
- **API URL**: Backend runs on port 8090 (not 8080 due to port conflict)
- **Environment**: Vite env vars must have `VITE_` prefix to be accessible in browser
- **Phase 2.8 Complete**: Project types and API client ready for UI components
- **Phase 2.9 Complete**: All 4 UI components implemented (ProjectList, ProjectCard, CreateProjectModal, ProjectDetailPage)
- **Phase 2.10 Complete**: WebSocket hook for real-time pod status updates (useProjectStatus + integration)
- **Phase 2.11 Complete**: Navigation menu with AppLayout component (Projects link, user email, logout)
- **Phase 3.7 Complete**: Task types and API client implemented (TaskStatus, TaskPriority, 6 API methods)
- **Next Phase**: Phase 3.8 - Kanban Board UI components
