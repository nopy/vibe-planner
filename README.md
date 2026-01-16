# OpenCode Project Manager

A comprehensive web application for managing software projects using OpenCode AI agent coding. Built with Go, React, Kubernetes, and PostgreSQL.

## 🎯 Project Overview

**OpenCode Project Manager** enables teams to:
- Create and manage projects with isolated Kubernetes pods
- Define and execute tasks using OpenCode AI agents
- View task progress on a Kanban board
- Explore and edit project files with Monaco editor
- Configure AI agent settings (models, providers, tools)
- Interact bidirectionally with the AI agent during execution

**Tech Stack:**
- **Backend:** Go 1.24+ (Gin framework)
- **Frontend:** React 18+ (TypeScript, Vite)
- **Database:** PostgreSQL 15+
- **Orchestration:** Kubernetes (kind for local development)
- **Authentication:** Keycloak (OIDC)
- **Container Registry:** registry.legal-suite.com
- **AI Model:** GPT-4o mini (configurable)
- **Production Build:** Single unified Docker image (29MB) with embedded frontend

**Team Size:** 3 developers
**Scope:** MVP + Optional features for future

---

## 📋 Documentation

### For Users
- **[IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md)** - High-level project roadmap and 10-phase implementation plan
- **[DEVELOPMENT.md](./DEVELOPMENT.md)** - Developer quick start and workflow guide

### For Architects & Leads
- **[ARCHITECTURE.md](./ARCHITECTURE.md)** - Detailed system architecture, component breakdown, data flows
- **[API_SPECIFICATION.md](./API_SPECIFICATION.md)** - Complete REST API documentation (coming soon)

---

## 🚀 Quick Start

### Prerequisites
```bash
go version          # 1.21+
node -v             # 16+
docker --version
kind version        # Kubernetes
kubectl version
```

### One-Command Setup
```bash
git clone <repo-url> opencode-project-manager
cd opencode-project-manager

# Start all services
make dev

# Access the application
# Frontend: http://localhost:5173
# Backend: http://localhost:8090
# Keycloak: http://localhost:8081
```

### Detailed Setup
See [DEVELOPMENT.md](./DEVELOPMENT.md) for complete setup instructions.

---

## 📁 Project Structure

```
.
├── Dockerfile                      # Unified production build (frontend + backend)
├── backend/                        # Go backend service
│   ├── cmd/api/                   # Entry point
│   ├── internal/                  # Core application code
│   │   ├── api/                   # HTTP handlers
│   │   ├── service/               # Business logic
│   │   ├── repository/            # Database access
│   │   ├── model/                 # Domain models
│   │   ├── middleware/            # HTTP middleware (auth, security, gzip)
│   │   ├── static/                # Embedded frontend serving (production)
│   │   ├── config/                # Configuration
│   │   ├── util/                  # Utilities
│   │   └── db/                    # Database migrations
│   ├── go.mod                     # Go dependencies
│   ├── Dockerfile                 # Backend-only build (development)
│   └── .gitignore
│
├── frontend/                       # React frontend application
│   ├── src/
│   │   ├── components/            # React components
│   │   ├── hooks/                 # Custom hooks
│   │   ├── contexts/              # React contexts
│   │   ├── services/              # API clients
│   │   ├── types/                 # TypeScript types
│   │   ├── utils/                 # Utilities
│   │   ├── App.tsx                # Root component
│   │   └── main.tsx               # Entry point
│   ├── package.json               # Node dependencies
│   ├── vite.config.ts             # Vite configuration
│   ├── tsconfig.json              # TypeScript config
│   ├── Dockerfile                 # Frontend-only build (development)
│   └── nginx.conf                 # Nginx config (development only)
│
├── sidecars/                   # Kubernetes sidecar services
│   ├── file-browser/          # File browsing service (Go)
│   │   ├── cmd/main.go
│   │   ├── internal/
│   │   └── Dockerfile
│   └── session-proxy/         # OpenCode session proxy (Go)
│       ├── cmd/main.go
│       ├── internal/
│       └── Dockerfile
│
├── k8s/                        # Kubernetes manifests
│   ├── base/                  # Base manifests
│   │   ├── namespace.yaml
│   │   ├── configmap.yaml
│   │   ├── secrets.yaml
│   │   ├── postgres-statefulset.yaml
│   │   ├── controller-deployment.yaml
│   │   ├── service.yaml
│   │   ├── ingress.yaml
│   │   └── rbac.yaml
│   ├── overlays/              # Environment-specific overrides
│   │   ├── dev/
│   │   └── prod/
│   └── kind-config.yaml       # Kind cluster configuration
│
├── db/                         # Database files
│   ├── migrations/            # SQL migration files
│   │   ├── 001_init.sql
│   │   ├── 002_projects.sql
│   │   └── ...
│   └── seeds/                 # Seed data (optional)
│
├── scripts/                    # Utility scripts
│   ├── setup-keycloak.sh      # Keycloak setup
│   ├── build-images.sh        # Docker image building
│   └── deploy-kind.sh         # Kind deployment
│
├── docs/                       # Additional documentation
│   ├── ARCHITECTURE.md
│   ├── DEVELOPMENT.md
│   ├── IMPLEMENTATION_PLAN.md
│   ├── API.md
│   └── TROUBLESHOOTING.md
│
├── docker-compose.yml          # Local development services
├── Makefile                    # Build and development commands
├── .env.example               # Environment variables template
├── .gitignore
└── README.md                   # This file

```

