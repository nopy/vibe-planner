# OPENCODE BACKEND KNOWLEDGE BASE

## OVERVIEW
Go 1.21 API server using Gin (HTTP) and GORM (PostgreSQL).

## STRUCTURE
```
.
├── cmd/api/           # Entry point (main.go)
├── internal/
│   ├── api/           # HTTP Handlers (currently logic is inline)
│   ├── model/         # GORM structs & Database schema
│   ├── service/       # Business logic (Planned - currently empty)
│   ├── repository/    # Data access (Planned - currently empty)
│   ├── config/        # Environment & App configuration
│   └── db/            # Connection & Migration logic
└── go.mod             # Module path: github.com/npinot/vibe/backend
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
- Currently, handlers in `internal/api/` contain inline logic (stubs).
- **Future State**: Handlers should only parse input, call Services, and return JSON.
- No direct DB access in handlers once Repositories are implemented.

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
