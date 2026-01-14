package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rootly/edge-connector/internal/api"
	"github.com/rootly/edge-connector/internal/config"
)

func TestConvertActionsToRegistrations_Callable(t *testing.T) {
	actions := []config.Action{
		{
			ID:          "restart_server",
			Name:        "Restart Server",
			Description: "Restart production server",
			Type:        "script",
			Timeout:     300,
			Trigger: config.TriggerConfig{
				EventType: "alert.action_triggered",
			},
			ParameterDefinitions: []config.ParameterDefinition{
				{
					Name:        "server_id",
					Type:        "string",
					Required:    true,
					Description: "Server ID to restart",
				},
				{
					Name:        "region",
					Type:        "string",
					Required:    false,
					Default:     "us-east-1",
					Options:     []string{"us-east-1", "us-west-2"},
					Description: "AWS region",
				},
			},
		},
		{
			ID:      "check_status",
			Name:    "Check Status",
			Type:    "http",
			Timeout: 60,
			Trigger: config.TriggerConfig{
				EventType: "action.triggered",
			},
			ParameterDefinitions: []config.ParameterDefinition{
				{
					Name:     "status_type",
					Type:     "string",
					Required: true,
				},
			},
		},
	}

	request := api.ConvertActionsToRegistrations(actions)

	// Both actions sent to backend (backend categorizes based on trigger)
	require.Len(t, request.Actions, 2)

	// First action (callable based on trigger pattern)
	assert.Equal(t, "restart_server", request.Actions[0].Slug)
	assert.Equal(t, "Restart Server", request.Actions[0].Name)
	assert.Equal(t, "script", request.Actions[0].ActionType)
	assert.Equal(t, "Restart production server", request.Actions[0].Description)
	assert.Equal(t, 300, request.Actions[0].Timeout)
	assert.Equal(t, "alert.action_triggered", request.Actions[0].Trigger)
	assert.Len(t, request.Actions[0].Parameters, 2)
	assert.Equal(t, "server_id", request.Actions[0].Parameters[0].Name)
	assert.True(t, request.Actions[0].Parameters[0].Required)

	// Second action (callable based on trigger pattern)
	assert.Equal(t, "check_status", request.Actions[1].Slug)
	assert.Equal(t, "Check Status", request.Actions[1].Name)
	assert.Equal(t, "action.triggered", request.Actions[1].Trigger)
}

func TestConvertActionsToRegistrations_MixedActions(t *testing.T) {
	actions := []config.Action{
		{
			ID:      "auto_action",
			Name:    "",
			Type:    "script",
			Timeout: 60,
			Trigger: config.TriggerConfig{
				EventType: "alert.created", // Automatic trigger
			},
		},
		{
			ID:      "callable_action",
			Name:    "Callable Action",
			Type:    "script",
			Timeout: 60,
			Trigger: config.TriggerConfig{
				EventType: "incident.action_triggered", // Callable trigger
			},
		},
	}

	request := api.ConvertActionsToRegistrations(actions)

	// Both actions sent to backend (backend categorizes)
	require.Len(t, request.Actions, 2)
	assert.Equal(t, "auto_action", request.Actions[0].Slug)
	assert.Equal(t, "alert.created", request.Actions[0].Trigger)
	assert.Equal(t, "callable_action", request.Actions[1].Slug)
	assert.Equal(t, "incident.action_triggered", request.Actions[1].Trigger)
}

func TestConvertActionsToRegistrations_Empty(t *testing.T) {
	actions := []config.Action{}

	request := api.ConvertActionsToRegistrations(actions)

	assert.Empty(t, request.Actions)
}

