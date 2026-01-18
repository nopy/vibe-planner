# OpenCode Project Manager - Development Guide

**Last Updated:** January 18, 2026  
**Status:** Phase 1 + Phase 2 Complete (Authentication + Project Management)

## Quick Start (5 minutes)

**Current Status:** Phase 1 (Auth) + Phase 2 (Project Management) Complete

### Prerequisites

```bash
# Check versions
go version          # 1.21+
node -v             # 16+
docker --version
docker-compose --version
kind version
kubectl version
```

### One-Command Setup

```bash
# Clone repository
git clone <repo-url> opencode-project-manager
cd opencode-project-manager

# Start everything
make dev

# Wait for logs: "All services ready!"
# Frontend: http://localhost:5173
# Backend: http://localhost:8080
# Keycloak: http://localhost:8081
```

---

## Detailed Setup (for first-time or clean install)

### 1. Install Dependencies

```bash
# Go
go mod download -m=readonly

# Node (from frontend directory)
cd frontend
npm install
cd ..

# Create .env file
cp .env.example .env
# Edit .env with your local values (see below)
```

### 2. Environment Configuration

**Create `.env` in project root:**

```bash
# Backend
DATABASE_URL=postgres://opencode:password@localhost:5432/opencode_dev
OIDC_ISSUER=http://localhost:8081/realms/opencode
OIDC_CLIENT_ID=opencode-app
OIDC_CLIENT_SECRET=opencode-secret
JWT_SECRET=your-secret-key-min-32-chars-long
JWT_EXPIRY=3600
KUBECONFIG=${HOME}/.kube/kind-config-opencode-dev
K8S_NAMESPACE=opencode
PORT=8080
LOG_LEVEL=debug
ENVIRONMENT=development

# OpenCode
OPENCODE_INSTALL_PATH=/usr/local/bin/opencode

# Frontend
VITE_API_URL=http://localhost:8080
VITE_OIDC_AUTHORITY=http://localhost:8081/realms/opencode
VITE_OIDC_CLIENT_ID=opencode-app
VITE_OIDC_REDIRECT_URI=http://localhost:5173/auth/callback
```

### 3. Start Services

**Terminal 1: Docker Compose (PostgreSQL + Keycloak)**

```bash
docker compose up -d postgres keycloak redis

# Wait for services
sleep 20

# Verify
docker ps
# Should show: postgres, keycloak, redis running
```

**Terminal 2: Configure Keycloak**

```bash
# Get Keycloak pod info
docker compose logs keycloak | grep "started in"

# Create realm, user, client (see Keycloak Setup section below)
# or use the provided setup script:
./scripts/setup-keycloak.sh
```

**Terminal 3: Backend**

```bash
cd backend

# Install/update migrations
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations
migrate -path db/migrations -database "$DATABASE_URL" up

# Start backend
go run cmd/api/main.go

# Should see: "Server listening on :8080"
```

**Terminal 4: Frontend**

```bash
cd frontend

# Start dev server
npm run dev

# Should see: "VITE v5.x.x ready in Xxx ms"
# Open: http://localhost:5173
```

### 4. Verify Everything Works

```bash
# Test backend
curl http://localhost:8080/healthz
# Expected: {"status":"ok"}

# Test frontend
open http://localhost:5173
# Should show login page

# Test Keycloak
open http://localhost:8081/realms/opencode/.well-known/openid-configuration
# Should return OIDC discovery JSON
```

---

## Keycloak Setup (Local Development)

### Automated Setup

```bash
./scripts/setup-keycloak.sh
```

### Manual Setup

1. **Access Keycloak Admin Console**
   - URL: http://localhost:8081/admin
   - Username: admin
   - Password: admin

2. **Create Realm**
   - Click "Create Realm" or "Add Realm"
   - Name: `opencode`
   - Enable
   - Save

3. **Create User**
   - Go to Realm: opencode
   - Users → Create user
   - Username: `testuser`
   - Email: `test@example.com`
   - Email verified: ON
   - Save
   - Go to Credentials tab
   - Set password: `password` (no expiry)

