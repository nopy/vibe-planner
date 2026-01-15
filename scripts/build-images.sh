#!/bin/bash
set -e

REGISTRY="${DOCKER_REGISTRY:-registry.legal-suite.com/opencode}"
VERSION="${1:-latest}"

echo "Building Docker images..."

echo "Building backend..."
docker build -t "${REGISTRY}/backend:${VERSION}" ./backend

echo "Building frontend..."
docker build -t "${REGISTRY}/frontend:${VERSION}" ./frontend

echo "Building file-browser sidecar..."
docker build -t "${REGISTRY}/file-browser-sidecar:${VERSION}" ./sidecars/file-browser

echo "Building session-proxy sidecar..."
docker build -t "${REGISTRY}/session-proxy-sidecar:${VERSION}" ./sidecars/session-proxy

echo "Build complete!"
echo "Images:"
echo "  - ${REGISTRY}/backend:${VERSION}"
echo "  - ${REGISTRY}/frontend:${VERSION}"
echo "  - ${REGISTRY}/file-browser-sidecar:${VERSION}"
echo "  - ${REGISTRY}/session-proxy-sidecar:${VERSION}"

echo ""
echo "To push images, run: docker push ${REGISTRY}/backend:${VERSION}"
