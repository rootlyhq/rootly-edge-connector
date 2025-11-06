package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rootly/edge-connector/internal/config"
)

func TestLoad_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	configContent := `
app:
  name: "test-connector"

rootly:
  api_url: "https://api.rootly.com"
  api_key: "test-api-key"

poller:
  polling_wait_interval_ms: 5000
  visibility_timeout_sec: 30
  max_number_of_messages: 10
  retry_on_error: true
  retry_backoff: "exponential"
  max_retries: 3

pool:
  max_number_of_workers: 10
  min_number_of_workers: 2
  queue_size: 1000

security:
  script_timeout: 300
  allowed_script_paths:
    - /opt/scripts
  global_env:
    ENV: "production"

logging:
  level: "info"
  format: "json"
  output: "stdout"
  max_size_mb: 100
  max_backups: 3
  max_age_days: 7
  compress: true

metrics:
  enabled: true
  port: 9090
  path: "/metrics"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := config.Load(configPath)
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// App config
	assert.Equal(t, "test-connector", cfg.App.Name)

	// Rootly config
	assert.Equal(t, "https://api.rootly.com", cfg.Rootly.APIURL)
	assert.Equal(t, "test-api-key", cfg.Rootly.APIKey)

	// Poller config
	assert.Equal(t, 5000, cfg.Poller.PollingWaitIntervalMs)
	assert.Equal(t, 30, cfg.Poller.VisibilityTimeoutSec)
	assert.Equal(t, 10, cfg.Poller.MaxNumberOfMessages)
	assert.True(t, cfg.Poller.RetryOnError)
	assert.Equal(t, "exponential", cfg.Poller.RetryBackoff)

	// Pool config
	assert.Equal(t, 10, cfg.Pool.MaxNumberOfWorkers)
	assert.Equal(t, 2, cfg.Pool.MinNumberOfWorkers)

	// Security config
	assert.Equal(t, 300, cfg.Security.ScriptTimeout)
	assert.Contains(t, cfg.Security.AllowedScriptPaths, "/opt/scripts")
	assert.Equal(t, "production", cfg.Security.GlobalEnv["ENV"])

	// Logging config
	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)
}

func TestLoad_EnvironmentVariableOverride(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	configContent := `
app:
  name: "test"
rootly:
  api_url: "https://api.rootly.com"
  api_key: "file-api-key"
poller:
  polling_wait_interval_ms: 5000
pool:
  max_number_of_workers: 5
  min_number_of_workers: 1
logging:
  level: "info"
  format: "text"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Set environment variables
	os.Setenv("REC_API_URL", "https://env-api.rootly.com")
	defer os.Unsetenv("REC_API_URL")

	os.Setenv("REC_API_PATH", "/rec/v2")
	defer os.Unsetenv("REC_API_PATH")

	os.Setenv("REC_API_KEY", "env-api-key")
	defer os.Unsetenv("REC_API_KEY")

	os.Setenv("REC_LOG_FORMAT_TYPE", "json")
	defer os.Unsetenv("REC_LOG_FORMAT_TYPE")

	os.Setenv("REC_LOG_LEVEL", "debug")
	defer os.Unsetenv("REC_LOG_LEVEL")

	cfg, err := config.Load(configPath)
	require.NoError(t, err)

	// Environment variables should override file values
	assert.Equal(t, "https://env-api.rootly.com", cfg.Rootly.APIURL)
	assert.Equal(t, "/rec/v2", cfg.Rootly.APIPath)
	assert.Equal(t, "env-api-key", cfg.Rootly.APIKey)
	assert.Equal(t, "json", cfg.Logging.Format)
	assert.Equal(t, "debug", cfg.Logging.Level)
}