4. **Create Client**
   - Clients → Create client
   - Client ID: `opencode-app`
   - Client type: OpenID Connect
   - Next
   - Client authentication: OFF
   - Authorization: OFF
   - Authentication flow: Standard flow, Direct access grants
   - Next → Save
   - Go to Valid Redirect URIs
   - Add: `http://localhost:5173/*`
   - Save

5. **Get Client Secret**
   - Clients → opencode-app → Credentials
   - Copy Secret
   - Update `.env`: `OIDC_CLIENT_SECRET=<secret>`

---

## Development Workflow

### Code Structure

**Frontend:**
```
src/
├── components/          # React components
│   ├── Auth/
│   ├── Projects/
│   ├── Kanban/
│   ├── Explorer/
│   ├── Config/
│   └── Common/
├── hooks/              # Custom hooks
├── contexts/           # React contexts
├── services/           # API clients
├── types/              # TypeScript types
├── utils/              # Utilities
├── styles/             # CSS
└── App.tsx
```

**Backend:**
```
internal/
├── api/                # Handlers
├── service/            # Business logic
├── repository/         # Database
├── model/              # Domain models
├── middleware/         # Middleware
├── config/             # Configuration
├── util/               # Utilities
└── db/                 # Migrations
```

### Making Changes

#### Frontend

```bash
# From frontend/ directory

# Start dev server
npm run dev

# Changes auto-reload

# Type checking
npm run typecheck

# Linting
npm run lint

# Build
npm run build
```

#### Backend

```bash
# From backend/ directory

# Build & run
go run cmd/api/main.go

# Or compile
go build -o opencode-api cmd/api/main.go
./opencode-api

# Test
go test ./...
go test -v ./internal/service

# Format
go fmt ./...
```

### Database Migrations

```bash
# Create new migration
migrate create -ext sql -dir db/migrations -seq add_users_table

# Run migrations
migrate -path db/migrations -database "$DATABASE_URL" up

# Rollback one step
migrate -path db/migrations -database "$DATABASE_URL" down

# Force version (dangerous!)
migrate -path db/migrations -database "$DATABASE_URL" force 5
```

### Testing

```bash
# Backend tests
cd backend
go test ./... -v

# Frontend tests
cd frontend
npm test

# Integration tests (requires services running)
npm run test:integration
```

---

## Kind Kubernetes Cluster

### Create Cluster

```bash
# Create kind cluster
kind create cluster --config k8s/kind-config.yaml --name opencode-dev

# Verify
kubectl cluster-info
kubectl get nodes

# Set context
kubectl config use-context kind-opencode-dev
```

### Deploy to Kind

```bash
# Create namespace
kubectl create namespace opencode

# Create secrets
kubectl apply -f k8s/base/secrets.yaml

# Apply manifests (Kustomize)
kubectl apply -k k8s/base/

# Wait for pods
kubectl wait --for=condition=ready pod -l app=opencode-controller -n opencode --timeout=300s

# Check status
kubectl get all -n opencode
kubectl logs -n opencode <pod-name>

# Port forward
kubectl port-forward -n opencode svc/opencode-controller 8080:80 &

# Access app
open http://localhost:8080
```

### Debugging

```bash
# Get pod info
kubectl describe pod <pod-name> -n opencode

# View logs
kubectl logs <pod-name> -n opencode
kubectl logs <pod-name> -n opencode -c <container-name>

# Execute command in pod
kubectl exec -it <pod-name> -n opencode -- /bin/sh

# Port forward
kubectl port-forward pod/<pod-name> 5432:5432 -n opencode

# Watch pod status
kubectl get pods -n opencode -w
```

### Clean Up

```bash
# Delete cluster
kind delete cluster --name opencode-dev

# Stop docker compose
docker compose down

# Remove volumes
docker volume prune
```

---

## Docker Build Modes

### Production Build (Unified Image)

**Single Docker image (29MB) with embedded frontend:**

