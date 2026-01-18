#!/bin/bash
set -e

CLUSTER_NAME="${KIND_CLUSTER_NAME:-opencode-dev}"

echo "Deploying to kind cluster '${CLUSTER_NAME}'..."

if ! kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
  echo "Cluster '${CLUSTER_NAME}' not found!"
  echo "Create it first with: kind create cluster --config k8s/kind-config.yaml --name ${CLUSTER_NAME}"
  exit 1
fi

kubectl config use-context "kind-${CLUSTER_NAME}"

# Load Docker images into kind cluster
echo "Loading Docker images into kind cluster..."
if docker images | grep -q "registry.legal-suite.com/opencode/app"; then
  echo "  Loading opencode/app..."
  kind load docker-image registry.legal-suite.com/opencode/app:latest --name ${CLUSTER_NAME}
else
  echo "  WARNING: opencode/app image not found locally. Build it first with: make docker-build-prod"
fi

if docker images | grep -q "registry.legal-suite.com/opencode/file-browser-sidecar"; then
  echo "  Loading file-browser-sidecar..."
  kind load docker-image registry.legal-suite.com/opencode/file-browser-sidecar:latest --name ${CLUSTER_NAME}
fi

if docker images | grep -q "registry.legal-suite.com/opencode/session-proxy-sidecar"; then
  echo "  Loading session-proxy-sidecar..."
  kind load docker-image registry.legal-suite.com/opencode/session-proxy-sidecar:latest --name ${CLUSTER_NAME}
fi

echo "Creating namespace..."
kubectl create namespace opencode --dry-run=client -o yaml | kubectl apply -f -

echo "Applying Kubernetes manifests..."
kubectl apply -k k8s/base/

echo "Waiting for pods to be ready..."
kubectl wait --for=condition=ready pod -l app=opencode --timeout=300s -n opencode || true

echo ""
echo "Deployment complete!"
echo "To view pods: kubectl get pods -n opencode"
echo "To view logs: kubectl logs -n opencode -l app=opencode -f"
echo "To port-forward: kubectl port-forward -n opencode svc/opencode-controller 8090:8090"
