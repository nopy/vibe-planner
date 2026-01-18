# Project Improvements & Future Enhancements

**Last Updated**: 2026-01-18 21:42 CET  
**Project**: OpenCode Project Manager  
**Current Phase**: Phase 3 (Task Management & Kanban) - Planning  
**Previous Phase**: Phase 2 (Project Management) - Complete ✅

---

## Overview

This document tracks optional improvements and enhancements that could be implemented to further improve code quality, test coverage, and developer experience. All items listed here are **non-blocking** - the project is production-ready in its current state.

**Sources:**
- Original testing and infrastructure improvements
- Phase 1 deferred improvements (from PHASE1.md)
- Phase 2 deferred improvements (from PHASE2.md)
- Ongoing quality enhancements

---

## 🚀 Phase 2 Deferred Improvements

### P2.1 Full Kubernetes Watch Integration

**Impact**: True real-time pod status updates without polling  
**Effort**: Medium (4-6 hours)  
**Priority**: Medium

**Current Implementation**:
- WebSocket endpoint sends current pod status on connect
- No live updates when pod status changes in Kubernetes
- Client must reconnect to get updated status

**What to Implement**:
```go
// backend/internal/api/projects.go

func (h *ProjectHandler) ProjectStatusWebSocket(c *gin.Context) {
    // ... existing auth and upgrade code ...
    
    // Start Kubernetes watch
    statusChan, err := h.projectService.WatchPodStatus(c.Request.Context(), projectID)
    if err != nil {
        ws.Close()
        return
    }
    
    // Stream updates to WebSocket
    for {
        select {
        case status := <-statusChan:
            if err := ws.WriteJSON(gin.H{"pod_status": status}); err != nil {
                return
            }
        case <-c.Request.Context().Done():
            return
        }
    }
}
```

**Files to Modify**:
- `backend/internal/api/projects.go` (enhance WebSocket handler)
- `backend/internal/service/kubernetes_service.go` (already has WatchPodStatus, needs integration)

**Referenced in**: PHASE2.md (Deferred Items - Medium Priority)

---

### P2.2 Pod Resource Limits Configuration UI

**Impact**: Allow per-project resource customization  
**Effort**: Low (2-3 hours)  
**Priority**: Low

**Current Implementation**:
- Resource limits hardcoded in pod template (CPU: 1000m, Memory: 1Gi)
- No way for users to customize per project

**What to Implement**:
```typescript
// frontend/src/components/Projects/CreateProjectModal.tsx

interface CreateProjectRequest {
    name: string;
    description?: string;
    repo_url?: string;
    resources?: {
        cpu_limit?: string;    // e.g., "500m", "2"
        memory_limit?: string; // e.g., "512Mi", "2Gi"
        storage_size?: string; // e.g., "1Gi", "10Gi"
    };
}
```

```go
// backend/internal/service/kubernetes_service.go

type ResourceConfig struct {
    CPULimit    string
    MemoryLimit string
    StorageSize string
}

func (s *KubernetesService) CreateProjectPod(ctx context.Context, project *model.Project, resources *ResourceConfig) error {
    // Use custom resources if provided, otherwise use defaults
    // ...
}
```

**Files to Modify**:
- `frontend/src/components/Projects/CreateProjectModal.tsx` (add resource fields)
- `frontend/src/types/index.ts` (add ResourceConfig interface)
- `backend/internal/service/kubernetes_service.go` (accept ResourceConfig parameter)
- `backend/internal/service/pod_template.go` (use dynamic resources)
- `backend/internal/model/project.go` (add resource fields)
- `db/migrations/003_add_project_resources.sql` (new migration)

**Referenced in**: PHASE2.md (Deferred Items - Low Priority)

---

### P2.3 Project Pagination

**Impact**: Improve performance for users with many projects  
**Effort**: Low (2-3 hours)  
**Priority**: Medium

**Current Implementation**:
- `GET /api/projects` returns all projects for a user
- No pagination support
- Could be slow with 100+ projects

**What to Implement**:
```go
// backend/internal/api/projects.go

type ListProjectsQuery struct {
    Page     int    `form:"page" binding:"min=1"`
    PageSize int    `form:"page_size" binding:"min=1,max=100"`
    SortBy   string `form:"sort_by" binding:"omitempty,oneof=name created_at updated_at"`
    Order    string `form:"order" binding:"omitempty,oneof=asc desc"`
}

type ListProjectsResponse struct {
    Projects   []model.Project `json:"projects"`
    TotalCount int             `json:"total_count"`
    Page       int             `json:"page"`
    PageSize   int             `json:"page_size"`
}

func (h *ProjectHandler) ListProjects(c *gin.Context) {
    var query ListProjectsQuery
    if err := c.ShouldBindQuery(&query); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // Default values
    if query.Page == 0 {
        query.Page = 1
    }
    if query.PageSize == 0 {
        query.PageSize = 20
    }
    
    projects, totalCount, err := h.projectService.ListProjectsPaginated(
        userID, query.Page, query.PageSize, query.SortBy, query.Order,
    )
    // ...
}
```

