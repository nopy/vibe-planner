# OpenCode Server Sidecar - Production Dockerfile
# Multi-stage build with Bun runtime for TypeScript execution
# Final image size target: <100MB

# ====================================
# Stage 1: Builder (OpenCode + dependencies)
# ====================================
FROM oven/bun:1-alpine AS builder

WORKDIR /build

# Install system dependencies for building
RUN apk add --no-cache \
    git \
    ca-certificates

# Clone OpenCode repository
# Using a specific commit for reproducible builds (update as needed)
RUN git clone --depth 1 https://github.com/anomalyco/opencode.git /build/opencode

# Install OpenCode dependencies
WORKDIR /build/opencode
RUN bun install --production --ignore-scripts

# Copy server implementation
WORKDIR /build/server
COPY server.ts .
COPY package.json .

# Install server dependencies
RUN bun install --production

# ====================================
# Stage 2: Runtime
# ====================================
FROM oven/bun:1-alpine

# Install runtime dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    wget \
    && rm -rf /var/cache/apk/*

# Create non-root user with workspace permissions
RUN addgroup -S opencode && \
    adduser -S opencode -G opencode && \
    mkdir -p /workspace && \
    chown -R opencode:opencode /workspace

# Set working directory
WORKDIR /app

# Copy OpenCode from builder
COPY --from=builder --chown=opencode:opencode /build/opencode ./opencode

# Copy server implementation and dependencies
COPY --from=builder --chown=opencode:opencode /build/server/server.ts ./server.ts
COPY --from=builder --chown=opencode:opencode /build/server/package.json ./package.json
COPY --from=builder --chown=opencode:opencode /build/server/node_modules ./node_modules

# Switch to non-root user
USER opencode

# Environment variables
ENV WORKSPACE_DIR=/workspace \
    PORT=3003 \
    LOG_LEVEL=info \
    NODE_ENV=production \
    SESSION_TIMEOUT=3600 \
    MAX_CONCURRENT_SESSIONS=5

# Expose server port
EXPOSE 3003

# Health check using wget to test HTTP endpoint
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3003/healthz || exit 1

# Start OpenCode server
CMD ["bun", "run", "server.ts"]
