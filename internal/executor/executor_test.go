package executor

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rootly/edge-connector/internal/api"
	"github.com/rootly/edge-connector/internal/config"
	"github.com/rootly/edge-connector/internal/reporter"
)

func TestMatchesAction_SingleTrigger(t *testing.T) {
	executor := &Executor{}

	action := &config.Action{
		Name: "test_action",
		Trigger: config.TriggerConfig{
			EventType: "alert.created",
		},
	}

	// Test matching event
	event := api.Event{
		Type: "alert.created",
		Data: map[string]interface{}{},
	}
	assert.True(t, executor.matchesAction(event, action))

	// Test non-matching event
	event.Type = "incident.created"
	assert.False(t, executor.matchesAction(event, action))
}

func TestMatchesAction_MultipleTriggers(t *testing.T) {
	executor := &Executor{}

	action := &config.Action{
		Name: "test_action",
		Trigger: config.TriggerConfig{
			EventTypes: []string{"alert.created", "incident.created", "alert.updated"},
		},
	}

	// Test all matching event types
	testCases := []struct {
		eventType string
		expected  bool
	}{
		{"alert.created", true},
		{"incident.created", true},
		{"alert.updated", true},
		{"incident.updated", false},
		{"alert.action_triggered", false},
	}

	for _, tc := range testCases {
		event := api.Event{
			Type: tc.eventType,
			Data: map[string]interface{}{},
		}
		assert.Equal(t, tc.expected, executor.matchesAction(event, action),
			"Event type %s should match: %v", tc.eventType, tc.expected)
	}
}

func TestMatchesAction_WithActionName(t *testing.T) {
	executor := &Executor{}

	action := &config.Action{
		Name: "test_action",
		Trigger: config.TriggerConfig{
			EventType:  "alert.action_triggered",
			ActionName: "deploy_hotfix",
		},
	}

	// Test matching event with correct action slug
	event := api.Event{
		Type: "alert.action_triggered",
		Action: &api.ActionMetadata{
			ID:   "action-uuid-123",
			Name: "Deploy Hotfix",
			Slug: "deploy_hotfix",
		},
		Data: map[string]interface{}{},
	}
	assert.True(t, executor.matchesAction(event, action))

	// Test non-matching action slug
	event.Action.Slug = "rollback"
	assert.False(t, executor.matchesAction(event, action))

	// Test missing action object
	event.Action = nil
	assert.False(t, executor.matchesAction(event, action))
}

func TestMatchesAction_ActionNameDefaultsToID(t *testing.T) {
	executor := &Executor{}

	action := &config.Action{
		ID:   "deploy_hotfix",
		Name: "Deploy Hotfix",
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
			// No ActionName specified - should default to ID
		},
	}

	// Test that action matches when event.action.slug equals action.ID
	event := api.Event{
		Type: "action.triggered",
		Action: &api.ActionMetadata{
			ID:   "action-uuid-123",
			Name: "Deploy Hotfix",
			Slug: "deploy_hotfix", // Matches action.ID
		},
		Data: map[string]interface{}{},
	}
	assert.True(t, executor.matchesAction(event, action))

	// Test non-matching slug
	event.Action.Slug = "other_action"
	assert.False(t, executor.matchesAction(event, action))
}

// REGRESSION TEST: action.triggered without action_name or Action metadata should not match
func TestMatchesAction_ActionTriggeredWithoutActionNameInEvent(t *testing.T) {
	executor := &Executor{}

	action := &config.Action{
		ID:   "my_action",
		Name: "My Action",
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
			// ActionName not specified - defaults to action.ID
		},
	}

	// Event WITHOUT Action metadata AND WITHOUT action_name in data
	// This is a malformed event - backend should send at least one
	event := api.Event{
		Type:   "action.triggered",
		Action: nil, // No Action metadata
		Data: map[string]interface{}{
			// MISSING: "action_name" field
			"parameters": map[string]interface{}{
				"message": "test",
			},
		},
	}

	// Should NOT match because we can't verify which action to execute
	// Without Action.Slug or data.action_name, we can't match the action
	result := executor.matchesAction(event, action)

	assert.False(t, result, "Should not match when neither Action metadata nor data.action_name is present")
}

func TestMatchesAction_LegacyCompatibility(t *testing.T) {
	executor := &Executor{}

	// Test that single event_type still works (legacy format)
	action := &config.Action{
		Name: "test_action",
		Trigger: config.TriggerConfig{
			EventType: "alert.created",
		},
	}

	event := api.Event{
		Type: "alert.created",
		Data: map[string]interface{}{},
	}
	assert.True(t, executor.matchesAction(event, action))
}

