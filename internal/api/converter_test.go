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

	// Both have parameter_definitions → both are callable
	assert.Len(t, request.Automatic, 0)
	assert.Len(t, request.Callable, 2)

	// First callable action
	assert.Equal(t, "restart_server", request.Callable[0].Slug)
	assert.Equal(t, "Restart Server", request.Callable[0].Name)
	assert.Equal(t, "script", request.Callable[0].ActionType)
	assert.Equal(t, "Restart production server", request.Callable[0].Description)
	assert.Equal(t, 300, request.Callable[0].Timeout)
	assert.Equal(t, "alert.action_triggered", request.Callable[0].Trigger)
	assert.Len(t, request.Callable[0].Parameters, 2)
	assert.Equal(t, "server_id", request.Callable[0].Parameters[0].Name)
	assert.True(t, request.Callable[0].Parameters[0].Required)

	// Second callable action
	assert.Equal(t, "check_status", request.Callable[1].Slug)
	assert.Equal(t, "Check Status", request.Callable[1].Name)
	assert.Equal(t, "action.triggered", request.Callable[1].Trigger)
}

func TestConvertActionsToRegistrations_AllActions(t *testing.T) {
	actions := []config.Action{
		{
			ID:      "auto_action",
			Name:    "auto_action",
			Type:    "script",
			Timeout: 60,
			Trigger: config.TriggerConfig{
				EventType: "alert.created", // Automatic trigger - now registered!
			},
		},
		{
			ID:      "callable_action",
			Name:    "callable_action",
			Type:    "script",
			Timeout: 60,
			Trigger: config.TriggerConfig{
				EventType: "incident.action_triggered", // User-triggered action
			},
		},
	}

	request := api.ConvertActionsToRegistrations(actions)

	// Both have no parameter_definitions → both are automatic
	assert.Len(t, request.Automatic, 2)
	assert.Len(t, request.Callable, 0)
	assert.Equal(t, "auto_action", request.Automatic[0].Slug)
	assert.Equal(t, "callable_action", request.Automatic[1].Slug)
}

func TestConvertActionsToRegistrations_Empty(t *testing.T) {
	actions := []config.Action{}

	request := api.ConvertActionsToRegistrations(actions)

	assert.Empty(t, request.Automatic)
	assert.Empty(t, request.Callable)
}

func TestConvertActionsToRegistrations_AutomaticActions(t *testing.T) {
	actions := []config.Action{
		{
			ID:   "auto_action1",
			Name: "auto_action1",
			Type: "script",
			Trigger: config.TriggerConfig{
				EventType: "alert.created",
			},
		},
		{
			ID:   "auto_action2",
			Name: "auto_action2",
			Type: "http",
			Trigger: config.TriggerConfig{
				EventType: "incident.created",
			},
		},
	}

	request := api.ConvertActionsToRegistrations(actions)

	// No parameter_definitions → all are automatic
	assert.Len(t, request.Automatic, 2)
	assert.Len(t, request.Callable, 0)
	assert.Equal(t, "auto_action1", request.Automatic[0].Slug)
	assert.Equal(t, "auto_action2", request.Automatic[1].Slug)
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

	// Has parameter_definitions → callable
	assert.Len(t, request.Automatic, 0)
	require.Len(t, request.Callable, 1)

	reg := request.Callable[0]
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

	// Has parameter_definitions → callable
	assert.Len(t, request.Automatic, 0)
	require.Len(t, request.Callable, 1)

	reg := request.Callable[0]
	assert.Equal(t, "send_webhook", reg.Slug) // id → slug
	assert.Equal(t, "", reg.Name)             // no name provided (backend humanizes)
	assert.Equal(t, "http", reg.ActionType)
	assert.Equal(t, "Send webhook notification", reg.Description)
	assert.Equal(t, 30, reg.Timeout)
	assert.Len(t, reg.Parameters, 1)

	// Note: HTTP configuration is NOT included in CallableActionRegistration
	// The connector doesn't send HTTP config to backend during registration
}
