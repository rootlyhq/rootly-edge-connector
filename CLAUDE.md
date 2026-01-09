# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Development Commands

### Setup & Building

```bash
# Install dependencies
go mod download
make deps

# Build binary
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
make test-coverage

# Run specific package tests
go test ./internal/api
go test ./internal/executor

# Run integration tests
go test -tags=integration ./tests/integration/...

# Run with race detector
go test -race ./...
```

### Code Quality - IMPORTANT: Run Before Each Commit

```bash
# Format code (REQUIRED before commit)
make fmt

# Fix imports
make goimports

# Run linter (REQUIRED before commit)
make lint

# Run all checks (fmt + vet + lint + test)
make check
```

**CRITICAL: Always run `make lint` before creating commits to ensure code quality.**

### Running Locally

```bash
# Copy dev config for local testing
cp config.example.dev.yml config.yml
cp actions.example.yml actions.yml

# Run with go run
go run cmd/rec/main.go -config config.yml -actions actions.yml

# Or build and run
make build
./bin/rootly-edge-connector -config config.yml -actions actions.yml

# With environment variable overrides
REC_LOG_LEVEL=debug REC_LOG_FORMAT_TYPE=colored ./bin/rootly-edge-connector
```

## Architecture Overview

### Technology Stack

- **Language**: Go 1.24
- **HTTP Client**: retryablehttp (with automatic retries)
- **Logging**: logrus with structured logging
- **Metrics**: Prometheus client
- **Config**: YAML-based configuration

### Project Structure

```
cmd/rec/              - Main application entry point
internal/
  api/                - Rootly API client
  config/             - Configuration loading and validation
  executor/           - Action execution (scripts, HTTP)
  metrics/            - Prometheus metrics
  poller/             - Event polling engine
  reporter/           - Execution result reporting
  worker/             - Worker pool for concurrent execution
pkg/
  git/                - Git repository management
tests/
  integration/        - End-to-end integration tests
```

### Key Components

#### 1. API Client (`internal/api/client.go`)
- Handles all HTTP communication with Rootly API
- Endpoints:
  - `GET /rec/v1/deliveries` - Poll for deliveries
  - `POST /rec/v1/deliveries/:id/acknowledge` - Acknowledge delivery
  - `PATCH /rec/v1/deliveries/:id` - Report execution (auto-acknowledges)
- Automatic retries with exponential backoff
- Rate limit handling and logging
- HTTP request/response logging at DEBUG level

#### 2. Poller (`internal/poller/poller.go`)
- Polls Rootly API for new events
- Configurable polling interval and visibility timeout
- Error handling with retry logic

#### 3. Executor (`internal/executor/`)
- Executes actions (scripts or HTTP requests)
- Script runner with parameter injection
- HTTP executor for webhook-style actions
- Timeout handling and error reporting

#### 4. Worker Pool (`internal/worker/pool.go`)
- Concurrent event processing
- Dynamic worker scaling
- Queue management

#### 5. Metrics (`internal/metrics/metrics.go`)
- Prometheus metrics for observability
- Supports custom labels (connector_id, environment, region, etc.)
- 12 metrics covering events, actions, workers, HTTP, and git

## Configuration

### Environment Variables

The following environment variables override config file settings:

| Variable | Override | Use Case |
|----------|----------|----------|
| `REC_API_URL` | `rootly.api_url` | Switch environments |
| `REC_API_PATH` | `rootly.api_path` | API version override |
| `REC_API_KEY` | `rootly.api_key` | Secret management |
| `REC_LOG_FORMAT_TYPE` | `logging.format` | Log format (json/text/colored) |
| `REC_LOG_LEVEL` | `logging.level` | Debug verbosity |

### Config Files

- `config.yml` - Main configuration (copy from `config.example.yml` or `config.example.dev.yml`)
- `actions.yml` - Action definitions (copy from `actions.example.yml`)

## Testing Patterns

