package config

// Config represents the main configuration file structure
type Config struct {
	Logging  LoggingConfig  `yaml:"logging"`
	Poller   PollerConfig   `yaml:"poller"`
	Pool     PoolConfig     `yaml:"pool"`
	Rootly   RootlyConfig   `yaml:"rootly"`
	App      AppConfig      `yaml:"app"`
	Metrics  MetricsConfig  `yaml:"metrics"`
	Security SecurityConfig `yaml:"security"`
}

// AppConfig contains application metadata
type AppConfig struct {
	Name string `yaml:"name"`
}

// RootlyConfig contains Rootly API configuration
type RootlyConfig struct {
	APIURL  string `yaml:"api_url"`  // API base URL (default: https://rec.rootly.com, staging: https://staging.rootly-ops.com)
	APIPath string `yaml:"api_path"` // API path prefix (default: /v1)
	APIKey  string `yaml:"api_key"`  // API key token (format: rec_xxxxx)
}

// PollerConfig contains polling engine configuration
type PollerConfig struct {
	RetryBackoff          string `yaml:"retry_backoff"` // exponential or linear
	PollingWaitIntervalMs int    `yaml:"polling_wait_interval_ms"`
	VisibilityTimeoutSec  int    `yaml:"visibility_timeout_sec"`
	MaxNumberOfMessages   int    `yaml:"max_number_of_messages"`
	MaxRetries            int    `yaml:"max_retries"`
	RetryOnError          bool   `yaml:"retry_on_error"`
}

// PoolConfig contains worker pool configuration
type PoolConfig struct {
	MaxNumberOfWorkers int `yaml:"max_number_of_workers"`
	MinNumberOfWorkers int `yaml:"min_number_of_workers"`
	QueueSize          int `yaml:"queue_size"`
	KeepAliveTimeMs    int `yaml:"keep_alive_time_ms"`
	MonitoringPeriodMs int `yaml:"monitoring_period_ms"`
}

// SecurityConfig contains security and script execution settings
type SecurityConfig struct {
	GlobalEnv          map[string]string `yaml:"global_env"`
	AllowedScriptPaths []string          `yaml:"allowed_script_paths"`
	ScriptTimeout      int               `yaml:"script_timeout"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`  // trace, debug, info, warn, error
	Format string `yaml:"format"` // json, text, colored
	Output string `yaml:"output"` // file path or stdout

	// Log rotation settings (using lumberjack)
	MaxSizeMB  int  `yaml:"max_size_mb"`  // Max size in megabytes before rotation (default: 100)
	MaxBackups int  `yaml:"max_backups"`  // Max number of old log files to retain (default: 3)
	MaxAgeDays int  `yaml:"max_age_days"` // Max days to retain old log files (default: 7)
	Compress   bool `yaml:"compress"`     // Compress rotated log files (default: true)
	LocalTime  bool `yaml:"local_time"`   // Use local time for log filenames (default: false, uses UTC)
}

// MetricsConfig contains Prometheus metrics configuration
type MetricsConfig struct {
	Path    string            `yaml:"path"`    // Path for metrics endpoint (default: /metrics)
	Port    int               `yaml:"port"`    // Port for metrics HTTP server (default: 9090)
	Enabled bool              `yaml:"enabled"` // Enable Prometheus metrics endpoint (default: true)
	Labels  map[string]string `yaml:"labels"`  // Optional custom labels for all metrics (e.g., connector_id, environment, region)
}

// ActionsConfig represents the actions configuration file structure
// New simplified format with on/callable sections
type ActionsConfig struct {
	On       map[string]OnAction       `yaml:"on"`       // Automatic actions (event type → action)
	Callable map[string]CallableAction `yaml:"callable"` // Callable actions (slug → action)
	Defaults ActionDefaults            `yaml:"defaults"` // Global defaults

	// Internal use: converted to unified array format after parsing
	Actions []Action `yaml:"-"`
}

// ActionDefaults contains global default values for all actions
type ActionDefaults struct {
	Timeout    int               `yaml:"timeout"`     // Default timeout (seconds)
	SourceType string            `yaml:"source_type"` // Default source type (local/git)
	Env        map[string]string `yaml:"env"`         // Default environment variables
}

// OnAction represents an automatic action (no UI, triggered by events)
type OnAction struct {
	Type       string            `yaml:"type"`        // "script" or "http" (default: script)
	SourceType string            `yaml:"source_type"` // "local" or "git" (default: local)
	Script     string            `yaml:"script"`      // Script path
	HTTP       *HTTPAction       `yaml:"http"`        // HTTP configuration
	GitOptions *GitOptions       `yaml:"git_options"` // Git options
	Parameters map[string]string `yaml:"parameters"`  // Template mappings
	Env        map[string]string `yaml:"env"`         // Environment variables
	Flags      map[string]string `yaml:"flags"`       // Command-line flags
	Args       []string          `yaml:"args"`        // Script arguments
	Timeout    int               `yaml:"timeout"`     // Timeout override
	Stdout     string            `yaml:"stdout"`      // Stdout redirect
	Stderr     string            `yaml:"stderr"`      // Stderr redirect
}

