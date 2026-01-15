package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gosimple/slug"
	"github.com/xeipuuv/gojsonschema"
)

//go:embed parameters_schema.json
var parametersSchemaJSON string

// Validate validates the main configuration
func Validate(cfg *Config) error {
	// Validate App config
	if cfg.App.Name == "" {
		return fmt.Errorf("app.name is required")
	}

	// Validate Rootly config
	if cfg.Rootly.APIURL == "" {
		return fmt.Errorf("rootly.api_url is required")
	}
	if _, err := url.Parse(cfg.Rootly.APIURL); err != nil {
		return fmt.Errorf("rootly.api_url is invalid: %w", err)
	}
	if cfg.Rootly.APIKey == "" {
		return fmt.Errorf("rootly.api_key is required (can be set via REC_API_KEY env var)")
	}

	// Validate Poller config
	if cfg.Poller.PollingWaitIntervalMs < 1000 {
		return fmt.Errorf("poller.polling_wait_interval_ms must be at least 1000")
	}
	if cfg.Poller.VisibilityTimeoutSec < 1 {
		return fmt.Errorf("poller.visibility_timeout_sec must be at least 1")
	}
	if cfg.Poller.MaxNumberOfMessages < 1 || cfg.Poller.MaxNumberOfMessages > 100 {
		return fmt.Errorf("poller.max_number_of_messages must be between 1 and 100")
	}
	if cfg.Poller.RetryBackoff != "exponential" && cfg.Poller.RetryBackoff != "linear" {
		return fmt.Errorf("poller.retry_backoff must be 'exponential' or 'linear'")
	}

	// Validate Pool config
	if cfg.Pool.MaxNumberOfWorkers < 1 {
		return fmt.Errorf("pool.max_number_of_workers must be at least 1")
	}
	if cfg.Pool.MinNumberOfWorkers < 1 {
		return fmt.Errorf("pool.min_number_of_workers must be at least 1")
	}
	if cfg.Pool.MinNumberOfWorkers > cfg.Pool.MaxNumberOfWorkers {
		return fmt.Errorf("pool.min_number_of_workers cannot exceed pool.max_number_of_workers")
	}
	if cfg.Pool.QueueSize < 1 {
		return fmt.Errorf("pool.queue_size must be at least 1")
	}

	// Validate Security config
	if cfg.Security.ScriptTimeout < 1 {
		return fmt.Errorf("security.script_timeout must be at least 1")
	}

	// Validate Logging config
	validLevels := []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}
	if !contains(validLevels, cfg.Logging.Level) {
		return fmt.Errorf("logging.level must be one of: %s", strings.Join(validLevels, ", "))
	}
	validFormats := []string{"json", "text", "colored"}
	if !contains(validFormats, cfg.Logging.Format) {
		return fmt.Errorf("logging.format must be one of: %s", strings.Join(validFormats, ", "))
	}

	// Validate Metrics config
	if cfg.Metrics.Enabled {
		if cfg.Metrics.Port < 1 || cfg.Metrics.Port > 65535 {
			return fmt.Errorf("metrics.port must be between 1 and 65535")
		}
		if !strings.HasPrefix(cfg.Metrics.Path, "/") {
			return fmt.Errorf("metrics.path must start with /")
		}
	}

	return nil
}

// ValidateActions validates the actions configuration
func ValidateActions(cfg *ActionsConfig) error {
	if len(cfg.Actions) == 0 {
		return fmt.Errorf("at least one action must be defined")
	}

	actionIDs := make(map[string]bool)
	for i, action := range cfg.Actions {
		// Check for duplicate action IDs (IDs must be unique)
		if actionIDs[action.ID] {
			return fmt.Errorf("duplicate action id: %s", action.ID)
		}
		actionIDs[action.ID] = true

		// Validate action
		if err := validateAction(&action); err != nil {
			return fmt.Errorf("action[%d] (%s): %w", i, action.ID, err)
		}
	}

	return nil
}