---

## 🔄 Development Workflow

### Start Local Development Environment

```bash
# Terminal 1: Services (PostgreSQL, Keycloak, Redis)
make dev-services

# Terminal 2: Backend
cd backend
make dev

# Terminal 3: Frontend
cd frontend
make dev
```

### Useful Make Commands

```bash
# All-in-one development start
make dev

# Individual services
make dev-services              # Start Docker services
make backend-dev               # Start Go backend
make frontend-dev              # Start React frontend

# Database
make db-migrate-up             # Run migrations
make db-migrate-down           # Rollback
make db-reset                  # Reset database

# Testing
make backend-test              # Run Go tests
make frontend-test             # Run React tests
make test                      # Run all tests

# Kubernetes (kind)
make kind-create               # Create kind cluster
make kind-deploy               # Deploy to kind
make kind-logs                 # View pod logs
make kind-delete               # Delete cluster

# Docker
make docker-build-prod         # Build production images (unified)
make docker-build-dev          # Build development images (separate)
make docker-push-prod          # Build and push production
make docker-push-dev           # Build and push development

# Cleanup
make clean                     # Stop services and cleanup
```

See [Makefile](./Makefile) for all available commands.

---

## 🏗️ Architecture Overview

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                   React SPA Frontend                            │
│              (Vite, TypeScript, Tailwind)                       │
└─────────────────────┬───────────────────────────────────────────┘
                      │ HTTPS + JWT
┌─────────────────────▼───────────────────────────────────────────┐
│           Kubernetes Cluster (kind/self-hosted)                 │
│                                                                  │
│  ┌────────────────────────────────────────────────────────┐    │
│  │  Main Controller Pod (Go/Gin API Server)              │    │
│  │  ├─ Project Management                                │    │
│  │  ├─ Task Management (state machine)                   │    │
│  │  ├─ OpenCode Integration                             │    │
│  │  ├─ File Browsing Proxy                              │    │
│  │  ├─ Real-time Updates (WebSocket)                    │    │
│  │  └─ Kubernetes Pod Lifecycle Management              │    │
│  └────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ┌────────────────────────────────────────────────────────┐    │
│  │  PostgreSQL (Persistent Storage)                      │    │
│  │  ├─ Users, Projects, Tasks                           │    │
│  │  ├─ Configurations, Sessions, Interactions           │    │
│  │  └─ Audit Trail                                      │    │
│  └────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ┌────────────────────────────────────────────────────────┐    │
│  │  Per-Project Pod (Created On-Demand)                 │    │
│  │  ├─ OpenCode Server (:3000)                          │    │
│  │  ├─ File Browser Sidecar (:3001)                     │    │
│  │  ├─ Session Proxy Sidecar (:3002)                    │    │
│  │  └─ Shared PVC (workspace)                           │    │
│  └────────────────────────────────────────────────────────┘    │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│  External Services                                               │
│  ├─ Keycloak (OIDC Authentication)                             │
│  └─ Private Docker Registry (registry.legal-suite.com)         │
└──────────────────────────────────────────────────────────────────┘
```

For detailed architecture information, see [ARCHITECTURE.md](./ARCHITECTURE.md).

---

## 📊 Implementation Timeline

10 phases over 20 weeks (MVP first, optional features after):

| Phase | Duration | Focus | Status |
|-------|----------|-------|--------|
| 1 | Weeks 1-2 | Foundation (Auth, DB, basic UI) | Planning |
| 2 | Weeks 3-4 | Project Management (K8s pods) | Planning |
| 3 | Weeks 5-6 | Kanban Board & Tasks | Planning |
| 4 | Weeks 7-8 | File Explorer (Monaco editor) | Planning |
| 5 | Weeks 9-10 | OpenCode Integration | Planning |
| 6 | Weeks 11-12 | Configuration UI | Planning |
| 7 | Weeks 13-14 | Two-Way Interactions | Planning |
| 8 | Weeks 15-16 | K8s Deployment | Planning |
| 9 | Weeks 17-18 | Testing & Documentation | Planning |
| 10 | Weeks 19-20 | Polish & Optimization | Planning |

See [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md) for detailed roadmap.

---

## 🔐 Security

### MVP Security Measures
✅ OIDC authentication via Keycloak
✅ JWT token validation
✅ Database credentials management
✅ Encrypted credential storage
✅ RBAC for Kubernetes access
✅ Path traversal prevention

### Future Hardening
- [ ] Network policies
- [ ] Pod security policies
- [ ] Rate limiting
- [ ] Audit logging
- [ ] Secrets encryption at rest

See [SECURITY.md](./docs/SECURITY.md) for detailed security guidelines.

---

## 🧪 Testing

```bash
# Backend unit tests
cd backend && go test ./... -v

