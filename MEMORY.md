# Project Memory - Lessons Learned

This document captures critical lessons learned during development to prevent future issues.

---

## Kubernetes Secrets & Encryption Keys

**Date:** 2026-01-19  
**Issue:** `make kind-deploy` failed with "Error: couldn't find key ENCRYPTION_KEY in Secret opencode/app-secrets"

### Root Causes
1. **Missing Secret Key**: The `app-secrets` Secret was missing the `ENCRYPTION_KEY` entry
2. **Environment Variable Mismatch**: Deployment manifest used `ENCRYPTION_KEY` but Go code expected `CONFIG_ENCRYPTION_KEY`
3. **Incorrect Key Generation**: Initially used `openssl rand -base64 32 | base64` which double-encoded the key

### Critical Lessons

#### 1. Kubernetes Secret Base64 Encoding
Kubernetes Secrets require **double base64 encoding**:
- The actual encryption key value: base64-encoded
- The Secret YAML `data` field: base64-encoded again

```yaml
# Secret YAML (k8s/base/secrets.yaml)
data:
  ENCRYPTION_KEY: SXN5VWJOMm11cVVIZjl2YmtueHZ6cW9oUUhmckdiN3lybjNmVzlJSXR3dz0=
  # ↑ This is base64(base64(32-random-bytes))
```

When the pod reads the env var, Kubernetes automatically decodes ONE layer, so the app receives the base64-encoded key string.

#### 2. Correct Encryption Key Generation

**✅ CORRECT:**
```bash
# Generate 32 random bytes, encode to base64 (44 chars)
KEY=$(openssl rand 32 | base64 -w 0)
echo $KEY  # Example: IsyUbN2muqUHf9vbknxvzqohQHfrGb7yrn3fW9IItww=

# Encode for Kubernetes Secret
echo -n "$KEY" | base64 -w 0  # Result: SXN5VWJOMm11cVVIZjl2YmtueHZ6cW9oUUhmckdiN3lybjNmVzlJSXR3dz0=
```

**❌ WRONG:**
```bash
# This produces 44 bytes when decoded, not 32!
openssl rand -base64 32  # Generates 32 base64 CHARACTERS, not 32 BYTES

# This double-encodes incorrectly
openssl rand -base64 32 | base64  # Wrong approach
```

#### 3. Verification Steps

After generating the key, verify it decodes to exactly 32 bytes:
```bash
# Check the Secret value in cluster
kubectl get secret app-secrets -n opencode -o jsonpath='{.data.ENCRYPTION_KEY}' | base64 -d

# Verify it decodes to 32 bytes
kubectl get secret app-secrets -n opencode -o jsonpath='{.data.ENCRYPTION_KEY}' | base64 -d | base64 -d | wc -c
# Output should be: 32
```

#### 4. Environment Variable Naming Consistency

**Always ensure deployment manifests match application code:**

```yaml
# k8s/base/deployment.yaml
env:
  - name: CONFIG_ENCRYPTION_KEY  # Must match Go code
    valueFrom:
      secretKeyRef:
        name: app-secrets
        key: ENCRYPTION_KEY  # Secret key name can differ
```

```go
// backend/internal/config/config.go
EncryptionKey: getEnv("CONFIG_ENCRYPTION_KEY", "")  // Must match deployment
```

**Mismatch causes:**
- Silent failures (env var empty, falls back to default)
- Cryptic errors like "encryption key must be base64-encoded 32 bytes"
- CrashLoopBackOff without clear indication

#### 5. ConfigService Validation

The `ConfigService` in `backend/internal/service/config_service.go` validates:
```go
key, err := base64.StdEncoding.DecodeString(encryptionKey)
if err != nil || len(key) != 32 {
    return nil, errors.New("encryption key must be base64-encoded 32 bytes")
}
```

**This expects:**
- Input: base64-encoded string (e.g., `IsyUbN2muqUHf9vbknxvzqohQHfrGb7yrn3fW9IItww=`)
- After decode: exactly 32 bytes of raw data

### Solution Applied

**Files Modified:**
1. `k8s/base/secrets.yaml` - Added `ENCRYPTION_KEY` with correct base64 encoding
2. `k8s/base/deployment.yaml` - Changed env var name from `ENCRYPTION_KEY` to `CONFIG_ENCRYPTION_KEY`
3. `.env.example` - Added `CONFIG_ENCRYPTION_KEY` with generation instructions

**Result:** Deployment successful, pod running healthy

### Prevention Checklist

When adding new secrets to Kubernetes:
- [ ] Generate secret value correctly (understand encoding requirements)
- [ ] Add to `k8s/base/secrets.yaml` with proper base64 encoding
- [ ] Add environment variable to deployment manifest with **correct name**
- [ ] Verify env var name matches application code (grep codebase)
- [ ] Update `.env.example` with clear generation instructions
- [ ] Document validation requirements in code comments
- [ ] Test in kind cluster before production deployment

### Reference Commands

```bash
# Generate 32-byte encryption key
openssl rand 32 | base64 -w 0

# Encode for Kubernetes Secret
echo -n "YOUR_KEY_HERE" | base64 -w 0

# Verify Secret in cluster
kubectl get secret app-secrets -n opencode -o yaml

# Check pod environment variables
kubectl exec -n opencode POD_NAME -- env | grep ENCRYPTION

# Restart deployment after Secret update
kubectl rollout restart deployment opencode-controller -n opencode

# Watch pod status
kubectl get pods -n opencode -w
```

---

## Template for Future Entries

```markdown
## [Category] - [Brief Issue Description]

**Date:** YYYY-MM-DD
**Issue:** [One-line problem statement]

### Root Causes
1. [Root cause 1]
2. [Root cause 2]

### Critical Lessons
[Key takeaways]

### Solution Applied
[What was changed]

### Prevention Checklist
- [ ] Item 1
- [ ] Item 2
```

---

**Last Updated:** 2026-01-19  
**Maintainer:** Development Team