// ValidateActionsConfig validates the on/callable sections before conversion
// This ensures trigger patterns match their section (callable vs automatic)
func ValidateActionsConfig(cfg *ActionsConfig) error {
	// Validate "on" section (automatic actions)
	for eventType := range cfg.On {
		if isCallableTriggerPattern(eventType) {
			return fmt.Errorf("on.%s: automatic actions cannot use callable trigger patterns (action.triggered or *.action_triggered)", eventType)
		}
	}

	// Validate "callable" section (callable actions)
	for slug, callableAction := range cfg.Callable {
		// Get trigger (defaults to action.triggered if not specified)
		trigger := callableAction.Trigger
		if trigger == "" {
			trigger = "action.triggered"
		}

		// Check if trigger matches callable pattern
		if !isCallableTriggerPattern(trigger) {
			return fmt.Errorf("callable.%s: callable actions must use callable trigger patterns (action.triggered or *.action_triggered), got: %s", slug, trigger)
		}

		// Callable actions must have a name for UI display
		if callableAction.Name == "" {
			return fmt.Errorf("callable.%s: name is required for callable actions (displayed in UI)", slug)
		}
	}

	return nil
}

// isCallableTriggerPattern checks if a trigger pattern indicates a callable action
// Matches: "action.triggered", "alert.action_triggered", "incident.action_triggered"
func isCallableTriggerPattern(trigger string) bool {
	return trigger == "action.triggered" || strings.HasSuffix(trigger, ".action_triggered")
}