### Unit Tests
- Use `testify/assert` and `testify/require` for assertions
- Create temp directories with `t.TempDir()`
- Mock external dependencies (API server, etc.)

### Integration Tests
- Tagged with `//go:build integration`
- Use MockAPIServer for end-to-end testing
- Test full workflows (poll → execute → report)

## Important Development Notes

### Before Every Commit

1. **Format code**: `make fmt`
2. **Run linter**: `make lint` (REQUIRED - catches issues early)
3. **Run tests**: `make test`
4. **Or use**: `make check` (runs all three)

### Logging Best Practices

- Use structured logging with `log.WithFields()`
- Log levels:
  - **TRACE**: Rate limits, full request/response bodies
  - **DEBUG**: HTTP requests/responses, event details
  - **INFO**: Action execution, major events
  - **WARN**: Retries, degraded performance
  - **ERROR**: Failures, errors

### HTTP Client Notes

- All API calls include rate limit handling
- Automatic retries (3 attempts with exponential backoff)
- Token redaction in logs (shows last 8 chars only)
- Request/response timing logged at DEBUG level

### Security

- Tokens must start with `rec_` prefix
- Scripts can be restricted to allowed paths
- Global environment variables can be set for all scripts
- Parameters are passed as `REC_PARAM_*` environment variables

## Common Tasks

### Adding a New Environment Variable Override

1. Add env var check in `internal/config/loader.go`
2. Add test in `internal/config/loader_test.go`
3. Document in README.md and docs/development.md

### Adding New Metrics

1. Define metric in `internal/metrics/metrics.go` (as var)
2. Initialize in `InitMetrics()` with custom labels
3. Register with `prometheus.MustRegister()`
4. Use in relevant code: `metrics.MyMetric.Inc()`
5. Document in README.md

### Adding HTTP Debug Logging

- Request: Log at DEBUG level with method, URL, key fields
- Response: Log at DEBUG level with status, duration
- Body: Log at TRACE level (can be large)
- Always redact sensitive data (tokens, passwords)

## CI/CD

### GitHub Actions Workflows

- **test.yml**: Runs on push/PR - tests on 4 OS × Go 1.24
- **release.yml**: Runs on version tags - builds multi-platform binaries

### Release Process

The release process uses semantic versioning with git tags. Pushing a version tag automatically triggers GitHub Actions to build and publish binaries for all platforms via GoReleaser.

#### Prerequisites

**CRITICAL**: Ensure your working directory is clean before releasing:
```bash
git status  # Must show "nothing to commit, working tree clean"
```

The release script will **fail** if there are uncommitted changes.

#### Quick Release Commands

```bash
# Patch release (1.2.3 → 1.2.4) - for bug fixes
make release-patch

# Minor release (1.2.3 → 1.3.0) - for new features
make release-minor

# Major release (1.2.3 → 2.0.0) - for breaking changes
make release-major
```

Each command will:
1. Check that your working directory is clean
2. Show current and next version
3. Ask for confirmation
4. Create and push a git tag (e.g., `v1.2.4`)
5. Trigger GitHub Actions to build and publish the release

#### Checking Version Information

```bash
# Show current version and all next versions
make version-show

# Example output:
# Current version: 1.2.3
# Next patch: 1.2.4
# Next minor: 1.3.0
# Next major: 2.0.0

# Show just the next patch version
make version-next
```

#### Manual Release Process

If you need more control, you can use the `bump-version.sh` script directly:

```bash
# Show what the next version would be (no changes)
./scripts/bump-version.sh show patch
./scripts/bump-version.sh show minor
./scripts/bump-version.sh show major

# Bump and tag (with confirmation prompt)
./scripts/bump-version.sh patch
./scripts/bump-version.sh minor
./scripts/bump-version.sh major

# Show help
./scripts/bump-version.sh help
```

#### What Happens After Tagging

1. **Git tag pushed**: e.g., `v1.2.4`
2. **GitHub Actions triggered**: `.github/workflows/release.yml` runs
3. **GoReleaser builds**:
   - Linux (amd64, arm64, arm)
   - macOS (amd64, arm64)
   - Windows (amd64)
