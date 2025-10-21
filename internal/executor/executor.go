package executor

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/osteele/liquid"
	log "github.com/sirupsen/logrus"

	"github.com/rootly/edge-connector/internal/api"
	"github.com/rootly/edge-connector/internal/config"
	"github.com/rootly/edge-connector/internal/metrics"
	"github.com/rootly/edge-connector/internal/reporter"
)

// Reporter interface for reporting execution results
type Reporter interface {
	Report(ctx context.Context, deliveryID, actionName, actionUUID string, result reporter.ScriptResult) error
}

// Executor coordinates action execution
type Executor struct {
	scriptRunner *ScriptRunner
	httpExecutor *HTTPExecutor
	reporter     Reporter
	actions      []config.Action
}

// New creates a new executor
func New(actions []config.Action, scriptRunner *ScriptRunner, httpExecutor *HTTPExecutor, rep Reporter) *Executor {
	return &Executor{
		actions:      actions,
		scriptRunner: scriptRunner,
		httpExecutor: httpExecutor,
		reporter:     rep,
	}
}

// Execute processes an event and executes matching actions
func (e *Executor) Execute(ctx context.Context, event api.Event) {
	// Find matching action for this event
	action := e.findMatchingAction(event)
	if action == nil {
		logFields := log.Fields{
			"delivery_id": event.ID,
			"event_id":    event.EventID,
			"event_type":  event.Type,
		}

		// Include action slug from event if present (for action_triggered events debugging)
		if event.Action != nil && event.Action.Slug != "" {
			logFields["event_action_slug"] = event.Action.Slug
		}

		// Check if action_name exists in data (for action_triggered events)
		reportedActionName := "none"
		if actionNameRaw, ok := event.Data["action_name"]; ok {
			if actionNameStr, ok := actionNameRaw.(string); ok && actionNameStr != "" {
				logFields["data_action_name"] = actionNameStr
				reportedActionName = actionNameStr // Use actual action name in report
			}
		}

		log.WithFields(logFields).Warn("No matching action for event")

		// Report failure so delivery doesn't stay in running state forever
		errorMsg := fmt.Sprintf("No action configured for event type: %s", event.Type)
		if reportedActionName != "none" {
			errorMsg = fmt.Sprintf("No action found matching action_name: %s", reportedActionName)
		}

		result := reporter.ScriptResult{
			ExitCode:   1,
			DurationMs: 0,
			Error:      fmt.Errorf("no matching action found for event type: %s", event.Type),
			Stderr:     errorMsg,
		}
		// Extract action UUID if present
		actionUUID := ""
		if event.Action != nil {
			actionUUID = event.Action.ID
		}
		if err := e.reporter.Report(ctx, event.ID, reportedActionName, actionUUID, result); err != nil {
			log.WithError(err).Error("Failed to report no-action failure")
		}
		return
	}

	log.WithFields(log.Fields{
		"action_name": action.Name,
		"action_type": action.Type,
		"delivery_id": event.ID,
		"event_id":    event.EventID,
		"event_type":  event.Type,
	}).Info("Executing action for event")

	// Track events currently running
	if metrics.EventsRunning != nil {
		metrics.EventsRunning.Inc()
		defer metrics.EventsRunning.Dec()
	}

	start := time.Now()
	var result reporter.ScriptResult

	// Prepare parameters with template substitution (for both script and HTTP actions)
	params := e.prepareParameters(action, event)

	// Execute based on action type
	if action.Type == "http" {
		result = e.httpExecutor.Execute(ctx, action, event, params)
	} else {
		// Script action
		result = e.scriptRunner.Run(ctx, action, params)
	}

	// Record execution metrics
	duration := time.Since(start)
	status := "completed"
	if result.Error != nil || result.ExitCode != 0 {
		status = "failed"
	}
	metrics.RecordActionExecution(action.Name, action.Type, status, duration)

	// Report result back to Rootly with action identifier and UUID
	// actionName = action.ID (slug from config like "test_manual_action_http")
	// actionUUID = event.Action.ID (UUID from backend, empty for non-action events)
	actionUUID := ""
	if event.Action != nil {
		actionUUID = event.Action.ID
	}
	if err := e.reporter.Report(ctx, event.ID, action.ID, actionUUID, result); err != nil {
		log.WithError(err).Error("Failed to report execution result")
	}
}

// findMatchingAction finds the first action that matches the event
func (e *Executor) findMatchingAction(event api.Event) *config.Action {
	for i := range e.actions {
		if e.matchesAction(event, &e.actions[i]) {
			return &e.actions[i]
		}
	}
	return nil
}

