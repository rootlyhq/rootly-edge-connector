package api

import (
	"github.com/rootly/edge-connector/internal/config"
)

// ConvertActionsToRegistrations converts config actions to API registration format
// Backend categorizes actions based on trigger patterns
func ConvertActionsToRegistrations(actions []config.Action) RegisterActionsRequest {
	registrations := make([]ActionRegistration, 0, len(actions))

	for _, action := range actions {
		eventTypes := action.Trigger.GetEventTypes()
		trigger := ""
		if len(eventTypes) > 0 {
			trigger = eventTypes[0] // Use first event type as trigger
		}

		registrations = append(registrations, ActionRegistration{
			Slug:        action.ID,
			Name:        action.Name,
			Description: action.Description,
			ActionType:  action.Type,
			Trigger:     trigger,
			Timeout:     action.Timeout,
			Parameters:  convertParameterDefinitions(action.ParameterDefinitions),
		})
	}

	return RegisterActionsRequest{
		Actions: registrations,
	}
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