func TestConvertActionsToRegistrations_AutomaticActions(t *testing.T) {
	actions := []config.Action{
		{
			ID:   "auto_action1",
			Name: "",
			Type: "script",
			Trigger: config.TriggerConfig{
				EventType: "alert.created",
			},
		},
		{
			ID:   "auto_action2",
			Name: "",
			Type: "http",
			Trigger: config.TriggerConfig{
				EventType: "incident.created",
			},
		},
	}

	request := api.ConvertActionsToRegistrations(actions)

	// Both actions sent to backend (automatic triggers)
	require.Len(t, request.Actions, 2)
	assert.Equal(t, "auto_action1", request.Actions[0].Slug)
	assert.Equal(t, "alert.created", request.Actions[0].Trigger)
	assert.Equal(t, "auto_action2", request.Actions[1].Slug)
	assert.Equal(t, "incident.created", request.Actions[1].Trigger)
}

func TestConvertActionsToRegistrations_WithID(t *testing.T) {
	actions := []config.Action{
		{
			Name:        "Send Webhook", // Human-readable
			ID:          "send_webhook", // Machine identifier
			Description: "Send a webhook notification to external systems.\n\nThis action posts JSON data to a configured webhook URL.",
			Type:        "script",
			Timeout:     60,
			Trigger: config.TriggerConfig{
				EventType:  "action.triggered",
				ActionName: "send_webhook",
			},
			ParameterDefinitions: []config.ParameterDefinition{
				{
					Name:        "message",
					Type:        "string",
					Required:    true,
					Description: "Message to send",
				},
			},
		},
	}

	request := api.ConvertActionsToRegistrations(actions)

	// Action sent to backend (callable trigger)
	require.Len(t, request.Actions, 1)

	reg := request.Actions[0]
	assert.Equal(t, "send_webhook", reg.Slug, "Should use id as slug")
	assert.Equal(t, "Send Webhook", reg.Name, "Should use explicit name")
	assert.Contains(t, reg.Description, "Send a webhook")
	assert.Contains(t, reg.Description, "This action posts") // Multi-line description
}

func TestConvertActionsToRegistrations_HTTPAction(t *testing.T) {
	actions := []config.Action{
		{
			ID:          "send_webhook", // Machine ID
			Name:        "",             // No human name
			Description: "Send webhook notification",
			Type:        "http",
			Timeout:     30,
			Trigger: config.TriggerConfig{
				EventType:  "action.triggered",
				ActionName: "send_webhook",
			},
			HTTP: &config.HTTPAction{
				URL:    "https://example.com/webhook",
				Method: "POST",
				Headers: map[string]string{
					"Content-Type": "application/json",
					"X-API-Key":    "{{ env.API_KEY }}",
				},
				Params: map[string]string{
					"source": "rootly",
				},
				Body: `{"message": "{{ summary }}"}`,
			},
			ParameterDefinitions: []config.ParameterDefinition{
				{
					Name:        "message",
					Type:        "string",
					Required:    false,
					Description: "Custom message",
				},
			},
		},
	}

	request := api.ConvertActionsToRegistrations(actions)

	// Action sent to backend (callable trigger)
	require.Len(t, request.Actions, 1)

	reg := request.Actions[0]
	assert.Equal(t, "send_webhook", reg.Slug) // id → slug
	assert.Equal(t, "", reg.Name)             // no name provided (backend humanizes)
	assert.Equal(t, "http", reg.ActionType)
	assert.Equal(t, "Send webhook notification", reg.Description)
	assert.Equal(t, 30, reg.Timeout)
	assert.Len(t, reg.Parameters, 1)

	// Note: HTTP configuration is NOT included in ActionRegistration
	// The connector doesn't send HTTP config to backend during registration
}

