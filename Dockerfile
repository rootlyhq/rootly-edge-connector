# Dockerfile
# Multi-stage build for Rootly Edge Connector

# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Install git for version information
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with version information
RUN export GIT_COMMIT=$(git rev-list -1 HEAD 2>/dev/null || echo "unknown") && \
    export VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev") && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo \
        -ldflags "-s -w -X main.version=$VERSION -X main.commit=$GIT_COMMIT -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        -o rootly-edge-connector cmd/rec/main.go

# Runtime stage
FROM alpine:3.19

# Install dependencies
# - bash: Required for script execution
# - git: For git-based configuration management
# - ca-certificates: For HTTPS connections
# - python3: For Python-based scripts
RUN apk update && \
    apk add --no-cache bash git ca-certificates python3 py3-pip && \
    update-ca-certificates

# Create non-root user
RUN addgroup -S rootly && \
    adduser -S rootly -G rootly

# Setup directories
RUN mkdir -p /opt/rootly-edge-connector/scripts && \
    mkdir -p /etc/rootly-edge-connector && \
    mkdir -p /var/log/rootly-edge-connector && \
    chown -R rootly:rootly /opt/rootly-edge-connector && \
    chown -R rootly:rootly /etc/rootly-edge-connector && \
    chown -R rootly:rootly /var/log/rootly-edge-connector

WORKDIR /opt/rootly-edge-connector

# Copy binary from builder
COPY --from=builder /build/rootly-edge-connector /opt/rootly-edge-connector/rootly-edge-connector
RUN chmod +x /opt/rootly-edge-connector/rootly-edge-connector

# Copy example configurations (can be overridden with volumes)
COPY config.example.yml /etc/rootly-edge-connector/config.yml
COPY actions.example.yml /etc/rootly-edge-connector/actions.yml

# Copy test scripts from builder to execution directory
COPY --from=builder /build/scripts /opt/rootly-edge-connector/scripts/
RUN find /opt/rootly-edge-connector/scripts -name "*.sh" -exec chmod +x {} \;

# Switch to non-root user
USER rootly

# Default configuration paths
ENV REC_CONFIG_PATH=/etc/rootly-edge-connector/config.yml
ENV REC_ACTIONS_PATH=/etc/rootly-edge-connector/actions.yml
ENV REC_LOG_OUTPUT=/var/log/rootly-edge-connector/connector.log

# Expose metrics port
EXPOSE 9090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD pgrep -f rootly-edge-connector || exit 1

# Entry point
ENTRYPOINT ["/opt/rootly-edge-connector/rootly-edge-connector"]
CMD ["-config", "/etc/rootly-edge-connector/config.yml", "-actions", "/etc/rootly-edge-connector/actions.yml"]