4. **GitHub Release created** with:
   - Binaries for all platforms
   - Checksums
   - Release notes (auto-generated from commits)
5. **Artifacts published** at `https://github.com/rootlyhq/rootly-edge-connector/releases`

#### Semantic Versioning Guidelines

Use **patch** (1.2.3 → 1.2.4) for:
- Bug fixes
- Security patches
- Performance improvements
- Documentation updates

Use **minor** (1.2.3 → 1.3.0) for:
- New features (backwards compatible)
- New configuration options
- New environment variables
- Deprecations (with backward compatibility)

Use **major** (1.2.3 → 2.0.0) for:
- Breaking changes
- API changes
- Configuration format changes
- Removing deprecated features

#### Troubleshooting

**Error: "Working directory is not clean"**
```bash
# Check what's uncommitted
git status

# Either commit changes:
git add .
git commit -m "fix: description"

# Or stash them:
git stash
```

**Error: "Tag already exists"**
```bash
# Delete local tag
git tag -d v1.2.4

# Delete remote tag (careful!)
git push origin :refs/tags/v1.2.4

# Try release again
make release-patch
```

**Release failed in CI**
- Check GitHub Actions: https://github.com/rootlyhq/rootly-edge-connector/actions
- View workflow logs for errors
- Common issues: Go build errors, missing dependencies
- Fix issues, then create a new patch release

#### Manual Tag and Push (Not Recommended)

If you need to manually create a tag without the script:

```bash
# Create annotated tag
git tag -a v1.2.4 -m "Release v1.2.4"

# Push tag to trigger CI
git push origin v1.2.4

# View release progress
# https://github.com/rootlyhq/rootly-edge-connector/actions
```

**Note**: Using `make release-*` commands is preferred as they include safety checks.

## Test Fixtures and Testdata Best Practices

### Directory Structure

Organize test data into `testdata/` directories following Go conventions:

```
internal/api/testdata/
  └── fixtures/
      ├── events_alert_and_incident.json
      └── events_action_triggered.json

internal/executor/testdata/
  ├── README.md
  └── fixtures/
      ├── test.py
      ├── test.js
      ├── test.sh
      └── ...

tests/integration/testdata/
  └── fixtures/
      └── event_incident_for_http.json

internal/config/testdata/
  └── fixtures/
      ├── valid_actions.yml
      ├── duplicate_ids.yml
      └── action_with_id_and_name.yml
```

### Best Practices

#### 1. Use JSON Fixtures for API Payloads

**Do:**
```go
// Load from file
fixture := loadFixture(t, "events_alert_and_incident.json")
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(fixture)
}))
```

**Don't:**
```go
// Inline JSON (hard to maintain, easy to make mistakes)
response := api.EventsResponse{
    Events: []api.Event{
        {ID: "...", EventID: "...", Type: "...", /* 50 more lines */},
    },
}
```

**Benefits:**
- ✅ Fixtures match real backend payloads
- ✅ Easy to update when API changes
- ✅ Reusable across multiple tests
- ✅ Can be validated against JSON schema
- ✅ Easier to review in PRs

#### 2. Use YAML Fixtures for Config Tests

**Do:**
```go
cfg, err := config.LoadActions("testdata/fixtures/valid_actions.yml")
require.NoError(t, err)
assert.Len(t, cfg.Actions, 2)
```

**Don't:**
```go
// Inline YAML in test code
actionsContent := `
actions:
  - id: test
    type: script
    # ... 20 more lines
`
```

**Benefits:**
- ✅ Fixtures can be tested manually
- ✅ IDE syntax highlighting
- ✅ Easier to spot YAML errors
- ✅ Can be used as example configs

#### 3. Organize by Purpose

**Script execution fixtures:**
- Location: `internal/executor/testdata/fixtures/`
- Content: Executable scripts (test.py, test.sh, test.js)
- Purpose: Test script runner with real scripts