```bash
# Build unified image
make docker-build-prod

# Or with custom version
./scripts/build-images.sh --mode prod --version v1.0.0

# Build and push
make docker-push-prod
```

**What it builds:**
- `registry.legal-suite.com/opencode/app:latest` - Backend + Frontend (29MB)
- `registry.legal-suite.com/opencode/file-browser-sidecar:latest`
- `registry.legal-suite.com/opencode/session-proxy-sidecar:latest`

**How it works:**
1. Stage 1: Build React frontend → `dist/` directory
2. Stage 2: Build Go backend with embedded frontend (using `//go:embed`)
3. Stage 3: Minimal alpine image with single binary

**When to use:**
- Production deployments
- Staging environments
- Integration testing
- Simpler Kubernetes manifests (one pod, one container)

### Development Build (Separate Images)

**Separate backend and frontend images:**

```bash
# Build separate images
make docker-build-dev

# Or with custom version
./scripts/build-images.sh --mode dev --version dev

# Build and push
make docker-push-dev
```

**What it builds:**
- `registry.legal-suite.com/opencode/backend:latest` - Backend only
- `registry.legal-suite.com/opencode/frontend:latest` - Frontend + nginx
- `registry.legal-suite.com/opencode/file-browser-sidecar:latest`
- `registry.legal-suite.com/opencode/session-proxy-sidecar:latest`

**When to use:**
- Local development with Docker
- Debugging individual services
- Independent scaling of frontend/backend
- CI/CD pipeline stages

### Build Script Options

```bash
./scripts/build-images.sh --help

# Examples:
./scripts/build-images.sh --mode prod --version v1.2.3
./scripts/build-images.sh --mode dev --version dev --push
./scripts/build-images.sh --mode prod --registry my-registry.com/app
```

**Available flags:**
- `--mode MODE` - Build mode: `prod` (unified) or `dev` (separate)
- `--version VERSION` - Image tag version (default: `latest`)
- `--push` - Push images to registry after building
- `--registry URL` - Docker registry URL
- `-h, --help` - Show help message

---

## Phase 2: Project Management Features

### Available APIs

**Project CRUD Operations:**

```bash
# List all user projects
curl -H "Authorization: Bearer $JWT_TOKEN" \
  http://localhost:8090/api/projects

# Create new project
curl -X POST -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"My Project","description":"Test project","repo_url":"https://github.com/user/repo"}' \
  http://localhost:8090/api/projects

# Get project details
curl -H "Authorization: Bearer $JWT_TOKEN" \
  http://localhost:8090/api/projects/{project-id}

# Update project
curl -X PATCH -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"description":"Updated description"}' \
  http://localhost:8090/api/projects/{project-id}

# Delete project (soft delete + pod cleanup)
curl -X DELETE -H "Authorization: Bearer $JWT_TOKEN" \
  http://localhost:8090/api/projects/{project-id}
```

**WebSocket Real-time Status Updates:**

```javascript
// Connect to WebSocket endpoint for pod status updates
const ws = new WebSocket('ws://localhost:8090/api/projects/{project-id}/status');

ws.onmessage = (event) => {
  const status = JSON.parse(event.data);
  console.log('Pod status:', status.pod_status); // "Pending", "Running", "Failed", etc.
};

ws.onerror = (error) => console.error('WebSocket error:', error);
ws.onclose = () => console.log('WebSocket disconnected');
```

### Project Pod Architecture

Each project spawns a dedicated Kubernetes pod with:
- **OpenCode server** (port 3000) - Main AI coding agent
- **File browser sidecar** (port 3001) - File operations
- **Session proxy sidecar** (port 3002) - Session management
- **Shared PVC** - Persistent workspace storage

Pod lifecycle managed automatically:
1. Project creation → Pod spawned in `opencode` namespace
2. Pod status tracked → WebSocket updates to frontend
3. Project deletion → Pod + PVC cleaned up

### Testing Project Management

