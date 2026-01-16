#!/bin/bash
set -e

# Configuration
REGISTRY="${DOCKER_REGISTRY:-registry.legal-suite.com/opencode}"
MODE="${MODE:-prod}"
VERSION="${VERSION:-latest}"
PUSH="${PUSH:-false}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

log_success() {
    echo -e "${GREEN}✓${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

log_error() {
    echo -e "${RED}✗${NC} $1"
}

usage() {
    cat <<EOF
Usage: $0 [OPTIONS]

Build Docker images for OpenCode Project Manager.

OPTIONS:
    --mode MODE         Build mode: 'prod' (unified) or 'dev' (separate) [default: prod]
    --version VERSION   Image tag version [default: latest]
    --push              Push images to registry after building
    --registry URL      Docker registry URL [default: registry.legal-suite.com/opencode]
    -h, --help          Show this help message

EXAMPLES:
    # Build production image (unified backend + frontend)
    $0 --mode prod --version v1.0.0

    # Build development images (separate backend/frontend)
    $0 --mode dev --version dev

    # Build and push production image
    $0 --mode prod --push

    # Build all images with custom registry
    $0 --registry myregistry.com/myproject --push

MODES:
    prod    - Single unified image with embedded frontend (29MB)
              Image: \${REGISTRY}/app:\${VERSION}
    
    dev     - Separate backend, frontend, and sidecar images
              Images: backend, frontend, file-browser-sidecar, session-proxy-sidecar

EOF
    exit 0
}

build_image() {
    local name="$1"
    local context="$2"
    local dockerfile="${3:-Dockerfile}"
    local tag="${REGISTRY}/${name}:${VERSION}"
    
    log_info "Building ${name}..."
    if docker build -t "${tag}" -f "${dockerfile}" "${context}"; then
        log_success "Built ${tag}"
        echo "${tag}"  # Return tag for push operations
    else
        log_error "Failed to build ${name}"
        exit 1
    fi
}

push_image() {
    local tag="$1"
    log_info "Pushing ${tag}..."
    if docker push "${tag}"; then
        log_success "Pushed ${tag}"
    else
        log_error "Failed to push ${tag}"
        exit 1
    fi
}

build_production() {
    log_info "Building PRODUCTION images (unified mode)"
    echo ""
    
    local images=()
    
    # Unified app image (backend + frontend)
    images+=("$(build_image "app" "." "Dockerfile")")
    
    # Sidecar images
    images+=("$(build_image "file-browser-sidecar" "./sidecars/file-browser" "./sidecars/file-browser/Dockerfile")")
    images+=("$(build_image "session-proxy-sidecar" "./sidecars/session-proxy" "./sidecars/session-proxy/Dockerfile")")
    
    echo ""
    log_success "Production build complete!"
    echo ""
    echo "Images built:"
    for img in "${images[@]}"; do
        echo "  - ${img}"
    done
    
    # Push if requested
    if [ "${PUSH}" = "true" ]; then
        echo ""
        log_info "Pushing images to registry..."
        for img in "${images[@]}"; do
            push_image "${img}"
        done
        log_success "All images pushed successfully!"
    fi
}

build_development() {
    log_info "Building DEVELOPMENT images (separate mode)"
    echo ""
    
    local images=()
    
    # Separate backend and frontend
    images+=("$(build_image "backend" "./backend" "./backend/Dockerfile")")
    images+=("$(build_image "frontend" "./frontend" "./frontend/Dockerfile")")
    
    # Sidecar images
    images+=("$(build_image "file-browser-sidecar" "./sidecars/file-browser" "./sidecars/file-browser/Dockerfile")")
    images+=("$(build_image "session-proxy-sidecar" "./sidecars/session-proxy" "./sidecars/session-proxy/Dockerfile")")
    
    echo ""
    log_success "Development build complete!"
    echo ""
    echo "Images built:"
    for img in "${images[@]}"; do
        echo "  - ${img}"
    done
    
    # Push if requested
    if [ "${PUSH}" = "true" ]; then
        echo ""
        log_info "Pushing images to registry..."
        for img in "${images[@]}"; do
            push_image "${img}"
        done
        log_success "All images pushed successfully!"
    fi
}

# Parse command-line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --mode)
            MODE="$2"
            shift 2
            ;;
        --version)
            VERSION="$2"
            shift 2
            ;;
        --push)
            PUSH="true"
            shift
            ;;
        --registry)
            REGISTRY="$2"
            shift 2
            ;;
        -h|--help)
            usage
            ;;
        *)
            log_error "Unknown option: $1"
            echo ""
            usage
            ;;
    esac
done

# Validate mode
if [ "${MODE}" != "prod" ] && [ "${MODE}" != "dev" ]; then
    log_error "Invalid mode: ${MODE}. Must be 'prod' or 'dev'"
    exit 1
fi

# Display configuration
echo "════════════════════════════════════════════════════════════"
log_info "OpenCode Project Manager - Docker Image Builder"
echo "════════════════════════════════════════════════════════════"
echo "  Mode:      ${MODE}"
echo "  Version:   ${VERSION}"
echo "  Registry:  ${REGISTRY}"
echo "  Push:      ${PUSH}"
echo "════════════════════════════════════════════════════════════"
echo ""

# Execute build based on mode
if [ "${MODE}" = "prod" ]; then
    build_production
else
    build_development
fi

echo ""
if [ "${PUSH}" != "true" ]; then
    log_info "To push images, run:"
    echo "  ${0} --mode ${MODE} --version ${VERSION} --push"
fi
echo ""
log_success "Done!"
