# Project Improvements & Future Enhancements

**Last Updated**: 2026-01-16  
**Project**: OpenCode Project Manager  
**Current Phase**: Phase 1 (OIDC Authentication) Complete

---

## Overview

This document tracks optional improvements and enhancements that could be implemented to further improve code quality, test coverage, and developer experience. All items listed here are **non-blocking** - the project is production-ready in its current state.

---

## Frontend Testing Improvements

**Current Status**: ✅ 83.51% overall test coverage (exceeds 80% target)

### 1. API Interceptor Tests

**Impact**: Increase `api.ts` coverage from 60% to ~90%  
**Effort**: Low (2-3 hours)  
**Priority**: Medium

**What to Test**:
```typescript
// src/services/api.test.ts

describe('API Interceptors', () => {
  describe('Request Interceptor', () => {
    it('adds Authorization header when token exists in localStorage')
    it('does not add Authorization header when no token')
    it('preserves existing headers')
  })

  describe('Response Interceptor', () => {
    it('passes through successful responses unchanged')
    it('clears token and redirects on 401 error')
    it('clears localStorage on 401')
    it('rejects promise with error for non-401 errors')
  })
})
```

**Implementation Approach**:
- Use `vi.spyOn(localStorage, 'getItem')` to mock token presence
- Use `vi.spyOn(localStorage, 'removeItem')` to verify token clearing
- Mock `window.location.href` to verify redirect
- Use axios mock adapter for response testing

**Files to Modify**:
- `/home/npinot/vibe/frontend/src/services/api.test.ts`

**Dependencies**:
- `axios-mock-adapter` (may need to install)

---

### 2. AuthContext Async Flow Tests

**Impact**: Increase `AuthContext.tsx` coverage from 73.73% to ~85%  
**Effort**: Medium (4-6 hours)  
**Priority**: Medium

**What to Test**:
```typescript
// src/contexts/AuthContext.test.tsx

describe('AuthContext - Async Flows', () => {
  describe('login()', () => {
    it('fetches OIDC login URL from backend')
    it('redirects to Keycloak authorization URL')
    it('handles API errors gracefully')
    it('logs errors to console')
  })

  describe('handleCallback()', () => {
    it('exchanges authorization code for JWT token')
    it('stores token in localStorage')
    it('fetches user data from /auth/me endpoint')
    it('updates user state with fetched data')
    it('handles token exchange failure')
    it('handles user fetch failure')
    it('clears state on error')
  })

  describe('Auto-login on mount', () => {
    it('validates existing token by calling /auth/me')
    it('sets user data if token is valid')
    it('clears invalid token from localStorage')
    it('sets isLoading to false after validation')
  })
})
```

**Implementation Approach**:
- Use `vi.mock('./services/api')` to mock axios calls
- Mock `api.get()` and `api.post()` with different responses
- Use `waitFor()` for async state updates
- Test both success and error paths

**Uncovered Lines** (from coverage report):
- Lines 41-44: `login()` function
- Lines 53-60: `handleCallback()` function  
- Lines 63-76: `useEffect` auto-login logic

**Files to Modify**:
- `/home/npinot/vibe/frontend/src/contexts/AuthContext.test.tsx`

---

### 3. App.tsx Route Tests

**Impact**: Increase `App.tsx` coverage from 75.6% to ~90%  
**Effort**: Low (2-3 hours)  
**Priority**: Low

**What to Test**:
```typescript
// src/App.test.tsx

describe('App - Route Rendering', () => {
  it('renders ProjectsPage at /projects route')
  it('renders ProjectDetailPage at /projects/:id route')
  it('requires authentication for /projects')
  it('requires authentication for /projects/:id')
  it('redirects unknown routes to home page')
  it('renders home page at / route')
})
```

**Implementation Approach**:
- Use `render(<App />)` with no wrapper (App has own router)
- Navigate to routes using `window.history.pushState()` or by rendering with initial entries
- Verify protected routes redirect to login when unauthenticated
- Verify protected routes render when authenticated

**Uncovered Lines** (from coverage report):
- Lines 60-69: `ProjectsPage` component
- Lines 71-80: `ProjectDetailPage` component

**Files to Modify**:
- `/home/npinot/vibe/frontend/src/App.test.tsx`