```bash
# Unit tests (55 tests across repository, service, and API layers)
cd backend
go test ./...

# Integration tests (requires kind cluster)
cd backend
export TEST_DATABASE_URL="postgres://opencode:password@localhost:5432/opencode_test"
export K8S_NAMESPACE="opencode-test"
go test -tags=integration -v ./internal/api

# Frontend tests
cd frontend
npm test
```

### Kubernetes Cluster Status

```bash
# Deploy to kind cluster
make kind-deploy

# Check deployment status
kubectl get all -n opencode

# Expected pods:
# - opencode-controller-XXXXX (1/1 Running)
# - postgres-0 (1/1 Running)

# View controller logs
kubectl logs -n opencode -l app=opencode-controller --tail=50

# Test health endpoints
kubectl port-forward -n opencode svc/opencode-controller 8090:8090 &
curl http://localhost:8090/healthz  # {"status":"ok"}
curl http://localhost:8090/ready    # {"status":"ready"}
```

---

## Common Issues & Troubleshooting

### 1. "Port already in use"

```bash
# Kill process on port
lsof -i :8080
kill -9 <PID>

# Or use different port
PORT=8081 go run cmd/api/main.go
```

### 2. "Database connection refused"

```bash
# Check postgres is running
docker compose ps postgres

# Check DATABASE_URL is correct
echo $DATABASE_URL

# Try connecting directly
psql $DATABASE_URL -c "SELECT 1"
```

### 3. "OIDC token validation failed"

```bash
# Verify Keycloak is running and accessible
curl http://localhost:8081/realms/opencode

# Check OIDC_ISSUER matches Keycloak URL
# Check OIDC_CLIENT_ID matches client in Keycloak
# Check OIDC_CLIENT_SECRET is correct
```

### 4. "Kubernetes error: connection refused"

```bash
# Check kubectl context
kubectl config current-context

# Switch to kind cluster
kubectl config use-context kind-opencode-dev

# Check KUBECONFIG
echo $KUBECONFIG

# Test connection
kubectl cluster-info
```

### 5. "Frontend blank page"

```bash
# Check console for errors
open http://localhost:5173
# Open DevTools (F12) → Console

# Check API is responding
curl http://localhost:8080/healthz

# Check VITE_API_URL is correct
echo $VITE_API_URL
```

### 6. "Pod fails to start"

```bash
# Check pod status
kubectl describe pod <pod-name> -n opencode

# View logs
kubectl logs <pod-name> -n opencode

# Check resource availability
kubectl top nodes
kubectl top pods -n opencode
```

---

## Useful Commands

```bash
# Backend development
make backend-dev          # Start backend with hot reload
make backend-test         # Run tests
make backend-lint         # Run linter
make backend-build        # Build binary
make backend-migrations   # Run DB migrations

# Frontend development
make frontend-dev         # Start dev server
make frontend-test        # Run tests
make frontend-build       # Build for production
make frontend-lint        # Run linter

# Services
make dev-services         # Start postgres, keycloak, redis
make dev-services-down    # Stop all services
make dev-services-logs    # View service logs

# Database
make db-migrate-up        # Run migrations
make db-migrate-down      # Rollback migrations
make db-reset             # Drop and recreate DB

# Kubernetes
make kind-create          # Create kind cluster
make kind-deploy          # Deploy to kind
make kind-logs            # View pod logs
make kind-delete          # Delete cluster

# Docker
make docker-build-prod    # Build production images (unified)
make docker-build-dev     # Build development images (separate)
make docker-push-prod     # Build and push production images
make docker-push-dev      # Build and push development images

# All-in-one
make dev                  # Start everything (services + backends)
make clean                # Clean up everything
```

---

## Git Workflow

### Branch Naming

```bash
feature/kanban-board
bugfix/auth-token-expiry
docs/api-documentation
test/e2e-coverage
refactor/service-layer
```

### Commit Messages

```bash
feat: add kanban board drag-drop functionality
fix: correct JWT token expiration logic
docs: update API documentation
test: add integration tests for projects
refactor: extract file service layer
chore: update dependencies to latest versions

# Format: type(scope): short description
# Types: feat, fix, docs, test, refactor, chore, perf, ci
```