# Frontend unit tests
cd frontend && npm test

# Integration tests
npm run test:integration

# E2E tests (requires services running)
npm run test:e2e
```

---

## 📝 API Documentation

Base URL: `http://localhost:8080/api`

### Key Endpoints

**Authentication:**
```
POST   /auth/oidc/login         - Get OIDC login URL
POST   /auth/oidc/callback      - Handle OIDC callback
GET    /auth/me                 - Get current user
POST   /auth/logout             - Logout
```

**Projects:**
```
GET    /projects                - List projects
POST   /projects                - Create project
GET    /projects/:id            - Get project
PATCH  /projects/:id            - Update project
DELETE /projects/:id            - Delete project
```

**Tasks:**
```
GET    /projects/:id/tasks      - List tasks
POST   /projects/:id/tasks      - Create task
PATCH  /projects/:id/tasks/:taskId     - Update task
POST   /projects/:id/tasks/:taskId/execute - Execute task
GET    /projects/:id/tasks/:taskId/output  - Stream output (SSE)
```

**Files:**
```
GET    /projects/:id/files/tree         - Get directory tree
GET    /projects/:id/files/content      - Get file content
POST   /projects/:id/files/write        - Write file
DELETE /projects/:id/files              - Delete file
```

**Configuration:**
```
GET    /projects/:id/config             - Get active config
POST   /projects/:id/config             - Create/update config
GET    /projects/:id/config/versions    - List config versions
```

See [API_SPECIFICATION.md](./docs/API_SPECIFICATION.md) for complete documentation.

---

## 🐳 Docker & Kubernetes

### Docker Images

**Production (Unified):**
```
registry.legal-suite.com/opencode/app:latest           # Backend + Frontend (29MB)
registry.legal-suite.com/opencode/file-browser-sidecar:latest
registry.legal-suite.com/opencode/session-proxy-sidecar:latest
```

**Development (Separate):**
```
registry.legal-suite.com/opencode/backend:latest       # Backend only
registry.legal-suite.com/opencode/frontend:latest      # Frontend + nginx
```

**Build Production Image:**
```bash
docker build -t registry.legal-suite.com/opencode/app:latest -f Dockerfile .
```

### Local Kubernetes (Kind)

```bash
# Create cluster
kind create cluster --config k8s/kind-config.yaml --name opencode-dev

# Deploy application
kubectl apply -k k8s/base/

# Port forward
kubectl port-forward -n opencode svc/opencode-controller 8090:8090
```

