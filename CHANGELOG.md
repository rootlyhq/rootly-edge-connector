# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.0.3] - 2026-01-16

### Fixed
- Improved error message when script path is not within allowed_script_paths to be more actionable and user-friendly
- Error message now includes the attempted path, list of allowed paths, and clear fix instructions

## [0.0.2] - 2026-01-14

### Added
- Trigger compatibility validation for callable vs automatic actions
- Specs for callable actions with zero parameters and trigger validation
- Homebrew installation instructions to documentation
- Workflow diagram to README
- Comprehensive release process documentation to CLAUDE.md

### Changed
- Migrated to Go 1.24 native tool management with golangci-lint v2
- Simplified action registration API to use single actions array
- Simplified REC_API_PATH documentation values to only show /v1
- Upgraded .goreleaser.yml to v2 format
- Updated release workflow to match standard format

### Fixed
- Removed invalid output.formats config from golangci-lint v2 config
- Added missing os import to validator_test.go
- Restored multi-stage Docker builds and improve configuration
- Preallocated actions slice to avoid reallocations
- Used rootlyhub Docker Hub organization in Makefile
- Used GORELEASER_PAT for GitHub and Homebrew tap authentication

## [0.0.1] - 2026-01-08

### Added
- Initial release of Rootly Edge Connector
- Event polling from Rootly API with configurable intervals
- Action execution support (scripts and HTTP requests)
- Script runner with parameter injection via environment variables
- HTTP executor for webhook-style actions
- Worker pool for concurrent event processing
- Prometheus metrics for observability
- Support for custom labels (connector_id, environment, region)
- Configuration via YAML files with environment variable overrides
- Automatic retry mechanism with exponential backoff
- Rate limit handling and logging
- Git repository management for script actions
- Semantic version management with make targets
- Multi-platform support (Linux, macOS, Windows)
- Docker Hub publishing
- Homebrew tap support for easy installation
- Comprehensive test suite with integration tests
- CI/CD with GitHub Actions
- Dependabot configuration for automated dependency updates

### Changed
- License changed from GPL v2 to Apache 2.0
- Upgraded Go dependencies to latest versions

### Fixed
- PowerShell test compatibility across all platforms
- Test ordering and CI matrix fail-fast behavior
- Logger file output with rotation on Windows
- Go script test timeout for Windows CI (increased to 30s)
- golangci-lint v2 configuration

[unreleased]: https://github.com/rootlyhq/rootly-edge-connector/compare/v0.0.3...HEAD
[0.0.3]: https://github.com/rootlyhq/rootly-edge-connector/compare/v0.0.2...v0.0.3
[0.0.2]: https://github.com/rootlyhq/rootly-edge-connector/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/rootlyhq/rootly-edge-connector/releases/tag/v0.0.1