// validateAction validates a single action
func validateAction(action *Action) error {
	// ID is required (machine identifier)
	if action.ID == "" {
		return fmt.Errorf("id is required")
	}

	// ID must be valid: lowercase alphanumeric with underscores/hyphens/dots
	// Allows: restart_server, send-webhook, clear_cache_v2, alert.created (for on: actions)
	if !regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]*[a-z0-9]$|^[a-z0-9]$`).MatchString(action.ID) {
		return fmt.Errorf("id must be lowercase alphanumeric with underscores/hyphens/dots: %s", action.ID)
	}

	// Name is optional (human-readable display name)

	// Validate type
	if action.Type != "script" && action.Type != "http" {
		return fmt.Errorf("type must be 'script' or 'http'")
	}

	// Validate trigger - must have either event_type or event_types
	eventTypes := action.Trigger.GetEventTypes()
	if len(eventTypes) == 0 {
		return fmt.Errorf("trigger.event_type or trigger.event_types is required")
	}

	// Validate trigger compatibility with action classification
	// Actions are classified as callable (user-initiated) or automatic (event-triggered) based on presence of Name field
	for _, eventType := range eventTypes {
		if action.Name != "" {
			// Action has Name field = callable intent
			// Must use callable trigger patterns: action.triggered, alert.action_triggered, incident.action_triggered
			if !isCallableTriggerPattern(eventType) {
				return fmt.Errorf("action '%s' has a name field (indicating callable intent) but uses trigger '%s' which is for automatic actions. Callable actions must use triggers: action.triggered, alert.action_triggered, or incident.action_triggered", action.ID, eventType)
			}
		} else {
			// Action has no Name field = automatic intent
			// Must NOT use callable trigger patterns
			if isCallableTriggerPattern(eventType) {
				return fmt.Errorf("action '%s' has no name field (indicating automatic intent) but uses callable trigger '%s'; automatic actions must use event triggers like alert.created or incident.updated", action.ID, eventType)
			}
		}
	}

	// Validate script action
	if action.Type == "script" {
		// Validate source type (only for script actions)
		if action.SourceType != "local" && action.SourceType != "git" {
			return fmt.Errorf("source_type must be 'local' or 'git'")
		}
		if action.Script == "" {
			return fmt.Errorf("script is required for script actions")
		}
		if action.SourceType == "local" {
			// Check if script file exists
			if !filepath.IsAbs(action.Script) {
				return fmt.Errorf("script path must be absolute for local scripts")
			}
			if _, err := os.Stat(action.Script); os.IsNotExist(err) {
				return fmt.Errorf("script file does not exist: %s", action.Script)
			}
		}
	}

	// Validate HTTP action
	if action.Type == "http" {
		if action.HTTP == nil {
			return fmt.Errorf("http configuration is required for http actions")
		}
		if action.HTTP.URL == "" {
			return fmt.Errorf("http.url is required")
		}
		if _, err := url.Parse(action.HTTP.URL); err != nil {
			return fmt.Errorf("http.url is invalid: %w", err)
		}
		validMethods := []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
		if !contains(validMethods, action.HTTP.Method) {
			return fmt.Errorf("http.method must be one of: %s", strings.Join(validMethods, ", "))
		}
	}

	// Validate git options
	if action.SourceType == "git" {
		if action.GitOptions == nil {
			return fmt.Errorf("git_options is required for git source type")
		}
		if action.GitOptions.URL == "" {
			return fmt.Errorf("git_options.url is required")
		}
		if action.GitOptions.PollIntervalSec < 1 {
			return fmt.Errorf("git_options.poll_interval_sec must be at least 1")
		}
	}

	// Validate timeout
	if action.Timeout < 1 {
		return fmt.Errorf("timeout must be at least 1")
	}

	// Validate parameter definitions
	if err := validateParameterDefinitions(action.ParameterDefinitions); err != nil {
		return fmt.Errorf("parameter_definitions: %w", err)
	}

	return nil
}

// validateParameterDefinitions validates parameter definitions against the backend's JSON Schema
// This ensures compatibility with the backend's validation rules:
// - All parameters must have name (non-empty) and type
// - Type must be one of: "string", "number", "boolean", "list"
// - List type MUST have options (with at least 1 item)
// - Non-list types MUST NOT have options
// - No additional properties allowed
func validateParameterDefinitions(params []ParameterDefinition) error {
	if len(params) == 0 {
		return nil // Empty parameter definitions are valid
	}

	// Use embedded JSON Schema matching backend's PARAMETER_SCHEMA
	schemaJSON := parametersSchemaJSON

	// Convert params to JSON for validation
	paramJSON, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	// Create schema loader
	schemaLoader := gojsonschema.NewStringLoader(schemaJSON)
	documentLoader := gojsonschema.NewBytesLoader(paramJSON)

	// Validate
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}

	if !result.Valid() {
		return formatParameterValidationErrors(result.Errors())
	}

	// Additional business logic validation
	if err := validateParameterDefaults(params); err != nil {
		return err
	}

	return nil
}

// validateParameterDefaults validates that default values are valid for their parameter type
// For list type:
// - default value must be one of the options
// - options must not contain duplicates
func validateParameterDefaults(params []ParameterDefinition) error {
	for i, param := range params {
		// For list type, validate options and defaults
		if param.Type == "list" {
			// Validate no duplicate options
			seen := make(map[string]bool)
			var duplicates []string
			for _, option := range param.Options {
				if seen[option] {
					if !contains(duplicates, option) {
						duplicates = append(duplicates, option)
					}
				}
				seen[option] = true
			}

			if len(duplicates) > 0 {
				return fmt.Errorf("[%d] (name=%s): options must be unique (duplicates: %s)",
					i, param.Name, strings.Join(duplicates, ", "))
			}

			// Validate default is in options (if default is set)
			if param.Default != nil {
				// Convert default to string for comparison
				defaultStr, ok := param.Default.(string)
				if !ok {
					return fmt.Errorf("[%d] (name=%s): default value must be a string for list type", i, param.Name)
				}

				// Check if default is in options
				found := false
				for _, option := range param.Options {
					if option == defaultStr {
						found = true
						break
					}
				}

				if !found {
					return fmt.Errorf("[%d] (name=%s): default value '%s' must be one of the available options: %v",
						i, param.Name, defaultStr, param.Options)
				}
			}
		}
	}

	return nil
}

// formatParameterValidationErrors formats JSON Schema validation errors with helpful context
func formatParameterValidationErrors(errors []gojsonschema.ResultError) error {
	if len(errors) == 0 {
		return nil
	}

	// Build detailed error message
	errMsgs := make([]string, 0, len(errors))
	for _, err := range errors {
		// Get the field path and description
		field := err.Field()
		description := err.Description()

		// Format error message with field context
		errMsgs = append(errMsgs, fmt.Sprintf("%s: %s", field, description))
	}

	return fmt.Errorf("validation failed: %s", strings.Join(errMsgs, "; "))
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// NormalizeActionName normalizes an action name to snake_case
// Matches Rails parameterize(separator: "_") behavior
// Handles accents, unicode, special characters
func NormalizeActionName(name string) string {
	// Use slug package for proper unicode/accent handling
	// slug.Make() lowercases, removes accents, handles unicode, replaces special chars with hyphens
	normalized := slug.Make(name)

	// Convert hyphens to underscores for snake_case
	normalized = strings.ReplaceAll(normalized, "-", "_")

	return normalized
}