func TestMatchesAction_MultipleTriggersPrecedence(t *testing.T) {
	executor := &Executor{}

	// Test that event_types takes precedence over event_type
	action := &config.Action{
		Name: "test_action",
		Trigger: config.TriggerConfig{
			EventType:  "alert.created",
			EventTypes: []string{"incident.created"},
		},
	}

	// Should match incident.created (from event_types)
	event := api.Event{
		Type: "incident.created",
		Data: map[string]interface{}{},
	}
	assert.True(t, executor.matchesAction(event, action))

	// Should NOT match alert.created (event_type is ignored when event_types is present)
	event.Type = "alert.created"
	assert.False(t, executor.matchesAction(event, action))
}

func TestPrepareParameters_UserInputPrecedence(t *testing.T) {
	executor := &Executor{}

	action := &config.Action{
		Name: "test_action",
		Parameters: map[string]string{
			"region":        "us-east-1", // Hardcoded default
			"user_provided": "{{ data.user_provided }}",
		},
		ParameterDefinitions: []config.ParameterDefinition{
			{Name: "region", Type: "string"},
			{Name: "user_provided", Type: "string"},
		},
	}

	event := api.Event{
		Data: map[string]interface{}{
			"region":        "eu-west-1", // User overrides hardcoded value
			"user_provided": "from_user",
		},
	}

	params := executor.prepareParameters(action, event)

	// User input should win
	assert.Equal(t, "eu-west-1", params["region"], "User input should override hardcoded value")
	assert.Equal(t, "from_user", params["user_provided"])
}

func TestPrepareParameters_HardcodedWhenNoUserInput(t *testing.T) {
	executor := &Executor{}

	action := &config.Action{
		Name: "test_action",
		Parameters: map[string]string{
			"region":  "us-east-1", // Hardcoded
			"timeout": "30",        // Hardcoded
		},
	}

	event := api.Event{
		Data: map[string]interface{}{
			// User provides nothing
		},
	}

	params := executor.prepareParameters(action, event)

	// Hardcoded values should be used when user provides nothing
	assert.Equal(t, "us-east-1", params["region"])
	assert.Equal(t, "30", params["timeout"])
}

func TestPrepareParameters_MixedUserAndHardcoded(t *testing.T) {
	executor := &Executor{}

	action := &config.Action{
		Name: "test_action",
		ParameterDefinitions: []config.ParameterDefinition{
			{Name: "service_name", Type: "string"},
		},
		Parameters: map[string]string{
			"service_name": "{{ service_name }}", // From user
			"environment":  "production",         // Hardcoded (not in UI)
			"max_retries":  "3",                  // Hardcoded (not in UI)
		},
	}

	event := api.Event{
		Data: map[string]interface{}{
			"service_name": "api",
		},
	}

	params := executor.prepareParameters(action, event)

	// Should have both user-provided and hardcoded
	assert.Equal(t, "api", params["service_name"])
	assert.Equal(t, "production", params["environment"])
	assert.Equal(t, "3", params["max_retries"])
}

func TestPrepareParameters_NonStringValues(t *testing.T) {
	executor := &Executor{}

	action := &config.Action{
		Name: "test_action",
		ParameterDefinitions: []config.ParameterDefinition{
			{Name: "count", Type: "number"},
			{Name: "enabled", Type: "boolean"},
		},
		Parameters: map[string]string{
			"count":   "{{ count }}",
			"enabled": "{{ enabled }}",
		},
	}

	event := api.Event{
		Data: map[string]interface{}{
			"count":   123,    // Number (not string)
			"enabled": true,   // Boolean (not string)
			"name":    "test", // String (valid)
		},
	}

	params := executor.prepareParameters(action, event)

	// Non-string values should be skipped (only strings are supported)
	// Only the template substitution result matters
	assert.Contains(t, params, "count")
	assert.Contains(t, params, "enabled")
}

func TestPrepareParameters_EmptyData(t *testing.T) {
	executor := &Executor{}

	action := &config.Action{
		Name: "test_action",
		Parameters: map[string]string{
			"default_value": "hardcoded",
			"template":      "{{ missing }}",
		},
	}

	event := api.Event{
		Data: map[string]interface{}{}, // Empty data
	}

	params := executor.prepareParameters(action, event)

	// Hardcoded values should be preserved
	assert.Equal(t, "hardcoded", params["default_value"])
	// Template with missing field should be empty
	assert.Equal(t, "", params["template"])
}

