# Phase 2 Deployment and Testing Guide

**Document Version:** 1.0  
**Last Updated:** 2026-01-17  
**Phase:** 2.12 - Infrastructure Setup and E2E Testing

---

## Overview

This guide provides step-by-step instructions for deploying the OpenCode Project Manager to a kind (Kubernetes in Docker) cluster and performing end-to-end testing of the project management features implemented in Phase 2.

### What Gets Deployed

1. **OpenCode Controller Pod** - Main backend API server with RBAC permissions
2. **Supporting Infrastructure** - ConfigMaps, Secrets, Services, RBAC
3. **Dynamic Project Pods** - Created on-demand via API calls (3 containers each)

### Architecture Summary

```
kind cluster (opencode-dev)
├── opencode namespace
│   ├── opencode-controller deployment (1 pod)
│   │   └── Container: opencode-api (port 8090)
│   ├── ServiceAccount: opencode-controller
│   ├── Role: opencode-controller (pod/pvc management)
│   └── Dynamic project pods (created via API)
│       ├── Container 1: opencode-server (port 3000)
│       ├── Container 2: file-browser (port 3001)
│       ├── Container 3: session-proxy (port 3002)
│       └── PVC: workspace-{project-id} (1Gi)
```

---

## Prerequisites

### Required Tools

```bash
# Verify all tools are installed
kind version          # v0.20.0+
kubectl version       # v1.28.0+
docker --version      # 20.10.0+
curl --version        # 7.68.0+
jq --version          # 1.6+ (optional, for JSON formatting)
```

### Required Images

Before deployment, ensure these images are available:

```bash
# Main application (production unified build)
registry.legal-suite.com/opencode/app:latest

# Sidecar images (for project pods)
registry.legal-suite.com/opencode/file-browser-sidecar:latest
registry.legal-suite.com/opencode/session-proxy-sidecar:latest

# Build images (if not already built)
cd /home/npinot/vibe
./scripts/build-images.sh --mode prod --push
```

---

## Step 1: Create kind Cluster

### 1.1 Create Cluster

```bash
cd /home/npinot/vibe

# Create 3-node cluster (1 control-plane, 2 workers)
kind create cluster --config k8s/kind-config.yaml --name opencode-dev
```

**Expected Output:**
```
Creating cluster "opencode-dev" ...
 ✓ Ensuring node image (kindest/node:v1.27.3)
 ✓ Preparing nodes 📦 📦 📦
 ✓ Writing configuration 📜
 ✓ Starting control-plane 🕹️
 ✓ Installing CNI 🔌
 ✓ Installing StorageClass 💾
 ✓ Joining worker nodes 🚜
Set kubectl context to "kind-opencode-dev"
```

### 1.2 Verify Cluster

```bash
# Check cluster exists
kind get clusters

# Check nodes are ready
kubectl get nodes

# Verify StorageClass exists (required for PVCs)
kubectl get storageclass
```

**Expected Output:**
```
NAME                 STATUS   ROLES           AGE
opencode-dev-control-plane   Ready    control-plane   1m
opencode-dev-worker          Ready    <none>          1m
opencode-dev-worker2         Ready    <none>          1m

NAME                 PROVISIONER             RECLAIMPOLICY
standard (default)   rancher.io/local-path   Delete
```

### 1.3 Load Images to kind (if using local builds)

```bash
# Load main application image
kind load docker-image registry.legal-suite.com/opencode/app:latest --name opencode-dev

# Load sidecar images
kind load docker-image registry.legal-suite.com/opencode/file-browser-sidecar:latest --name opencode-dev
kind load docker-image registry.legal-suite.com/opencode/session-proxy-sidecar:latest --name opencode-dev
```

---

## Step 2: Deploy Application

### 2.1 Review Configuration

Before deploying, review and customize if needed:

**ConfigMap** (`k8s/base/configmap.yaml`):
```yaml
# Key settings to verify:
ENVIRONMENT: "production"
PORT: "8090"
K8S_NAMESPACE: "opencode"
DATABASE_URL: "postgres://..." # Update if using external DB
OIDC_ISSUER: "http://keycloak:8080/realms/opencode" # Update for your setup
```

**Secrets** (`k8s/base/secrets.yaml`):
```yaml
# IMPORTANT: Update these base64-encoded secrets for production!
JWT_SECRET: <base64-encoded-secret>
OIDC_CLIENT_SECRET: <base64-encoded-secret>
```

To generate new secrets:
```bash
# Generate random secret
openssl rand -base64 32

# Encode to base64 (for secrets.yaml)
echo -n "your-secret-here" | base64
```