// CallableAction represents a user-triggered action (shows in UI)
type CallableAction struct {
	Name                 string                `yaml:"name"`                  // Display name (required)
	Description          string                `yaml:"description"`           // Description for UI
	Type                 string                `yaml:"type"`                  // "script" or "http" (default: script)
	SourceType           string                `yaml:"source_type"`           // "local" or "git" (default: local)
	Script               string                `yaml:"script"`                // Script path
	HTTP                 *HTTPAction           `yaml:"http"`                  // HTTP configuration
	GitOptions           *GitOptions           `yaml:"git_options"`           // Git options
	ParameterDefinitions []ParameterDefinition `yaml:"parameter_definitions"` // UI form fields
	Parameters           map[string]string     `yaml:"parameters"`            // Template mappings (auto-generated if not specified)
	Env                  map[string]string     `yaml:"env"`                   // Environment variables
	Flags                map[string]string     `yaml:"flags"`                 // Command-line flags
	Args                 []string              `yaml:"args"`                  // Script arguments
	Trigger              string                `yaml:"trigger"`               // Event type (default: action.triggered)
	Timeout              int                   `yaml:"timeout"`               // Timeout override
	Stdout               string                `yaml:"stdout"`                // Stdout redirect
	Stderr               string                `yaml:"stderr"`                // Stderr redirect
	Auth                 Authorization         `yaml:"authorization"`         // Authorization rules
}

// ParameterDefinition represents a parameter definition for callable actions
type ParameterDefinition struct {
	Name        string      `yaml:"name" json:"name"`                                   // Parameter name
	Type        string      `yaml:"type" json:"type"`                                   // "string", "number", "boolean", "list"
	Required    bool        `yaml:"required,omitempty" json:"required,omitempty"`       // Whether parameter is required
	Description string      `yaml:"description,omitempty" json:"description,omitempty"` // Parameter description
	Default     interface{} `yaml:"default,omitempty" json:"default,omitempty"`         // Default value
	Options     []string    `yaml:"options,omitempty" json:"options,omitempty"`         // Valid options (required for "list" type, not allowed for other types)
}

// Action represents a single action configuration
type Action struct {
	HTTP                 *HTTPAction           `yaml:"http,omitempty"`
	GitOptions           *GitOptions           `yaml:"git_options,omitempty"`
	ParameterDefinitions []ParameterDefinition `yaml:"parameter_definitions,omitempty"` // For callable actions (UI metadata)
	Parameters           map[string]string     `yaml:"parameters"`                      // Template mappings (execution time)
	Env                  map[string]string     `yaml:"env"`                             // Environment variables
	Flags                map[string]string     `yaml:"flags"`                           // Command-line flags (e.g., --verbose, --config=value)
	Args                 []string              `yaml:"args"`                            // Script arguments
	ID                   string                `yaml:"id"`                              // REQUIRED: Machine identifier for lookups (e.g., "send_webhook")
	Name                 string                `yaml:"name,omitempty"`                  // OPTIONAL: Human-readable name for UI (e.g., "Send Webhook")
	Description          string                `yaml:"description,omitempty"`           // OPTIONAL: Multi-line description for UI
	Type                 string                `yaml:"type"`                            // "script", "http" (default: "script")
	SourceType           string                `yaml:"source_type"`                     // "local", "git" (default: "local")
	Script               string                `yaml:"script"`                          // Path to script (local or relative to git repo)
	Stdout               string                `yaml:"stdout"`
	Stderr               string                `yaml:"stderr"`
	Timeout              int                   `yaml:"timeout"`
	Trigger              TriggerConfig         `yaml:"trigger"`
	Auth                 Authorization         `yaml:"authorization"`
}

// HTTPAction represents HTTP action configuration
type HTTPAction struct {
	URL     string            `yaml:"url"`
	Method  string            `yaml:"method"`  // GET, POST, PUT, PATCH, DELETE (default: POST)
	Headers map[string]string `yaml:"headers"` // HTTP headers
	Params  map[string]string `yaml:"params"`  // Query parameters
	Body    string            `yaml:"body"`    // Request body template
}

// GitOptions represents git repository configuration
type GitOptions struct {
	URL             string `yaml:"url"`               // Git repository URL
	PrivateKeyPath  string `yaml:"private_key_path"`  // Path to SSH private key
	Passphrase      string `yaml:"passphrase"`        // Passphrase for private key
	Branch          string `yaml:"branch"`            // Branch to checkout (default: "main")
	PollIntervalSec int    `yaml:"poll_interval_sec"` // How often to pull updates (default: 300)
}

// TriggerConfig represents event trigger configuration
// Supports both single trigger (legacy) and multiple triggers (new)
type TriggerConfig struct {
	EventType  string   `yaml:"event_type"`  // Single event type (legacy, deprecated)
	EventTypes []string `yaml:"event_types"` // Multiple event types (new)
	ActionName string   `yaml:"action_name"` // Optional: Filter by action name (for action_triggered events)
}