---

## Backend Testing Improvements

**Current Status**: ⚠️ Minimal test coverage

### 4. Auth Service Unit Tests

**Impact**: Establish baseline test coverage for critical auth logic  
**Effort**: High (8-10 hours)  
**Priority**: High

**What to Test**:
```go
// backend/internal/service/auth_service_test.go

func TestAuthService_GetOIDCLoginURL(t *testing.T) {
    // Test OIDC authorization URL generation
    // Test state parameter generation
    // Test URL encoding
}

func TestAuthService_HandleOIDCCallback(t *testing.T) {
    // Test authorization code exchange
    // Test JWT token generation
    // Test user creation/update via repository
    // Test error handling for invalid code
    // Test error handling for OIDC provider errors
}

func TestAuthService_ValidateToken(t *testing.T) {
    // Test valid JWT validation
    // Test expired token rejection
    // Test invalid signature rejection
    // Test missing claims rejection
}
```

**Implementation Approach**:
- Mock OIDC provider using `httptest.Server`
- Use `github.com/stretchr/testify/mock` for repository mocking
- Test both success and error paths
- Verify JWT claims structure

**Files to Create**:
- `/home/npinot/vibe/backend/internal/service/auth_service_test.go`

**Current Files**:
- Existing test file present but incomplete

---

### 5. User Repository Tests

**Impact**: Ensure database operations work correctly  
**Effort**: Medium (4-6 hours)  
**Priority**: Medium

**What to Test**:
```go
// backend/internal/repository/user_repository_test.go

func TestUserRepository_FindByOIDCSubject(t *testing.T) {
    // Test finding existing user
    // Test user not found returns nil
    // Test database error handling
}

func TestUserRepository_CreateUser(t *testing.T) {
    // Test user creation with valid data
    // Test duplicate OIDC subject constraint
    // Test required field validation
}

func TestUserRepository_UpdateUser(t *testing.T) {
    // Test updating existing user
    // Test updating non-existent user
}

func TestUserRepository_UpsertByOIDCSubject(t *testing.T) {
    // Test insert when user doesn't exist
    // Test update when user exists
    // Test concurrent upsert handling
}
```

**Implementation Approach**:
- Use `sqlmock` for database mocking
- Test SQL query generation
- Verify GORM behaviors
- Test transaction handling

**Files to Create/Modify**:
- `/home/npinot/vibe/backend/internal/repository/user_repository_test.go` (already exists, expand)

---

### 6. Auth Middleware Tests

**Impact**: Verify JWT validation in HTTP layer  
**Effort**: Low (2-3 hours)  
**Priority**: Medium

**What to Test**:
```go
// backend/internal/middleware/auth_test.go

func TestAuthMiddleware_ValidToken(t *testing.T) {
    // Test request proceeds with valid token
    // Test user context is set
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
    // Test 401 response
    // Test error message format
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
    // Test malformed token rejection
    // Test expired token rejection
    // Test invalid signature rejection
}

func TestAuthMiddleware_HeaderFormats(t *testing.T) {
    // Test "Bearer <token>" format
    // Test missing "Bearer" prefix rejection
}
```

**Implementation Approach**:
- Use `httptest.NewRecorder()` for response testing
- Create valid/invalid JWT tokens for testing
- Verify HTTP status codes
- Verify response body structure

**Files to Create/Modify**:
- `/home/npinot/vibe/backend/internal/middleware/auth_test.go` (already exists, expand)

---

## Infrastructure Improvements

### 7. CI/CD Pipeline

**Impact**: Automate testing and deployment  
**Effort**: High (1-2 days)  
**Priority**: High

**What to Implement**:
```yaml
# .github/workflows/ci.yml

name: CI Pipeline

on: [push, pull_request]

jobs:
  backend-tests:
    - Run: go test ./...
    - Run: go vet ./...
    - Run: golangci-lint run
    - Upload coverage to Codecov
  
  frontend-tests:
    - Run: npm test -- --run
    - Run: npm run lint
    - Run: npm run build
    - Upload coverage to Codecov
  
  docker-build:
    - Build production image
    - Run security scan
    - Push to registry (on main branch)
```

**Files to Create**:
- `.github/workflows/ci.yml`
- `.github/workflows/deploy.yml`

