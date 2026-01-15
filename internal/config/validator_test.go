package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rootly/edge-connector/internal/config"
)

func TestValidateAction_SingleEventType(t *testing.T) {
	// Test with single event_type (legacy format)
	action := &config.Action{
		ID:   "test_action",
		Type: "http",
		Trigger: config.TriggerConfig{
			EventType: "alert.created",
		},
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{
		Actions: []config.Action{*action},
	})

	// Should pass validation (event_type is set)
	assert.NoError(t, err)
}

func TestValidateAction_MultipleEventTypes(t *testing.T) {
	// Test with event_types (new format)
	action := &config.Action{
		ID:   "test_action",
		Type: "http",
		Trigger: config.TriggerConfig{
			EventTypes: []string{"alert.created", "incident.created"},
		},
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{
		Actions: []config.Action{*action},
	})

	// Should pass validation (event_types is set)
	assert.NoError(t, err)
}

func TestValidateAction_NoEventType(t *testing.T) {
	// Test with neither event_type nor event_types
	action := &config.Action{
		ID:      "test_action",
		Type:    "http",
		Trigger: config.TriggerConfig{},
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{
		Actions: []config.Action{*action},
	})

	// Should fail validation
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "trigger.event_type or trigger.event_types is required")
}

func TestValidateAction_BothEventTypeFormats(t *testing.T) {
	// Test with BOTH event_type and event_types (event_types should take precedence)
	action := &config.Action{
		ID:   "test_action",
		Type: "http",
		Trigger: config.TriggerConfig{
			EventType:  "alert.created",
			EventTypes: []string{"incident.created"},
		},
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{
		Actions: []config.Action{*action},
	})

	// Should pass validation (both are set, event_types takes precedence)
	assert.NoError(t, err)
}

func TestNormalizeActionName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"restart_server", "restart_server"},          // Already valid
		{"Restart Server", "restart_server"},          // Spaces
		{"restart-server", "restart_server"},          // Hyphens
		{"Restart-Server-123", "restart_server_123"},  // Mixed
		{"restart__server", "restart__server"},        // Multiple underscores OK
		{"restart.server", "restart_server"},          // Dots
		{"restart@server#123", "restartatserver_123"}, // @ becomes "at" (slug behavior)
		{"RESTART_SERVER", "restart_server"},          // Uppercase
		{"très|Jolie-- ", "tres_jolie"},               // Accents and special chars
		{"Café Münchën", "cafe_munchen"},              // Unicode accents
		{"hello   world", "hello_world"},              // Multiple spaces
	}

	for _, tt := range tests {
		// Test the normalization function directly
		result := config.NormalizeActionName(tt.input)
		assert.Equal(t, tt.expected, result, "Normalization of %s", tt.input)
	}
}

func TestLoadActions_OnFormat(t *testing.T) {
	// Load fixture with new on: format
	cfg, err := config.LoadActions("testdata/fixtures/action_with_id_and_name.yml")
	require.NoError(t, err)

	// Event type becomes the ID for on: actions
	assert.Equal(t, "alert.created", cfg.Actions[0].ID)
	assert.Equal(t, "", cfg.Actions[0].Name) // No name for automatic actions
	assert.Equal(t, "http", cfg.Actions[0].Type)
}

func TestValidateActions_DuplicateIDsYAMLOverwrite(t *testing.T) {
	// With map format, duplicate keys cause a YAML parse error (yaml.v3 behavior)
	// This tests that duplicate keys are caught at parse time
	_, err := config.LoadActions("testdata/fixtures/duplicate_ids.yml")

	// Should fail with YAML parse error for duplicate keys
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already defined")
}

func TestValidateActions_UniqueIDs(t *testing.T) {
	actions := []config.Action{
		{
			ID:   "action_one",
			Name: "Action One", // Names CAN be duplicate
			Type: "http",
			HTTP: &config.HTTPAction{
				URL:    "https://example.com",
				Method: "POST",
			},
			Trigger: config.TriggerConfig{
				EventType: "alert.created",
			},
			Timeout: 10,
		},
		{
			ID:   "action_two",
			Name: "Action One", // Same name is OK, id must be unique
			Type: "http",
			HTTP: &config.HTTPAction{
				URL:    "https://example.com",
				Method: "POST",
			},
			Trigger: config.TriggerConfig{
				EventType: "alert.created",
			},
			Timeout: 10,
		},
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: actions})

	assert.NoError(t, err)
}

