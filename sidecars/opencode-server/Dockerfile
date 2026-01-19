# OpenCode Server Container
# This container runs the OpenCode AI agent runtime for project execution

FROM node:20-alpine

WORKDIR /app

# Install OpenCode CLI globally
# TODO: Replace with actual opencode npm package when available
# For now, this is a placeholder that will need the real opencode installation
RUN npm install -g @opencode/cli || echo "OpenCode CLI not yet published - placeholder image"

# Create workspace directory
RUN mkdir -p /workspace

# Health check script
RUN echo '#!/bin/sh' > /healthcheck.sh && \
    echo 'exit 0' >> /healthcheck.sh && \
    chmod +x /healthcheck.sh

# Expose OpenCode server port
EXPOSE 3000

# Environment variables
ENV WORKSPACE_DIR=/workspace
ENV PORT=3000
ENV NODE_ENV=production

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD /healthcheck.sh || exit 1

# Start OpenCode server
# TODO: Replace with actual opencode server start command
CMD ["sh", "-c", "echo 'OpenCode Server placeholder - waiting for actual OpenCode CLI' && sleep infinity"]