func TestPrepareParameters_NilData(t *testing.T) {
	executor := &Executor{}

	action := &config.Action{
		Name: "test_action",
		Parameters: map[string]string{
			"default": "value",
		},
	}

	event := api.Event{
		Data: nil, // Nil data
	}

	// Should not panic
	assert.NotPanics(t, func() {
		params := executor.prepareParameters(action, event)
		assert.Equal(t, "value", params["default"])
	})
}

func TestPrepareParameters_ParameterDefinitionsOnly(t *testing.T) {
	executor := &Executor{}

	action := &config.Action{
		Name: "test_action",
		ParameterDefinitions: []config.ParameterDefinition{
			{Name: "user_input", Type: "string"},
			{Name: "description", Type: "string"},
		},
		Parameters: map[string]string{}, // No parameters in config
	}

	event := api.Event{
		Data: map[string]interface{}{
			"user_input":  "from_user",
			"description": "test description",
		},
	}

	params := executor.prepareParameters(action, event)

	// User-provided values should be included via parameter_definitions
	assert.Equal(t, "from_user", params["user_input"])
	assert.Equal(t, "test description", params["description"])
}

func TestExecute_NoMatchingAction(t *testing.T) {
	// Mock reporter to capture the failure report
	reportCalled := false
	var reportedResult reporter.ScriptResult
	var reportedDeliveryID string
	var reportedActionName string

	mockReporter := &mockReporter{
		reportFunc: func(ctx context.Context, deliveryID, actionName, actionUUID string, result reporter.ScriptResult) error {
			reportCalled = true
			reportedDeliveryID = deliveryID
			reportedActionName = actionName
			reportedResult = result
			return nil
		},
	}

	executor := &Executor{
		actions:  []config.Action{}, // No actions configured
		reporter: mockReporter,
	}

	event := api.Event{
		ID:      "delivery-123",
		EventID: "event-456",
		Type:    "action.triggered",
		Data:    map[string]interface{}{},
	}

	executor.Execute(context.Background(), event)

	// Should have reported a failure
	assert.True(t, reportCalled, "Reporter should be called")
	assert.Equal(t, "delivery-123", reportedDeliveryID)
	assert.Equal(t, "none", reportedActionName)
	assert.Equal(t, 1, reportedResult.ExitCode)
	assert.NotNil(t, reportedResult.Error)
	assert.Contains(t, reportedResult.Error.Error(), "no matching action found")
	assert.Contains(t, reportedResult.Stderr, "No action configured")
}

func TestExecute_NoMatchingActionWithActionName(t *testing.T) {
	var reportedActionName string

	mockReporter := &mockReporter{
		reportFunc: func(ctx context.Context, deliveryID, actionName, actionUUID string, result reporter.ScriptResult) error {
			reportedActionName = actionName
			return nil
		},
	}

	executor := &Executor{
		actions:  []config.Action{}, // No actions configured
		reporter: mockReporter,
	}

	// Event with action_name in data
	event := api.Event{
		ID:      "delivery-456",
		EventID: "event-789",
		Type:    "action.triggered",
		Data: map[string]interface{}{
			"action_name": "missing_action", // This action isn't configured
		},
	}

	executor.Execute(context.Background(), event)

	// Should report with the actual action_name from event
	assert.Equal(t, "missing_action", reportedActionName,
		"Should use action_name from event data when no action matches")
}

// Test matching via data.action_name (fallback when event.Action is null)
func TestMatchesAction_DataActionNameFallback(t *testing.T) {
	executor := &Executor{}

	action := &config.Action{
		ID:   "test_action",
		Name: "Test Action",
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
			// ActionName defaults to ID ("test_action")
		},
	}

	// Event with action_name in data (but no Action metadata)
	event := api.Event{
		Type:   "action.triggered",
		Action: nil, // Backend doesn't send Action object for action.triggered
		Data: map[string]interface{}{
			"action_name": "test_action", // Matches action.ID
			"parameters": map[string]interface{}{
				"message": "test",
			},
		},
	}

	// Should match via data.action_name fallback
	assert.True(t, executor.matchesAction(event, action),
		"Should match when data.action_name equals action.ID (fallback when Action is null)")
}

func TestMatchesAction_DataActionNameMismatch(t *testing.T) {
	executor := &Executor{}

	action := &config.Action{
		ID:   "test_action",
		Name: "Test Action",
		Trigger: config.TriggerConfig{
			EventType: "action.triggered",
		},
	}

	// Event with different action_name
	event := api.Event{
		Type:   "action.triggered",
		Action: nil,
		Data: map[string]interface{}{
			"action_name": "different_action", // Doesn't match
		},
	}

	// Should NOT match
	assert.False(t, executor.matchesAction(event, action),
		"Should not match when data.action_name doesn't equal expected action name")
}