### Pull Requests

```bash
# Create feature branch
git checkout -b feature/your-feature

# Make changes, commit regularly
git add .
git commit -m "feat: your feature"

# Push
git push origin feature/your-feature

# Create PR on GitHub
# Wait for:
# - All tests pass
# - Code review approval
# - No conflicts

# Merge when ready
# Squash or rebase depending on branch history
```

---

## Performance Tips

### Frontend

```bash
# Use React DevTools Profiler
# Check bundle size
npm run build
npm run analyze

# Monitor network requests
# Open DevTools → Network
# Look for slow API calls or large assets
```

### Backend

```bash
# Benchmark tests
go test -bench=. ./...

# Memory profiling
go run -cpuprofile=cpu.prof cmd/api/main.go
go tool pprof cpu.prof

# Check database queries
# Enable query logging in config
# Look for N+1 queries
```

### Database

```bash
# Monitor slow queries
# In PostgreSQL:
# SET log_min_duration_statement = 1000; -- 1 second

# Analyze query plans
EXPLAIN ANALYZE SELECT * FROM tasks WHERE project_id = '...';
```

---

## Deployment Preparation

### Before Going to Production

```bash
# Security checklist
- [ ] Enable HTTPS (TLS certificates)
- [ ] Set strong JWT_SECRET (32+ chars)
- [ ] Set strong OIDC_CLIENT_SECRET
- [ ] Enable DB password encryption
- [ ] Configure RBAC in K8s
- [ ] Set resource limits on pods
- [ ] Configure network policies
- [ ] Enable audit logging
- [ ] Setup monitoring and alerting
- [ ] Review and test error handling
- [ ] Load test with realistic data
- [ ] Backup strategy for database
- [ ] Rollback procedure documented

# Build & push production images (unified)
make docker-build-prod
make docker-push-prod

# Deploy to production K8s
kubectl apply -k k8s/overlays/prod/

# Run smoke tests
npm run test:smoke

# Monitor logs and metrics
# Check dashboards
# Alert on errors
```

---

## Documentation

### Updating Documentation

When you make changes:

1. Update relevant `.md` files
2. Update API docs (OpenAPI/Swagger if applicable)
3. Update inline code comments
4. Update ARCHITECTURE.md if changing design
5. Update IMPLEMENTATION_PLAN.md if changing roadmap

### Code Comments

```go
// Good: explains why, not what
// We cache config for 5 minutes to reduce DB queries
// during high task execution periods
func getCachedConfig() {
  ...
}

// Bad: obvious from code
// Get the config
func getConfig() {
  ...
}
```

---

## IDE Setup

### VSCode Extensions

```
Go:
- golang.go
- golang.tools

Frontend:
- ES7+ React/Redux/React-Native snippets
- ESLint
- Prettier - Code formatter
- TypeScript Vue Plugin

Database:
- PostgreSQL
- SQL Formatter

General:
- GitLens
- Docker
- Kubernetes
```

### Configuration

**.vscode/settings.json**
```json
{
  "[go]": {
    "editor.defaultFormatter": "golang.go",
    "editor.formatOnSave": true
  },
  "[javascript][typescript]": {
    "editor.defaultFormatter": "esbenp.prettier-vscode",
    "editor.formatOnSave": true
  },
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "package"
}
```

---

## Resources

- [Go Documentation](https://pkg.go.dev/std)
- [React Documentation](https://react.dev/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Keycloak Documentation](https://www.keycloak.org/documentation)
- [Gin Framework](https://gin-gonic.com/)
- [TypeScript Handbook](https://www.typescriptlang.org/docs/)

---

## Getting Help

1. Check relevant `.md` files (ARCHITECTURE.md, IMPLEMENTATION_PLAN.md)
2. Check inline code comments
3. Check Troubleshooting section above
4. Ask in team Slack/Discord
5. Search GitHub issues
6. Create new issue if problem is new

