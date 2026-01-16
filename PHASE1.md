# Phase 1: OIDC Authentication - ARCHIVED

**Completion Date:** 2026-01-16 21:28 CET  
**Status:** ✅ COMPLETE - All E2E tests passing  
**Branch:** main

---

## Summary

Phase 1 implemented complete OIDC authentication flow with Keycloak integration. All critical infrastructure issues were resolved, and the foundation is ready for Phase 2 development.

**Implementation Highlights:**
- OIDC provider integration (go-oidc v3)
- JWT token generation and validation (HS256)
- User repository with OIDC upsert logic
- Auth service layer with middleware
- Complete React auth flow (AuthContext, hooks, protected routes)
- E2E testing with Playwright verification

---

## Completed Tasks (21/28)

### Backend ✅
- [x] OIDC provider integration (go-oidc v3)
- [x] JWT token generation and validation (HS256)
- [x] User repository with OIDC upsert logic
- [x] Auth service layer
- [x] Auth middleware for protected routes
- [x] All auth endpoints implemented:
  - `GET /api/auth/oidc/login` - Get Keycloak authorization URL
  - `GET /api/auth/oidc/callback` - Exchange code for JWT
  - `GET /api/auth/me` - Get current user (protected)
  - `POST /api/auth/logout` - Logout

### Frontend ✅
- [x] AuthContext for global auth state
- [x] useAuth hook
- [x] LoginPage component
- [x] OidcCallbackPage component
- [x] ProtectedRoute wrapper
- [x] App.tsx integration with route protection

### Infrastructure ✅
- [x] Keycloak realm `opencode` created
- [x] Keycloak client `opencode-app` configured
- [x] PostgreSQL running with `users` table migrated
- [x] Backend server running on port 8090
- [x] All linting passes (ESLint + TypeScript)
- [x] Backend compiles successfully

---

## Critical Bugs Fixed During E2E Testing

### 1. Database Column Mismatch (FIXED ✅)
**Issue:** GORM auto-migration created column `o_id_c_subject` instead of `oidc_subject`  
**Impact:** Callback endpoint returned 401 - "column oidc_subject does not exist"  
**Fix:** 
- Renamed column: `ALTER TABLE users RENAME COLUMN o_id_c_subject TO oidc_subject`
- Added explicit GORM tags: `gorm:"column:oidc_subject"` to User model
**Root Cause:** GORM's automatic snake_case conversion conflicts with migration file

### 2. User Repository Error Handling (FIXED ✅)
**Issue:** `CreateOrUpdateFromOIDC` string-compared error instead of using `errors.Is()`  
**Impact:** New users couldn't be created, callback always returned 401  
**Fix:** Changed to `errors.Is(err, gorm.ErrRecordNotFound)` + import "errors" package  
**Location:** `backend/internal/repository/user_repository.go:70`

### 3. Frontend API URL Configuration (FIXED ✅)
**Issue:** Frontend hardcoded to `http://localhost:8080` but backend runs on port 8090  
**Fix:** Created `frontend/.env.local` with `VITE_API_URL=http://localhost:8090`  
**Root Cause:** Port conflict with SearXNG service

### 4. Keycloak Test User (CREATED ✅)
- Username: `testuser`
- Password: `testpass123`
- Email: `testuser@example.com`
- Created via: `docker exec` commands to Keycloak admin CLI

### 5. Backend Environment Loading (FIXED ✅)
**Issue:** `godotenv.Load()` only looks for `.env` in `backend/` directory, but file is in project root  
**Impact:** Backend failed to start - "unsupported protocol scheme" error (OIDC_ISSUER not loaded)  
**Fix:** Updated `backend/cmd/api/main.go` to load from `../.env` first, then fallback to current directory  
**Location:** `backend/cmd/api/main.go:19-26`

### 6. React StrictMode Double Code Exchange (FIXED ✅)
**Issue:** Keycloak reported "Code already used" error (CODE_TO_TOKEN_ERROR) causing 403 responses  
**Impact:** OAuth callback failed after successful Keycloak login  
**Root Cause:** React.StrictMode in dev mode double-invokes useEffect, causing OidcCallbackPage to exchange the authorization code twice  
**Fix:** Added `useRef` guard in `OidcCallbackPage.tsx` to prevent duplicate code exchange  
**Location:** `frontend/src/pages/OidcCallbackPage.tsx:10-17`