func TestMatchesAction_DataActionNameWithExplicitFilter(t *testing.T) {
	executor := &Executor{}

	action := &config.Action{
		ID:   "my_action_v2",
		Name: "My Action V2",
		Trigger: config.TriggerConfig{
			EventType:  "action.triggered",
			ActionName: "my_action", // Explicit name (different from ID)
		},
	}

	// Event with matching action_name
	event := api.Event{
		Type:   "action.triggered",
		Action: nil,
		Data: map[string]interface{}{
			"action_name": "my_action", // Matches trigger.action_name
		},
	}

	// Should match
	assert.True(t, executor.matchesAction(event, action),
		"Should match when data.action_name equals trigger.action_name")

	// Event with action_name matching ID instead of trigger.action_name
	event2 := api.Event{
		Type:   "action.triggered",
		Action: nil,
		Data: map[string]interface{}{
			"action_name": "my_action_v2", // Matches ID but not trigger.action_name
		},
	}

	// Should NOT match (trigger.action_name takes precedence over ID)
	assert.False(t, executor.matchesAction(event2, action),
		"Should not match when data.action_name doesn't equal trigger.action_name")
}

// REGRESSION TEST: action.triggered without action_name should report as "none"
// This test verifies that if the backend regresses and stops sending action_name,
// the connector properly reports "none" instead of incorrectly matching an action
func TestExecute_ActionTriggeredMissingActionNameReportsUnknown(t *testing.T) {
	var reportedActionName string

	mockReporter := &mockReporter{
		reportFunc: func(ctx context.Context, deliveryID, actionName, actionUUID string, result reporter.ScriptResult) error {
			reportedActionName = actionName
			return nil
		},
	}

	// Configure action expecting action.triggered with specific action_name
	executor := &Executor{
		actions: []config.Action{
			{
				ID:   "my_action",
				Name: "My Action",
				Type: "script",
				Trigger: config.TriggerConfig{
					EventType:  "action.triggered",
					ActionName: "my_action", // Explicit action_name to match
				},
			},
		},
		scriptRunner: nil,
		httpExecutor: nil,
		reporter:     mockReporter,
	}

	// Event without Action metadata or action_name in data
	event := api.Event{
		ID:      "delivery-bug",
		EventID: "event-bug",
		Type:    "action.triggered",
		Action:  nil, // No action metadata
		Data: map[string]interface{}{
			// Missing action_name in data
			"parameters": map[string]interface{}{
				"test": "value",
			},
		},
	}

	executor.Execute(context.Background(), event)

	// Should report as "none" because action can't be matched
	assert.Equal(t, "none", reportedActionName,
		"Should report 'unknown' when action.triggered lacks action metadata - prevents incorrect matching")
}

// mockReporter is a mock implementation of the reporter interface
type mockReporter struct {
	reportFunc func(ctx context.Context, deliveryID, actionName, actionUUID string, result reporter.ScriptResult) error
}

func (m *mockReporter) Report(ctx context.Context, deliveryID, actionName, actionUUID string, result reporter.ScriptResult) error {
	if m.reportFunc != nil {
		return m.reportFunc(ctx, deliveryID, actionName, actionUUID, result)
	}
	return nil
}

// Helper to get absolute path to test fixtures
func getFixturePath(filename string) (scriptPath, allowedDir string, err error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", "", err
	}
	fixturesDir := filepath.Join(wd, "testdata/fixtures")
	scriptPath = filepath.Join(fixturesDir, filename)
	return scriptPath, fixturesDir, nil
}

func TestNew(t *testing.T) {
	actions := []config.Action{
		{Name: "test_action", Type: "script"},
	}
	scriptRunner := NewScriptRunner([]string{}, map[string]string{})
	httpExecutor := NewHTTPExecutor()
	reporter := &mockReporter{}

	executor := New(actions, scriptRunner, httpExecutor, reporter)

	assert.NotNil(t, executor)
	assert.Len(t, executor.actions, 1)
}