**Files to Modify**:
- `backend/internal/repository/project_repository.go` (add FindByUserIDPaginated)
- `backend/internal/service/project_service.go` (add ListProjectsPaginated)
- `backend/internal/api/projects.go` (update ListProjects handler)
- `frontend/src/services/api.ts` (accept pagination params)
- `frontend/src/components/Projects/ProjectList.tsx` (add pagination UI)

**Referenced in**: PHASE2.md (Deferred Items - Medium Priority)

---

### P2.4 Project Search and Filtering

**Impact**: Easier project discovery for large project lists  
**Effort**: Medium (3-4 hours)  
**Priority**: Low

**Current Implementation**:
- No search or filter functionality
- All projects displayed in grid (sorted by created_at DESC)

**What to Implement**:
```typescript
// frontend/src/components/Projects/ProjectList.tsx

const [searchTerm, setSearchTerm] = useState('');
const [statusFilter, setStatusFilter] = useState<string | null>(null);

// Search input
<input
  type="text"
  placeholder="Search projects..."
  value={searchTerm}
  onChange={(e) => setSearchTerm(e.target.value)}
/>

// Status filter dropdown
<select onChange={(e) => setStatusFilter(e.target.value || null)}>
  <option value="">All Statuses</option>
  <option value="ready">Ready</option>
  <option value="initializing">Initializing</option>
  <option value="error">Error</option>
  <option value="archived">Archived</option>
</select>
```

```go
// backend/internal/repository/project_repository.go

type ProjectFilter struct {
    SearchTerm string
    Status     string
}

func (r *ProjectRepository) FindByUserIDWithFilter(ctx context.Context, userID uuid.UUID, filter ProjectFilter) ([]model.Project, error) {
    query := r.db.Where("user_id = ?", userID)
    
    if filter.SearchTerm != "" {
        query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+filter.SearchTerm+"%", "%"+filter.SearchTerm+"%")
    }
    
    if filter.Status != "" {
        query = query.Where("pod_status = ?", filter.Status)
    }
    
    // ...
}
```

**Files to Modify**:
- `backend/internal/repository/project_repository.go` (add filtering logic)
- `backend/internal/service/project_service.go` (accept filter params)
- `backend/internal/api/projects.go` (parse filter query params)
- `frontend/src/components/Projects/ProjectList.tsx` (add search/filter UI)

**Referenced in**: PHASE2.md (Deferred Items - Low Priority)

---

## 🔐 Authentication & Security Improvements (Phase 1 Deferred)

### A1. Token Refresh Logic

**Impact**: Improve user experience by avoiding forced re-login  
**Effort**: Medium (4-6 hours)  
**Priority**: High (before production)

**Current Behavior**:
- 401 response clears token and redirects to login
- Users must re-authenticate after 1-hour token expiry

**What to Implement**:
```typescript
// frontend/src/services/api.ts

// Add refresh token storage
const setRefreshToken = (token: string) => {
  localStorage.setItem('refresh_token', token);
};

// Modify response interceptor
api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;
    
    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true;
      
      try {
        // Attempt token refresh
        const refreshToken = localStorage.getItem('refresh_token');
        const { data } = await api.post('/auth/refresh', { refreshToken });
        
        setAuthToken(data.token);
        setRefreshToken(data.refreshToken);
        
        // Retry original request
        return api(originalRequest);
      } catch (refreshError) {
        // Refresh failed, logout
        clearAuth();
        window.location.href = '/login';
        return Promise.reject(refreshError);
      }
    }
    
    return Promise.reject(error);
  }
);
```

**Backend Changes Needed**:
```go
// backend/internal/api/auth.go

// Add refresh endpoint
func (h *AuthHandler) RefreshToken(c *gin.Context) {
    // Validate refresh token
    // Generate new access token + refresh token
    // Return both tokens
}
```

**Files to Modify**:
- `frontend/src/services/api.ts`
- `backend/internal/api/auth.go`
- `backend/internal/service/auth_service.go`

**Referenced in**: PHASE1.md, TODO.md (Phase 1 deferred)