func TestLoad_Defaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// Minimal config - most values should get defaults
	configContent := `
app:
  name: "test"
rootly:
  api_url: "https://api.rootly.com"
  api_key: "test-key"
pool:
  max_number_of_workers: 5
  min_number_of_workers: 1
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := config.Load(configPath)
	require.NoError(t, err)

	// Check defaults are applied
	assert.Equal(t, 5000, cfg.Poller.PollingWaitIntervalMs, "Default polling interval")
	assert.Equal(t, 30, cfg.Poller.VisibilityTimeoutSec, "Default visibility timeout")
	assert.Equal(t, 10, cfg.Poller.MaxNumberOfMessages, "Default max messages")
	assert.Equal(t, "exponential", cfg.Poller.RetryBackoff, "Default backoff strategy")
	assert.Equal(t, 300, cfg.Security.ScriptTimeout, "Default script timeout")
	assert.Equal(t, "info", cfg.Logging.Level, "Default log level")
	assert.Equal(t, "text", cfg.Logging.Format, "Default log format")
	assert.Equal(t, "stdout", cfg.Logging.Output, "Default log output")
	assert.Equal(t, 9090, cfg.Metrics.Port, "Default metrics port")
	assert.Equal(t, "/metrics", cfg.Metrics.Path, "Default metrics path")
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	invalidContent := `
app:
  name: test
  invalid yaml here {{
`

	err := os.WriteFile(configPath, []byte(invalidContent), 0644)
	require.NoError(t, err)

	_, err = config.Load(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := config.Load("/nonexistent/config.yml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read")
}

func TestLoadActions_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	actionsPath := filepath.Join(tmpDir, "actions.yml")

	// Create a dummy script for validation
	scriptPath := filepath.Join(tmpDir, "test.sh")
	err := os.WriteFile(scriptPath, []byte("#!/bin/bash\necho test"), 0755)
	require.NoError(t, err)

	actionsContent := `
on:
  alert.created:
    type: script
    source_type: local
    script: ` + scriptPath + `
    parameters:
      host: "{{ data.host }}"
    timeout: 60

  incident.created:
    type: http
    http:
      url: "https://api.example.com/notify"
      method: POST
      headers:
        Content-Type: "application/json"
      body: |
        {"message": "{{ title }}"}
    timeout: 10
`

	err = os.WriteFile(actionsPath, []byte(actionsContent), 0644)
	require.NoError(t, err)

	actions, err := config.LoadActions(actionsPath)
	require.NoError(t, err)
	assert.NotNil(t, actions)
	assert.Len(t, actions.Actions, 2)

	// Find actions by ID (order is non-deterministic from map iteration)
	var alertAction, incidentAction *config.Action
	for i := range actions.Actions {
		if actions.Actions[i].ID == "alert.created" {
			alertAction = &actions.Actions[i]
		} else if actions.Actions[i].ID == "incident.created" {
			incidentAction = &actions.Actions[i]
		}
	}

	// Verify alert.created action
	require.NotNil(t, alertAction, "Should have alert.created action")
	assert.Equal(t, "alert.created", alertAction.ID)
	assert.Equal(t, "", alertAction.Name) // No name for automatic actions
	assert.Equal(t, "script", alertAction.Type)
	assert.Equal(t, "local", alertAction.SourceType)
	assert.Equal(t, scriptPath, alertAction.Script)
	assert.Equal(t, "alert.created", alertAction.Trigger.EventType)
	assert.Equal(t, 60, alertAction.Timeout)

	// Verify incident.created action
	require.NotNil(t, incidentAction, "Should have incident.created action")
	assert.Equal(t, "incident.created", incidentAction.ID)
	assert.Equal(t, "", incidentAction.Name) // No name for automatic actions
	assert.Equal(t, "http", incidentAction.Type)
	assert.NotNil(t, incidentAction.HTTP)
	assert.Equal(t, "https://api.example.com/notify", incidentAction.HTTP.URL)
	assert.Equal(t, "POST", incidentAction.HTTP.Method)
}

func TestLoadActions_Defaults(t *testing.T) {
	tmpDir := t.TempDir()
	actionsPath := filepath.Join(tmpDir, "actions.yml")

	scriptPath := filepath.Join(tmpDir, "test.sh")
	err := os.WriteFile(scriptPath, []byte("#!/bin/bash\necho test"), 0755)
	require.NoError(t, err)

	// Minimal action config
	actionsContent := `
on:
  alert.created:
    script: ` + scriptPath + `
`

	err = os.WriteFile(actionsPath, []byte(actionsContent), 0644)
	require.NoError(t, err)

	actions, err := config.LoadActions(actionsPath)
	require.NoError(t, err)

	action := actions.Actions[0]
	assert.Equal(t, "script", action.Type, "Default type")
	assert.Equal(t, "local", action.SourceType, "Default source type")
	assert.Equal(t, 30, action.Timeout, "Default timeout for on: format")
}

func TestLoadActions_GitOptions(t *testing.T) {
	tmpDir := t.TempDir()
	actionsPath := filepath.Join(tmpDir, "actions.yml")

	actionsContent := `
on:
  alert.created:
    type: script
    source_type: git
    script: scripts/test.sh
    git_options:
      url: "git@github.com:org/repo.git"
      private_key_path: "/etc/ssh/id_rsa"
      branch: "main"
      poll_interval_sec: 300
`

	err := os.WriteFile(actionsPath, []byte(actionsContent), 0644)
	require.NoError(t, err)

	actions, err := config.LoadActions(actionsPath)
	require.NoError(t, err)

	action := actions.Actions[0]
	assert.Equal(t, "git", action.SourceType)
	assert.NotNil(t, action.GitOptions)
	assert.Equal(t, "git@github.com:org/repo.git", action.GitOptions.URL)
	assert.Equal(t, "/etc/ssh/id_rsa", action.GitOptions.PrivateKeyPath)
	assert.Equal(t, "main", action.GitOptions.Branch)
	assert.Equal(t, 300, action.GitOptions.PollIntervalSec)
}

func TestLoadActions_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	actionsPath := filepath.Join(tmpDir, "actions.yml")

	invalidContent := `
on:
  alert.created:
    invalid yaml {{
`

	err := os.WriteFile(actionsPath, []byte(invalidContent), 0644)
	require.NoError(t, err)

	_, err = config.LoadActions(actionsPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
}

func TestLoadActions_EmptyActions(t *testing.T) {
	tmpDir := t.TempDir()
	actionsPath := filepath.Join(tmpDir, "actions.yml")

	emptyContent := `
on: {}
`

	err := os.WriteFile(actionsPath, []byte(emptyContent), 0644)
	require.NoError(t, err)

	_, err = config.LoadActions(actionsPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one action")
}
