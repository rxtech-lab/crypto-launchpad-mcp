# Multi-stage Dockerfile for launchpad-mcp streamable-http service
# Supports linux/amd64 and linux/arm64 architectures

# Build stage - compile Go binary and frontend assets
FROM --platform=$BUILDPLATFORM node:20-alpine AS frontend-builder

# Install curl and bash for bun installation
RUN apk add --no-cache curl bash

# Install bun using official installer (supports ARM64)
RUN curl -fsSL https://bun.sh/install | bash
ENV PATH="/root/.bun/bin:$PATH"

# Copy frontend source
WORKDIR /app/frontend/signing

# Copy package files (handle missing bun.lock gracefully)
COPY frontend/signing/package.json ./
COPY frontend/signing/bun.lock* ./

# Install dependencies
RUN bun install

# Copy frontend source code and build
COPY frontend/signing/ ./
RUN bun run build

# Go build stage - use bullseye for better CGO compatibility
# Use target platform for native compilation to support CGO dependencies
FROM golang:1.25-bookworm AS golang-builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

# Set working directory
WORKDIR /app

# Copy source code (excluding Dockerfile to avoid circular dependency)
COPY . .
RUN go mod download
RUN go generate ./...

# Copy built frontend assets from previous stage
COPY --from=frontend-builder /app/frontend/signing/dist/app.js ./internal/assets/signing_app.js
COPY --from=frontend-builder /app/frontend/signing/dist/app.css ./internal/assets/signing_app.css


# Build arguments for version info
ARG VERSION=docker
ARG COMMIT_HASH
ARG BUILD_TIME

# Build the streamable-http binary (CGO enabled for v8go dependency)
# Use native compilation instead of cross-compilation for CGO compatibility
RUN CGO_ENABLED=1 go build \
    -ldflags "-X main.Version=${VERSION} -X main.CommitHash=${COMMIT_HASH} -X main.BuildTime=${BUILD_TIME}" \
    -o launchpad-mcp-http \
    ./cmd/streamable-http/main.go

RUN ls -la

# Final runtime stage
FROM alpine:3.20

# Install ca-certificates for HTTPS requests and wget for health check
RUN apk --no-cache add ca-certificates tzdata wget

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy the binary from build stage
COPY --from=golang-builder /app/launchpad-mcp-http ./

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port (default 8080, configurable via PORT env var)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT:-8080}/health || exit 1

# Environment variables with defaults
ENV PORT=8080
ENV GIN_MODE=release

# Run the binary
CMD ["./launchpad-mcp-http"]