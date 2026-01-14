package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const (
	defaultActionType       = "script"
	defaultActionTypeHTTP   = "http"
	defaultActionSourceType = "local"
	defaultHTTPMethod       = "POST"
	defaultGitBranch        = "main"
)

// Load loads and parses the main configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Apply environment variable overrides
	if apiURL := os.Getenv("REC_API_URL"); apiURL != "" {
		cfg.Rootly.APIURL = apiURL
	}

	if apiPath := os.Getenv("REC_API_PATH"); apiPath != "" {
		cfg.Rootly.APIPath = apiPath
	}

	if apiKey := os.Getenv("REC_API_KEY"); apiKey != "" {
		cfg.Rootly.APIKey = apiKey
	}

	if logFormat := os.Getenv("REC_LOG_FORMAT_TYPE"); logFormat != "" {
		cfg.Logging.Format = logFormat
	}

	if logLevel := os.Getenv("REC_LOG_LEVEL"); logLevel != "" {
		cfg.Logging.Level = logLevel
	}

	// Apply defaults
	applyDefaults(&cfg)

	// Validate configuration
	if err := Validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// LoadActions loads and parses the actions configuration file
// Supports new on/callable format
func LoadActions(path string) (*ActionsConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read actions file: %w", err)
	}

	var cfg ActionsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse actions: %w", err)
	}

	// Validate on/callable sections before conversion
	// This checks that trigger patterns match their section (callable vs automatic)
	if err := ValidateActionsConfig(&cfg); err != nil {
		return nil, fmt.Errorf("invalid actions config: %w", err)
	}

	// Convert on/callable format to internal Actions array
	cfg.ConvertToActions()

	// Apply defaults to actions
	for i := range cfg.Actions {
		applyActionDefaults(&cfg.Actions[i])
	}

	// Validate actions
	if err := ValidateActions(&cfg); err != nil {
		return nil, fmt.Errorf("invalid actions config: %w", err)
	}

	return &cfg, nil
}

// applyDefaults sets default values for configuration
func applyDefaults(cfg *Config) {
	// Rootly defaults
	if cfg.Rootly.APIPath == "" {
		cfg.Rootly.APIPath = "/v1"
	}

	// Poller defaults
	if cfg.Poller.PollingWaitIntervalMs == 0 {
		cfg.Poller.PollingWaitIntervalMs = 5000
	}
	if cfg.Poller.VisibilityTimeoutSec == 0 {
		cfg.Poller.VisibilityTimeoutSec = 30
	}
	if cfg.Poller.MaxNumberOfMessages == 0 {
		cfg.Poller.MaxNumberOfMessages = 10
	}
	if cfg.Poller.RetryBackoff == "" {
		cfg.Poller.RetryBackoff = "exponential"
	}
	if cfg.Poller.MaxRetries == 0 {
		cfg.Poller.MaxRetries = 3
	}

	// Pool defaults
	if cfg.Pool.MaxNumberOfWorkers == 0 {
		cfg.Pool.MaxNumberOfWorkers = 10
	}
	if cfg.Pool.MinNumberOfWorkers == 0 {
		cfg.Pool.MinNumberOfWorkers = 2
	}
	if cfg.Pool.QueueSize == 0 {
		cfg.Pool.QueueSize = 1000
	}
	if cfg.Pool.KeepAliveTimeMs == 0 {
		cfg.Pool.KeepAliveTimeMs = 60000
	}
	if cfg.Pool.MonitoringPeriodMs == 0 {
		cfg.Pool.MonitoringPeriodMs = 30000
	}

	// Security defaults
	if cfg.Security.ScriptTimeout == 0 {
		cfg.Security.ScriptTimeout = 300
	}

	// Logging defaults
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.Format == "" {
		cfg.Logging.Format = "text"
	}
	if cfg.Logging.Output == "" {
		cfg.Logging.Output = "stdout"
	}
	if cfg.Logging.MaxSizeMB == 0 {
		cfg.Logging.MaxSizeMB = 100
	}
	if cfg.Logging.MaxBackups == 0 {
		cfg.Logging.MaxBackups = 3
	}
	if cfg.Logging.MaxAgeDays == 0 {
		cfg.Logging.MaxAgeDays = 7
	}

	// Metrics defaults
	if cfg.Metrics.Port == 0 {
		cfg.Metrics.Port = 9090
	}
	if cfg.Metrics.Path == "" {
		cfg.Metrics.Path = "/metrics"
	}
}

// applyActionDefaults sets default values for an action
func applyActionDefaults(action *Action) {
	if action.Type == "" {
		action.Type = defaultActionType
	}
	if action.SourceType == "" {
		action.SourceType = defaultActionSourceType
	}
	if action.Timeout == 0 {
		action.Timeout = 300 // 5 minutes default
	}
	if action.HTTP != nil && action.HTTP.Method == "" {
		action.HTTP.Method = defaultHTTPMethod
	}
	if action.GitOptions != nil {
		if action.GitOptions.Branch == "" {
			action.GitOptions.Branch = defaultGitBranch
		}
		if action.GitOptions.PollIntervalSec == 0 {
			action.GitOptions.PollIntervalSec = 300
		}
	}
}
