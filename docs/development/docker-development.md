# Docker Development Guide

This guide covers using Docker for local development of the Rootly Edge Connector.

## Quick Start

### Development Mode

```bash
# Build dev image
docker build -f Dockerfile.dev -t rootly-edge-connector:dev .

# Run container interactively (foreground)
docker run --rm -it \
  -e REC_API_KEY=${REC_API_KEY} \
  -v $(pwd)/config.example.dev.yml:/etc/rootly-edge-connector/config.yml \
  -v $(pwd)/actions.example.dev.yml:/etc/rootly-edge-connector/actions.yml \
  -v $(pwd)/scripts:/opt/rootly-edge-connector/scripts \
  rootly-edge-connector:dev

# OR run in background (daemon)
docker run -d \
  --name rec-dev \
  -e REC_API_KEY=${REC_API_KEY} \
  -v $(pwd)/config.example.dev.yml:/etc/rootly-edge-connector/config.yml \
  -v $(pwd)/actions.example.dev.yml:/etc/rootly-edge-connector/actions.yml \
  -v $(pwd)/scripts:/opt/rootly-edge-connector/scripts \
  rootly-edge-connector:dev

# To override API URL (e.g., connect to production instead of localhost):
# docker run --rm -it \
#   -e REC_API_KEY=${REC_API_KEY} \
#   -e REC_API_URL=https://rec.rootly.com \
#   -v $(pwd)/config.example.dev.yml:/etc/rootly-edge-connector/config.yml \
#   ...

# If you need metrics, enable them in config and expose the port:
# docker run --rm -it -p 9090:9090 -e REC_API_KEY=${REC_API_KEY} ...

# View logs (only works if container is running in background)
docker logs -f rec-dev

# Note: To see colors in Docker logs, use one of these methods:
# 1. Run interactively with -it (recommended for dev)
# 2. Use: docker logs -f rec-dev 2>&1 | cat
# 3. Use a tool like lazydocker or stern

# Stop and remove
docker stop rec-dev
docker rm rec-dev
```

### Production Mode

```bash
# Build production image
docker build -t rootly-edge-connector:latest .

# Run container
docker run --rm -it \
  -p 9090:9090 \
  -e REC_API_KEY=your-api-key \
  -v $(pwd)/config.yml:/etc/rootly-edge-connector/config.yml:ro \
  -v $(pwd)/actions.yml:/etc/rootly-edge-connector/actions.yml:ro \
  -v $(pwd)/scripts:/opt/rootly-edge-connector/scripts:ro \
  rootly-edge-connector:latest
```

## Docker Files Overview

### `Dockerfile` (Production)

Multi-stage build optimized for production:
- **Build stage**: Compiles Go binary with version information
- **Runtime stage**: Minimal Alpine image with only runtime dependencies
- **Size**: ~164MB final image (includes Python3 + pip for script support)
- **User**: Runs as non-root `rootly` user
- **Configs**: Uses `config.example.yml` and `actions.example.yml`
- **Scripts**: Includes test scripts from /scripts directory

### `Dockerfile.dev` (Development)

Multi-stage build optimized for development with live reloading:
- **Base stage**: Dependencies and Go module caching
- **Development stage**: Full `golang:1.24-alpine` with toolchain
- **Includes**: make, git, curl, debugging tools
- **Runtime**: Uses `go run` for live reloading and easier debugging
- **Configs**: Uses `config.example.dev.yml` and `actions.example.dev.yml`
- **Logging**: Colored, trace-level output by default
- **Size**: ~759MB (includes Go toolchain, dependencies, and source code for development)


## Development Workflows

### 1. Basic Development

Run with example configurations and test scripts:

```bash
# Build and run
docker build -f Dockerfile.dev -t rootly-edge-connector:dev .
docker run --rm -it \
  -v $(pwd)/config.example.dev.yml:/etc/rootly-edge-connector/config.yml \
  -v $(pwd)/actions.example.dev.yml:/etc/rootly-edge-connector/actions.yml \
  -v $(pwd)/scripts:/opt/rootly-edge-connector/scripts \
  rootly-edge-connector:dev

# The connector will:
# - Use config.example.dev.yml (host.docker.internal:3000 mock server)
# - Execute test scripts from ./scripts/
# - Output colored trace logs

# Note: The config uses host.docker.internal to reach your host machine
# This allows the container to connect to services running on your laptop
```

### 2. With Source Code Mounting

Edit code on your host machine and rebuild inside container:

```bash
# Run with source code mounted
docker run --rm -it \
  -v $(pwd):/app \
  -v $(pwd)/config.example.dev.yml:/etc/rootly-edge-connector/config.yml \
  -v $(pwd)/actions.example.dev.yml:/etc/rootly-edge-connector/actions.yml \
  -v $(pwd)/scripts:/opt/rootly-edge-connector/scripts \
  rootly-edge-connector:dev

# Or exec into running container to rebuild
docker exec -it rec-dev sh

# Inside container:
go build -o /tmp/rec cmd/rec/main.go
/tmp/rec -config /etc/rootly-edge-connector/config.yml -actions /etc/rootly-edge-connector/actions.yml
```

