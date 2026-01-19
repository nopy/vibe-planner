# Integration Testing Guide

This document describes how to run the end-to-end integration tests for the OpenCode Project Manager backend.

## Overview

Integration tests verify the complete application lifecycle by testing:
1. **Project Management**: Project creation, Kubernetes pod orchestration, PVC creation, deletion and cleanup
2. **Task Execution**: Task lifecycle, OpenCode session execution, real-time output streaming
3. **Configuration Management**: Config versioning, API key encryption, rollback functionality

**Test Files:**
- `backend/internal/api/projects_integration_test.go` - Project lifecycle tests
- `backend/internal/api/tasks_execution_integration_test.go` - Task execution tests
- `backend/internal/api/config_integration_test.go` - Configuration management tests (Phase 6)

## Prerequisites

### 1. PostgreSQL Database

You need a running PostgreSQL instance for integration tests. **Use a separate test database** to avoid data conflicts.

```bash
# Option 1: Use docker-compose test database
docker run -d \
  --name postgres-test \
  -e POSTGRES_DB=opencode_test \
  -e POSTGRES_USER=opencode \
  -e POSTGRES_PASSWORD=password \
  -p 5433:5432 \
  postgres:15-alpine

# Option 2: Use existing PostgreSQL server
# Create test database manually:
psql -U postgres -c "CREATE DATABASE opencode_test;"
```

### 2. Kubernetes Cluster

Integration tests require access to a Kubernetes cluster. **Use a test namespace** to isolate test resources.

#### Option A: Local Kind Cluster (Recommended)

```bash
# Create kind cluster
kind create cluster --name opencode-test

# Create test namespace
kubectl create namespace opencode-test

# Verify cluster access
kubectl cluster-info
kubectl get nodes
```

#### Option B: Other Kubernetes Cluster

If you have access to another cluster (minikube, k3s, cloud provider, etc.):

```bash
# Verify cluster access
kubectl cluster-info

# Create test namespace
kubectl create namespace opencode-test

# Verify RBAC permissions
kubectl auth can-i create pods -n opencode-test
kubectl auth can-i create pvc -n opencode-test
```

### 3. Environment Variables

Set the following environment variables before running tests:

```bash
# Database connection
export TEST_DATABASE_URL="postgres://opencode:password@localhost:5433/opencode_test"
# Or fallback to regular database URL (not recommended for safety)
export DATABASE_URL="postgres://opencode:password@localhost:5432/opencode_dev"

# Kubernetes configuration
export KUBECONFIG="$HOME/.kube/config"  # Path to kubeconfig (omit for in-cluster)
export K8S_NAMESPACE="opencode-test"    # Kubernetes namespace for tests

# Configuration encryption key (for config integration tests)
export CONFIG_ENCRYPTION_KEY="$(openssl rand -base64 32)"  # Generate 32-byte key for tests

# Optional: Database migration path (if needed)
export MIGRATION_PATH="../db/migrations"
```

## Running Integration Tests

### Run All Integration Tests

```bash
cd backend

# Run with verbose output
go test -tags=integration -v ./internal/api

# Run with timeout (recommended for long-running tests)
go test -tags=integration -v -timeout 10m ./internal/api
```

### Run Specific Test

```bash
# Run only the project lifecycle test
go test -tags=integration -v -run TestProjectLifecycle ./internal/api

# Run only the pod failure test
go test -tags=integration -v -run TestProjectCreation_PodFailure ./internal/api

# Run only the config lifecycle test
go test -tags=integration -v -run TestConfigLifecycle ./internal/api

# Run only the config encryption test
go test -tags=integration -v -run TestConfigAPIKeyEncryption ./internal/api
```

### Skip Integration Tests (Default Behavior)

Integration tests are automatically skipped in the following scenarios:

1. **Build tag not specified** - Running `go test ./...` without `-tags=integration` will skip these tests
2. **Short mode** - Running `go test -short` will skip integration tests
3. **Missing prerequisites** - Tests will be skipped with a message if:
   - Database connection cannot be established
   - Kubernetes client cannot be initialized

```bash
# These commands will NOT run integration tests
go test ./...              # No build tag
go test -short ./...       # Short mode
go test -v ./internal/api  # No build tag
```

## Test Scenarios

### 1. TestProjectLifecycle_Integration

**Full end-to-end project lifecycle test:**