// Comprehensive validation tests for Validate() function

// Helper to create a valid config with all defaults set
func validConfig() *config.Config {
	return &config.Config{
		App: config.AppConfig{
			Name: "test-app",
		},
		Rootly: config.RootlyConfig{
			APIURL: "https://api.rootly.com",
			APIKey: "test-key",
		},
		Poller: config.PollerConfig{
			PollingWaitIntervalMs: 5000,
			VisibilityTimeoutSec:  30,
			MaxNumberOfMessages:   10,
			RetryBackoff:          "exponential",
		},
		Pool: config.PoolConfig{
			MinNumberOfWorkers: 1,
			MaxNumberOfWorkers: 10,
			QueueSize:          100,
		},
		Security: config.SecurityConfig{
			ScriptTimeout: 300,
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		Metrics: config.MetricsConfig{
			Enabled: false,
			Port:    9090,
			Path:    "/metrics",
		},
	}
}

func TestValidate_MissingAppName(t *testing.T) {
	cfg := validConfig()
	cfg.App.Name = "" // Make invalid

	err := config.Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "app.name is required")
}

func TestValidate_MissingAPIURL(t *testing.T) {
	cfg := validConfig()
	cfg.Rootly.APIURL = ""

	err := config.Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rootly.api_url is required")
}

func TestValidate_InvalidAPIURL(t *testing.T) {
	cfg := validConfig()
	cfg.Rootly.APIURL = "://invalid url with spaces"

	err := config.Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rootly.api_url is invalid")
}

func TestValidate_MissingAPIKey(t *testing.T) {
	cfg := validConfig()
	cfg.Rootly.APIKey = ""

	err := config.Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rootly.api_key is required")
}

func TestValidate_PollingIntervalTooLow(t *testing.T) {
	cfg := validConfig()
	cfg.Poller.PollingWaitIntervalMs = 500 // Too low

	err := config.Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "polling_wait_interval_ms must be at least 1000")
}

func TestValidate_VisibilityTimeoutTooLow(t *testing.T) {
	cfg := validConfig()
	cfg.Poller.VisibilityTimeoutSec = 0 // Too low

	err := config.Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "visibility_timeout_sec must be at least 1")
}

func TestValidate_MaxMessagesTooLow(t *testing.T) {
	cfg := validConfig()
	cfg.Poller.MaxNumberOfMessages = 0 // Too low

	err := config.Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max_number_of_messages must be between 1 and 100")
}

func TestValidate_MaxMessagesTooHigh(t *testing.T) {
	cfg := validConfig()
	cfg.Poller.MaxNumberOfMessages = 101 // Too high

	err := config.Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max_number_of_messages must be between 1 and 100")
}

func TestValidate_InvalidRetryBackoff(t *testing.T) {
	cfg := validConfig()
	cfg.Poller.RetryBackoff = "invalid" // Invalid value

	err := config.Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retry_backoff must be 'exponential' or 'linear'")
}

func TestValidate_PoolMaxWorkersTooLow(t *testing.T) {
	cfg := validConfig()
	cfg.Pool.MaxNumberOfWorkers = 0 // Too low

	err := config.Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max_number_of_workers must be at least 1")
}

func TestValidate_PoolMinWorkersTooLow(t *testing.T) {
	cfg := validConfig()
	cfg.Pool.MinNumberOfWorkers = 0 // Too low

	err := config.Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "min_number_of_workers must be at least 1")
}

func TestValidate_MinWorkersExceedsMaxWorkers(t *testing.T) {
	cfg := validConfig()
	cfg.Pool.MinNumberOfWorkers = 10
	cfg.Pool.MaxNumberOfWorkers = 5 // Min > Max

	err := config.Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot exceed")
}

func TestValidate_QueueSizeTooLow(t *testing.T) {
	cfg := validConfig()
	cfg.Pool.QueueSize = 0 // Too low

	err := config.Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "queue_size must be at least 1")
}

func TestValidate_ScriptTimeoutTooLow(t *testing.T) {
	cfg := validConfig()
	cfg.Security.ScriptTimeout = 0 // Too low

	err := config.Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "script_timeout must be at least 1")
}

func TestValidate_InvalidLogLevel(t *testing.T) {
	cfg := validConfig()
	cfg.Logging.Level = "invalid" // Invalid level

	err := config.Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "logging.level must be one of")
}