// matchesAction checks if an event matches an action's trigger conditions
func (e *Executor) matchesAction(event api.Event, action *config.Action) bool {
	// Get all event types from trigger (handles both single and multiple)
	eventTypes := action.Trigger.GetEventTypes()

	// Check if any of the configured event types match
	matched := false
	for _, eventType := range eventTypes {
		if eventType == event.Type {
			matched = true
			break
		}
	}

	if !matched {
		return false
	}

	// Check action slug/name (for action_triggered events)
	// Supports two matching strategies:
	// 1. event.Action.Slug (preferred - from Action metadata object)
	// 2. event.Data["action_name"] (fallback - when Action metadata is null)
	// Matches against trigger.action_name (or defaults to action.ID if not specified)

	// Get expected action name from trigger (defaults to action.ID if not specified)
	expectedActionName := action.Trigger.ActionName
	if expectedActionName == "" {
		expectedActionName = action.ID
	}

	// Strategy 1: Check Action metadata object (preferred)
	if event.Action != nil {
		return event.Action.Slug == expectedActionName
	}

	// Strategy 2: Check data.action_name field (fallback for when Action is null)
	// This handles cases where backend sends action_name in data but not as Action metadata
	if actionNameRaw, ok := event.Data["action_name"]; ok {
		if actionNameStr, ok := actionNameRaw.(string); ok {
			// If action_name is specified in trigger (or defaults to ID), check it matches
			if expectedActionName != "" && actionNameStr != expectedActionName {
				return false
			}
			return true
		}
	}

	// No Action metadata and no action_name in data
	// For action_triggered event types, we need action identity to match
	// If expectedActionName is set (from trigger or defaulted to ID), require matching
	eventType := event.Type
	isActionTriggered := strings.HasSuffix(eventType, ".action_triggered") || eventType == "action.triggered"

	if isActionTriggered && expectedActionName != "" {
		// This is an action_triggered event but we can't verify which action
		// Don't match - prevents wrong action from executing
		return false
	}

	// No action filtering needed - matches any event of this type
	return true
}

// prepareParameters prepares parameters for script execution with template substitution
// User-provided values from event.data take precedence over config-defined templates
func (e *Executor) prepareParameters(action *config.Action, event api.Event) map[string]string {
	params := make(map[string]string)

	// First, apply template substitution for all configured parameters
	for key, template := range action.Parameters {
		value := e.substituteTemplate(template, event)
		params[key] = value
	}

	// Then, override with direct user-provided values from event.data
	// This allows user input to take precedence over config defaults
	if event.Data != nil {
		for key, value := range event.Data {
			// Only override if the key exists in our parameters OR parameter_definitions
			if _, existsInParams := action.Parameters[key]; existsInParams {
				// Convert value to string
				if strValue, ok := value.(string); ok {
					params[key] = strValue
				}
			} else {
				// Check if it's in parameter_definitions (for callable actions)
				for _, paramDef := range action.ParameterDefinitions {
					if paramDef.Name == key {
						if strValue, ok := value.(string); ok {
							params[key] = strValue
						}
						break
					}
				}
			}
		}
	}

	return params
}

// substituteTemplate performs template substitution using Liquid template engine
// Supports:
// - Simple fields: {{ field }}
// - Nested fields: {{ nested.field }}
// - Array access: {{ services[0].name }} or {{ services.first.name }}
// - Environment variables: {{ env.VAR }}
// - Filters: {{ services | map: "name" | join: ", " }}
func (e *Executor) substituteTemplate(tmplStr string, event api.Event) string {
	// Create Liquid engine
	engine := liquid.NewEngine()

	// Prepare template context with event data + env variables
	context := e.prepareTemplateContext(tmplStr, event)

	// Render template
	result, err := engine.ParseAndRenderString(tmplStr, context)
	if err != nil {
		// If template rendering fails, log and return empty string
		log.WithError(err).WithField("template", tmplStr).Warn("Template rendering failed, returning empty string")
		return ""
	}

	return result
}

// prepareTemplateContext creates the template context with event data and environment variables
func (e *Executor) prepareTemplateContext(tmplStr string, event api.Event) map[string]interface{} {
	context := make(map[string]interface{})

	// Add all event data to root context
	// This allows {{ field }} access directly
	for key, value := range event.Data {
		context[key] = value
	}

	// Add action metadata if present (for action_triggered events)
	// Allows templates to use {{ action.name }}, {{ action.slug }}, etc.
	if event.Action != nil {
		context["action"] = map[string]interface{}{
			"id":   event.Action.ID,
			"name": event.Action.Name,
			"slug": event.Action.Slug,
		}
	}

	// Add special "event" namespace for backward compatibility
	context["event"] = event.Data

	// Add environment variables under "env" namespace
	// Extract env vars used in templates (scan for env.* pattern)
	envVars := make(map[string]string)
	re := regexp.MustCompile(`\{\{\s*env\.([A-Z_][A-Z0-9_]*)\s*[}|]`)
	matches := re.FindAllStringSubmatch(tmplStr, -1)
	for _, match := range matches {
		if len(match) > 1 {
			envVarName := match[1]
			if value := os.Getenv(envVarName); value != "" {
				envVars[envVarName] = value
			}
		}
	}
	if len(envVars) > 0 {
		context["env"] = envVars
	}

	return context
}

// getFieldValue retrieves a nested field value from a map using dot notation
// e.g., "alert.host" -> event.Data["alert"]["host"]
func (e *Executor) getFieldValue(data map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	var current interface{} = data

	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[part]
		} else {
			return nil
		}
	}

	return current
}