### 2.2 Deploy Using Script

```bash
cd /home/npinot/vibe

# Deploy all manifests
./scripts/deploy-kind.sh
```

**Expected Output:**
```
Deploying to kind cluster 'opencode-dev'...
Creating namespace...
namespace/opencode created
Applying Kubernetes manifests...
namespace/opencode unchanged
serviceaccount/opencode-controller created
role.rbac.authorization.k8s.io/opencode-controller created
rolebinding.rbac.authorization.k8s.io/opencode-controller created
configmap/app-config created
secret/app-secrets created
deployment.apps/opencode-controller created
service/opencode-controller created
Waiting for pods to be ready...
pod/opencode-controller-xxx condition met
Deployment complete!
```

### 2.3 Manual Deployment (Alternative)

```bash
# Deploy manifests manually
kubectl create namespace opencode
kubectl apply -k k8s/base/

# Watch deployment progress
kubectl get pods -n opencode -w
```

---

## Step 3: Verify Deployment

### 3.1 Check All Resources

```bash
# Check all resources in namespace
kubectl get all -n opencode

# Expected output:
# - 1 deployment (opencode-controller)
# - 1 pod (opencode-controller-xxx)
# - 1 service (opencode-controller)
# - 1 replicaset
```

### 3.2 Verify RBAC

```bash
# Check ServiceAccount
kubectl get sa -n opencode opencode-controller

# Check Role
kubectl get role -n opencode opencode-controller

# Check RoleBinding
kubectl get rolebinding -n opencode opencode-controller

# Verify deployment uses ServiceAccount
kubectl get deployment opencode-controller -n opencode -o jsonpath='{.spec.template.spec.serviceAccountName}'
```

**Expected:** `opencode-controller`

### 3.3 Check Pod Status

```bash
# Check pod is running
kubectl get pods -n opencode

# Check pod events for errors
kubectl describe pod -n opencode -l app=opencode

# View logs
kubectl logs -n opencode -l app=opencode --tail=50
```

**Healthy Pod Indicators:**
- Status: `Running`
- Ready: `1/1`
- Restarts: `0`
- No error events in `kubectl describe`

### 3.4 Test Health Endpoints

```bash
# Port-forward to access API
kubectl port-forward -n opencode svc/opencode-controller 8090:8090 &

# Test health endpoint
curl http://localhost:8090/healthz
# Expected: {"status":"ok"}

# Test readiness endpoint
curl http://localhost:8090/ready
# Expected: {"status":"ready"}
```

---

## Step 4: End-to-End Testing

### 4.1 Prerequisites for E2E Tests

Before testing project creation, ensure you have:

1. **Authentication Token**: Get a valid JWT token
   ```bash
   # Option 1: Login via frontend and extract token from browser DevTools
   # Option 2: Use OIDC flow to get token (see Phase 1 docs)
   export TOKEN="your-jwt-token-here"
   ```

2. **Port-forward running**:
   ```bash
   kubectl port-forward -n opencode svc/opencode-controller 8090:8090
   ```

### 4.2 Test 1: Create Project

```bash
# Create a test project
curl -X POST http://localhost:8090/api/projects \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-project",
    "description": "E2E test project",
    "repo_url": "https://github.com/example/repo.git"
  }' | jq

# Expected response (201 Created):
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "test-project",
  "slug": "test-project",
  "description": "E2E test project",
  "repo_url": "https://github.com/example/repo.git",
  "pod_status": "Pending",
  "pod_name": "project-550e8400-...",
  "pod_namespace": "opencode",
  "workspace_pvc_name": "workspace-550e8400-...",
  "created_at": "2026-01-17T13:00:00Z",
  "updated_at": "2026-01-17T13:00:00Z"
}

# Save project ID for later tests
export PROJECT_ID="550e8400-e29b-41d4-a716-446655440000"
```

### 4.3 Test 2: Verify Pod Created

```bash
# List all pods (should see new project pod)
kubectl get pods -n opencode

# Expected output:
# opencode-controller-xxx    1/1   Running   0   10m
# project-550e8400-xxx       0/3   Pending   0   5s   <- NEW POD

# Wait for pod to be ready (3/3 containers)
kubectl wait --for=condition=ready pod -l project-id=$PROJECT_ID -n opencode --timeout=120s

# Check pod details
kubectl describe pod -l project-id=$PROJECT_ID -n opencode
```