See [DEVELOPMENT.md](./DEVELOPMENT.md#kind-kubernetes-cluster) for detailed K8s instructions.

---

## 🚢 Production Deployment

```bash
# Build and push production images
make docker-build-prod
make docker-push-prod

# Deploy to production K8s
kubectl apply -k k8s/overlays/prod/

# Verify deployment
kubectl get all -n opencode
kubectl logs -n opencode <pod-name>
```

See [DEPLOYMENT.md](./docs/DEPLOYMENT.md) for production deployment guide.

---

## 📖 Team Resources

- **[IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md)** - Project roadmap and phase breakdown
- **[DEVELOPMENT.md](./DEVELOPMENT.md)** - Developer setup and workflow
- **[ARCHITECTURE.md](./ARCHITECTURE.md)** - System design and component details
- **[API_SPECIFICATION.md](./docs/API_SPECIFICATION.md)** - API reference (coming soon)
- **[DEPLOYMENT.md](./docs/DEPLOYMENT.md)** - Production deployment guide (coming soon)
- **[TROUBLESHOOTING.md](./docs/TROUBLESHOOTING.md)** - Common issues and solutions (coming soon)

---

## 🤝 Contributing

### Code Style

**Go:**
```bash
go fmt ./...
go vet ./...
```

**TypeScript/React:**
```bash
npm run lint
npm run format
```

### Git Workflow

```bash
# Create feature branch
git checkout -b feature/your-feature

# Make changes and commit
git commit -m "feat: description"

# Push and create PR
git push origin feature/your-feature
```

### Pull Request Checklist
- [ ] Tests pass locally
- [ ] Code follows style guide
- [ ] No security issues
- [ ] Documentation updated
- [ ] PR description clear

See [DEVELOPMENT.md#git-workflow](./DEVELOPMENT.md#git-workflow) for detailed guidelines.

---

## ❓ Troubleshooting

### Common Issues

**Services won't start:**
```bash
# Check ports
lsof -i :8080
lsof -i :5173
lsof -i :8081

# Check Docker
docker ps
docker logs <container-name>
```

**Database connection error:**
```bash
# Test connection
psql $DATABASE_URL -c "SELECT 1"

# Check migration status
migrate -path db/migrations -database "$DATABASE_URL" version
```

**OIDC token validation fails:**
```bash
# Check Keycloak is accessible
curl http://localhost:8081/realms/opencode

# Verify environment variables
echo $OIDC_ISSUER
echo $OIDC_CLIENT_ID
```

See [DEVELOPMENT.md#troubleshooting](./DEVELOPMENT.md#common-issues--troubleshooting) for more solutions.

---

## 📞 Support & Questions

1. Check [DEVELOPMENT.md](./DEVELOPMENT.md) for common questions
2. Check [ARCHITECTURE.md](./ARCHITECTURE.md) for design questions
3. Check [TROUBLESHOOTING.md](./docs/TROUBLESHOOTING.md) for issues
4. Ask team members in Slack/Discord
5. Create GitHub issue for bugs/features

---

## 📄 License

[To be determined]

---

## 👥 Team

- **3 Developers**
- **Sprint Duration:** Flexible (no fixed timeframe)
- **MVP Scope:** Core features (see [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md))
- **Optional Features:** Advanced features (defer for Phase 2+)

---

## 🔗 External References

- **OpenCode:** https://github.com/anomalyco/opencode
- **Keycloak:** https://www.keycloak.org/
- **Kubernetes:** https://kubernetes.io/
- **Gin Framework:** https://gin-gonic.com/
- **React:** https://react.dev/
- **PostgreSQL:** https://www.postgresql.org/

---

## ✅ Bootstrap Status

**COMPLETED** - All foundational structure in place (January 2026)

### What's Ready:
- ✅ Complete directory structure (backend, frontend, sidecars, k8s, db, scripts)
- ✅ All Go modules compile successfully (backend + 2 sidecars)
- ✅ Database schema defined (001_init.sql with all tables and migrations)
- ✅ Docker Compose for local services (PostgreSQL, Keycloak, Redis)
- ✅ Kubernetes manifests (base + dev/prod overlays)
- ✅ Frontend structure (React + TypeScript + Vite + Tailwind)
- ✅ Utility scripts (Keycloak setup, image building, Kind deployment)
- ✅ All Dockerfiles (multi-stage builds for all components)

### Next Steps:
1. **Install Dependencies:** `cd frontend && npm install`
2. **Start Services:** `make dev-services`
3. **Run Migrations:** `make db-migrate-up`
4. **Begin Phase 1:** Implement OIDC authentication flow

### Files Created: 43 source files
- Go: 14 files (backend + sidecars)
- TypeScript/React: 12 files
- SQL: 2 migration files
- Kubernetes: 8 manifests
- Docker: 4 Dockerfiles + 1 docker-compose.yml
- Scripts: 3 shell scripts
- Config: 9 configuration files

---

**Last Updated:** January 15, 2026
**Version:** 1.0.0 (Bootstrap Complete - Ready for Phase 1)