---

### A2. Environment-Specific Configuration

**Impact**: Enable proper multi-environment deployment  
**Effort**: Low (1-2 hours)  
**Priority**: High (before production)

**Current Issue**:
- Hardcoded redirect URI: `http://localhost:5173/auth/callback`
- Should be environment-specific

**What to Implement**:
```bash
# .env (add)
OIDC_REDIRECT_URI=http://localhost:5173/auth/callback

# .env.production (create)
OIDC_REDIRECT_URI=https://opencode.example.com/auth/callback
```

```go
// backend/internal/service/auth_service.go

func (s *AuthService) GetOIDCLoginURL() (string, error) {
    redirectURI := os.Getenv("OIDC_REDIRECT_URI") // Read from env
    
    authURL := s.oauth2Config.AuthCodeURL(
        state,
        oauth2.SetAuthURLParam("redirect_uri", redirectURI),
    )
    
    return authURL, nil
}
```

**Files to Modify**:
- `backend/internal/service/auth_service.go:35`
- `.env.example`

**Referenced in**: PHASE1.md (Deferred Improvements - High Priority)

---

### A3. Enhanced Error Handling

**Impact**: Better user experience during auth failures  
**Effort**: Medium (3-4 hours)  
**Priority**: Medium

**Current Issue**:
- Generic error messages in frontend
- No error boundary for React components

**What to Implement**:

**1. Error Boundary Component**:
```typescript
// frontend/src/components/ErrorBoundary.tsx

class ErrorBoundary extends React.Component {
  state = { hasError: false, error: null };
  
  static getDerivedStateFromError(error) {
    return { hasError: true, error };
  }
  
  componentDidCatch(error, errorInfo) {
    console.error('Error caught by boundary:', error, errorInfo);
  }
  
  render() {
    if (this.state.hasError) {
      return <ErrorFallback error={this.state.error} />;
    }
    return this.props.children;
  }
}
```

**2. User-Friendly Error Messages**:
```typescript
// frontend/src/utils/errorMessages.ts

export const getAuthErrorMessage = (error: any): string => {
  if (error.response?.status === 401) {
    return 'Invalid credentials. Please try again.';
  }
  if (error.response?.status === 403) {
    return 'Access denied. Please contact administrator.';
  }
  if (error.code === 'NETWORK_ERROR') {
    return 'Network error. Please check your connection.';
  }
  return 'An unexpected error occurred. Please try again.';
};
```

**Files to Create**:
- `frontend/src/components/ErrorBoundary.tsx`
- `frontend/src/utils/errorMessages.ts`

**Files to Modify**:
- `frontend/src/App.tsx` (wrap with ErrorBoundary)
- `frontend/src/contexts/AuthContext.tsx` (use error messages)

**Referenced in**: PHASE1.md (Deferred Improvements - High Priority)

---

### A4. Security Hardening

**Impact**: Production-ready security  
**Effort**: Medium (4-6 hours)  
**Priority**: High (before production)

**Current Issues**:
1. JWT secret should be 32+ characters (currently placeholder)
2. Tokens stored in localStorage (vulnerable to XSS)
3. No CSRF protection
4. No rate limiting on auth endpoints

**What to Implement**:

**1. Strong JWT Secret**:
```bash
# Generate secure secret
openssl rand -base64 32

# .env
JWT_SECRET=<generated-32-char-secret>
```

**2. httpOnly Cookies** (alternative to localStorage):
```go
// backend/internal/api/auth.go

func (h *AuthHandler) OIDCCallback(c *gin.Context) {
    // ... existing code to generate token ...
    
    // Set httpOnly cookie instead of returning token in JSON
    c.SetSameSite(http.SameSiteStrictMode)
    c.SetCookie(
        "auth_token",
        token,
        3600, // maxAge
        "/",
        "",
        true,  // secure (HTTPS only)
        true,  // httpOnly
    )
    
    c.JSON(200, gin.H{"message": "Login successful"})
}
```

**3. Rate Limiting**:
```go
// backend/internal/middleware/rate_limit.go

import "github.com/ulule/limiter/v3"

func RateLimitAuth() gin.HandlerFunc {
    rate := limiter.Rate{
        Period: 1 * time.Minute,
        Limit:  5, // 5 login attempts per minute
    }
    
    store := memory.NewStore()
    instance := limiter.New(store, rate)
    
    return func(c *gin.Context) {
        context, err := instance.Get(c, c.ClientIP())
        if err != nil {
            c.AbortWithStatus(500)
            return
        }
        
        if context.Reached {
            c.AbortWithStatusJSON(429, gin.H{
                "error": "Too many requests",
            })
            return
        }
        
        c.Next()
    }
}
```