---

## E2E Test Results (7/7 PASSING ✅)

**Test Suite:** Automated via Playwright + curl  
**Completion Date:** 2026-01-16 21:28 CET

### ✅ Test 1: Navigate to Frontend
- Page loaded successfully at http://localhost:5173
- Landing page displayed with "Get Started" button
- No console errors

### ✅ Test 2: Login Page Navigation
- Clicking "Get Started" → redirects to /login
- Login page displays with "Login with Keycloak" button
- No errors

### ✅ Test 3: Backend OIDC Login Endpoint
- `GET /api/auth/oidc/login` returns valid Keycloak authorization URL
- URL contains correct client_id, redirect_uri, scopes, and state parameter

### ✅ Test 4: Backend Health & Protected Endpoints
- `GET /healthz` returns 200 with `{"status":"ok"}`
- `GET /ready` returns 200 with `{"status":"ready"}`
- `GET /api/auth/me` (without token) returns 401 with proper error message

### ✅ Test 5: Database Schema Verification
- `users` table has correct column `oidc_subject` (not `o_id_c_subject`)
- All expected columns present: id, oidc_subject, email, name, picture_url, last_login_at
- Unique index on oidc_subject exists

### ✅ Test 6: Code Quality Verification
- Backend compiles without errors
- User repository uses correct `errors.Is()` pattern
- User model has explicit GORM column tags
- Backend loads .env from correct location

### ✅ Test 7: Complete OAuth Flow E2E
- ✓ Login button triggers API call to `/api/auth/oidc/login`
- ✓ Redirects to Keycloak login page
- ✓ User authentication with testuser/testpass123
- ✓ Redirect to `/auth/callback?code=...`
- ✓ Code exchange returns 200 (no duplicate exchange error)
- ✓ JWT stored in localStorage
- ✓ Redirect to `/projects` page
- ✓ User created in database with correct OIDC claims

**Test User Verified in Database:**
```
ID: 53bf0971-6915-4858-92eb-233c74f134cc
OIDC Subject: 5afce404-06b9-4400-80f1-8aed9bbb621b
Email: testuser@example.com
Name: Test User
Created: 2026-01-16 20:41:44 UTC
```

---

## Services Status (Final)

- ✅ Backend: Running on port 8090
- ✅ Frontend: Running on port 5173 (Vite dev server)
- ✅ Keycloak: Running on port 8081 (Docker container)
- ✅ PostgreSQL: Running on port 5432 (Docker container)
- ✅ Test user created in Keycloak

---

## Deferred Improvements (NOT in Phase 1 scope)

### High Priority (Before Production)
- [ ] **Token Refresh Logic**
  - Current: 401 clears token and redirects to login
  - Needed: Silent token refresh with retry logic
  - Location: `frontend/src/services/api.ts`

- [ ] **Keycloak User Management**
  - Setup script currently only creates realm/client
  - Need: Initial admin user creation
  - Need: User registration flow or manual user creation docs

- [ ] **Environment-Specific Configuration**
  - Hardcoded redirect URI in backend
  - Should read from env: `OIDC_REDIRECT_URI`
  - Location: `backend/internal/service/auth_service.go:35`

- [ ] **Error Handling Improvements**
  - Generic error messages in frontend
  - Need: User-friendly error messages
  - Need: Error boundary for React components

- [ ] **Security Hardening**
  - JWT secret should be 32+ chars (currently using placeholder)
  - Consider httpOnly cookies instead of localStorage
  - Add CSRF protection
  - Add rate limiting to auth endpoints

### Medium Priority
- [ ] **Testing**
  - Unit tests for AuthService (token validation, OIDC flow)
  - Unit tests for UserRepository (upsert logic)
  - Frontend component tests (AuthContext, useAuth, LoginPage)
  - Integration tests for complete auth flow

- [ ] **Logging & Monitoring**
  - Structured logging (JSON format)
  - Log auth events (login success/failure, token refresh)
  - Metrics for auth endpoint latency

