# OpenCode Server Implementation TODO

**Status:** Placeholder container created (2026-01-19)  
**Current Image:** `registry.legal-suite.com/opencode/opencode-server-sidecar:latest` (135MB)  
**Location:** `sidecars/opencode-server/`

---

## 🎯 Overview

The opencode-server container is currently a **placeholder** that runs a Node.js 20 Alpine base with a mock health check. This document outlines the steps needed to replace it with a fully functional OpenCode AI agent runtime.

---

## 📋 Next Steps (Priority Order)

### Phase 1: Research & Planning (HIGH PRIORITY)

- [x] **1.1 Investigate OpenCode CLI/Server** ✅ COMPLETE (2026-01-19)
  - [x] Research official OpenCode documentation for deployment
  - [x] Determine if OpenCode has an npm package (`@opencode/cli`) or alternative installation method
  - [x] Identify actual port requirements (currently assumes :3000)
  - [x] Document required environment variables
  - [x] Check if OpenCode requires additional dependencies (Python, git, etc.)
  
  **FINDINGS (Phase 1.1):**
  
  #### Installation Method
  - **Repository:** https://github.com/anomalyco/opencode
  - **Installation:** Git clone + npm install (no published npm package as of 2026-01-19)
  - **Runtime:** Bun (not Node.js) - OpenCode uses Bun runtime for TypeScript execution
  - **System Dependencies:** git, ca-certificates
  - **Build Tools:** Not required at runtime (TypeScript executed directly via Bun)
  
  #### Port Configuration
  - **CRITICAL PORT CONFLICT FOUND:** 
    - Current codebase has inconsistent port usage: :3000 vs :3003
    - Pod template specifies port 3000 (`backend/internal/service/pod_template.go`)
    - Backend session service uses port 3003 (`backend/internal/service/session_service.go:216`)
    - Backend task output stream uses port 3003 (`backend/internal/api/tasks.go:666`)
    - **RECOMMENDATION:** Standardize to port **3003** (aligns with IMPLEMENTATION_PLAN.md)
    - **ACTION REQUIRED:** Update pod_template.go and README to use 3003
  
  #### Session Management & Concurrency
  - **Execution Model:** Per-session serialization with in-memory AbortController state
  - **Concurrent Sessions:** Unlimited sessions supported, but each session runs single loop
  - **Session Isolation:** Logical isolation via project/sandbox directories (no OS-level containers)
  - **State Persistence:** Sessions/messages persisted to disk (JSON files under storage dir)
  - **Resource Limits:** No built-in CPU/RAM quotas - must enforce at Kubernetes pod level
  - **Cleanup:** Explicit APIs for session removal and cancellation (Session.remove, SessionPrompt.cancel)
  
  #### AI Model Configuration
  - **Credentials Storage:** Backend ConfigService encrypts API keys with AES-256-GCM
  - **ENCRYPTION_KEY Required:** 32-byte base64-encoded secret (Kubernetes Secret)
  - **API Key Injection:** Currently NOT passed to OpenCode start request
  - **ACTION REQUIRED:** Extend session start payload to include decrypted API key + model config
  - **Supported Providers:** OpenAI, Anthropic, local models (configurable per project)
  
  #### Environment Variables (OpenCode)
  - `WORKSPACE_DIR` - Project workspace path (default: /workspace)
  - `OPENCODE_URL` - URL to OpenCode server for session-proxy (default: http://localhost:3000)
  - Model-specific API keys passed via session start request (not environment)
  
  #### Critical Action Items from Research
  1. ✅ **Port Standardization (HIGH):** Changed opencode-server port 3000 → 3003 in pod_template.go (2026-01-19)
     - Updated ContainerPort to 3003
     - Updated LivenessProbe and ReadinessProbe ports to 3003
     - Updated session-proxy OPENCODE_URL to http://localhost:3003
  2. ✅ **API Key Injection (HIGH):** Modified session_service.go to include decrypted API key in start request (2026-01-19)
     - Added ConfigServiceInterface dependency to SessionService
     - Extended callOpenCodeStart() to fetch active config and decrypt API key
     - Request body now includes model_config with: provider, model, api_key, temperature, max_tokens, enabled_tools
     - Added support for optional fields: model_version, api_endpoint, system_prompt
  3. **Encryption Key Secret (HIGH):** Create Kubernetes Secret with ENCRYPTION_KEY for ConfigService
  4. **Session Concurrency (MEDIUM):** Document that one OpenCode instance can handle multiple sessions
  5. **Resource Limits (MEDIUM):** Profile memory usage and set appropriate pod limits

- [ ] **1.2 Define API Contract**
  - [ ] Document expected REST endpoints (e.g., `/health`, `/ready`, `/tasks/execute`)
  - [ ] Document WebSocket protocol for real-time task execution
  - [ ] Define request/response schemas for task submission
  - [ ] Define streaming output format for task progress
  - [ ] Specify authentication mechanism (JWT from main backend?)

- [ ] **1.3 Determine Resource Requirements**
  - [ ] Profile OpenCode memory usage during typical tasks
  - [ ] Determine appropriate CPU/memory limits for pod spec
  - [ ] Decide on persistent storage needs beyond workspace PVC
  - [ ] Identify if GPU access is needed for certain AI models

### Phase 2: Implementation (HIGH PRIORITY)

- [ ] **2.1 Replace Dockerfile**
  - [ ] Update `sidecars/opencode-server/Dockerfile` with actual OpenCode installation
  - [ ] Install required system dependencies (git, build-essential, etc.)
  - [ ] Configure proper user/permissions for workspace access
  - [ ] Optimize image size (multi-stage build if needed)
  - [ ] Add proper HEALTHCHECK with HTTP endpoint validation

- [ ] **2.2 Implement Health Check Endpoints**
  - [ ] Create `/health` endpoint (liveness probe)
  - [ ] Create `/ready` endpoint (readiness probe)
  - [ ] Return appropriate HTTP status codes (200, 503, etc.)
  - [ ] Include basic system checks (workspace writable, dependencies available)

- [ ] **2.3 Implement Task Execution API**
  - [ ] Create REST endpoint for task submission: `POST /tasks/execute`
  - [ ] Implement WebSocket endpoint for streaming output: `WS /tasks/{taskId}/stream`
  - [ ] Handle task state management (pending, running, completed, failed)
  - [ ] Implement proper error handling and recovery
  - [ ] Add logging with structured output (JSON logs)

- [ ] **2.4 Session Proxy Integration**
  - [ ] Define communication protocol with session-proxy sidecar (:3002)
  - [ ] Implement bidirectional message passing
  - [ ] Handle session lifecycle (create, attach, detach, destroy)
  - [ ] Support multiple concurrent task executions

### Phase 3: Backend Integration (MEDIUM PRIORITY)

- [ ] **3.1 Update Backend Proxy Layer**
  - [ ] Review `backend/internal/api/tasks.go` - `ExecuteTask` handler (currently stub)
  - [ ] Implement HTTP proxy to opencode-server `/tasks/execute` endpoint
  - [ ] Add WebSocket proxy for streaming task output to frontend
  - [ ] Handle pod IP resolution (already implemented in `files.go`)
  - [ ] Add proper error mapping (502 for sidecar errors, etc.)

- [ ] **3.2 Update Task Service**
  - [ ] Extend `backend/internal/service/task_service.go` with execution logic
  - [ ] Implement task state transitions during execution
  - [ ] Add timeout handling for long-running tasks
  - [ ] Store execution logs/output in database (new table?)
  - [ ] Implement task cancellation support

- [ ] **3.3 Database Schema Updates**
  - [ ] Create migration for task execution logs table
  - [ ] Add columns: `execution_started_at`, `execution_completed_at`, `execution_output`
  - [ ] Consider separate `task_executions` table for retry history
  - [ ] Add indexes for querying by task_id and status

### Phase 4: Frontend Integration (MEDIUM PRIORITY)

- [ ] **4.1 Task Execution UI**
  - [ ] Add "Execute" button to TaskCard component
  - [ ] Create TaskExecutionPanel component for streaming output
  - [ ] Implement WebSocket client for real-time output display
  - [ ] Add terminal-like output view with ANSI color support
  - [ ] Show execution progress indicators (spinner, progress bar)

- [ ] **4.2 Task Output Viewer**
  - [ ] Create modal/panel for viewing task execution history
  - [ ] Display logs with syntax highlighting (if applicable)
  - [ ] Add download/copy buttons for output
  - [ ] Show execution duration and resource usage metrics
  - [ ] Support filtering logs by severity level

### Phase 5: Testing & Validation (HIGH PRIORITY)

- [ ] **5.1 Unit Tests**
  - [ ] Test health check endpoints return correct status codes
  - [ ] Test task submission API with valid/invalid payloads
  - [ ] Test WebSocket connection lifecycle (connect, send, receive, close)
  - [ ] Test error handling for missing workspace, permission errors
  - [ ] Mock OpenCode CLI responses for deterministic testing

- [ ] **5.2 Integration Tests**
  - [ ] Test full task execution flow (backend → opencode-server → response)
  - [ ] Test concurrent task executions in same pod
  - [ ] Test task cancellation mid-execution
  - [ ] Test pod restart recovery (in-flight tasks)
  - [ ] Test session-proxy communication

- [ ] **5.3 E2E Tests**
  - [ ] Create test project in kind cluster
  - [ ] Submit real coding task via frontend
  - [ ] Verify output streams to browser
  - [ ] Verify workspace files are modified correctly
  - [ ] Test file-browser sidecar sees changes in real-time

### Phase 6: Security & Production Readiness (CRITICAL)

- [ ] **6.1 Security Hardening**
  - [ ] Run container as non-root user (already in base Alpine?)
  - [ ] Implement workspace sandboxing (prevent escaping /workspace)
  - [ ] Add resource limits to prevent DoS (CPU, memory, disk I/O)
  - [ ] Validate task inputs to prevent code injection
  - [ ] Add authentication for opencode-server API (shared secret with backend?)

- [ ] **6.2 Monitoring & Observability**
  - [ ] Add Prometheus metrics endpoint (`/metrics`)
  - [ ] Expose task execution metrics (count, duration, success rate)
  - [ ] Add structured logging with correlation IDs
  - [ ] Implement distributed tracing (if using OpenTelemetry)
  - [ ] Add alerts for high failure rates

- [ ] **6.3 Production Configuration**
  - [ ] Update pod resource limits in `pod_template.go` based on profiling
  - [ ] Configure proper restart policies (handle transient failures)
  - [ ] Add pod disruption budgets for controlled rollouts
  - [ ] Document deployment procedures in PHASE5.md
  - [ ] Create rollback plan for failed deployments

### Phase 7: Documentation (MEDIUM PRIORITY)

- [ ] **7.1 Developer Documentation**
  - [ ] Update `sidecars/opencode-server/README.md` with actual implementation details
  - [ ] Document API endpoints with request/response examples
  - [ ] Add architecture diagram showing opencode-server in context
  - [ ] Document environment variables and configuration options
  - [ ] Create troubleshooting guide for common issues

- [ ] **7.2 User Documentation**
  - [ ] Update main README.md with task execution features
  - [ ] Create user guide for submitting and monitoring tasks
  - [ ] Document supported task types and limitations
  - [ ] Add examples of common coding tasks
  - [ ] Create FAQ for task execution errors

---

## 🔗 Related Files

| File | Purpose | Current Status |
|------|---------|----------------|
| `sidecars/opencode-server/Dockerfile` | Container build definition | ✅ Placeholder |
| `sidecars/opencode-server/README.md` | Documentation | ✅ Basic outline |
| `backend/internal/service/pod_template.go` | Pod spec with opencode-server container | ✅ Complete |
| `backend/internal/api/tasks.go` | Task execution API handler (stub) | ⚠️ Needs implementation |
| `backend/internal/service/task_service.go` | Task business logic | ⚠️ Needs execution logic |
| `backend/internal/config/config.go` | Container image configuration | ✅ Complete |
| `Makefile` | Build targets | ✅ Complete |
| `scripts/build-images.sh` | Multi-image build script | ✅ Complete |

---

## 🚧 Current Blockers

1. **OpenCode CLI Availability:** Need to determine official installation method
2. **API Specification:** No official OpenCode server API documentation found
3. **Resource Profiling:** Cannot determine optimal pod limits without running actual workload

---

## 💡 Implementation Notes

### Dockerfile Structure (Proposed)

```dockerfile
# Multi-stage build for smaller image
FROM node:20-alpine AS builder

WORKDIR /app

# Install OpenCode CLI (method TBD based on research)
RUN npm install -g @opencode/cli || \
    apk add --no-cache git python3 build-base && \
    git clone https://github.com/anomalyco/opencode.git && \
    cd opencode && npm install && npm run build

# Runtime stage
FROM node:20-alpine

# Install runtime dependencies only
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy built artifacts from builder
COPY --from=builder /usr/local/lib/node_modules/@opencode /usr/local/lib/node_modules/@opencode
COPY --from=builder /usr/local/bin/opencode /usr/local/bin/opencode

# Create non-root user
RUN addgroup -S opencode && adduser -S opencode -G opencode
USER opencode

EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:3000/health || exit 1

CMD ["opencode", "server", "--port=3000", "--workspace=/workspace"]
```

### API Endpoints (Proposed)

```
GET  /health                          - Liveness probe (200 OK)
GET  /ready                           - Readiness probe (200 OK / 503 Service Unavailable)
POST /tasks/execute                   - Submit task for execution
  Request:  { "task_id": "uuid", "code": "...", "context": "..." }
  Response: { "execution_id": "uuid", "status": "pending" }

GET  /tasks/{execution_id}/status     - Get execution status
  Response: { "status": "running|completed|failed", "progress": 45 }

WS   /tasks/{execution_id}/stream     - WebSocket stream for real-time output
  Messages: { "type": "stdout|stderr|status", "data": "...", "timestamp": "..." }

POST /tasks/{execution_id}/cancel     - Cancel running task
DELETE /tasks/{execution_id}          - Cleanup execution artifacts
```

### Environment Variables (Proposed)

```bash
WORKSPACE_DIR=/workspace              # Shared workspace path
PORT=3000                             # Server port
PROJECT_ID=<uuid>                     # Associated project UUID
LOG_LEVEL=info                        # Logging verbosity
OPENCODE_API_KEY=<optional>           # If OpenCode requires API key for AI models
MAX_CONCURRENT_TASKS=5                # Limit concurrent executions
TASK_TIMEOUT=3600                     # Max execution time in seconds
```

---

## 📞 Questions to Resolve

1. **OpenCode Installation:** What is the official method to install/run OpenCode in a container?
2. **Port Requirements:** Does OpenCode expose an HTTP server on a configurable port?
3. **Session Management:** How does OpenCode handle multiple concurrent tasks?
4. **AI Model Configuration:** Where do AI model credentials (OpenAI API key, etc.) come from?
5. **Workspace Isolation:** Does OpenCode provide built-in sandboxing or do we need Docker-in-Docker?

---

## 🎯 Success Criteria

This implementation will be considered **complete** when:

- ✅ OpenCode server starts successfully in the container
- ✅ Health/readiness probes return correct status
- ✅ Tasks can be submitted via REST API
- ✅ Real-time output streams via WebSocket to frontend
- ✅ File changes are visible in file-browser sidecar
- ✅ Multiple tasks can run concurrently in same pod
- ✅ All unit/integration/E2E tests pass
- ✅ Pod resource limits are optimized based on profiling
- ✅ Documentation is complete and accurate

---

**Last Updated:** 2026-01-19  
**Status:** Ready for Phase 1 (Research & Planning)
