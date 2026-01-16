# OPENCODE BACKEND KNOWLEDGE BASE

## OVERVIEW
Go 1.24 API server using Gin (HTTP), GORM (PostgreSQL), and Keycloak (OIDC).

**Phase 1 Status**: ✅ OIDC Authentication Complete

## STRUCTURE
```
.
├── cmd/api/           # Entry point (main.go) - wired with auth dependencies
├── internal/
│   ├── api/           # HTTP Handlers - auth.go fully implemented
│   ├── model/         # GORM structs (User model with OIDC fields)
│   ├── service/       # ✅ auth_service.go - OIDC provider, token exchange, JWT
│   ├── repository/    # ✅ user_repository.go - User CRUD with OIDC upsert
│   ├── middleware/    # ✅ auth.go - JWT validation middleware
│   ├── config/        # Environment & App configuration
│   └── db/            # Connection & Migration logic (User auto-migrate added)
└── go.mod             # Module path: github.com/npinot/vibe/backend
```

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
