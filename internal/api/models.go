package api

import "time"

// ActionMetadata represents action metadata in events
type ActionMetadata struct {
	ID   string `json:"id"`   // Action UUID in Rootly
	Name string `json:"name"` // Human-readable display name
	Slug string `json:"slug"` // Machine identifier (matches trigger.action_name)
}

// Event represents an event from the Rootly API
// Matches backend response format from GET /rec/v1/deliveries
type Event struct {
	Data      map[string]interface{} `json:"data"`       // Event payload
	Action    *ActionMetadata        `json:"action"`     // Action metadata (only for action_triggered events)
	Timestamp time.Time              `json:"timestamp"`  // ISO8601
	ID        string                 `json:"id"`         // Delivery UUID (unique per connector)
	EventID   string                 `json:"event_id"`   // Event UUID (shared across connectors)
	Type      string                 `json:"event_type"` // e.g., "alert.created", "incident.created", "action.triggered"
}

// EventsResponse represents the response from GET /rec/v1/deliveries
type EventsResponse struct {
	NextCursor *int64  `json:"next_cursor"` // Unix timestamp or null
	Events     []Event `json:"events"`      // Note: Still called "events" in response for backward compatibility
}

// ExecutionResult represents execution results to send to PATCH /rec/v1/deliveries/:id
// Field names match database columns exactly
type ExecutionResult struct {
	// Note: DeliveryID is NOT sent in JSON body - it's in the URL path
	DeliveryID          string `json:"-"`                               // Delivery UUID (used for URL path, not in body)
	ExecutionStatus     string `json:"execution_status"`                // "running", "completed", "failed" (required)
	CompletedAt         string `json:"completed_at,omitempty"`          // ISO8601 timestamp when completed (for completed status)
	FailedAt            string `json:"failed_at,omitempty"`             // ISO8601 timestamp when failed (for failed status)
	RunningAt           string `json:"running_at,omitempty"`            // ISO8601 timestamp when started running
	ExecutionStdout     string `json:"execution_stdout,omitempty"`      // Script stdout (optional, truncated to 10k chars)
	ExecutionStderr     string `json:"execution_stderr,omitempty"`      // Script stderr (optional, truncated to 10k chars)
	ExecutionError      string `json:"execution_error,omitempty"`       // Error message if failed (optional)
	ExecutionActionID   string `json:"execution_action_id,omitempty"`   // Action UUID from event.action.id (optional)
	ExecutionActionName string `json:"execution_action_name,omitempty"` // Action slug/identifier from config.id (e.g., "test_manual_action_http")
	ExecutionDurationMs int64  `json:"execution_duration_ms,omitempty"` // Execution duration in milliseconds (optional)
	ExecutionExitCode   int    `json:"execution_exit_code,omitempty"`   // Exit code (optional, 0 for success)
}

// ExecutionResponse represents the response from PATCH /rec/v1/deliveries/:id
type ExecutionResponse struct {
	Success bool `json:"success"`
}

// ActionParameter represents a parameter definition for an action
// Matches Rails PARAMETER_SCHEMA validation
type ActionParameter struct {
	Name        string      `json:"name"`                  // Required: Parameter name
	Type        string      `json:"type"`                  // Required: "string", "number", or "boolean"
	Required    bool        `json:"required,omitempty"`    // Optional: Whether parameter is required
	Description string      `json:"description,omitempty"` // Optional: Parameter description
	Default     interface{} `json:"default,omitempty"`     // Optional: Default value (any type)
	Options     []string    `json:"options,omitempty"`     // Optional: List of valid options
}

// HTTPActionRegistration represents HTTP configuration for action registration
type HTTPActionRegistration struct {
	URL     string            `json:"url,omitempty"`
	Method  string            `json:"method,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Params  map[string]string `json:"params,omitempty"`
	Body    string            `json:"body,omitempty"`
}

// TriggerRegistration represents trigger configuration for action registration
type TriggerRegistration struct {
	EventTypes []string `json:"event_types_trigger"` // Event types that trigger this action (always an array)
}

// ActionRegistration represents an action registration sent to the backend
// Backend categorizes as automatic or callable based on trigger pattern
type ActionRegistration struct {
	Slug        string            `json:"slug"`                  // Action slug (machine identifier)
	Name        string            `json:"name,omitempty"`        // Display name in UI (optional, backend humanizes if empty)
	Description string            `json:"description,omitempty"` // Description in UI (optional)
	ActionType  string            `json:"action_type"`           // "script" or "http"
	Trigger     string            `json:"trigger"`               // Event type (e.g., "action.triggered", "alert.created")
	Timeout     int               `json:"timeout"`               // Execution timeout
	Parameters  []ActionParameter `json:"parameters,omitempty"`  // UI form fields (optional, for callable actions)
}

// RegisterActionsRequest represents the request body for POST /rec/v1/actions
// Backend categorizes actions based on trigger patterns
type RegisterActionsRequest struct {
	Actions []ActionRegistration `json:"actions"`
}

// RegisterActionsResponse represents the response from POST /rec/v1/actions
type RegisterActionsResponse struct {
	Registered struct {
		Automatic int `json:"automatic"`
		Callable  int `json:"callable"`
		Total     int `json:"total"`
	} `json:"registered"`
	RegisteredActions struct {
		Automatic []string `json:"automatic"`
		Callable  []string `json:"callable"`
	} `json:"registered_actions"`
	Failed   int             `json:"failed"`
	Failures []ActionFailure `json:"failures"`
}

// ActionFailure represents a failed action registration
type ActionFailure struct {
	Slug   string `json:"slug"`
	Reason string `json:"reason"`
}