### 3. With Mock API Server

Test without connecting to real Rootly API:

**Option A: Rootly Rails server on host (recommended)**
```bash
# Start Rootly Rails server on your host machine (in one terminal)
# Run this from your rootly/rootly Rails project directory
WEB_CONCURRENCY=1 BASE_URL=http://localhost bundle exec rails s -p 3000

# Run connector in Docker (in another terminal)
docker run --rm -it \
  -e REC_API_KEY=${REC_API_KEY} \
  -v $(pwd)/config.example.dev.yml:/etc/rootly-edge-connector/config.yml \
  -v $(pwd)/actions.example.dev.yml:/etc/rootly-edge-connector/actions.yml \
  -v $(pwd)/scripts:/opt/rootly-edge-connector/scripts \
  rootly-edge-connector:dev

# The connector uses host.docker.internal:3000 to reach your Rails server
```

**Option B: Mock server in Docker**
```bash
# Start httpbin on port 3000
docker run --rm -p 3000:80 kennethreitz/httpbin

# Run connector with --network host (Linux only)
docker run --rm -it \
  --network host \
  -v $(pwd)/config.example.dev.yml:/etc/rootly-edge-connector/config.yml \
  -v $(pwd)/actions.example.dev.yml:/etc/rootly-edge-connector/actions.yml \
  -v $(pwd)/scripts:/opt/rootly-edge-connector/scripts \
  rootly-edge-connector:dev
```

### 4. Interactive Development

Run commands inside the container:

```bash
# Start container in background
docker run -d --name rec-dev \
  -v $(pwd):/app \
  -v $(pwd)/config.example.dev.yml:/etc/rootly-edge-connector/config.yml \
  -v $(pwd)/actions.example.dev.yml:/etc/rootly-edge-connector/actions.yml \
  rootly-edge-connector:dev

# Shell access
docker exec -it rec-dev sh

# Run tests
docker exec -it rec-dev make test

# Run linter
docker exec -it rec-dev make lint

# Format code
docker exec -it rec-dev make fmt

# Build binary
docker exec -it rec-dev make build

# Stop and remove
docker stop rec-dev && docker rm rec-dev
```

### 5. Debugging with Delve

Add debugger support for step-through debugging:

**Update `Dockerfile.dev` CMD:**
```dockerfile
# Install delve
RUN go install github.com/go-delve/delve/cmd/dlv@latest

# Change CMD to use delve
CMD ["dlv", "debug", "cmd/rec/main.go", "--headless", "--listen=:2345", "--api-version=2", "--accept-multiclient", "--", "-config", "/etc/rootly-edge-connector/config.yml", "-actions", "/etc/rootly-edge-connector/actions.yml"]
```

**Run with debugger port:**
```bash
docker run --rm -it \
  -p 2345:2345 \
  -v $(pwd):/app \
  rootly-edge-connector:dev
```

**Connect from VS Code:**
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Connect to Docker",
      "type": "go",
      "request": "attach",
      "mode": "remote",
      "remotePath": "/app",
      "port": 2345,
      "host": "localhost"
    }
  ]
}
```

## Environment Variables

### Passing Environment Variables

You can override configuration using `-e` flags:

```bash
# Set API key from your shell environment
export REC_API_KEY="rec_your_actual_key_here"

docker run --rm -it \
  -e REC_API_KEY=${REC_API_KEY} \
  -v $(pwd)/config.example.dev.yml:/etc/rootly-edge-connector/config.yml \
  rootly-edge-connector:dev

# Or pass directly
docker run --rm -it \
  -e REC_API_KEY=rec_your_actual_key \
  -e REC_API_URL=https://rec.rootly.com \
  -e REC_LOG_LEVEL=trace \
  -v $(pwd)/config.example.dev.yml:/etc/rootly-edge-connector/config.yml \
  rootly-edge-connector:dev
```

### Common Variables

| Variable | Dev Default | Production Default |
|----------|-------------|-------------------|
| `REC_API_URL` | `http://host.docker.internal:3000` (Docker) or `http://localhost:3000` (local) | `https://rec.rootly.com` |
| `REC_LOG_LEVEL` | `debug` | `info` |
| `REC_LOG_FORMAT_TYPE` | `colored` | `json` |
| `REC_API_KEY` | `test-key` | Required from env |

## Volume Mounts

### Development Volumes

