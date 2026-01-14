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