**Status**: Deferred to Phase 9 (see IMPLEMENTATION_PLAN.md)

---

### 8. E2E Testing Setup

**Impact**: Catch integration issues before production  
**Effort**: High (2-3 days)  
**Priority**: Medium

**What to Test**:
```typescript
// e2e/auth-flow.spec.ts

describe('Authentication Flow E2E', () => {
  it('complete login flow via Keycloak')
  it('token refresh on expiration')
  it('logout clears session')
  it('protected routes redirect to login')
})

// e2e/project-management.spec.ts

describe('Project Management E2E', () => {
  it('create project creates Kubernetes pod')
  it('delete project cleans up resources')
  it('task execution shows real-time output')
})
```

**Tools to Use**:
- Playwright (already referenced in docs)
- Docker Compose for test environment
- Keycloak test container

**Files to Create**:
- `/home/npinot/vibe/e2e/` directory structure
- `playwright.config.ts`

**Status**: Deferred to Phase 9

---

## Code Quality Improvements

### 9. TypeScript Strict Mode Fixes

**Impact**: Catch type errors at compile time  
**Effort**: Medium (4-6 hours)  
**Priority**: Low

**Current Issues**:
```
src/contexts/AuthContext.test.tsx(24,17): Property 'mockResolvedValue' does not exist
src/hooks/useAuth.test.ts(25,38): No overload matches this call
src/tests/setupTests.ts(47,1): Cannot find name 'global'
```

**What to Fix**:
- Add proper type definitions for mocked functions
- Fix `global` reference in setupTests.ts (use `globalThis`)
- Add `@types/node` for global types if needed

**Files to Modify**:
- `/home/npinot/vibe/frontend/src/contexts/AuthContext.test.tsx`
- `/home/npinot/vibe/frontend/src/hooks/useAuth.test.ts`
- `/home/npinot/vibe/frontend/src/tests/setupTests.ts`

---

### 10. Linter Configuration Enhancements

**Impact**: Enforce consistent code style  
**Effort**: Low (1-2 hours)  
**Priority**: Low

**What to Add**:

**Frontend (ESLint)**:
```json
{
  "rules": {
    "no-console": ["warn", { "allow": ["error"] }],
    "prefer-const": "error",
    "no-var": "error",
    "@typescript-eslint/explicit-function-return-type": "warn"
  }
}
```

**Backend (golangci-lint)**:
```yaml
# .golangci.yml
linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused
    - gocyclo
    - gofmt
```

**Files to Create/Modify**:
- `/home/npinot/vibe/frontend/.eslintrc.json` (update rules)
- `/home/npinot/vibe/.golangci.yml` (create)

---

## Performance Improvements

### 11. Frontend Bundle Optimization

**Impact**: Reduce initial load time  
**Effort**: Low (2-3 hours)  
**Priority**: Low

**What to Implement**:
- Code splitting for routes
- Lazy loading for heavy components
- Tree shaking verification
- Bundle size analysis

**Files to Modify**:
- `/home/npinot/vibe/frontend/vite.config.ts`
- `/home/npinot/vibe/frontend/src/App.tsx` (add lazy loading)

**Commands to Add**:
```json
{
  "scripts": {
    "analyze": "vite-bundle-visualizer"
  }
}
```

---

### 12. Docker Image Optimization

**Impact**: Faster deployments, smaller registry footprint  
**Effort**: Medium (3-4 hours)  
**Priority**: Low

**Current Size**: 29MB (production unified image) - already excellent!

**Potential Further Optimizations**:
- Multi-stage build review (already implemented)
- Alpine base image usage (already implemented)
- Layer caching optimization
- Remove unnecessary build dependencies

**Files to Review**:
- `/home/npinot/vibe/Dockerfile`
- `/home/npinot/vibe/backend/Dockerfile`
- `/home/npinot/vibe/frontend/Dockerfile`

**Note**: Current 29MB is already exceptional. Further optimization has diminishing returns.

---

## Documentation Improvements

### 13. API Documentation Generation

**Impact**: Easier API consumption for frontend developers  
**Effort**: Medium (4-6 hours)  
**Priority**: Medium

**What to Implement**:
- OpenAPI/Swagger specification
- Auto-generated from Go code comments
- Interactive API explorer UI