```yaml
volumes:
  # Configuration (read-write for easy editing)
  - ./config.example.dev.yml:/etc/rootly-edge-connector/config.yml
  - ./actions.example.dev.yml:/etc/rootly-edge-connector/actions.yml

  # Scripts (read-write for development)
  - ./scripts:/opt/rootly-edge-connector/scripts

  # Logs (read-write)
  - ./logs:/var/log/rootly-edge-connector
```

### Production Volumes

```yaml
volumes:
  # Configuration (read-only for security)
  - ./config.yml:/etc/rootly-edge-connector/config.yml:ro
  - ./actions.yml:/etc/rootly-edge-connector/actions.yml:ro

  # Scripts (read-only for security)
  - ./scripts:/opt/rootly-edge-connector/scripts:ro

  # Logs (read-write)
  - ./logs:/var/log/rootly-edge-connector
```

## Testing

### Run Tests in Container

```bash
# Unit tests
docker exec -it rec-dev make test

# Integration tests
docker exec -it rec-dev go test -tags=integration ./tests/integration/...

# With coverage
docker exec -it rec-dev make test-coverage
```

### Run Linter

```bash
docker exec -it rec-dev make lint
```

## Monitoring

### Prometheus Metrics

Metrics are **disabled by default** in dev mode. To enable:

1. Set `metrics.enabled: true` in `config.example.dev.yml`
2. Expose port when running: `-p 9090:9090`

```bash
# Run with metrics enabled
docker run --rm -it \
  -p 9090:9090 \
  -v $(pwd)/config.example.dev.yml:/etc/rootly-edge-connector/config.yml \
  rootly-edge-connector:dev

# View metrics
curl http://localhost:9090/metrics

# Watch metrics
watch -n 1 'curl -s http://localhost:9090/metrics | grep rec_'
```

### Health Check

```bash
# Check container health
docker ps --filter name=rec-dev

# Manual health check
docker exec rec-dev pgrep -f "go run"
```

## Troubleshooting

### "No such container" Error

If you get `Error response from daemon: No such container: rec-dev`:

```bash
# Check if container exists
docker ps -a | grep rec-dev

# If not found, you need to start it first with -d flag and --name rec-dev
docker run -d --name rec-dev \
  -p 9090:9090 \
  -v $(pwd)/config.example.dev.yml:/etc/rootly-edge-connector/config.yml \
  -v $(pwd)/actions.example.dev.yml:/etc/rootly-edge-connector/actions.yml \
  -v $(pwd)/scripts:/opt/rootly-edge-connector/scripts \
  rootly-edge-connector:dev

# Now you can view logs
docker logs -f rec-dev
```

**Note**: `docker logs` only works for containers running in background (`-d` flag). If you're running with `--rm -it`, logs are already streaming to your terminal.

### Container Won't Start

```bash
# Check logs (if container exists)
docker logs rec-dev

# Rebuild from scratch
docker stop rec-dev && docker rm rec-dev
docker build -f Dockerfile.dev --no-cache -t rootly-edge-connector:dev .
docker run --rm -it rootly-edge-connector:dev
```

### Permission Issues

```bash
# Fix script permissions
chmod +x scripts/*.sh

# Fix log directory permissions
mkdir -p logs
chmod 777 logs  # Or chown to appropriate user
```

### Out of Sync

```bash
# Stop and remove container
docker stop rec-dev && docker rm rec-dev

# Remove image and rebuild
docker rmi rootly-edge-connector:dev
docker build -f Dockerfile.dev --no-cache -t rootly-edge-connector:dev .

# Start fresh
docker run --rm -it rootly-edge-connector:dev
```

## Best Practices

### Development

1. **Use dev Dockerfile**: Always use `Dockerfile.dev` for development
2. **Mount volumes**: Mount configs and scripts for easy editing
3. **Use trace logs**: Maximum verbosity for debugging
4. **Keep containers running**: Use `-d` flag and `docker logs -f`
5. **Rebuild when needed**: After Go dependency changes, rebuild image

### Production

1. **Use prod Dockerfile**: The multi-stage build is much smaller
2. **Read-only mounts**: Mount configs and scripts as read-only (`:ro`)
3. **Set resource limits**: Prevent runaway containers
4. **Use secrets**: Never commit `REC_API_KEY` to git
5. **Enable health checks**: Monitor container health
6. **Use JSON logs**: For log aggregation systems

## Integration with CI/CD

### GitHub Actions Example

```yaml
name: Docker Build Test

on: [push, pull_request]

jobs:
  docker-dev:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Build dev image
        run: docker build -f Dockerfile.dev -t rootly-edge-connector:dev .

      - name: Run tests in container
        run: |
          docker run -d --name rec-dev -v $(pwd):/app rootly-edge-connector:dev
          docker exec rec-dev make test
          docker stop rec-dev && docker rm rec-dev
```

## Next Steps

- See [development.md](development.md) for non-Docker local development
- See [systemd-installation.md](systemd-installation.md) for production deployment
- See main [README.md](../README.md) for general usage