func TestFindMatchingAction(t *testing.T) {
	executor := &Executor{
		actions: []config.Action{
			{
				Name: "alert_action",
				Trigger: config.TriggerConfig{
					EventType: "alert.created",
				},
			},
			{
				Name: "incident_action",
				Trigger: config.TriggerConfig{
					EventType: "incident.created",
				},
			},
		},
	}

	// Test finding matching action
	event := api.Event{Type: "alert.created"}
	action := executor.findMatchingAction(event)
	assert.NotNil(t, action)
	assert.Equal(t, "alert_action", action.Name)

	// Test no matching action
	event = api.Event{Type: "unknown.event"}
	action = executor.findMatchingAction(event)
	assert.Nil(t, action)
}

func TestSubstituteTemplate(t *testing.T) {
	executor := &Executor{}

	tests := []struct {
		name     string
		template string
		event    api.Event
		expected string
	}{
		{
			name:     "simple field",
			template: "{{ severity }}",
			event: api.Event{
				Data: map[string]interface{}{
					"severity": "critical",
				},
			},
			expected: "critical",
		},
		{
			name:     "nested field",
			template: "{{ alert.host }}",
			event: api.Event{
				Data: map[string]interface{}{
					"alert": map[string]interface{}{
						"host": "server-01",
					},
				},
			},
			expected: "server-01",
		},
		{
			name:     "event prefix",
			template: "{{ event.title }}",
			event: api.Event{
				Data: map[string]interface{}{
					"title": "Test Alert",
				},
			},
			expected: "Test Alert",
		},
		{
			name:     "missing field",
			template: "{{ missing }}",
			event: api.Event{
				Data: map[string]interface{}{},
			},
			expected: "",
		},
		{
			name:     "multiple substitutions",
			template: "{{ severity }} - {{ title }}",
			event: api.Event{
				Data: map[string]interface{}{
					"severity": "high",
					"title":    "Database Down",
				},
			},
			expected: "high - Database Down",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.substituteTemplate(tt.template, tt.event)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSubstituteTemplate_ArrayAccess(t *testing.T) {
	executor := &Executor{}

	tests := []struct {
		name     string
		template string
		event    api.Event
		expected string
	}{
		{
			name:     "array index access",
			template: "{{ services[0].name }}",
			event: api.Event{
				Data: map[string]interface{}{
					"services": []interface{}{
						map[string]interface{}{"name": "Database"},
						map[string]interface{}{"name": "API"},
					},
				},
			},
			expected: "Database",
		},
		{
			name:     "array first helper",
			template: "{{ services.first.name }}",
			event: api.Event{
				Data: map[string]interface{}{
					"services": []interface{}{
						map[string]interface{}{"name": "Database"},
					},
				},
			},
			expected: "Database",
		},
		{
			name:     "array last helper",
			template: "{{ services.last.name }}",
			event: api.Event{
				Data: map[string]interface{}{
					"services": []interface{}{
						map[string]interface{}{"name": "Database"},
						map[string]interface{}{"name": "API Gateway"},
					},
				},
			},
			expected: "API Gateway",
		},
		{
			name:     "environment array access",
			template: "{{ environments[0].slug }}",
			event: api.Event{
				Data: map[string]interface{}{
					"environments": []interface{}{
						map[string]interface{}{"slug": "production"},
					},
				},
			},
			expected: "production",
		},
		{
			name:     "combined with simple field",
			template: "[{{ labels.severity }}] {{ summary }} - {{ services[0].name }}",
			event: api.Event{
				Data: map[string]interface{}{
					"summary": "High latency",
					"labels":  map[string]interface{}{"severity": "critical"},
					"services": []interface{}{
						map[string]interface{}{"name": "Database"},
					},
				},
			},
			expected: "[critical] High latency - Database",
		},
		{
			name:     "array out of bounds",
			template: "{{ services[99].name }}",
			event: api.Event{
				Data: map[string]interface{}{
					"services": []interface{}{
						map[string]interface{}{"name": "Database"},
					},
				},
			},
			expected: "", // Should return empty for out of bounds
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.substituteTemplate(tt.template, tt.event)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSubstituteTemplate_LiquidFilters(t *testing.T) {
	executor := &Executor{}

	// Test a few key filters to verify Liquid integration works
	// Full filter list: https://github.com/osteele/liquid/blob/main/filters/standard_filters.go
	tests := []struct {
		name     string
		template string
		event    api.Event
		expected string
	}{
		{
			name:     "map and join filters (most common)",
			template: `{{ services | map: "name" | join: ", " }}`,
			event: api.Event{
				Data: map[string]interface{}{
					"services": []interface{}{
						map[string]interface{}{"name": "Database"},
						map[string]interface{}{"name": "API"},
						map[string]interface{}{"name": "Cache"},
					},
				},
			},
			expected: "Database, API, Cache",
		},
		{
			name:     "default filter for missing values",
			template: "{{ missing | default: 'N/A' }}",
			event: api.Event{
				Data: map[string]interface{}{},
			},
			expected: "N/A",
		},
		{
			name:     "upcase filter",
			template: "{{ status | upcase }}",
			event: api.Event{
				Data: map[string]interface{}{"status": "open"},
			},
			expected: "OPEN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.substituteTemplate(tt.template, tt.event)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSubstituteTemplate_EnvironmentVariables(t *testing.T) {
	executor := &Executor{}

	// Set environment variables for testing
	os.Setenv("TEST_ENV_VAR", "env_value")
	os.Setenv("API_KEY", "secret_key_123")
	defer func() {
		os.Unsetenv("TEST_ENV_VAR")
		os.Unsetenv("API_KEY")
	}()

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "single env var",
			template: "{{ env.TEST_ENV_VAR }}",
			expected: "env_value",
		},
		{
			name:     "env var in string",
			template: "API key: {{ env.API_KEY }}",
			expected: "API key: secret_key_123",
		},
		{
			name:     "multiple env vars",
			template: "{{ env.TEST_ENV_VAR }}-{{ env.API_KEY }}",
			expected: "env_value-secret_key_123",
		},
		{
			name:     "missing env var",
			template: "{{ env.NONEXISTENT }}",
			expected: "",
		},
		{
			name:     "mixed env and data",
			template: "{{ env.TEST_ENV_VAR }}-{{ title }}",
			expected: "env_value-Test Title",
		},
	}

	event := api.Event{
		Data: map[string]interface{}{
			"title": "Test Title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.substituteTemplate(tt.template, event)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSubstituteTemplate_ComplexTemplates(t *testing.T) {
	executor := &Executor{}

	event := api.Event{
		Data: map[string]interface{}{
			"severity": "critical",
			"alert": map[string]interface{}{
				"title": "Server Down",
				"details": map[string]interface{}{
					"host":   "prod-server-01",
					"region": "us-west-2",
				},
			},
		},
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "multiple nested fields",
			template: "[{{ severity }}] {{ alert.title }} on {{ alert.details.host }}",
			expected: "[critical] Server Down on prod-server-01",
		},
		{
			name:     "all levels of nesting",
			template: "{{ severity }}/{{ alert.title }}/{{ alert.details.host }}/{{ alert.details.region }}",
			expected: "critical/Server Down/prod-server-01/us-west-2",
		},
		{
			name:     "repeated field",
			template: "{{ severity }} {{ severity }}",
			expected: "critical critical",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.substituteTemplate(tt.template, event)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetFieldValue(t *testing.T) {
	executor := &Executor{}

	data := map[string]interface{}{
		"simple": "value",
		"nested": map[string]interface{}{
			"field": "nested_value",
			"deep": map[string]interface{}{
				"field": "deep_value",
			},
		},
	}

	tests := []struct {
		name     string
		path     string
		expected interface{}
	}{
		{
			name:     "simple field",
			path:     "simple",
			expected: "value",
		},
		{
			name:     "nested field",
			path:     "nested.field",
			expected: "nested_value",
		},
		{
			name:     "deeply nested field",
			path:     "nested.deep.field",
			expected: "deep_value",
		},
		{
			name:     "missing field",
			path:     "missing",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.getFieldValue(data, tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExecute_ScriptActionSuccess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping shell script test on Windows")
	}

	scriptPath, allowedDir, err := getFixturePath("test.sh")
	require.NoError(t, err)

	var reportedResult reporter.ScriptResult
	var reportedActionName string

	mockRep := &mockReporter{
		reportFunc: func(ctx context.Context, deliveryID, actionName, actionUUID string, result reporter.ScriptResult) error {
			reportedResult = result
			reportedActionName = actionName
			return nil
		},
	}

	scriptRunner := NewScriptRunner([]string{allowedDir}, nil)
	httpExecutor := NewHTTPExecutor()

	executor := New([]config.Action{
		{
			ID:      "test_script_id",
			Name:    "Test Script Display Name",
			Type:    "script",
			Script:  scriptPath,
			Timeout: 5,
			Parameters: map[string]string{
				"message": "{{ message }}",
			},
			Trigger: config.TriggerConfig{
				EventType: "test.event",
			},
		},
	}, scriptRunner, httpExecutor, mockRep)

	event := api.Event{
		ID:      "delivery-123",
		EventID: "event-456",
		Type:    "test.event",
		Data: map[string]interface{}{
			"message": "ExecuteTest",
		},
	}

	executor.Execute(context.Background(), event)

	// Verify the result was reported with action ID, not display name
	assert.Equal(t, "test_script_id", reportedActionName)
	assert.Equal(t, 0, reportedResult.ExitCode)
	assert.Contains(t, reportedResult.Stdout, "Shell: ExecuteTest")
	assert.Nil(t, reportedResult.Error)
}

func TestExecute_ScriptActionFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping shell script test on Windows")
	}

	// Create a script that will fail
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "fail.sh")
	err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nexit 1\n"), 0755)
	require.NoError(t, err)

	var reportedResult reporter.ScriptResult

	mockRep := &mockReporter{
		reportFunc: func(ctx context.Context, deliveryID, actionName, actionUUID string, result reporter.ScriptResult) error {
			reportedResult = result
			return nil
		},
	}

	scriptRunner := NewScriptRunner([]string{tmpDir}, nil)
	httpExecutor := NewHTTPExecutor()

	executor := New([]config.Action{
		{
			ID:      "failing_script_id",
			Name:    "Failing Script",
			Type:    "script",
			Script:  scriptPath,
			Timeout: 5,
			Trigger: config.TriggerConfig{
				EventType: "test.event",
			},
		},
	}, scriptRunner, httpExecutor, mockRep)

	event := api.Event{
		ID:      "delivery-456",
		EventID: "event-789",
		Type:    "test.event",
		Data:    map[string]interface{}{},
	}

	executor.Execute(context.Background(), event)

	// Verify failure was reported
	assert.Equal(t, 1, reportedResult.ExitCode)
	assert.NotNil(t, reportedResult.Error)
}

func TestExecute_HTTPActionSuccess(t *testing.T) {
	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	}))
	defer server.Close()

	var reportedResult reporter.ScriptResult
	var reportedActionName string

	mockRep := &mockReporter{
		reportFunc: func(ctx context.Context, deliveryID, actionName, actionUUID string, result reporter.ScriptResult) error {
			reportedResult = result
			reportedActionName = actionName
			return nil
		},
	}

	scriptRunner := NewScriptRunner(nil, nil)
	httpExecutor := NewHTTPExecutor()

	executor := New([]config.Action{
		{
			ID:      "http_webhook_id",
			Name:    "HTTP Webhook",
			Type:    "http",
			Timeout: 10,
			HTTP: &config.HTTPAction{
				URL:    server.URL,
				Method: "POST",
			},
			Trigger: config.TriggerConfig{
				EventType: "alert.created",
			},
		},
	}, scriptRunner, httpExecutor, mockRep)

	event := api.Event{
		ID:      "delivery-789",
		EventID: "event-012",
		Type:    "alert.created",
		Data:    map[string]interface{}{},
	}

	executor.Execute(context.Background(), event)

	// Verify the result was reported with action ID
	assert.Equal(t, "http_webhook_id", reportedActionName)
	assert.Equal(t, 200, reportedResult.ExitCode) // HTTP executor returns status code
	assert.Contains(t, reportedResult.Stdout, "200")
	assert.Nil(t, reportedResult.Error)
}

func TestExecute_ReporterFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping shell script test on Windows")
	}

	scriptPath, allowedDir, err := getFixturePath("test.sh")
	require.NoError(t, err)

	reporterCalled := false

	mockRep := &mockReporter{
		reportFunc: func(ctx context.Context, deliveryID, actionName, actionUUID string, result reporter.ScriptResult) error {
			reporterCalled = true
			return errors.New("reporter failed")
		},
	}

	scriptRunner := NewScriptRunner([]string{allowedDir}, nil)
	httpExecutor := NewHTTPExecutor()

	executor := New([]config.Action{
		{
			ID:      "test_action_id",
			Name:    "Test Action",
			Type:    "script",
			Script:  scriptPath,
			Timeout: 5,
			Trigger: config.TriggerConfig{
				EventType: "test.event",
			},
		},
	}, scriptRunner, httpExecutor, mockRep)

	event := api.Event{
		ID:      "delivery-111",
		EventID: "event-111",
		Type:    "test.event",
		Data: map[string]interface{}{
			"message": "test",
		},
	}

	// Should not panic even when reporter fails
	assert.NotPanics(t, func() {
		executor.Execute(context.Background(), event)
	})

	assert.True(t, reporterCalled)
}

func TestSubstituteTemplate_InvalidTemplate(t *testing.T) {
	scriptRunner := NewScriptRunner(nil, nil)
	httpExecutor := NewHTTPExecutor()
	mockRep := &mockReporter{}

	executor := New([]config.Action{}, scriptRunner, httpExecutor, mockRep)

	event := api.Event{
		Data: map[string]interface{}{
			"test": "value",
		},
	}

	// Invalid Liquid template syntax
	result := executor.substituteTemplate("{{ invalid | unknown_filter }}", event)

	// Should return empty string on error and log warning
	assert.Empty(t, result, "Invalid template should return empty string")
}

func TestSubstituteTemplate_NoEnvVars(t *testing.T) {
	scriptRunner := NewScriptRunner(nil, nil)
	httpExecutor := NewHTTPExecutor()
	mockRep := &mockReporter{}

	executor := New([]config.Action{}, scriptRunner, httpExecutor, mockRep)

	event := api.Event{
		Data: map[string]interface{}{
			"message": "hello",
		},
	}

	// Template with no env vars
	result := executor.substituteTemplate("{{ message }}", event)
	assert.Equal(t, "hello", result)
}

func TestPrepareTemplateContext_MultipleEnvVars(t *testing.T) {
	scriptRunner := NewScriptRunner(nil, nil)
	httpExecutor := NewHTTPExecutor()
	mockRep := &mockReporter{}

	executor := New([]config.Action{}, scriptRunner, httpExecutor, mockRep)

	// Set multiple environment variables
	os.Setenv("TEST_VAR_1", "value1")
	os.Setenv("TEST_VAR_2", "value2")
	os.Setenv("TEST_VAR_3", "value3")
	defer os.Unsetenv("TEST_VAR_1")
	defer os.Unsetenv("TEST_VAR_2")
	defer os.Unsetenv("TEST_VAR_3")

	event := api.Event{
		Data: map[string]interface{}{
			"field": "data",
		},
	}

	tmpl := "{{ env.TEST_VAR_1 }} {{ env.TEST_VAR_2 }} {{ env.TEST_VAR_3 }}"
	result := executor.substituteTemplate(tmpl, event)

	assert.Contains(t, result, "value1")
	assert.Contains(t, result, "value2")
	assert.Contains(t, result, "value3")
}

func TestPrepareTemplateContext_EnvVarNotSet(t *testing.T) {
	scriptRunner := NewScriptRunner(nil, nil)
	httpExecutor := NewHTTPExecutor()
	mockRep := &mockReporter{}

	executor := New([]config.Action{}, scriptRunner, httpExecutor, mockRep)

	// Ensure env var is not set
	os.Unsetenv("NONEXISTENT_VAR")

	event := api.Event{
		Data: map[string]interface{}{},
	}

	// Template references non-existent env var
	result := executor.substituteTemplate("{{ env.NONEXISTENT_VAR }}", event)

	// Should render but env var won't be available
	assert.NotContains(t, result, "NONEXISTENT_VAR")
}

// Edge case tests for improved coverage

func TestExecute_NoMatchingAction_WithActionSlugInMetadata(t *testing.T) {
	// Test Execute when event has Action metadata with Slug - covers logging path
	mockRep := &mockReporter{}
	exec := New([]config.Action{}, nil, nil, mockRep)

	event := api.Event{
		ID:      "delivery-1",
		EventID: "event-1",
		Type:    "action.triggered",
		Data:    map[string]interface{}{},
		Action: &api.ActionMetadata{
			Slug: "some_action_slug",
			ID:   "action-uuid-123",
		},
	}

	// Execute - no matching action, should log event_action_slug
	exec.Execute(context.Background(), event)
}

func TestExecute_NoMatchingAction_ReporterError(t *testing.T) {
	// Test when reporter fails - covers error logging path
	reporterCalled := false
	mockRep := &mockReporter{
		reportFunc: func(ctx context.Context, deliveryID, actionName, actionUUID string, result reporter.ScriptResult) error {
			reporterCalled = true
			return errors.New("reporter failed")
		},
	}
	exec := New([]config.Action{}, nil, nil, mockRep)

	event := api.Event{
		ID:      "delivery-1",
		EventID: "event-1",
		Type:    "unknown.event",
		Data:    map[string]interface{}{},
	}

	// Should not panic when reporter fails
	exec.Execute(context.Background(), event)
	assert.True(t, reporterCalled)
}

func TestGetFieldValue_NonMapInPath(t *testing.T) {
	// Test getFieldValue when encountering non-map during traversal
	exec := &Executor{}

	data := map[string]interface{}{
		"level1": "string_value", // Not a map
	}

	// Try to access level1.level2 - should return nil
	result := exec.getFieldValue(data, "level1.level2")
	assert.Nil(t, result)
}
