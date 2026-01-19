.PHONY: help dev dev-services backend-dev frontend-dev db-migrate-up db-migrate-down db-reset test backend-test frontend-test kind-create kind-deploy kind-delete docker-build-prod docker-build-dev docker-push-prod docker-push-dev clean

# Default target
help:
	@echo "OpenCode Project Manager - Development Commands"
	@echo ""
	@echo "Setup:"
	@echo "  make setup              - Install all dependencies"
	@echo "  make dev                - Start all services (one command)"
	@echo ""
	@echo "Development:"
	@echo "  make dev-services       - Start Docker services (PostgreSQL, Keycloak, Redis)"
	@echo "  make backend-dev        - Start Go backend"
	@echo "  make frontend-dev       - Start React frontend"
	@echo ""
	@echo "Database:"
	@echo "  make db-migrate-up      - Run database migrations"
	@echo "  make db-migrate-down    - Rollback database migrations"
	@echo "  make db-reset           - Reset database"
	@echo ""
	@echo "Testing:"
	@echo "  make test               - Run all tests"
	@echo "  make backend-test       - Run Go tests"
	@echo "  make frontend-test      - Run React tests"
	@echo ""
	@echo "Kubernetes (kind):"
	@echo "  make kind-create        - Create kind cluster"
	@echo "  make kind-deploy        - Deploy to kind"
	@echo "  make kind-logs          - View pod logs"
	@echo "  make kind-delete        - Delete kind cluster"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build-prod    - Build production images (unified)"
	@echo "  make docker-build-dev     - Build development images (separate)"
	@echo "  make docker-push-prod     - Build and push production images"
	@echo "  make docker-push-dev      - Build and push development images"
	@echo ""
	@echo "Sidecars:"
	@echo "  make opencode-server-build  - Build opencode-server sidecar"
	@echo "  make file-browser-build     - Build file-browser sidecar"
	@echo "  make session-proxy-build    - Build session-proxy sidecar"
	@echo "  make sidecars-build         - Build all sidecars"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clean              - Stop all services and cleanup"

# Setup
setup:
	@echo "Installing dependencies..."
	@cd backend && go mod download
	@cd frontend && npm install
	@echo "Setup complete!"

# Development - Start everything
dev: dev-services
	@echo "Starting backend and frontend..."
	@echo "Backend will be available at http://localhost:8080"
	@echo "Frontend will be available at http://localhost:5173"
	@echo "Press Ctrl+C to stop"
	@$(MAKE) -j2 backend-dev frontend-dev

# Start Docker services
dev-services:
	@echo "Starting Docker services..."
	@docker compose up -d postgres keycloak redis
	@echo "Waiting for services to be ready..."
	@sleep 10
	@echo "Services started!"

dev-services-down:
	@docker compose down

dev-services-logs:
	@docker compose logs -f

# Backend development
backend-dev:
	@echo "Starting backend..."
	@cd backend && go run cmd/api/main.go

backend-build:
	@echo "Building backend..."
	@cd backend && go build -o opencode-api cmd/api/main.go

backend-test:
	@echo "Running backend tests..."
	@cd backend && go test ./... -v

backend-lint:
	@echo "Linting backend..."
	@cd backend && go fmt ./...
	@cd backend && go vet ./...

# Frontend development
frontend-dev:
	@echo "Starting frontend..."
	@cd frontend && npm run dev

frontend-build:
	@echo "Building frontend..."
	@cd frontend && npm run build

frontend-test:
	@echo "Running frontend tests..."
	@cd frontend && npm test

frontend-lint:
	@echo "Linting frontend..."
	@cd frontend && npm run lint

# Database migrations
db-migrate-up:
	@echo "Running database migrations..."
	@migrate -path db/migrations -database "${DATABASE_URL}" up

db-migrate-down:
	@echo "Rolling back database migrations..."
	@migrate -path db/migrations -database "${DATABASE_URL}" down 1

db-reset:
	@echo "Resetting database..."
	@migrate -path db/migrations -database "${DATABASE_URL}" drop -f
	@migrate -path db/migrations -database "${DATABASE_URL}" up

# Testing
test: backend-test frontend-test

# Kubernetes (kind)
kind-create:
	@echo "Creating kind cluster..."
	@kind create cluster --config k8s/kind-config.yaml --name opencode-dev
	@kubectl cluster-info

kind-deploy:
	@echo "Deploying to kind..."
	@./scripts/deploy-kind.sh

kind-logs:
	@kubectl logs -n opencode -l app=opencode-controller -f

kind-delete:
	@echo "Deleting kind cluster..."
	@kind delete cluster --name opencode-dev

# Docker
docker-build-prod:
	@./scripts/build-images.sh --mode prod

docker-build-dev:
	@./scripts/build-images.sh --mode dev

docker-push-prod:
	@./scripts/build-images.sh --mode prod --push

docker-push-dev:
	@./scripts/build-images.sh --mode dev --push

# Sidecars
opencode-server-build:
	@echo "Building opencode-server sidecar..."
	@docker build -t registry.legal-suite.com/opencode/opencode-server-sidecar:latest sidecars/opencode-server/

file-browser-build:
	@echo "Building file-browser sidecar..."
	@docker build -t registry.legal-suite.com/opencode/file-browser-sidecar:latest sidecars/file-browser/

session-proxy-build:
	@echo "Building session-proxy sidecar..."
	@docker build -t registry.legal-suite.com/opencode/session-proxy-sidecar:latest sidecars/session-proxy/

sidecars-build: opencode-server-build file-browser-build session-proxy-build
	@echo "All sidecars built successfully!"

# Cleanup
clean:
	@echo "Cleaning up..."
	@docker compose down -v
	@rm -rf backend/opencode-api
	@rm -rf frontend/dist
	@rm -rf frontend/node_modules/.vite
	@echo "Cleanup complete!"