**Tools**:
- `swaggo/swag` for Go annotation parsing
- Swagger UI for documentation serving

**Files to Create**:
- `/home/npinot/vibe/docs/swagger.yaml`
- API endpoint annotations in handler files

**Status**: Mentioned in TODO.md, not yet implemented

---

### 14. Architecture Decision Records (ADRs)

**Impact**: Document key technical decisions  
**Effort**: Low (1-2 hours)  
**Priority**: Low

**What to Document**:
- ADR-001: Choice of OIDC for authentication
- ADR-002: Unified Docker image for production
- ADR-003: Kind for local Kubernetes development
- ADR-004: React Router over alternative routing solutions

**Files to Create**:
- `/home/npinot/vibe/docs/adr/` directory
- Individual ADR markdown files

---

## Security Improvements

### 15. Security Headers Middleware

**Impact**: Prevent common web vulnerabilities  
**Effort**: Low (1-2 hours)  
**Priority**: Medium

**What to Implement**:
```go
// backend/internal/middleware/security.go

func SecurityHeaders() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("X-Content-Type-Options", "nosniff")
        c.Header("X-Frame-Options", "DENY")
        c.Header("X-XSS-Protection", "1; mode=block")
        c.Header("Strict-Transport-Security", "max-age=31536000")
        c.Header("Content-Security-Policy", "default-src 'self'")
        c.Next()
    }
}
```

**Files to Create/Modify**:
- `/home/npinot/vibe/backend/internal/middleware/security.go` (already exists)
- `/home/npinot/vibe/backend/cmd/api/main.go` (add middleware)

---

### 16. Dependency Vulnerability Scanning

**Impact**: Catch vulnerable dependencies early  
**Effort**: Low (1-2 hours)  
**Priority**: High

**What to Implement**:

**Frontend**:
```bash
npm audit
npm audit fix
```

**Backend**:
```bash
go list -m all | nancy sleuth
```

**CI Integration**:
```yaml
- name: Security Scan
  run: |
    npm audit --audit-level=moderate
    go install github.com/sonatype-nexus-community/nancy@latest
    go list -m all | nancy sleuth
```

**Files to Modify**:
- `.github/workflows/ci.yml`

**Current Status**: 7 vulnerabilities detected in frontend (6 moderate, 1 critical) - should be addressed

---

## Summary

### High Priority (Should Do)
1. ✅ Backend Auth Service Unit Tests (establishes test baseline)
2. ✅ CI/CD Pipeline (automates quality checks)
3. ✅ Dependency Vulnerability Scanning (security)

### Medium Priority (Nice to Have)
1. API Interceptor Tests (frontend coverage)
2. AuthContext Async Flow Tests (frontend coverage)
3. User Repository Tests (backend coverage)
4. Auth Middleware Tests (backend coverage)
5. Security Headers Middleware (security hardening)
6. API Documentation Generation (developer experience)
7. E2E Testing Setup (integration confidence)

### Low Priority (Optional)
1. App.tsx Route Tests (marginal coverage increase)
2. TypeScript Strict Mode Fixes (type safety)
3. Linter Configuration Enhancements (code quality)
4. Frontend Bundle Optimization (already fast)
5. Docker Image Optimization (already optimized)
6. Architecture Decision Records (documentation)

---

## Progress Tracking

| Item | Status | Completion Date | Notes |
|------|--------|----------------|-------|
| Frontend Test Infrastructure | ✅ Complete | 2026-01-16 | 83.51% coverage achieved |
| ProtectedRoute Tests | ✅ Complete | 2026-01-16 | All 5 tests passing |
| API Interceptor Tests | 📋 Planned | - | Would increase api.ts to ~90% |
| AuthContext Async Tests | 📋 Planned | - | Would increase AuthContext to ~85% |
| Backend Auth Service Tests | 📋 Planned | - | Critical for Phase 2 |
| CI/CD Pipeline | 📋 Planned | - | Deferred to Phase 9 |
| E2E Testing | 📋 Planned | - | Deferred to Phase 9 |

---

**Last Updated**: 2026-01-16  
**Next Review**: Before Phase 2 kickoff

**Note**: This document is a living guide. Prioritization should be revisited as project needs evolve.