**Expected Pod Structure:**
- **3 containers**: `opencode-server`, `file-browser`, `session-proxy`
- **1 volume**: `workspace` (from PVC)
- **Labels**: `project-id=<uuid>`, `app=opencode-project`

### 4.4 Test 3: Verify PVC Created

```bash
# List PVCs in namespace
kubectl get pvc -n opencode

# Expected:
# workspace-550e8400-...   Bound   pvc-xxx   1Gi   RWO   standard   10s

# Check PVC details
kubectl describe pvc workspace-$PROJECT_ID -n opencode
```

**PVC Verification Checklist:**
- ✅ Status: `Bound`
- ✅ Size: `1Gi`
- ✅ AccessMode: `ReadWriteOnce`
- ✅ StorageClass: `standard` (kind default)
- ✅ Naming: `workspace-{project-id}`

### 4.5 Test 4: Check Container Logs

```bash
# Get pod name
POD_NAME=$(kubectl get pod -n opencode -l project-id=$PROJECT_ID -o jsonpath='{.items[0].metadata.name}')

# Check opencode-server logs
kubectl logs -n opencode $POD_NAME -c opencode-server --tail=20

# Check file-browser logs
kubectl logs -n opencode $POD_NAME -c file-browser --tail=20

# Check session-proxy logs
kubectl logs -n opencode $POD_NAME -c session-proxy --tail=20
```

**What to Look For:**
- ✅ No error messages
- ✅ Server started successfully
- ✅ Listening on expected ports (3000, 3001, 3002)
- ✅ Workspace mounted correctly

### 4.6 Test 5: Get Project Status

```bash
# Get project details
curl http://localhost:8090/api/projects/$PROJECT_ID \
  -H "Authorization: Bearer $TOKEN" | jq

# Expected:
# pod_status should be "Running" (if all containers started)
# pod_name, pod_namespace, workspace_pvc_name should be populated
```

### 4.7 Test 6: List Projects

```bash
# List all user's projects
curl http://localhost:8090/api/projects \
  -H "Authorization: Bearer $TOKEN" | jq

# Expected: Array with at least one project (the one we created)
```

### 4.8 Test 7: Delete Project

```bash
# Delete project
curl -X DELETE http://localhost:8090/api/projects/$PROJECT_ID \
  -H "Authorization: Bearer $TOKEN"

# Expected: 204 No Content

# Verify pod deleted
kubectl get pods -n opencode -l project-id=$PROJECT_ID
# Expected: No resources found

# Verify PVC deleted
kubectl get pvc -n opencode workspace-$PROJECT_ID
# Expected: Error from server (NotFound)

# Verify soft delete in API
curl http://localhost:8090/api/projects/$PROJECT_ID \
  -H "Authorization: Bearer $TOKEN"
# Expected: 404 Not Found
```

---

## Step 5: Integration Tests (Optional)

The backend includes integration tests that perform automated E2E testing.

### 5.1 Prerequisites

```bash
# Ensure PostgreSQL test database exists
docker run -d --name postgres-test \
  -e POSTGRES_DB=opencode_test \
  -e POSTGRES_USER=opencode \
  -e POSTGRES_PASSWORD=password \
  -p 5433:5432 postgres:15-alpine

# Set environment variables
export TEST_DATABASE_URL="postgres://opencode:password@localhost:5433/opencode_test"
export K8S_NAMESPACE="opencode"
export KUBECONFIG="$HOME/.kube/config"  # or kind kubeconfig path
```

### 5.2 Run Integration Tests

```bash
cd backend

# Run integration tests
go test -tags=integration -v ./internal/api -timeout 10m

# Expected output:
# === RUN   TestProjectLifecycle_Integration
# --- PASS: TestProjectLifecycle_Integration (30.00s)
# === RUN   TestProjectCreation_PodFailure_Integration
# --- PASS: TestProjectCreation_PodFailure_Integration (5.00s)
# PASS
```

**Tests Covered:**
1. Complete project lifecycle (create → verify pod → verify PVC → delete → cleanup)
2. Pod failure graceful handling (partial success model)

See `backend/INTEGRATION_TESTING.md` for detailed integration test documentation.

---

## Troubleshooting

### Issue: Pod Stuck in Pending

**Symptoms:**
```bash
kubectl get pods -n opencode
# project-xxx   0/3   Pending   0   5m
```

**Diagnosis:**
```bash
kubectl describe pod -n opencode -l app=opencode-project
# Check Events section for errors
```

**Common Causes:**
1. **PVC not bound**: StorageClass missing or provisioner not working
   ```bash
   kubectl get pvc -n opencode
   kubectl get storageclass
   ```
   