- [ ] **Documentation**
  - API endpoint documentation (Swagger/OpenAPI)
  - Auth flow diagram
  - Deployment guide for Keycloak configuration

### Low Priority (Nice to Have)
- [ ] **UI/UX Polish**
  - Loading states for auth operations
  - Better error messages
  - Toast notifications for auth events
  - Remember me functionality

- [ ] **Developer Experience**
  - Docker Compose profiles for frontend dev (optional backend in container)
  - Hot reload for backend (air or similar)
  - VS Code debug configurations

- [ ] **Alternative Auth Methods**
  - Social login (Google, GitHub)
  - SSO with other providers
  - API key authentication for CLI tools

---

## Known Limitations

1. **Port Conflict**: 
   - Default port 8080 conflicts with SearXNG
   - Using 8090 for backend
   - Docs updated to reflect this

2. **No Token Refresh**:
   - Tokens expire after 1 hour (configurable via JWT_EXPIRY)
   - Users must re-login after expiry
   - No silent refresh mechanism

3. **No User Profile Management**:
   - Users created automatically via OIDC
   - No UI to view/edit profile
   - No way to delete users

4. **Keycloak Setup Script Incomplete**:
   - Creates realm and client only
   - Doesn't create test users
   - Doesn't configure email/password policies

---

## Key Files Implemented

### Backend
```
backend/
├── cmd/api/main.go (entry point with .env loading fix)
├── internal/
│   ├── api/auth.go (OIDC endpoints)
│   ├── service/auth_service.go (OIDC + JWT logic)
│   ├── repository/user_repository.go (CRUD + upsert)
│   ├── middleware/auth.go (JWT validation)
│   └── model/user.go (GORM model with explicit column tags)
```

### Frontend
```
frontend/
├── src/
│   ├── contexts/AuthContext.tsx (global auth state)
│   ├── hooks/useAuth.ts (auth hook)
│   ├── pages/LoginPage.tsx (login UI)
│   ├── pages/OidcCallbackPage.tsx (callback handler with useRef guard)
│   ├── components/ProtectedRoute.tsx (route guard)
│   ├── services/api.ts (axios client with JWT interceptor)
│   └── App.tsx (route definitions)
└── .env.local (VITE_API_URL=http://localhost:8090)
```

### Database
```
db/migrations/001_init.up.sql (users table schema)
```

---

## Configuration Files

**Backend (.env at project root):**
```
DATABASE_URL=postgres://opencode:opencode@localhost:5432/opencode_dev
OIDC_ISSUER=http://localhost:8081/realms/opencode
OIDC_CLIENT_ID=opencode-app
OIDC_CLIENT_SECRET=your-client-secret
JWT_SECRET=local-dev-secret-change-in-production
JWT_EXPIRY=3600
PORT=8090
ENVIRONMENT=development
```

**Frontend (.env.local):**
```
VITE_API_URL=http://localhost:8090
```

---

## Lessons Learned

1. **GORM Auto-Migration vs Manual Migrations:**
   - GORM's snake_case conversion can conflict with manual migration files
   - Always use explicit `gorm:"column:name"` tags for clarity

2. **Error Handling in Repositories:**
   - Use `errors.Is()` for GORM error comparison, not string matching
   - Import "errors" package explicitly

3. **React StrictMode Gotcha:**
   - StrictMode double-invokes effects in dev mode
   - Use `useRef` guards for non-idempotent operations (OAuth code exchange)

4. **Environment Variable Loading:**
   - `godotenv.Load()` is CWD-relative
   - Load from parent directory explicitly when .env is at project root

5. **Port Conflicts:**
   - Always check for port conflicts in multi-service environments
   - Document actual ports used (not just default ports)

---

## Acknowledgments

- **Keycloak:** OIDC provider for authentication
- **go-oidc v3:** Simplified OIDC integration in Go
- **golang-jwt/jwt v5:** JWT token generation and validation
- **Playwright:** E2E testing automation

---

**Archived by:** Sisyphus (OpenCode AI Agent)  
**Date:** 2026-01-16 23:44 CET  
**Next Phase:** Phase 2 - Project Management (K8s pod lifecycle)
