package api

import (
	"github.com/rootly/edge-connector/internal/config"
)

// ConvertActionsToRegistrations converts config actions to NEW API registration format
// Separates actions into automatic and callable based on parameter_definitions
func ConvertActionsToRegistrations(actions []config.Action) RegisterActionsRequest {
	automatic := make([]AutomaticActionRegistration, 0)
	callable := make([]CallableActionRegistration, 0)

	for _, action := range actions {
		eventTypes := action.Trigger.GetEventTypes()
		trigger := ""
		if len(eventTypes) > 0 {
			trigger = eventTypes[0] // Use first event type as trigger
		}

		// Automatic actions: no parameter_definitions
		if len(action.ParameterDefinitions) == 0 {
			automatic = append(automatic, AutomaticActionRegistration{
				Slug:        action.ID,
				ActionType:  action.Type,
				Trigger:     trigger,
				Timeout:     action.Timeout,
				Description: generateAutomaticDescription(action.ID),
			})
		} else {
			// Callable actions: have parameter_definitions
			callable = append(callable, CallableActionRegistration{
				Slug:        action.ID,
				Name:        action.Name,
				Description: action.Description,
				ActionType:  action.Type,
				Trigger:     trigger,
				Timeout:     action.Timeout,
				Parameters:  convertParameterDefinitions(action.ParameterDefinitions),
			})
		}
	}

	return RegisterActionsRequest{
		Automatic: automatic,
		Callable:  callable,
	}
}

// generateAutomaticDescription creates a description for automatic actions
func generateAutomaticDescription(eventType string) string {
	return "Handles " + eventType + " events"
}

// convertParameterDefinitions converts config parameter definitions to API format
func convertParameterDefinitions(params []config.ParameterDefinition) []ActionParameter {
	apiParams := make([]ActionParameter, 0, len(params))

	for _, param := range params {
		apiParams = append(apiParams, ActionParameter{
			Name:        param.Name,
			Type:        param.Type,
			Required:    param.Required,
			Description: param.Description,
			Default:     param.Default,
			Options:     param.Options,
		})
	}

	return apiParams
}