1. **CreateProject**: POST /api/projects
   - Validates request body
   - Creates project in database
   - Spawns Kubernetes pod with 3 containers
   - Creates PersistentVolumeClaim
   - Returns project with pod metadata

2. **VerifyPodCreated**: Kubernetes pod status check
   - Queries Kubernetes API for pod status
   - Asserts pod is in "Pending" or "Running" state

3. **VerifyPVCCreated**: PersistentVolumeClaim verification
   - Validates PVC naming convention: `workspace-{project-id}`
   - Verifies PVC metadata stored in project

4. **GetProjectByID**: GET /api/projects/:id
   - Retrieves project by UUID
   - Verifies all fields match created project

5. **ListProjects**: GET /api/projects
   - Lists all projects for user
   - Verifies test project appears in list

6. **DeleteProjectAndVerifyCleanup**: DELETE /api/projects/:id
   - Deletes project via API
   - Verifies pod deleted from Kubernetes
   - Verifies PVC deleted from Kubernetes
   - Verifies project soft-deleted in database
   - Verifies deleted project not returned in list

**Expected Duration:** ~10-15 seconds (depends on K8s cluster responsiveness)

### 2. TestProjectCreation_PodFailure_Integration

**Tests graceful handling of pod creation failures:**

1. Creates project with potentially invalid configuration
2. Verifies project still created in database (partial success model)
3. Checks pod error is stored in `project.pod_error` field
4. Verifies project status is set appropriately

**Expected Duration:** ~5 seconds

### 3. TestConfigLifecycle_Integration (Phase 6)

**Complete configuration management lifecycle test:**

1. **CreateInitialConfig**: POST /api/projects/:id/config
   - Creates config with version=1 and API key encryption
   - Verifies config saved and active

2. **VerifyEncryption**: Direct database query
   - Validates API key is encrypted in database (not plaintext)
   - Confirms sanitization in API responses

3. **UpdateConfig**: POST /api/projects/:id/config (again)
   - Creates new version=2 with different model/settings
   - Verifies version=1 deactivated automatically

4. **GetConfigHistory**: GET /api/projects/:id/config/versions
   - Retrieves all versions in reverse chronological order
   - Verifies API keys sanitized in history

5. **RollbackToVersion**: POST /api/projects/:id/config/rollback/:version
   - Rolls back to version=1 by creating version=3 (copy of v1)
   - Verifies version=2 deactivated

6. **DeleteProjectAndVerifyCleanup**: DELETE /api/projects/:id
   - Deletes project
   - Verifies all configs cascade deleted via foreign key

**Expected Duration:** ~3 seconds

### 4. TestConfigAPIKeyEncryption_Integration (Phase 6)

**Tests API key security and encryption:**

1. **CreateConfigWithAPIKey**: Encrypt API key during config creation
2. **VerifyNotPlaintext**: Query database and verify encrypted bytes don't contain plaintext
3. **VerifyAPISanitization**: GetActiveConfig and GetConfigHistory return null API keys
4. **GetDecryptedAPIKey**: Internal service method successfully decrypts original key
5. **TestNoKeyScenario**: Config without API key returns error on decryption attempt
6. **TestSpecialCharacters**: Encrypt/decrypt keys with special characters (round-trip)
7. **TestNonDeterminism**: Verify same key encrypted twice produces different ciphertext (nonce randomness)

**Expected Duration:** ~2 seconds

## Troubleshooting

### Common Issues

#### 1. Database Connection Errors

**Error:** `TEST_DATABASE_URL or DATABASE_URL environment variable not set`

**Solution:**
```bash
export TEST_DATABASE_URL="postgres://opencode:password@localhost:5433/opencode_test"
```

**Error:** `Failed to connect to database: connection refused`

**Solution:**
```bash
# Check if PostgreSQL is running
docker ps | grep postgres

# Verify connection manually
psql $TEST_DATABASE_URL -c "SELECT 1"
```

#### 2. Kubernetes Connection Errors

**Error:** `Failed to initialize Kubernetes service`

**Solution:**
```bash
# Verify kubeconfig is valid
kubectl cluster-info

# Check KUBECONFIG environment variable
echo $KUBECONFIG

# Verify namespace exists
kubectl get namespace $K8S_NAMESPACE

# Create namespace if missing
kubectl create namespace opencode-test
```

#### 3. RBAC Permission Errors