// GetEventTypes returns all configured event types
// Handles both single event_type (legacy) and event_types (multiple) fields
func (t TriggerConfig) GetEventTypes() []string {
	// If event_types is set, use it (new format)
	if len(t.EventTypes) > 0 {
		return t.EventTypes
	}

	// Fall back to single event_type (legacy format)
	if t.EventType != "" {
		return []string{t.EventType}
	}

	return []string{}
}

// Authorization represents authorization configuration
type Authorization struct {
	AllowedTeams         []string `yaml:"allowed_teams"`
	RequiresIncidentRole []string `yaml:"requires_incident_role"`
	RequiresApproval     bool     `yaml:"requires_approval"`
}

// ConvertToActions converts the new on/callable format to internal Action array
func (cfg *ActionsConfig) ConvertToActions() {
	actions := make([]Action, 0, len(cfg.On)+len(cfg.Callable))

	// Convert "on" actions (automatic triggers)
	for eventType, onAction := range cfg.On {
		action := onActionToAction(eventType, onAction, &cfg.Defaults)
		actions = append(actions, action)
	}

	// Convert "callable" actions (user-triggered)
	for slug, callableAction := range cfg.Callable {
		action := callableActionToAction(slug, callableAction, &cfg.Defaults)
		actions = append(actions, action)
	}

	cfg.Actions = actions
}

// onActionToAction converts an OnAction to the internal Action format
func onActionToAction(eventType string, on OnAction, defaults *ActionDefaults) Action {
	// Auto-detect type from configuration
	actionType := on.Type
	if actionType == "" {
		if on.HTTP != nil {
			actionType = defaultActionTypeHTTP
		} else {
			actionType = defaultActionType
		}
	}

	action := Action{
		ID:          eventType, // Use event type as ID for on actions
		Name:        "",        // No name for automatic actions
		Description: "",
		Type:        actionType,
		SourceType:  getOrDefault(on.SourceType, getOrDefault(defaults.SourceType, "local")),
		Script:      on.Script,
		HTTP:        on.HTTP,
		GitOptions:  on.GitOptions,
		Parameters:  on.Parameters,
		Env:         mergeEnv(defaults.Env, on.Env),
		Flags:       on.Flags,
		Args:        on.Args,
		Timeout:     getTimeoutOrDefault(on.Timeout, defaults.Timeout, 30),
		Stdout:      on.Stdout,
		Stderr:      on.Stderr,
		Trigger: TriggerConfig{
			EventType: eventType,
		},
	}
	return action
}

// callableActionToAction converts a CallableAction to the internal Action format
func callableActionToAction(slug string, callable CallableAction, defaults *ActionDefaults) Action {
	// Determine event type from trigger (default: action.triggered)
	eventType := getOrDefault(callable.Trigger, "action.triggered")

	// Auto-detect type from configuration
	actionType := callable.Type
	if actionType == "" {
		if callable.HTTP != nil {
			actionType = defaultActionTypeHTTP
		} else {
			actionType = defaultActionType
		}
	}

	// Always auto-generate parameters from parameter_definitions, then merge with manual parameters
	// This allows users to only specify EXTRA parameters, not repeat the auto-generated ones
	parameters := make(map[string]string)
	if len(callable.ParameterDefinitions) > 0 {
		parameters = autoGenerateParameters(callable.ParameterDefinitions)
	}
	// Merge manual parameters (manual ones can override or add extras)
	for k, v := range callable.Parameters {
		parameters[k] = v
	}

	action := Action{
		ID:                   slug,
		Name:                 callable.Name,
		Description:          callable.Description,
		Type:                 actionType,
		SourceType:           getOrDefault(callable.SourceType, getOrDefault(defaults.SourceType, "local")),
		Script:               callable.Script,
		HTTP:                 callable.HTTP,
		GitOptions:           callable.GitOptions,
		ParameterDefinitions: callable.ParameterDefinitions,
		Parameters:           parameters,
		Env:                  mergeEnv(defaults.Env, callable.Env),
		Flags:                callable.Flags,
		Args:                 callable.Args,
		Timeout:              getTimeoutOrDefault(callable.Timeout, defaults.Timeout, 30),
		Stdout:               callable.Stdout,
		Stderr:               callable.Stderr,
		Auth:                 callable.Auth,
		Trigger: TriggerConfig{
			EventType: eventType,
		},
	}
	return action
}

// Helper functions

func getOrDefault(value, defaultValue string) string {
	if value != "" {
		return value
	}
	return defaultValue
}

func getTimeoutOrDefault(value, defaultValue, fallback int) int {
	if value > 0 {
		return value
	}
	if defaultValue > 0 {
		return defaultValue
	}
	return fallback
}

func mergeEnv(global, local map[string]string) map[string]string {
	if global == nil && local == nil {
		return nil
	}
	result := make(map[string]string)
	for k, v := range global {
		result[k] = v
	}
	for k, v := range local {
		result[k] = v // Local overrides global
	}
	return result
}

func autoGenerateParameters(paramDefs []ParameterDefinition) map[string]string {
	params := make(map[string]string)
	for _, def := range paramDefs {
		// Auto-map: parameter_name → "{{ parameters.parameter_name }}"
		params[def.Name] = "{{ parameters." + def.Name + " }}"
	}
	return params
}
