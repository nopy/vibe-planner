# FRONTEND AGENT KNOWLEDGE BASE (React/TS)

## OVERVIEW
React 18 + TypeScript SPA powered by Vite, providing the project management dashboard.

**Phase 1 Status**: ✅ OIDC Authentication Complete

## STRUCTURE
- `src/App.tsx`: Central router with AuthProvider and protected routes
- `src/services/api.ts`: Axios-based API client with JWT interceptors
- `src/types/index.ts`: Single source of truth for all shared TypeScript interfaces
- `src/contexts/AuthContext.tsx`: ✅ Global auth state (login, callback, logout)
- `src/hooks/useAuth.ts`: ✅ Auth hook for components
- `src/pages/LoginPage.tsx`: ✅ Login UI with Keycloak integration
- `src/pages/OidcCallbackPage.tsx`: ✅ OIDC redirect handler
- `src/components/ProtectedRoute.tsx`: ✅ Route protection wrapper
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

### Environment Variables (VITE_* prefix required)
```bash
VITE_API_URL=http://localhost:8090
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