**Error:** `pods is forbidden: User "..." cannot create resource "pods" in API group "" in namespace "opencode-test"`

**Solution:**
```bash
# Grant permissions to your user (for local testing only)
kubectl create clusterrolebinding test-admin-binding \
  --clusterrole=cluster-admin \
  --user=your-username

# Or create a service account with proper RBAC
kubectl apply -f k8s/base/rbac.yaml
```

#### 4. Pod Stays in "Pending" State

**Cause:** Docker images not available in cluster

**Solution:**
```bash
# For kind cluster, load images manually
kind load docker-image registry.legal-suite.com/opencode/server:latest --name opencode-test
kind load docker-image registry.legal-suite.com/opencode/file-browser-sidecar:latest --name opencode-test
kind load docker-image registry.legal-suite.com/opencode/session-proxy-sidecar:latest --name opencode-test

# Or use publicly available test images
# (Modify KubernetesConfig in test setup to use busybox or nginx for testing)
```

#### 5. Tests Timeout

**Error:** `test timed out after 2m0s`

**Solution:**
```bash
# Increase timeout
go test -tags=integration -v -timeout 10m ./internal/api

# Check Kubernetes cluster performance
kubectl top nodes
kubectl get events -n opencode-test --sort-by='.lastTimestamp'
```

### Debugging Tips

#### 1. Enable Verbose Logging

```bash
# Run tests with verbose output
go test -tags=integration -v ./internal/api

# Run with Go race detector (slower but catches concurrency issues)
go test -tags=integration -race -v ./internal/api
```

#### 2. Inspect Kubernetes Resources

```bash
# List all pods in test namespace
kubectl get pods -n opencode-test

# Describe specific pod (replace POD_NAME)
kubectl describe pod POD_NAME -n opencode-test

# Get pod logs
kubectl logs POD_NAME -n opencode-test

# List PVCs
kubectl get pvc -n opencode-test

# Describe PVC
kubectl describe pvc PVC_NAME -n opencode-test
```

#### 3. Manual Cleanup

If tests fail and leave resources behind:

```bash
# Delete all pods in test namespace
kubectl delete pods --all -n opencode-test

# Delete all PVCs in test namespace
kubectl delete pvc --all -n opencode-test

# Clean up database
psql $TEST_DATABASE_URL -c "DELETE FROM opencode_configs WHERE project_id IN (SELECT id FROM projects WHERE name LIKE 'config-integration-test-%');"
psql $TEST_DATABASE_URL -c "DELETE FROM projects WHERE name LIKE 'integration-test-%' OR name LIKE 'config-integration-test-%';"
psql $TEST_DATABASE_URL -c "DELETE FROM users WHERE email LIKE 'test-%@integration.test' OR email LIKE 'config-test-%@integration.test';"
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  integration:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15-alpine
        env:
          POSTGRES_DB: opencode_test
          POSTGRES_USER: opencode
          POSTGRES_PASSWORD: password
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.24'

      - name: Set up Kind
        uses: helm/kind-action@v1.5.0
        with:
          cluster_name: opencode-test

      - name: Create test namespace
        run: kubectl create namespace opencode-test

      - name: Run integration tests
        env:
          TEST_DATABASE_URL: postgres://opencode:password@localhost:5432/opencode_test
          K8S_NAMESPACE: opencode-test
        run: |
          cd backend
          go test -tags=integration -v -timeout 10m ./internal/api
```

## Best Practices

1. **Always use a separate test database** - Never run integration tests against production or development databases
2. **Use a dedicated namespace** - Isolate test resources in `opencode-test` namespace
3. **Clean up resources** - Tests include cleanup logic, but verify manually if tests fail
4. **Run tests in CI/CD** - Automate integration tests in your CI pipeline
5. **Monitor test duration** - Integration tests should complete within 2-5 minutes
6. **Parallelize carefully** - Tests create real K8s resources; avoid overwhelming cluster

## Additional Resources

- [Go Testing Documentation](https://pkg.go.dev/testing)
- [Testify Assertions](https://pkg.go.dev/github.com/stretchr/testify/assert)
- [Kubernetes Client-Go](https://github.com/kubernetes/client-go)
- [Kind Documentation](https://kind.sigs.k8s.io/)

---

**Last Updated:** 2026-01-19
**Maintainer:** Sisyphus (OpenCode AI Agent)