**Files to Modify**:
- `.env` (update JWT_SECRET)
- `backend/internal/api/auth.go` (cookie-based auth)
- `backend/cmd/api/main.go` (add rate limiter to auth routes)

**Files to Create**:
- `backend/internal/middleware/rate_limit.go`

**Dependencies to Add**:
- `github.com/ulule/limiter/v3`

**Referenced in**: PHASE1.md (Deferred Improvements - High Priority)

---

### A5. Keycloak User Management

**Impact**: Complete Keycloak setup automation  
**Effort**: Low (2-3 hours)  
**Priority**: Medium

**Current Limitation**:
- Setup script only creates realm and client
- Test users must be created manually

**What to Implement**:
```bash
# scripts/setup-keycloak.sh (extend)

# Create test user
docker exec opencode-keycloak /opt/keycloak/bin/kcadm.sh create users \
  -r opencode \
  -s username=testuser \
  -s email=testuser@example.com \
  -s enabled=true

# Set password
docker exec opencode-keycloak /opt/keycloak/bin/kcadm.sh set-password \
  -r opencode \
  --username testuser \
  --new-password testpass123

# Create admin user
docker exec opencode-keycloak /opt/keycloak/bin/kcadm.sh create users \
  -r opencode \
  -s username=admin \
  -s email=admin@example.com \
  -s enabled=true

docker exec opencode-keycloak /opt/keycloak/bin/kcadm.sh set-password \
  -r opencode \
  --username admin \
  --new-password adminpass123
```

**Files to Modify**:
- `scripts/setup-keycloak.sh`

**Documentation to Add**:
- User registration flow instructions
- Manual user creation guide

**Referenced in**: PHASE1.md (Deferred Improvements - High Priority)

---

## 🧪 Testing Improvements

### T1. Auth Service Unit Tests (Backend)

**Impact**: Establish baseline test coverage for critical auth logic  
**Effort**: High (8-10 hours)  
**Priority**: High (from PHASE1.md + original IMPROVEMENTS.md)

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

**Files to Create/Modify**:
- `/home/npinot/vibe/backend/internal/service/auth_service_test.go` (expand existing)

