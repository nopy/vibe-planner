# Stage 1: Build frontend
FROM node:18-alpine AS frontend-builder

WORKDIR /app/frontend

COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci

COPY frontend/ .
RUN npm run build

# Stage 2: Build backend with embedded frontend
FROM golang:1.24-alpine AS backend-builder

WORKDIR /app/backend

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ .

COPY --from=frontend-builder /app/frontend/dist ./internal/static/dist

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/opencode-api ./cmd/api/main.go

# Stage 3: Final runtime image
FROM alpine:3.18

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=backend-builder /app/opencode-api .

ENV GIN_MODE=release
ENV ENVIRONMENT=production

EXPOSE 8090

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8090/healthz || exit 1

ENTRYPOINT ["/app/opencode-api"]