func TestConvertActionsToRegistrations_CallableWithZeroParameters(t *testing.T) {
	tests := []struct {
		name        string
		action      config.Action
		expectedLen int
	}{
		{
			name: "standalone callable action with no parameters",
			action: config.Action{
				ID:          "clear_cache",
				Name:        "Clear Cache",
				Description: "Clear all application caches",
				Type:        "script",
				Timeout:     60,
				Trigger: config.TriggerConfig{
					EventType:  "action.triggered",
					ActionName: "clear_cache",
				},
				ParameterDefinitions: []config.ParameterDefinition{}, // Empty parameters
			},
			expectedLen: 0,
		},
		{
			name: "alert action with no parameters",
			action: config.Action{
				ID:          "restart_all",
				Name:        "Restart All Services",
				Description: "Restart all services without configuration",
				Type:        "script",
				Timeout:     120,
				Trigger: config.TriggerConfig{
					EventType:  "alert.action_triggered",
					ActionName: "restart_all",
				},
				ParameterDefinitions: nil, // Nil parameters
			},
			expectedLen: 0,
		},
		{
			name: "incident action with no parameters",
			action: config.Action{
				ID:          "page_oncall",
				Name:        "Page On-Call Engineer",
				Description: "Page the on-call engineer immediately",
				Type:        "http",
				Timeout:     30,
				Trigger: config.TriggerConfig{
					EventType:  "incident.action_triggered",
					ActionName: "page_oncall",
				},
				// No ParameterDefinitions field at all
			},
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actions := []config.Action{tt.action}
			request := api.ConvertActionsToRegistrations(actions)

			// Action should be sent to backend (callable based on trigger pattern)
			require.Len(t, request.Actions, 1)

			reg := request.Actions[0]
			assert.Equal(t, tt.action.ID, reg.Slug)
			assert.Equal(t, tt.action.Name, reg.Name)
			assert.Equal(t, tt.action.Type, reg.ActionType)
			assert.Equal(t, tt.action.Description, reg.Description)
			assert.Equal(t, tt.action.Timeout, reg.Timeout)
			assert.Equal(t, tt.action.Trigger.EventType, reg.Trigger)

			// Parameters should be empty (0 length)
			assert.Len(t, reg.Parameters, tt.expectedLen, "Should have zero parameters")

			// Verify trigger pattern is callable
			isCallableTrigger := reg.Trigger == "action.triggered" ||
				reg.Trigger == "alert.action_triggered" ||
				reg.Trigger == "incident.action_triggered"
			assert.True(t, isCallableTrigger, "Trigger should be a callable pattern")
		})
	}
}

func TestConvertActionsToRegistrations_MixedParameterCounts(t *testing.T) {
	actions := []config.Action{
		{
			ID:      "action_with_params",
			Name:    "Action With Parameters",
			Type:    "script",
			Timeout: 60,
			Trigger: config.TriggerConfig{
				EventType: "action.triggered",
			},
			ParameterDefinitions: []config.ParameterDefinition{
				{Name: "param1", Type: "string", Required: true},
				{Name: "param2", Type: "boolean", Default: false},
			},
		},
		{
			ID:      "action_no_params",
			Name:    "Action Without Parameters",
			Type:    "script",
			Timeout: 60,
			Trigger: config.TriggerConfig{
				EventType: "action.triggered",
			},
			ParameterDefinitions: []config.ParameterDefinition{}, // Zero params
		},
		{
			ID:      "auto_action",
			Name:    "",
			Type:    "script",
			Timeout: 60,
			Trigger: config.TriggerConfig{
				EventType: "alert.created", // Automatic action
			},
			// No parameters (automatic actions don't need them)
		},
	}

	request := api.ConvertActionsToRegistrations(actions)

	// All actions sent to backend
	require.Len(t, request.Actions, 3)

	// First action: callable with 2 parameters
	assert.Equal(t, "action_with_params", request.Actions[0].Slug)
	assert.Equal(t, "action.triggered", request.Actions[0].Trigger)
	assert.Len(t, request.Actions[0].Parameters, 2)

	// Second action: callable with 0 parameters
	assert.Equal(t, "action_no_params", request.Actions[1].Slug)
	assert.Equal(t, "action.triggered", request.Actions[1].Trigger)
	assert.Len(t, request.Actions[1].Parameters, 0)

	// Third action: automatic with 0 parameters (expected)
	assert.Equal(t, "auto_action", request.Actions[2].Slug)
	assert.Equal(t, "alert.created", request.Actions[2].Trigger)
	assert.Len(t, request.Actions[2].Parameters, 0)
}
