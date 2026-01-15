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
echo "To port-forward: kubectl port-forward -n opencode svc/opencode-controller 8080:80"