**API payload fixtures:**
- Location: `internal/api/testdata/fixtures/`
- Content: JSON event payloads
- Purpose: Test client parsing and serialization

**Config fixtures:**
- Location: `internal/config/testdata/fixtures/`
- Content: YAML configuration files
- Purpose: Test config loading and validation

**Integration fixtures:**
- Location: `tests/integration/testdata/fixtures/`
- Content: End-to-end test data
- Purpose: Full workflow testing

#### 4. Naming Conventions

**Use descriptive names:**
- ✅ `events_action_triggered.json` (what it contains)
- ✅ `duplicate_ids.yml` (what it tests)
- ✅ `action_with_id_and_name.yml` (scenario)
- ❌ `test1.json` (unclear)
- ❌ `data.yml` (too generic)

**Use underscores for readability:**
- ✅ `valid_actions.yml`
- ❌ `validactions.yml`

#### 5. Add README.md to Fixture Directories

Document what each fixture is for:

```markdown
# Test Fixtures

- `events_alert_and_incident.json` - Alert and incident events matching serializers
- `events_action_triggered.json` - All 3 action types (alert, incident, standalone)
- `valid_actions.yml` - Minimal valid action config for testing defaults
```

#### 6. Make Scripts Executable

For script fixtures:
```bash
chmod +x internal/executor/testdata/fixtures/*.sh
chmod +x internal/executor/testdata/fixtures/*.py
```

**In tests:**
```go
scriptPath, allowedDir, err := getFixturePath("test.sh")
require.NoError(t, err)
runner := executor.NewScriptRunner([]string{allowedDir}, nil)
```

#### 7. Helper Functions

Create helpers to reduce duplication:

```go
// Load JSON fixture
func loadFixture(t *testing.T, filename string) api.EventsResponse {
    data, err := os.ReadFile("testdata/fixtures/" + filename)
    require.NoError(t, err)

    var response api.EventsResponse
    err = json.Unmarshal(data, &response)
    require.NoError(t, err)

    return response
}

// Get script fixture path
func getFixturePath(filename string) (scriptPath, allowedDir string, err error) {
    wd, err := os.Getwd()
    if err != nil {
        return "", "", err
    }
    fixturesDir := filepath.Join(wd, "testdata/fixtures")
    scriptPath = filepath.Join(fixturesDir, filename)
    return scriptPath, fixturesDir, nil
}
```

#### 8. Keep Fixtures Realistic

**Match actual backend payloads:**
- Use real UUIDs format
- Use realistic timestamps
- Include all fields from serializers
- Match actual data structures

**Example:**
```json
{
  "id": "6aeb35ae-ca31-4bcf-91bd-c4ecce44dedc",
  "event_id": "82a9699f-3a68-49a8-9ae6-d0c04e91f956",
  "event_type": "alert.created",
  "timestamp": "2025-10-26T21:30:00Z",
  "data": {
    "id": "alert-uuid-1",
    "source": "datadog",
    "services": [...],
    "environments": [...]
  }
}
```

#### 9. Version Control

**Do commit:**
- ✅ JSON/YAML fixtures
- ✅ README.md in testdata/
- ✅ Script fixtures (.sh, .py, .js)

**Don't commit:**
- ❌ Generated test output
- ❌ Coverage files (*.out)
- ❌ Temporary test files

#### 10. Update Fixtures When API Changes

When backend API changes:
1. Update fixture files first
2. Run tests (they'll fail with clear diffs)
3. Update test assertions
4. Update code if needed

**This ensures tests accurately reflect real API.**

---

## Additional Resources

- Main README: [../README.md](../README.md)
- Development Guide: [docs/development/development.md](docs/development/development.md)
- Systemd Installation: [docs/user-guide/systemd-installation.md](docs/user-guide/systemd-installation.md)
- Test Coverage Report: [docs/development/test-coverage-improvements.md](docs/development/test-coverage-improvements.md)
