# Developer Guide

This guide covers how to set up, run, and develop the Rootly Edge Connector locally.

## Prerequisites

- Go 1.24+ ([installation guide](https://golang.org/doc/install))
- Git
- (Optional) Docker for testing containerized deployment
- (Optional) [mise](https://mise.jdx.dev/) for managing Go version

### Using mise (Recommended)

The project includes a `mise.toml` file for automatic Go version management:

```bash
# Install mise
curl https://mise.run | sh

# Or on macOS
brew install mise

# Install project dependencies (including Go 1.24)
mise install

# Regenerate shims (if you installed new tools)
mise reshim

# mise will now automatically use the correct Go version
# when you're in the project directory
```

## Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/rootlyhq/rootly-edge-connector.git
cd rootly-edge-connector
```

### 2. Install Dependencies

```bash
# Download Go modules
go mod download

# Or use make
make deps
```

### 3. Create Local Configuration

**For local development (recommended):**
```bash
# Use the dev config pre-configured for localhost
cp config.example.dev.yml config.yml
cp actions.example.yml actions.yml
```

The dev config is optimized for local development:
- Points to `http://localhost:3000` (mock server)
- Fast polling (1 second)
- Debug logging with colors
- No script path restrictions
- Fewer workers and smaller queues

**For testing against staging/production:**
```bash
# Use the production config template
cp config.example.yml config.yml
cp actions.example.yml actions.yml

# Then edit config.yml and set your API key
# Get your API key from the Rootly UI or use REC_API_KEY env var
```

### 4. Create a Test Script

Create a simple test script at `/tmp/test-script.sh`:

```bash
#!/bin/bash
echo "Script executed successfully"
echo "Host: $REC_PARAM_HOST"
echo "Event ID: $REC_PARAM_EVENT_ID"
exit 0
```

```bash
chmod +x /tmp/test-script.sh
```

Update `actions.yml`:

```yaml
actions:
  - name: test_action
    type: script
    script: /tmp/test-script.sh
    trigger:
      event_type: "alert.created"
    parameters:
      host: "{{ alert.host }}"
      event_id: "{{ alert.id }}"
    timeout: 30
```

### 5. Run Locally

```bash
# Set API key
export REC_API_KEY="your-api-key"

# Run with go run
go run cmd/rec/main.go -config config.yml -actions actions.yml

# Or build and run
make build
./bin/rootly-edge-connector -config config.yml -actions actions.yml
```

## Development Workflow

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Clean build artifacts
make clean
```

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/api
go test ./internal/executor

# Run with race detector
go test -race ./...

# Run with verbose output
go test -v ./...
```

### Code Quality

```bash
# Format code
make fmt

# Run goimports
make goimports

# Run linter
make lint

# Run all checks
make check  # Runs fmt, vet, lint, test
```

### Running with Different Configs

```bash
# Development mode (verbose logging)
go run cmd/rec/main.go \
    -config config.yml \
    -actions actions.yml

# With custom log level
REC_LOG_FORMAT_TYPE=colored go run cmd/rec/main.go \
    -config config.yml \
    -actions actions.yml

# Test with mock configuration
go run cmd/rec/main.go \
    -config config.example.yml \
    -actions actions.example.yml
```

## Testing Locally Without Rootly API

### Option 1: Rootly Mock Server (Recommended)

Use the Rootly Rails server as your mock API:

```bash
# In a separate terminal, start the Rootly Rails server on port 3000
WEB_CONCURRENCY=1 BASE_URL=http://localhost bundle exec rails s -p 3000
```

Your `config.yml` should already be configured for localhost:3000 (see Quick Start step 3 above).

### Option 2: Integration Tests

Run the integration tests which include a mock server:

```bash
go test -tags=integration ./tests/integration/...
```

## Monitoring During Development

### View Prometheus Metrics

While the connector is running:

```bash
# View all metrics
curl http://localhost:9090/metrics

# View specific metrics
curl http://localhost:9090/metrics | grep rec_events

# Watch metrics in real-time
watch -n 1 'curl -s http://localhost:9090/metrics | grep rec_'
```

### View Logs

```bash
# If using JSON logging
tail -f /var/log/rootly-edge-connector/connector.log | jq

# If using text/colored logging
tail -f /var/log/rootly-edge-connector/connector.log
```

## Docker Development

### Build Docker Image

```bash
# Development image
docker build -f Dockerfile.dev -t rootly-edge-connector:dev .

# Production image
docker build -f Dockerfile -t rootly-edge-connector:prod .
```

### Run Docker Container

```bash
# Run with local config
docker run --rm \
    -v $(pwd)/config.yml:/etc/rootly-edge-connector/config.yml:ro \
    -v $(pwd)/actions.yml:/etc/rootly-edge-connector/actions.yml:ro \
    -p 9090:9090 \
    rootly-edge-connector:dev
```

## Debugging

### Enable Debug Logging

In `config.yml`:
```yaml
logging:
  level: "debug"  # or "trace" for even more detail
  format: "colored"  # Easier to read during development
  output: "stdout"
```

### Debug Specific Components

Add log statements using logrus:

```go
import log "github.com/sirupsen/logrus"

log.WithFields(log.Fields{
    "event_id": event.ID,
    "action": action.Name,
}).Debug("Processing event")
```

### Common Issues

**"Failed to load config":**
- Check file paths are correct
- Verify YAML syntax with `yamllint config.yml`
- Ensure required fields are present

**"Script path not allowed":**
- Add script directory to `security.allowed_script_paths` in config.yml
- Or leave `allowed_script_paths` empty to allow all paths

**"Rate limit exceeded":**
- Check logs for X-RateLimit-* headers
- Increase polling interval in config
- Contact Rootly for higher limits

## IDE Setup

### VS Code

Recommended extensions:
- Go (golang.go)
- YAML (redhat.vscode-yaml)
- Docker (ms-azuretools.vscode-docker)

`.vscode/settings.json`:
```json
{
  "go.useLanguageServer": true,
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "workspace",
  "[go]": {
    "editor.formatOnSave": true,
    "editor.codeActionsOnSave": {
      "source.organizeImports": true
    }
  }
}
```

### GoLand / IntelliJ IDEA

1. Enable Go modules support
2. Configure golangci-lint integration
3. Enable goimports on save

## Contributing

### Before Committing

```bash
# 1. Format code
make fmt
make goimports

# 2. Run tests
make test

# 3. Run linter
make lint

# 4. Or run all checks
make check
```

### Commit Message Format

Follow conventional commits:

```
feat: add new feature
fix: fix a bug
docs: update documentation
test: add tests
chore: update dependencies
```

## Useful Make Targets

```bash
make help          # Show all available targets
make build         # Build binary
make test          # Run tests
make lint          # Run linter
make fmt           # Format code
make goimports     # Fix imports
make check         # Run all checks
make clean         # Remove build artifacts
make tidy          # Tidy go modules
make run           # Run with example configs
```

## Performance Profiling

### CPU Profiling

```bash
# Build with profiling
go build -o bin/rootly-edge-connector cmd/rec/main.go

# Run with CPU profiling
./bin/rootly-edge-connector -cpuprofile=cpu.prof \
    -config config.yml -actions actions.yml

# Analyze profile
go tool pprof cpu.prof
```

### Memory Profiling

```bash
# Build with profiling
go build -o bin/rootly-edge-connector cmd/rec/main.go

# Run with memory profiling
./bin/rootly-edge-connector -memprofile=mem.prof \
    -config config.yml -actions actions.yml

# Analyze profile
go tool pprof mem.prof
```

## Environment Variables

All environment variables supported:

| Variable | Description | Example |
|----------|-------------|---------|
| `REC_API_URL` | Rootly API base URL | `https://rec.rootly.com` |
| `REC_API_PATH` | API path prefix | `/v1` or `/rec/v1` |
| `REC_API_KEY` | Rootly API key | `rec_xxxxx` |
| `REC_LOG_FORMAT_TYPE` | Log format override | `json`, `text`, `colored` |
| `REC_LOG_LEVEL` | Log level override | `trace`, `debug`, `info`, `warn`, `error` |

Parameters passed to scripts:

| Variable Pattern | Description | Example |
|-----------------|-------------|---------|
| `REC_PARAM_*` | Action parameters | `REC_PARAM_HOST=prod-db-01` |

## Troubleshooting

### Dependency Issues

```bash
# Clean and re-download
go clean -modcache
go mod download
go mod tidy
```

### Build Issues

```bash
# Clean all caches
go clean -cache -modcache -testcache
make clean
make build
```

### Test Issues

```bash
# Clear test cache
go clean -testcache

# Run specific test
go test -v -run TestScriptRunner ./internal/executor

# Run with more timeout
go test -timeout 60s ./...
```

## Next Steps

- Read the [systemd installation guide](systemd-installation.md)
- Check the [main README](../README.md) for usage examples
- Review [actions.example.yml](../actions.example.yml) for action configuration
- See [config.example.yml](../config.example.yml) for all configuration options
