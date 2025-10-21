package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTriggerConfig_GetEventTypes_Single(t *testing.T) {
	// Test single event type (legacy format)
	trigger := TriggerConfig{
		EventType: "alert.created",
	}

	eventTypes := trigger.GetEventTypes()
	assert.Equal(t, []string{"alert.created"}, eventTypes)
}

func TestTriggerConfig_GetEventTypes_Multiple(t *testing.T) {
	// Test multiple event types (new format)
	trigger := TriggerConfig{
		EventTypes: []string{"alert.created", "incident.created", "alert.action_triggered"},
	}

	eventTypes := trigger.GetEventTypes()
	assert.Equal(t, []string{"alert.created", "incident.created", "alert.action_triggered"}, eventTypes)
}

func TestTriggerConfig_GetEventTypes_PreferMultiple(t *testing.T) {
	// Test that event_types takes precedence over event_type
	trigger := TriggerConfig{
		EventType:  "alert.created",
		EventTypes: []string{"incident.created", "alert.updated"},
	}

	eventTypes := trigger.GetEventTypes()
	assert.Equal(t, []string{"incident.created", "alert.updated"}, eventTypes)
}

func TestTriggerConfig_GetEventTypes_Empty(t *testing.T) {
	// Test empty trigger
	trigger := TriggerConfig{}

	eventTypes := trigger.GetEventTypes()
	assert.Equal(t, []string{}, eventTypes)
}

// Tests for new on/callable format

func TestConvertToActions_OnActions(t *testing.T) {
	cfg := &ActionsConfig{
		On: map[string]OnAction{
			"alert.created": {
				Script: "/opt/scripts/handle-alert.sh",
				Parameters: map[string]string{
					"alert_id": "{{ id }}",
				},
			},
		},
	}

	cfg.ConvertToActions()

	assert.Len(t, cfg.Actions, 1)
	assert.Equal(t, "alert.created", cfg.Actions[0].ID)
	assert.Equal(t, "script", cfg.Actions[0].Type)
	assert.Equal(t, "/opt/scripts/handle-alert.sh", cfg.Actions[0].Script)
	assert.Equal(t, "alert.created", cfg.Actions[0].Trigger.EventType)
	assert.Equal(t, "{{ id }}", cfg.Actions[0].Parameters["alert_id"])
}

func TestConvertToActions_OnActionsHTTP(t *testing.T) {
	cfg := &ActionsConfig{
		On: map[string]OnAction{
			"incident.created": {
				HTTP: &HTTPAction{
					URL:    "https://httpbin.org/post",
					Method: "POST",
				},
			},
		},
	}

	cfg.ConvertToActions()

	assert.Len(t, cfg.Actions, 1)
	assert.Equal(t, "incident.created", cfg.Actions[0].ID)
	assert.Equal(t, "http", cfg.Actions[0].Type) // Auto-detected from HTTP field
	assert.NotNil(t, cfg.Actions[0].HTTP)
	assert.Equal(t, "https://httpbin.org/post", cfg.Actions[0].HTTP.URL)
}

func TestConvertToActions_CallableActions(t *testing.T) {
	cfg := &ActionsConfig{
		Callable: map[string]CallableAction{
			"restart_service": {
				Name:        "Restart Service",
				Description: "Restart a service",
				Script:      "/opt/scripts/restart.sh",
				ParameterDefinitions: []ParameterDefinition{
					{Name: "service_name", Type: "string", Required: true},
				},
			},
		},
	}

	cfg.ConvertToActions()

	assert.Len(t, cfg.Actions, 1)
	assert.Equal(t, "restart_service", cfg.Actions[0].ID)
	assert.Equal(t, "Restart Service", cfg.Actions[0].Name)
	assert.Equal(t, "script", cfg.Actions[0].Type)
	assert.Equal(t, "action.triggered", cfg.Actions[0].Trigger.EventType) // Default trigger
	assert.Len(t, cfg.Actions[0].ParameterDefinitions, 1)
	// Auto-generated parameters
	assert.Equal(t, "{{ parameters.service_name }}", cfg.Actions[0].Parameters["service_name"])
}

func TestConvertToActions_CallableWithCustomTrigger(t *testing.T) {
	cfg := &ActionsConfig{
		Callable: map[string]CallableAction{
			"restart_on_alert": {
				Name:    "Restart",
				Script:  "/opt/scripts/restart.sh",
				Trigger: "alert.action_triggered", // Custom trigger
				ParameterDefinitions: []ParameterDefinition{
					{Name: "service", Type: "string"},
				},
			},
		},
	}

	cfg.ConvertToActions()

	assert.Len(t, cfg.Actions, 1)
	assert.Equal(t, "alert.action_triggered", cfg.Actions[0].Trigger.EventType)
}

func TestConvertToActions_CallableHTTP(t *testing.T) {
	cfg := &ActionsConfig{
		Callable: map[string]CallableAction{
			"webhook": {
				Name: "Send Webhook",
				HTTP: &HTTPAction{
					URL:    "https://example.com",
					Method: "POST",
				},
				ParameterDefinitions: []ParameterDefinition{
					{Name: "message", Type: "string"},
				},
			},
		},
	}

	cfg.ConvertToActions()

	assert.Len(t, cfg.Actions, 1)
	assert.Equal(t, "http", cfg.Actions[0].Type) // Auto-detected
	assert.NotNil(t, cfg.Actions[0].HTTP)
}

