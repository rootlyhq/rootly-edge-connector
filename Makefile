.PHONY: build build-all clean test lint run install help

# Build variables
BINARY_NAME=rootly-edge-connector
VERSION?=0.1.0
BUILD_DIR=bin
PLATFORMS=linux/amd64 linux/arm64 linux/arm linux/386 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64 freebsd/amd64

# Go build flags
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"
BUILD_FLAGS=-trimpath

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

build: ## Build the binary for current platform
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) cmd/rec/main.go
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

build-all: ## Build binaries for all platforms
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} \
		go build $(BUILD_FLAGS) $(LDFLAGS) \
		-o $(BUILD_DIR)/$(BINARY_NAME)-$${platform%/*}-$${platform#*/} \
		cmd/rec/main.go; \
		echo "Built: $(BUILD_DIR)/$(BINARY_NAME)-$${platform%/*}-$${platform#*/}"; \
	done
	@echo "All builds complete!"

clean: ## Remove build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@go clean
	@echo "Clean complete!"

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race -cover ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out

test-coverage-html: ## Generate HTML coverage report
	@echo "Generating HTML coverage report..."
	@go test -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo ""
	@echo "Coverage report generated: coverage.html"
	@echo "Open with: open coverage.html (macOS) or xdg-open coverage.html (Linux)"
	@echo ""
	@go tool cover -func=coverage.out | grep total

test-integration: ## Run integration tests (tagged with //go:build integration)
	@echo "Running integration tests..."
	@go test -v -tags=integration ./tests/integration/...

test-all: test test-integration ## Run both unit and integration tests

lint: ## Run linter
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --timeout=5m; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Format complete!"

goimports: ## Run goimports
	@echo "Running goimports..."
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w -local github.com/rootly/edge-connector ./cmd ./internal ./pkg ./tests; \
	else \
		echo "goimports not installed. Install with: go install golang.org/x/tools/cmd/goimports@latest"; \
		exit 1; \
	fi

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

tidy: ## Tidy go.mod and go.sum
	@echo "Tidying modules..."
	@go mod tidy
	@echo "Tidy complete!"

run: ## Run the application (requires config.yml and actions.yml)
	@echo "Running $(BINARY_NAME)..."
	@go run cmd/rec/main.go -config config.yml -actions actions.yml

install: build ## Install the binary to /usr/local/bin
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	@sudo install -m 755 $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "Install complete!"

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@echo "Dependencies downloaded!"

vendor: ## Vendor dependencies
	@echo "Vendoring dependencies..."
	@go mod vendor
	@echo "Vendor complete!"

check: fmt vet lint test ## Run all checks (format, vet, lint, test)

version: ## Show version
	@echo "Version: $(VERSION)"

# Development targets
dev-setup: ## Setup development environment
	@echo "Setting up development environment..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go mod download
	@echo "Development setup complete!"

docker-build-dev: ## Build development Docker image
	@echo "Building development Docker image..."
	@docker build -f Dockerfile.dev -t rootly-edge-connector:dev .
	@echo "Development Docker image built: rootly-edge-connector:dev"

docker-build-prod: ## Build production Docker image
	@echo "Building production Docker image..."
	@docker build -f Dockerfile -t rootly-edge-connector:$(VERSION) .
	@docker tag rootly-edge-connector:$(VERSION) rootly-edge-connector:latest
	@echo "Production Docker image built: rootly-edge-connector:$(VERSION)"

docker-build: docker-build-prod ## Build production Docker image (alias for docker-build-prod)

docker-run-dev: ## Run development Docker container
	@echo "Running development container..."
	@docker run --rm \
		-e REC_API_KEY=$${REC_API_KEY} \
		-v $(PWD)/config.example.dev.yml:/etc/rootly-edge-connector/config.yml:ro \
		-v $(PWD)/actions.example.dev.yml:/etc/rootly-edge-connector/actions.yml:ro \
		-v $(PWD)/scripts:/opt/rootly-edge-connector/scripts:ro \
		-p 9090:9090 \
		rootly-edge-connector:dev

docker-run-prod: ## Run production Docker container
	@echo "Running production container..."
	@docker run --rm \
		-e REC_API_KEY=$${REC_API_KEY} \
		-v $(PWD)/config.example.yml:/etc/rootly-edge-connector/config.yml:ro \
		-v $(PWD)/actions.example.yml:/etc/rootly-edge-connector/actions.yml:ro \
		-v $(PWD)/scripts:/opt/rootly-edge-connector/scripts:ro \
		-p 9090:9090 \
		rootly-edge-connector:latest

# Release targets
release: clean build-all ## Create a release (clean + build for all platforms)
	@echo "Release $(VERSION) complete!"
	@ls -lh $(BUILD_DIR)/

# Version management targets
# These targets manage semantic versioning using git tags
.PHONY: version-patch version-minor version-major version-show version-next version-help

version-show: ## Show current and next versions
	@echo "Current version: $$(git describe --tags --abbrev=0 2>/dev/null || echo 'No tags found')"
	@echo "Next patch: $$(scripts/bump-version.sh show patch)"
	@echo "Next minor: $$(scripts/bump-version.sh show minor)"
	@echo "Next major: $$(scripts/bump-version.sh show major)"

version-patch: ## Bump patch version (1.2.3 → 1.2.4)
	@scripts/bump-version.sh patch

version-minor: ## Bump minor version (1.2.3 → 1.3.0)
	@scripts/bump-version.sh minor

version-major: ## Bump major version (1.2.3 → 2.0.0)
	@scripts/bump-version.sh major

version-next: ## Show next patch version
	@scripts/bump-version.sh show patch

version-help: ## Show detailed version help
	@scripts/bump-version.sh help

# Release targets - these create git tags which trigger CI releases
.PHONY: release-patch release-minor release-major

release-patch: version-patch ## Bump patch version and push tag (triggers CI release)
	@echo "✅ Patch version bumped and tagged"
	@echo "🚀 GitHub Actions will automatically build and publish the release"

release-minor: version-minor ## Bump minor version and push tag (triggers CI release)
	@echo "✅ Minor version bumped and tagged"
	@echo "🚀 GitHub Actions will automatically build and publish the release"

release-major: version-major ## Bump major version and push tag (triggers CI release)
	@echo "✅ Major version bumped and tagged"
	@echo "🚀 GitHub Actions will automatically build and publish the release"