**Referenced in**: PHASE1.md (Deferred - Medium Priority), Original IMPROVEMENTS.md (#4)

---

### T2. User Repository Tests (Backend)

**Impact**: Ensure database operations work correctly  
**Effort**: Medium (4-6 hours)  
**Priority**: Medium (from PHASE1.md + original IMPROVEMENTS.md)

**What to Test** (see original IMPROVEMENTS.md section 5 for full details)

**Referenced in**: PHASE1.md (Deferred - Medium Priority), Original IMPROVEMENTS.md (#5)

---

### T3. Auth Middleware Tests (Backend)

**Impact**: Verify JWT validation in HTTP layer  
**Effort**: Low (2-3 hours)  
**Priority**: Medium (from PHASE1.md + original IMPROVEMENTS.md)

**What to Test** (see original IMPROVEMENTS.md section 6 for full details)

**Referenced in**: PHASE1.md (Deferred - Medium Priority), Original IMPROVEMENTS.md (#6)

---

### T4. Frontend Component Tests

**Impact**: Increase frontend test coverage  
**Effort**: Medium (6-8 hours)  
**Priority**: Medium (from PHASE1.md)

**What to Test**:
1. **AuthContext Component Tests** - see original IMPROVEMENTS.md (#2)
2. **API Interceptor Tests** - see original IMPROVEMENTS.md (#1)
3. **LoginPage Component Tests**
4. **OidcCallbackPage Component Tests**
5. **useAuth Hook Tests**

**Referenced in**: PHASE1.md (Deferred - Medium Priority)

---

### T5. Integration Tests

**Impact**: Ensure complete auth flow works end-to-end  
**Effort**: High (6-8 hours)  
**Priority**: Medium (from PHASE1.md)

**What to Test**:
```go
// backend/internal/api/integration_test.go

func TestAuthFlow_Complete(t *testing.T) {
    // 1. GET /api/auth/oidc/login → returns Keycloak URL
    // 2. Simulate Keycloak callback with code
    // 3. GET /api/auth/oidc/callback?code=... → returns JWT
    // 4. GET /api/auth/me with JWT → returns user data
    // 5. Verify user created in database
}
```

**Referenced in**: PHASE1.md (Deferred - Medium Priority)

---

## 📊 Monitoring & Logging (Phase 1 Deferred)

### M1. Structured Logging

**Impact**: Easier debugging and monitoring  
**Effort**: Low (2-3 hours)  
**Priority**: Medium

**What to Implement**:
```go
// backend/internal/util/logger.go

import "go.uber.org/zap"

var logger *zap.Logger

func InitLogger(env string) {
    var err error
    if env == "production" {
        logger, err = zap.NewProduction()
    } else {
        logger, err = zap.NewDevelopment()
    }
    
    if err != nil {
        panic(err)
    }
}

func Info(msg string, fields ...zap.Field) {
    logger.Info(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
    logger.Error(msg, fields...)
}
```

**Usage**:
```go
// backend/internal/service/auth_service.go

import "github.com/npinot/vibe/backend/internal/util"

func (s *AuthService) HandleOIDCCallback(code string) (string, error) {
    util.Info("OIDC callback initiated", 
        zap.String("code", code[:10]+"..."))
    
    // ... rest of implementation
    
    util.Info("User authenticated successfully",
        zap.String("user_id", user.ID.String()),
        zap.String("email", user.Email))
}
```

**Files to Create**:
- `backend/internal/util/logger.go`

**Files to Modify**:
- `backend/cmd/api/main.go` (initialize logger)
- `backend/internal/service/auth_service.go` (add logging)
- `backend/internal/api/auth.go` (add logging)

**Referenced in**: PHASE1.md (Deferred - Medium Priority)

---

### M2. Metrics for Auth Endpoints

**Impact**: Track auth performance  
**Effort**: Medium (3-4 hours)  
**Priority**: Low

**What to Implement**:
- Track login success/failure rates
- Track token validation latency
- Track OIDC callback latency

**Files to Create**:
- `backend/internal/middleware/metrics.go`

**Referenced in**: PHASE1.md (Deferred - Medium Priority)

---

## 📚 Documentation (Phase 1 Deferred)

### D1. API Endpoint Documentation

**Impact**: Easier API consumption for frontend developers  
**Effort**: Medium (4-6 hours)  
**Priority**: Medium

**What to Implement**:
- OpenAPI/Swagger specification
- Interactive API explorer UI

**Tools**: `swaggo/swag` (see original IMPROVEMENTS.md #13 for details)

**Referenced in**: PHASE1.md (Deferred - Medium Priority), Original IMPROVEMENTS.md (#13)

---

### D2. Auth Flow Diagram

**Impact**: Visual documentation of OIDC flow  
**Effort**: Low (1-2 hours)  
**Priority**: Low

**What to Create**:
```
┌─────────┐          ┌─────────┐          ┌──────────┐          ┌──────────┐
│ Browser │          │ Backend │          │ Keycloak │          │ Database │
└────┬────┘          └────┬────┘          └────┬─────┘          └────┬─────┘
     │                    │                    │                     │
     │ 1. GET /login      │                    │                     │
     ├───────────────────>│                    │                     │
     │                    │                    │                     │
     │ 2. Redirect to KC  │                    │                     │
     │<───────────────────┤                    │                     │
     │                    │                    │                     │
     │ 3. Authenticate    │                    │                     │
     ├────────────────────┼───────────────────>│                     │
     │                    │                    │                     │
     │ 4. Code in callback│                    │                     │
     │<───────────────────┼────────────────────┤                     │
     │                    │                    │                     │
     │ 5. Exchange code   │                    │                     │
     ├───────────────────>│                    │                     │
     │                    │                    │                     │
     │                    │ 6. Validate code   │                     │
     │                    ├───────────────────>│                     │
     │                    │                    │                     │
     │                    │ 7. User info       │                     │
     │                    │<───────────────────┤                     │
     │                    │                    │                     │
     │                    │ 8. Create/update user                    │
     │                    ├──────────────────────────────────────────>│
     │                    │                    │                     │
     │ 9. JWT token       │                    │                     │
     │<───────────────────┤                    │                     │
     │                    │                    │                     │
```

**Files to Create**:
- `docs/auth-flow.md`
- `docs/diagrams/auth-flow.svg`

**Referenced in**: PHASE1.md (Deferred - Medium Priority)

---

### D3. Deployment Guide for Keycloak

**Impact**: Easier production setup  
**Effort**: Low (2-3 hours)  
**Priority**: Medium

**What to Document**:
- Keycloak installation steps
- Realm configuration
- Client configuration
- User management
- SSL/TLS setup
- Backup and recovery

**Files to Create**:
- `docs/keycloak-deployment.md`

**Referenced in**: PHASE1.md (Deferred - Medium Priority)

---

## 🎨 UI/UX Improvements (Phase 1 Deferred)

### UI1. Loading States

**Impact**: Better user feedback  
**Effort**: Low (2-3 hours)  
**Priority**: Low

**What to Implement**:
```typescript
// frontend/src/pages/LoginPage.tsx

const [isLoading, setIsLoading] = useState(false);

const handleLogin = async () => {
  setIsLoading(true);
  try {
    await login();
  } finally {
    setIsLoading(false);
  }
};

return (
  <button onClick={handleLogin} disabled={isLoading}>
    {isLoading ? <Spinner /> : 'Login with Keycloak'}
  </button>
);
```

**Files to Modify**:
- `frontend/src/pages/LoginPage.tsx`
- `frontend/src/pages/OidcCallbackPage.tsx`

**Referenced in**: PHASE1.md (Deferred - Low Priority)

---

### UI2. Toast Notifications

**Impact**: Better error/success feedback  
**Effort**: Low (2-3 hours)  
**Priority**: Low

**What to Implement**:
- Use `react-hot-toast` or similar
- Show notifications for auth events (login success, logout, errors)

**Files to Modify**:
- `frontend/src/contexts/AuthContext.tsx`

**Referenced in**: PHASE1.md (Deferred - Low Priority)

---

### UI3. Remember Me Functionality

**Impact**: Convenience for users  
**Effort**: Medium (3-4 hours)  
**Priority**: Low

**What to Implement**:
- Checkbox on login page
- Store preference in localStorage
- Extend token expiry if enabled

**Referenced in**: PHASE1.md (Deferred - Low Priority)

---

## 🛠️ Developer Experience (Phase 1 Deferred)

### DX1. Docker Compose Profiles

**Impact**: Flexible development setup  
**Effort**: Low (1-2 hours)  
**Priority**: Low

**What to Implement**:
```yaml
# docker-compose.yml

services:
  backend:
    profiles: ["backend", "full"]
    # ... existing config
  
  frontend:
    profiles: ["frontend", "full"]
    # ... existing config
```

**Usage**:
```bash
# Run only services
docker-compose up

# Run with backend in container
docker-compose --profile backend up

# Run full stack
docker-compose --profile full up
```

**Referenced in**: PHASE1.md (Deferred - Low Priority)

---

### DX2. Hot Reload for Backend

**Impact**: Faster development iteration  
**Effort**: Low (1-2 hours)  
**Priority**: Low

**What to Implement**:
```bash
# Install air
go install github.com/cosmtrek/air@latest

# Create .air.toml config
```

**Files to Create**:
- `.air.toml`

**Referenced in**: PHASE1.md (Deferred - Low Priority)

---

### DX3. VS Code Debug Configurations

**Impact**: Easier debugging  
**Effort**: Low (1 hour)  
**Priority**: Low

**What to Create**:
```json
// .vscode/launch.json

{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug Backend",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/backend/cmd/api",
      "env": {
        "DATABASE_URL": "..."
      }
    }
  ]
}
```

**Files to Create**:
- `.vscode/launch.json`

**Referenced in**: PHASE1.md (Deferred - Low Priority)

---

## 🔌 Alternative Auth Methods (Phase 1 Deferred)

### ALT1. Social Login (Google, GitHub)

**Impact**: More login options for users  
**Effort**: Medium (4-6 hours per provider)  
**Priority**: Low

**What to Implement**:
- Add Google OAuth provider to Keycloak
- Add GitHub OAuth provider to Keycloak
- Update login page UI

**Referenced in**: PHASE1.md (Deferred - Low Priority)

---

### ALT2. API Key Authentication

**Impact**: Enable CLI tool authentication  
**Effort**: High (8-10 hours)  
**Priority**: Low

**What to Implement**:
- API key generation endpoint
- API key validation middleware
- API key management UI

**Referenced in**: PHASE1.md (Deferred - Low Priority)

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

## 🔄 Original Improvements (Pre-Phase 1 Archive)

The following sections preserve the original improvement tracking from before Phase 1 completion. Many of these are now integrated into the categorized sections above.

<details>
<summary><b>Click to expand original sections</b></summary>

### Frontend Testing Improvements

**Current Status**: ✅ 83.51% overall test coverage (exceeds 80% target)

#### 1. API Interceptor Tests
(See T4 above - now categorized under Frontend Component Tests)

#### 2. AuthContext Async Flow Tests
(See T4 above - now categorized under Frontend Component Tests)

#### 3. App.tsx Route Tests
(See original IMPROVEMENTS.md section 3 for full details)

### Backend Testing Improvements

**Current Status**: ⚠️ Minimal test coverage

#### 4. Auth Service Unit Tests
(See T1 above - now elevated to High Priority)

#### 5. User Repository Tests
(See T2 above)

#### 6. Auth Middleware Tests
(See T3 above)

### Infrastructure Improvements

#### 7. CI/CD Pipeline
(See original IMPROVEMENTS.md section 7 for full details)
**Status**: Deferred to Phase 9

#### 8. E2E Testing Setup
(See original IMPROVEMENTS.md section 8 for full details)
**Status**: Deferred to Phase 9

### Code Quality Improvements

#### 9. TypeScript Strict Mode Fixes
(See original IMPROVEMENTS.md section 9 for full details)

#### 10. Linter Configuration Enhancements
(See original IMPROVEMENTS.md section 10 for full details)

### Performance Improvements

#### 11. Frontend Bundle Optimization
(See original IMPROVEMENTS.md section 11 for full details)

#### 12. Docker Image Optimization
(See original IMPROVEMENTS.md section 12 for full details)
**Note**: Current 29MB is already exceptional

### Documentation Improvements

#### 13. API Documentation Generation
(See D1 above)

#### 14. Architecture Decision Records (ADRs)
(See original IMPROVEMENTS.md section 14 for full details)

### Security Improvements

#### 15. Security Headers Middleware
(See original IMPROVEMENTS.md section 15 for full details)

#### 16. Dependency Vulnerability Scanning
(See original IMPROVEMENTS.md section 16 for full details)
**Current Status**: 7 vulnerabilities detected in frontend (6 moderate, 1 critical) - should be addressed

</details>

---

## 📊 Summary & Prioritization

### 🔴 High Priority (Before Production)

**Phase 1 Deferred (Must Address)**:
1. **A1** - Token Refresh Logic (user experience)
2. **A2** - Environment-Specific Configuration (deployment)
3. **A4** - Security Hardening (JWT secret, rate limiting, CSRF)
4. **T1** - Auth Service Unit Tests (test baseline)

**Original High Priority**:
5. **CI/CD Pipeline** (#7 - deferred to Phase 9)
6. **Dependency Vulnerability Scanning** (#16 - 7 frontend vulnerabilities)

### 🟡 Medium Priority (Improve Quality)

**Phase 2 Deferred**:
1. **P2.1** - Full Kubernetes Watch Integration (real-time updates)
2. **P2.3** - Project Pagination (performance)

**Phase 1 Deferred**:
3. **A3** - Enhanced Error Handling (UX)
4. **A5** - Keycloak User Management (setup automation)
5. **T2** - User Repository Tests (coverage)
6. **T3** - Auth Middleware Tests (coverage)
7. **T4** - Frontend Component Tests (coverage)
8. **T5** - Integration Tests (e2e confidence)
9. **M1** - Structured Logging (debugging)
10. **D1** - API Documentation (developer experience)
11. **D3** - Keycloak Deployment Guide (production setup)

**Original Medium Priority**:
12. **Security Headers Middleware** (#15)
13. **E2E Testing Setup** (#8 - deferred to Phase 9)

### 🟢 Low Priority (Nice to Have)

**Phase 2 Deferred**:
1. **P2.2** - Pod Resource Limits Configuration UI (flexibility)
2. **P2.4** - Project Search and Filtering (UX)

**Phase 1 Deferred**:
3. **M2** - Auth Metrics (monitoring)
4. **D2** - Auth Flow Diagram (visual docs)
5. **UI1** - Loading States (UX polish)
6. **UI2** - Toast Notifications (UX polish)
7. **UI3** - Remember Me (convenience)
8. **DX1** - Docker Compose Profiles (flexibility)
9. **DX2** - Hot Reload for Backend (dev speed)
10. **DX3** - VS Code Debug Configs (debugging)
11. **ALT1** - Social Login (alternative auth)
12. **ALT2** - API Key Auth (CLI tools)

**Original Low Priority**:
13. App.tsx Route Tests (#3)
14. TypeScript Strict Mode Fixes (#9)
15. Linter Configuration Enhancements (#10)
16. Frontend Bundle Optimization (#11)
17. Docker Image Optimization (#12)
18. Architecture Decision Records (#14)

---

## 📈 Progress Tracking

| Category | Item | Status | Completion Date | Priority | Notes |
|----------|------|--------|----------------|----------|-------|
| **Phase 2** | Full K8s Watch (P2.1) | 📋 Planned | - | Medium | Real-time updates |
| **Phase 2** | Pod Resources UI (P2.2) | 📋 Planned | - | Low | Custom limits |
| **Phase 2** | Project Pagination (P2.3) | 📋 Planned | - | Medium | Performance |
| **Phase 2** | Search/Filter (P2.4) | 📋 Planned | - | Low | UX improvement |
| **Auth** | Token Refresh Logic (A1) | 📋 Planned | - | High | Phase 1 deferred |
| **Auth** | Environment Config (A2) | 📋 Planned | - | High | Phase 1 deferred |
| **Auth** | Error Handling (A3) | 📋 Planned | - | Medium | Phase 1 deferred |
| **Auth** | Security Hardening (A4) | 📋 Planned | - | High | Phase 1 deferred |
| **Auth** | Keycloak User Mgmt (A5) | 📋 Planned | - | Medium | Phase 1 deferred |
| **Testing** | Frontend Infrastructure | ✅ Complete | 2026-01-16 | - | 83.51% coverage |
| **Testing** | ProtectedRoute Tests | ✅ Complete | 2026-01-16 | - | All tests passing |
| **Testing** | Backend Unit Tests | ✅ Complete | 2026-01-18 | - | 55 tests (Phase 2) |
| **Testing** | Integration Tests | ✅ Complete | 2026-01-18 | - | E2E lifecycle (Phase 2) |
| **Testing** | Auth Service Tests (T1) | 📋 Planned | - | High | Phase 1 deferred |
| **Testing** | User Repo Tests (T2) | 📋 Planned | - | Medium | Phase 1 deferred |
| **Testing** | Auth Middleware (T3) | 📋 Planned | - | Medium | Phase 1 deferred |
| **Testing** | Frontend Components (T4) | 📋 Planned | - | Medium | Phase 1 deferred |
| **Infra** | CI/CD Pipeline (#7) | 📋 Planned | - | High | Deferred to Phase 9 |
| **Infra** | E2E Testing (#8) | 📋 Planned | - | Medium | Deferred to Phase 9 |
| **Logging** | Structured Logging (M1) | 📋 Planned | - | Medium | Phase 1 deferred |
| **Logging** | Auth Metrics (M2) | 📋 Planned | - | Low | Phase 1 deferred |
| **Docs** | API Documentation (D1) | 📋 Planned | - | Medium | Phase 1 deferred |
| **Docs** | Auth Flow Diagram (D2) | 📋 Planned | - | Low | Phase 1 deferred |
| **Docs** | Keycloak Deploy (D3) | 📋 Planned | - | Medium | Phase 1 deferred |
| **Security** | Security Headers (#15) | 📋 Planned | - | Medium | Original |
| **Security** | Vuln Scanning (#16) | 📋 Planned | - | High | 7 frontend vulns |
| **UI/UX** | Loading States (UI1) | 📋 Planned | - | Low | Phase 1 deferred |
| **UI/UX** | Toast Notifications (UI2) | 📋 Planned | - | Low | Phase 1 deferred |
| **UI/UX** | Remember Me (UI3) | 📋 Planned | - | Low | Phase 1 deferred |

---

## 📝 Notes

1. **Phase 2 Deferred Items**: Items P2.1-P2.4 were identified during Phase 2 implementation but deferred to maintain focus on core functionality. Medium priority items (P2.1, P2.3) should be addressed before Phase 9.

2. **Phase 1 Deferred Items**: All items marked as "Phase 1 deferred" were identified during Phase 1 implementation but deferred to maintain focus on core authentication functionality. These should be addressed before production deployment.

3. **Testing Strategy**: Focus on high-priority tests (T1, T2, T3) before Phase 3 begins to establish a solid testing baseline for new features. Note: Phase 2 already has 55 backend unit tests and integration test suite.

4. **Security First**: Items A2, A4, and #16 should be addressed together as a security hardening sprint before production.

5. **Documentation**: Items D1, D2, D3 can be bundled together in a documentation sprint, likely during Phase 9 (Testing & Documentation).

6. **UX Polish**: Items UI1, UI2, UI3, P2.4 are low priority and can be addressed in Phase 10 (Polish & Optimization).

---

**Last Updated**: 2026-01-18 21:42 CET  
**Next Review**: Before Phase 3 kickoff  
**Source Documents**: PHASE1.md + PHASE2.md (deferred improvements) + original IMPROVEMENTS.md

**Note**: This document is a living guide. Prioritization should be revisited as project needs evolve. Items from Phase 1 and Phase 2 are now centralized here to maintain a single source of truth for all improvements.