2. **Image pull error**: Image not available in kind cluster
   ```bash
   kind load docker-image <image-name> --name opencode-dev
   ```

3. **Insufficient resources**: Node out of CPU/memory
   ```bash
   kubectl describe nodes
   ```

### Issue: Pod Crashes (CrashLoopBackOff)

**Symptoms:**
```bash
kubectl get pods -n opencode
# project-xxx   1/3   CrashLoopBackOff   5   10m
```

**Diagnosis:**
```bash
# Check which container is crashing
kubectl describe pod -n opencode <pod-name>

# View logs of crashed container
kubectl logs -n opencode <pod-name> -c <container-name>
kubectl logs -n opencode <pod-name> -c <container-name> --previous
```

**Common Causes:**
1. **Missing environment variables**: Check container env
2. **Volume mount issues**: Verify PVC is bound
3. **Image misconfiguration**: Check Dockerfile and entrypoint

### Issue: API Returns 500 on Project Creation

**Diagnosis:**
```bash
# Check controller logs
kubectl logs -n opencode -l app=opencode,component=controller --tail=50

# Look for Kubernetes API errors
grep -i "kubernetes" <logs>
grep -i "error" <logs>
```

**Common Causes:**
1. **RBAC permissions**: ServiceAccount lacks permissions
   ```bash
   kubectl auth can-i create pods --as=system:serviceaccount:opencode:opencode-controller -n opencode
   ```

2. **Namespace mismatch**: K8S_NAMESPACE env var doesn't match deployment namespace
   ```bash
   kubectl get configmap app-config -n opencode -o yaml | grep K8S_NAMESPACE
   ```

3. **API server unreachable**: In-cluster config not working
   ```bash
   kubectl exec -n opencode deployment/opencode-controller -- env | grep KUBERNETES
   ```

### Issue: PVC Not Deleted After Project Deletion

**Diagnosis:**
```bash
# Check if PVC still exists
kubectl get pvc -n opencode

# Check PVC finalizers (may prevent deletion)
kubectl get pvc <pvc-name> -n opencode -o yaml | grep finalizers
```

**Solution:**
```bash
# Manually delete PVC if stuck
kubectl delete pvc <pvc-name> -n opencode --force --grace-period=0
```

### Issue: Authentication Fails (401)

**Diagnosis:**
```bash
# Verify token is valid
echo $TOKEN | cut -d'.' -f2 | base64 -d | jq

# Check token expiry
date -d @$(echo $TOKEN | cut -d'.' -f2 | base64 -d | jq -r .exp)
```

**Solution:**
1. Get new token via OIDC login flow
2. Verify JWT_SECRET matches between auth service and middleware

---

## Cleanup

### Remove kind Cluster

```bash
# Delete entire cluster
kind delete cluster --name opencode-dev
```

### Remove Specific Resources (keep cluster)

```bash
# Delete all resources in namespace
kubectl delete namespace opencode

# Or delete specific project pods
kubectl delete pod -n opencode -l app=opencode-project
kubectl delete pvc -n opencode -l app=opencode-project
```

---

## Success Criteria

Phase 2.12 is complete when all of the following are verified:

- [x] ✅ kind cluster created successfully with 3 nodes
- [x] ✅ Default StorageClass available for PVCs
- [ ] ✅ All base manifests deployed without errors
- [ ] ✅ opencode-controller pod running and healthy
- [ ] ✅ RBAC permissions configured correctly
- [ ] ✅ POST /api/projects creates new project in database
- [ ] ✅ Project pod spawns with 3 containers (opencode, file-browser, session-proxy)
- [ ] ✅ PVC created with naming convention `workspace-{project-id}`
- [ ] ✅ All 3 containers in project pod start successfully (no crashes)
- [ ] ✅ GET /api/projects/:id returns correct pod status
- [ ] ✅ DELETE /api/projects/:id removes pod and PVC from cluster
- [ ] ✅ Soft delete works (project not returned in list after deletion)

---

## Next Steps

After Phase 2.12 completion:

1. **Phase 2.13**: Update documentation (AGENTS.md, README.md, TODO.md)
2. **Phase 3**: Task Management (Kanban board, state machine)
3. **Production Deployment**: Deploy to real Kubernetes cluster with proper ingress

---

**Document Maintained By:** Sisyphus (OpenCode AI Agent)  
**Last Tested:** 2026-01-17  
**Kubernetes Version:** v1.27+ (kind)  
**Tested With:** kind v0.20.0, kubectl v1.28.0