func TestValidate_InvalidLogFormat(t *testing.T) {
	cfg := validConfig()
	cfg.Logging.Format = "invalid" // Invalid format

	err := config.Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "logging.format must be one of")
}

func TestValidate_MetricsPortTooLow(t *testing.T) {
	cfg := validConfig()
	cfg.Metrics.Enabled = true
	cfg.Metrics.Port = 0 // Too low

	err := config.Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metrics.port must be between 1 and 65535")
}

func TestValidate_MetricsPortTooHigh(t *testing.T) {
	cfg := validConfig()
	cfg.Metrics.Enabled = true
	cfg.Metrics.Port = 70000 // Too high

	err := config.Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metrics.port must be between 1 and 65535")
}

func TestValidate_MetricsPathInvalid(t *testing.T) {
	cfg := validConfig()
	cfg.Metrics.Enabled = true
	cfg.Metrics.Path = "metrics" // Missing leading /

	err := config.Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metrics.path must start with /")
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := validConfig()

	err := config.Validate(cfg)
	assert.NoError(t, err)
}

// Comprehensive validateAction tests

func TestValidateAction_MissingID(t *testing.T) {
	action := &config.Action{
		ID:   "", // Missing
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "alert.created",
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "id is required")
}

func TestValidateAction_InvalidIDFormat(t *testing.T) {
	tests := []struct {
		id      string
		wantErr bool
	}{
		{"valid_action", false},
		{"valid-action", false},
		{"valid123", false},
		{"valid.action", false},     // Dot now allowed (for event types like alert.created)
		{"alert.created", false},    // Event types are valid IDs
		{"Valid_Action", true},      // Uppercase
		{"valid action", true},      // Space
		{"_invalid", true},          // Leading underscore
		{"invalid_", true},          // Trailing underscore
		{"-invalid", true},          // Leading hyphen
		{"invalid-", true},          // Trailing hyphen
		{".invalid", true},          // Leading dot
		{"invalid.", true},          // Trailing dot
		{"valid_action_123", false}, // Valid
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			action := &config.Action{
				ID:   tt.id,
				Type: "http",
				HTTP: &config.HTTPAction{
					URL:    "https://example.com",
					Method: "POST",
				},
				Trigger: config.TriggerConfig{
					EventType: "alert.created",
				},
				Timeout: 10,
			}

			err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "id must be lowercase alphanumeric")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAction_InvalidType(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "invalid", // Invalid type
		Trigger: config.TriggerConfig{
			EventType: "alert.created",
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "type must be 'script' or 'http'")
}

func TestValidateAction_ScriptMissingScript(t *testing.T) {
	action := &config.Action{
		ID:         "test",
		Type:       "script",
		SourceType: "local",
		Script:     "", // Missing
		Trigger: config.TriggerConfig{
			EventType: "alert.created",
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "script is required for script actions")
}

func TestValidateAction_ScriptInvalidSourceType(t *testing.T) {
	action := &config.Action{
		ID:         "test",
		Type:       "script",
		SourceType: "invalid", // Invalid
		Script:     "/tmp/test.sh",
		Trigger: config.TriggerConfig{
			EventType: "alert.created",
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "source_type must be 'local' or 'git'")
}

func TestValidateAction_ScriptPathNotAbsolute(t *testing.T) {
	action := &config.Action{
		ID:         "test",
		Type:       "script",
		SourceType: "local",
		Script:     "relative/path.sh", // Not absolute
		Trigger: config.TriggerConfig{
			EventType: "alert.created",
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "script path must be absolute")
}

func TestValidateAction_ScriptFileNotExists(t *testing.T) {
	// Use a cross-platform absolute path that doesn't exist
	nonexistentPath := filepath.Join(t.TempDir(), "nonexistent", "script.sh")

	action := &config.Action{
		ID:         "test",
		Type:       "script",
		SourceType: "local",
		Script:     nonexistentPath, // Absolute path that doesn't exist
		Trigger: config.TriggerConfig{
			EventType: "alert.created",
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "script file does not exist")
}

func TestValidateAction_HTTPMissingConfig(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: nil, // Missing
		Trigger: config.TriggerConfig{
			EventType: "alert.created",
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "http configuration is required")
}

func TestValidateAction_HTTPMissingURL(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "", // Missing
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "alert.created",
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "http.url is required")
}

func TestValidateAction_HTTPInvalidURL(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "://invalid url", // Invalid
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "alert.created",
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "http.url is invalid")
}

func TestValidateAction_HTTPInvalidMethod(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "INVALID", // Invalid
		},
		Trigger: config.TriggerConfig{
			EventType: "alert.created",
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "http.method must be one of")
}

func TestValidateAction_GitMissingOptions(t *testing.T) {
	action := &config.Action{
		ID:         "test",
		Type:       "script",
		SourceType: "git",
		Script:     "scripts/test.sh",
		GitOptions: nil, // Missing
		Trigger: config.TriggerConfig{
			EventType: "alert.created",
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "git_options is required")
}

func TestValidateAction_GitMissingURL(t *testing.T) {
	action := &config.Action{
		ID:         "test",
		Type:       "script",
		SourceType: "git",
		Script:     "scripts/test.sh",
		GitOptions: &config.GitOptions{
			URL: "", // Missing
		},
		Trigger: config.TriggerConfig{
			EventType: "alert.created",
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "git_options.url is required")
}

func TestValidateAction_GitPollIntervalTooLow(t *testing.T) {
	action := &config.Action{
		ID:         "test",
		Type:       "script",
		SourceType: "git",
		Script:     "scripts/test.sh",
		GitOptions: &config.GitOptions{
			URL:             "https://github.com/example/repo",
			PollIntervalSec: 0, // Too low
		},
		Trigger: config.TriggerConfig{
			EventType: "alert.created",
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "poll_interval_sec must be at least 1")
}

func TestValidateAction_TimeoutTooLow(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "alert.created",
		},
		Timeout: 0, // Too low
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout must be at least 1")
}

func TestValidateActions_EmptyActions(t *testing.T) {
	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one action must be defined")
}

// Parameter Definitions Validation Tests
// These tests validate against the backend's JSON Schema for parameter_definitions

func TestParameterValidation_ValidStringType(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
		},
		ParameterDefinitions: []config.ParameterDefinition{
			{
				Name:        "message",
				Type:        "string",
				Required:    true,
				Description: "Message to send",
			},
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	assert.NoError(t, err)
}

func TestParameterValidation_ValidNumberType(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
		},
		ParameterDefinitions: []config.ParameterDefinition{
			{
				Name:        "count",
				Type:        "number",
				Required:    false,
				Description: "Number of items",
				Default:     10,
			},
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	assert.NoError(t, err)
}

func TestParameterValidation_ValidBooleanType(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
		},
		ParameterDefinitions: []config.ParameterDefinition{
			{
				Name:        "enabled",
				Type:        "boolean",
				Required:    false,
				Description: "Enable feature",
				Default:     true,
			},
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	assert.NoError(t, err)
}

func TestParameterValidation_ValidListType(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
		},
		ParameterDefinitions: []config.ParameterDefinition{
			{
				Name:        "environment",
				Type:        "list",
				Required:    true,
				Description: "Target environment",
				Options:     []string{"development", "staging", "production"},
			},
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	assert.NoError(t, err)
}

func TestParameterValidation_EmptyParameterDefinitions(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
		},
		ParameterDefinitions: []config.ParameterDefinition{}, // Empty is valid
		Timeout:              10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	assert.NoError(t, err)
}

func TestParameterValidation_MultipleParameters(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
		},
		ParameterDefinitions: []config.ParameterDefinition{
			{
				Name:        "message",
				Type:        "string",
				Required:    true,
				Description: "Message to send",
			},
			{
				Name:     "count",
				Type:     "number",
				Required: false,
				Default:  10,
			},
			{
				Name:    "enabled",
				Type:    "boolean",
				Default: true,
			},
			{
				Name:    "environment",
				Type:    "list",
				Options: []string{"dev", "prod"},
			},
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	assert.NoError(t, err)
}

func TestParameterValidation_MissingName(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
		},
		ParameterDefinitions: []config.ParameterDefinition{
			{
				Name: "", // Missing name
				Type: "string",
			},
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parameter_definitions")
}

func TestParameterValidation_MissingType(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
		},
		ParameterDefinitions: []config.ParameterDefinition{
			{
				Name: "message",
				Type: "", // Missing type
			},
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parameter_definitions")
}

func TestParameterValidation_InvalidType(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
		},
		ParameterDefinitions: []config.ParameterDefinition{
			{
				Name: "message",
				Type: "invalid_type", // Invalid type
			},
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parameter_definitions")
}

func TestParameterValidation_ListTypeMissingOptions(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
		},
		ParameterDefinitions: []config.ParameterDefinition{
			{
				Name:    "environment",
				Type:    "list",
				Options: nil, // List type MUST have options
			},
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parameter_definitions")
}

func TestParameterValidation_ListTypeEmptyOptions(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
		},
		ParameterDefinitions: []config.ParameterDefinition{
			{
				Name:    "environment",
				Type:    "list",
				Options: []string{}, // List type needs at least 1 option
			},
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parameter_definitions")
}

func TestParameterValidation_StringTypeWithOptions(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
		},
		ParameterDefinitions: []config.ParameterDefinition{
			{
				Name:    "message",
				Type:    "string",
				Options: []string{"option1", "option2"}, // String type CANNOT have options
			},
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parameter_definitions")
}

func TestParameterValidation_NumberTypeWithOptions(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
		},
		ParameterDefinitions: []config.ParameterDefinition{
			{
				Name:    "count",
				Type:    "number",
				Options: []string{"1", "2", "3"}, // Number type CANNOT have options
			},
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parameter_definitions")
}

func TestParameterValidation_BooleanTypeWithOptions(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
		},
		ParameterDefinitions: []config.ParameterDefinition{
			{
				Name:    "enabled",
				Type:    "boolean",
				Options: []string{"true", "false"}, // Boolean type CANNOT have options
			},
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parameter_definitions")
}

// Default Value Validation Tests

func TestParameterValidation_ListTypeDefaultInOptions(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
		},
		ParameterDefinitions: []config.ParameterDefinition{
			{
				Name:    "environment",
				Type:    "list",
				Default: "staging", // Valid - "staging" is in options
				Options: []string{"dev", "staging", "production"},
			},
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	assert.NoError(t, err)
}

func TestParameterValidation_ListTypeDefaultNotInOptions(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
		},
		ParameterDefinitions: []config.ParameterDefinition{
			{
				Name:    "environment",
				Type:    "list",
				Default: "invalid_value", // Invalid - not in options
				Options: []string{"dev", "staging", "production"},
			},
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parameter_definitions")
	assert.Contains(t, err.Error(), "default value 'invalid_value' must be one of the available options")
	assert.Contains(t, err.Error(), "environment")
}

func TestParameterValidation_ListTypeNoDefault(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
		},
		ParameterDefinitions: []config.ParameterDefinition{
			{
				Name:    "environment",
				Type:    "list",
				Options: []string{"dev", "staging", "production"},
				// No default - this is valid
			},
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	assert.NoError(t, err)
}

func TestParameterValidation_StringTypeWithDefault(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
		},
		ParameterDefinitions: []config.ParameterDefinition{
			{
				Name:    "message",
				Type:    "string",
				Default: "Hello World", // String can have any default
			},
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	assert.NoError(t, err)
}

func TestParameterValidation_NumberTypeWithDefault(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
		},
		ParameterDefinitions: []config.ParameterDefinition{
			{
				Name:    "count",
				Type:    "number",
				Default: 42, // Number can have any numeric default
			},
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	assert.NoError(t, err)
}

func TestParameterValidation_BooleanTypeWithDefault(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
		},
		ParameterDefinitions: []config.ParameterDefinition{
			{
				Name:    "enabled",
				Type:    "boolean",
				Default: true, // Boolean can have true/false default
			},
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	assert.NoError(t, err)
}

func TestParameterValidation_MultipleListsWithDefaults(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
		},
		ParameterDefinitions: []config.ParameterDefinition{
			{
				Name:    "environment",
				Type:    "list",
				Default: "staging",
				Options: []string{"dev", "staging", "production"},
			},
			{
				Name:    "severity",
				Type:    "list",
				Default: "warning",
				Options: []string{"info", "warning", "critical"},
			},
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	assert.NoError(t, err)
}

func TestParameterValidation_MultipleListsOneInvalidDefault(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
		},
		ParameterDefinitions: []config.ParameterDefinition{
			{
				Name:    "environment",
				Type:    "list",
				Default: "staging", // Valid
				Options: []string{"dev", "staging", "production"},
			},
			{
				Name:    "severity",
				Type:    "list",
				Default: "invalid", // Invalid
				Options: []string{"info", "warning", "critical"},
			},
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parameter_definitions")
	assert.Contains(t, err.Error(), "default value 'invalid' must be one of the available options")
	assert.Contains(t, err.Error(), "severity")
}

// Duplicate Options Validation Tests

func TestParameterValidation_ListTypeDuplicateOptions(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
		},
		ParameterDefinitions: []config.ParameterDefinition{
			{
				Name:    "environment",
				Type:    "list",
				Options: []string{"dev", "staging", "production", "dev", "staging"}, // Duplicates: dev, staging
			},
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parameter_definitions")
	assert.Contains(t, err.Error(), "options must be unique")
	assert.Contains(t, err.Error(), "environment")
	assert.Contains(t, err.Error(), "dev")
	assert.Contains(t, err.Error(), "staging")
}

func TestParameterValidation_ListTypeUniqueOptions(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
		},
		ParameterDefinitions: []config.ParameterDefinition{
			{
				Name:    "environment",
				Type:    "list",
				Options: []string{"dev", "staging", "production"}, // All unique
			},
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	assert.NoError(t, err)
}

func TestParameterValidation_ListTypeSingleDuplicate(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
		},
		ParameterDefinitions: []config.ParameterDefinition{
			{
				Name:    "severity",
				Type:    "list",
				Options: []string{"info", "warning", "critical", "info"}, // One duplicate: info
			},
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parameter_definitions")
	assert.Contains(t, err.Error(), "options must be unique")
	assert.Contains(t, err.Error(), "severity")
	assert.Contains(t, err.Error(), "info")
}

func TestParameterValidation_ListTypeTripleDuplicate(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
		},
		ParameterDefinitions: []config.ParameterDefinition{
			{
				Name:    "priority",
				Type:    "list",
				Options: []string{"low", "medium", "high", "low", "low"}, // "low" appears 3 times
			},
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parameter_definitions")
	assert.Contains(t, err.Error(), "options must be unique")
	assert.Contains(t, err.Error(), "priority")
	assert.Contains(t, err.Error(), "low")
}

func TestParameterValidation_ListTypeDuplicatesAndInvalidDefault(t *testing.T) {
	action := &config.Action{
		ID:   "test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "https://example.com",
			Method: "POST",
		},
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
		},
		ParameterDefinitions: []config.ParameterDefinition{
			{
				Name:    "environment",
				Type:    "list",
				Default: "invalid",
				Options: []string{"dev", "staging", "production", "dev"}, // Has duplicate
			},
		},
		Timeout: 10,
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parameter_definitions")
	// Should fail on duplicates first
	assert.Contains(t, err.Error(), "options must be unique")
}

// Callable Action Trigger Validation Tests

func TestValidateAction_CallableWithValidTriggers(t *testing.T) {
	tests := []struct {
		name    string
		trigger string
	}{
		{"standalone callable", "action.triggered"},
		{"alert callable", "alert.action_triggered"},
		{"incident callable", "incident.action_triggered"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := &config.Action{
				ID:         "test_action",
				Name:       "Test Action", // Has name = callable intent
				Type:       "script",
				Script:     filepath.Join(t.TempDir(), "test.sh"),
				SourceType: "local",
				Trigger: config.TriggerConfig{
					EventType: tt.trigger,
				},
				Timeout: 10,
			}

			// Create dummy script file
			err := createDummyScript(action.Script)
			require.NoError(t, err)

			err = config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
			assert.NoError(t, err, "Callable action with trigger %s should be valid", tt.trigger)
		})
	}
}

func TestValidateAction_CallableWithBadTriggers(t *testing.T) {
	tests := []struct {
		name        string
		trigger     string
		expectedErr string
	}{
		{
			name:        "callable with alert.created trigger",
			trigger:     "alert.created",
			expectedErr: "action 'test_action' has a name field (indicating callable intent) but uses trigger 'alert.created' which is for automatic actions",
		},
		{
			name:        "callable with incident.created trigger",
			trigger:     "incident.created",
			expectedErr: "action 'test_action' has a name field (indicating callable intent) but uses trigger 'incident.created' which is for automatic actions",
		},
		{
			name:        "callable with alert.updated trigger",
			trigger:     "alert.updated",
			expectedErr: "action 'test_action' has a name field (indicating callable intent) but uses trigger 'alert.updated' which is for automatic actions",
		},
		{
			name:        "callable with incident.updated trigger",
			trigger:     "incident.updated",
			expectedErr: "action 'test_action' has a name field (indicating callable intent) but uses trigger 'incident.updated' which is for automatic actions",
		},
		{
			name:        "callable with custom event trigger",
			trigger:     "custom.event.type",
			expectedErr: "action 'test_action' has a name field (indicating callable intent) but uses trigger 'custom.event.type' which is for automatic actions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := &config.Action{
				ID:         "test_action",
				Name:       "Test Action", // Has name = callable intent
				Type:       "script",
				Script:     filepath.Join(t.TempDir(), "test.sh"),
				SourceType: "local",
				Trigger: config.TriggerConfig{
					EventType: tt.trigger,
				},
				Timeout: 10,
			}

			// Create dummy script file
			err := createDummyScript(action.Script)
			require.NoError(t, err)

			err = config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestValidateAction_CallableWithZeroParameters(t *testing.T) {
	tests := []struct {
		name    string
		trigger string
		params  []config.ParameterDefinition
	}{
		{
			name:    "callable with no parameters (nil)",
			trigger: "action.triggered",
			params:  nil,
		},
		{
			name:    "callable with empty parameters",
			trigger: "alert.action_triggered",
			params:  []config.ParameterDefinition{},
		},
		{
			name:    "callable incident action with no parameters",
			trigger: "incident.action_triggered",
			params:  []config.ParameterDefinition{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := &config.Action{
				ID:         "test_action",
				Name:       "Test Action", // Has name = callable intent
				Type:       "script",
				Script:     filepath.Join(t.TempDir(), "test.sh"),
				SourceType: "local",
				Trigger: config.TriggerConfig{
					EventType: tt.trigger,
				},
				ParameterDefinitions: tt.params,
				Timeout:              10,
			}

			// Create dummy script file
			err := createDummyScript(action.Script)
			require.NoError(t, err)

			err = config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
			assert.NoError(t, err, "Callable action with 0 parameters should be valid")
		})
	}
}

func TestValidateAction_AutomaticWithoutName(t *testing.T) {
	// Automatic actions (without name) can use any event trigger
	tests := []struct {
		name    string
		trigger string
	}{
		{"automatic alert.created", "alert.created"},
		{"automatic alert.updated", "alert.updated"},
		{"automatic incident.created", "incident.created"},
		{"automatic incident.updated", "incident.updated"},
		{"automatic custom.event", "custom.event.type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := &config.Action{
				ID:         "test_action",
				Name:       "", // No name = automatic action
				Type:       "script",
				Script:     filepath.Join(t.TempDir(), "test.sh"),
				SourceType: "local",
				Trigger: config.TriggerConfig{
					EventType: tt.trigger,
				},
				Timeout: 10,
			}

			// Create dummy script file
			err := createDummyScript(action.Script)
			require.NoError(t, err)

			err = config.ValidateActions(&config.ActionsConfig{Actions: []config.Action{*action}})
			assert.NoError(t, err, "Automatic action (no name) with trigger %s should be valid", tt.trigger)
		})
	}
}

func TestValidateAction_MixedCallableAndAutomatic(t *testing.T) {
	tempDir := t.TempDir()
	script1 := filepath.Join(tempDir, "callable.sh")
	script2 := filepath.Join(tempDir, "automatic.sh")

	// Create dummy scripts
	require.NoError(t, createDummyScript(script1))
	require.NoError(t, createDummyScript(script2))

	actions := []config.Action{
		{
			ID:         "callable_action",
			Name:       "Callable Action", // Has name = callable
			Type:       "script",
			Script:     script1,
			SourceType: "local",
			Trigger: config.TriggerConfig{
				EventType: "action.triggered", // Valid callable trigger
			},
			ParameterDefinitions: []config.ParameterDefinition{},
			Timeout:              10,
		},
		{
			ID:         "automatic_action",
			Name:       "", // No name = automatic
			Type:       "script",
			Script:     script2,
			SourceType: "local",
			Trigger: config.TriggerConfig{
				EventType: "alert.created", // Valid automatic trigger
			},
			Timeout: 10,
		},
	}

	err := config.ValidateActions(&config.ActionsConfig{Actions: actions})
	assert.NoError(t, err, "Mix of callable and automatic actions should be valid")
}

// Helper function to create dummy script files for testing
func createDummyScript(path string) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	// Create empty file
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return nil
}