func TestConvertToActions_Defaults(t *testing.T) {
	cfg := &ActionsConfig{
		Defaults: ActionDefaults{
			Timeout:    60,
			SourceType: "git",
			Env: map[string]string{
				"GLOBAL_VAR": "global_value",
			},
		},
		On: map[string]OnAction{
			"alert.created": {
				Script: "/opt/scripts/alert.sh",
				Env: map[string]string{
					"LOCAL_VAR": "local_value",
				},
			},
		},
	}

	cfg.ConvertToActions()

	assert.Len(t, cfg.Actions, 1)
	assert.Equal(t, 60, cfg.Actions[0].Timeout)       // From defaults
	assert.Equal(t, "git", cfg.Actions[0].SourceType) // From defaults
	// Env merged: global + local
	assert.Equal(t, "global_value", cfg.Actions[0].Env["GLOBAL_VAR"])
	assert.Equal(t, "local_value", cfg.Actions[0].Env["LOCAL_VAR"])
}

func TestConvertToActions_AutoGenerateParameters(t *testing.T) {
	cfg := &ActionsConfig{
		Callable: map[string]CallableAction{
			"test_action": {
				Name: "Test",
				HTTP: &HTTPAction{URL: "https://example.com"},
				ParameterDefinitions: []ParameterDefinition{
					{Name: "param1", Type: "string"},
					{Name: "param2", Type: "number"},
				},
				// No parameters specified - should auto-generate
			},
		},
	}

	cfg.ConvertToActions()

	assert.Len(t, cfg.Actions, 1)
	// Auto-generated mappings
	assert.Equal(t, "{{ parameters.param1 }}", cfg.Actions[0].Parameters["param1"])
	assert.Equal(t, "{{ parameters.param2 }}", cfg.Actions[0].Parameters["param2"])
}

func TestConvertToActions_AutoGenerateWithExtras(t *testing.T) {
	cfg := &ActionsConfig{
		Callable: map[string]CallableAction{
			"test_action": {
				Name: "Test",
				HTTP: &HTTPAction{URL: "https://example.com"},
				ParameterDefinitions: []ParameterDefinition{
					{Name: "service_name", Type: "string"},
					{Name: "environment", Type: "string"},
				},
				// Specify only EXTRA parameters (auto-generated ones are merged in)
				Parameters: map[string]string{
					"entity_id":    "{{ entity_id }}",
					"triggered_by": "{{ triggered_by.email }}",
					"region":       "us-west-2",
				},
			},
		},
	}

	cfg.ConvertToActions()

	assert.Len(t, cfg.Actions, 1)
	// Auto-generated from parameter_definitions
	assert.Equal(t, "{{ parameters.service_name }}", cfg.Actions[0].Parameters["service_name"])
	assert.Equal(t, "{{ parameters.environment }}", cfg.Actions[0].Parameters["environment"])
	// Manual extras merged in
	assert.Equal(t, "{{ entity_id }}", cfg.Actions[0].Parameters["entity_id"])
	assert.Equal(t, "{{ triggered_by.email }}", cfg.Actions[0].Parameters["triggered_by"])
	assert.Equal(t, "us-west-2", cfg.Actions[0].Parameters["region"])
}

func TestConvertToActions_ParametersOverrideAutoGenerated(t *testing.T) {
	cfg := &ActionsConfig{
		Callable: map[string]CallableAction{
			"test_action": {
				Name: "Test",
				HTTP: &HTTPAction{URL: "https://example.com"},
				ParameterDefinitions: []ParameterDefinition{
					{Name: "service_name", Type: "string"},
				},
				// Override auto-generated mapping with custom one
				Parameters: map[string]string{
					"service_name": "{{ parameters.service_name | upcase }}", // Custom mapping overrides
				},
			},
		},
	}

	cfg.ConvertToActions()

	assert.Len(t, cfg.Actions, 1)
	// Manual parameter overrides auto-generated
	assert.Equal(t, "{{ parameters.service_name | upcase }}", cfg.Actions[0].Parameters["service_name"])
}

func TestConvertToActions_MixedOnAndCallable(t *testing.T) {
	cfg := &ActionsConfig{
		On: map[string]OnAction{
			"alert.created": {
				Script: "/opt/scripts/alert.sh",
			},
			"incident.created": {
				HTTP: &HTTPAction{URL: "https://example.com"},
			},
		},
		Callable: map[string]CallableAction{
			"action1": {
				Name:   "Action 1",
				Script: "/opt/scripts/action1.sh",
				ParameterDefinitions: []ParameterDefinition{
					{Name: "p1", Type: "string"},
				},
			},
			"action2": {
				Name: "Action 2",
				HTTP: &HTTPAction{URL: "https://example.com"},
				ParameterDefinitions: []ParameterDefinition{
					{Name: "p2", Type: "string"},
				},
			},
		},
	}

	cfg.ConvertToActions()

	// Should have 4 actions total (2 from on, 2 from callable)
	assert.Len(t, cfg.Actions, 4)

	// Check we have both automatic and callable actions
	hasAlertCreated := false
	hasAction1 := false
	for _, action := range cfg.Actions {
		if action.ID == "alert.created" {
			hasAlertCreated = true
			assert.Len(t, action.ParameterDefinitions, 0) // No params for automatic
		}
		if action.ID == "action1" {
			hasAction1 = true
			assert.Len(t, action.ParameterDefinitions, 1) // Has params for callable
		}
	}
	assert.True(t, hasAlertCreated)
	assert.True(t, hasAction1)
}
