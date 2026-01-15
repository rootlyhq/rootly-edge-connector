//go:build integration
// +build integration

package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rootly/edge-connector/internal/api"
	"github.com/rootly/edge-connector/internal/config"
	"github.com/rootly/edge-connector/internal/executor"
	"github.com/rootly/edge-connector/internal/poller"
	"github.com/rootly/edge-connector/internal/reporter"
	"github.com/rootly/edge-connector/internal/worker"
)

// MockAPIServer simulates the Rootly API for integration testing
type MockAPIServer struct {
	*httptest.Server
	events     []api.Event
	executions []api.ExecutionResult
	mu         sync.RWMutex
}

// loadEventFixture loads an event fixture from JSON file
func loadEventFixture(t *testing.T, filename string) api.EventsResponse {
	data, err := os.ReadFile("testdata/fixtures/" + filename)
	require.NoError(t, err, "Failed to read fixture file")

	var response api.EventsResponse
	err = json.Unmarshal(data, &response)
	require.NoError(t, err, "Failed to parse fixture JSON")

	return response
}

func NewMockAPIServer() *MockAPIServer {
	mock := &MockAPIServer{
		events:     []api.Event{},
		executions: []api.ExecutionResult{},
	}

	mux := http.NewServeMux()

	// GET /rec/v1/deliveries
	mux.HandleFunc("/rec/v1/deliveries", func(w http.ResponseWriter, r *http.Request) {
		mock.mu.RLock()
		events := make([]api.Event, len(mock.events))
		copy(events, mock.events)
		mock.mu.RUnlock()

		response := api.EventsResponse{Events: events}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

		// Clear events after fetching
		mock.mu.Lock()
		mock.events = []api.Event{}
		mock.mu.Unlock()
	})

	// PATCH /rec/v1/deliveries/:id (report execution or mark as running)
	mux.HandleFunc("/rec/v1/deliveries/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PATCH" {
			// Extract delivery ID from URL path
			// URL format: /rec/v1/deliveries/{delivery_id}
			pathParts := strings.Split(r.URL.Path, "/")
			var deliveryID string
			for i, part := range pathParts {
				if part == "deliveries" && i+1 < len(pathParts) {
					deliveryID = pathParts[i+1]
					break
				}
			}

			var execution api.ExecutionResult
			if err := json.NewDecoder(r.Body).Decode(&execution); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// Set delivery ID from URL (since it's not in JSON body)
			execution.DeliveryID = deliveryID

			mock.mu.Lock()
			mock.executions = append(mock.executions, execution)
			mock.mu.Unlock()

			w.WriteHeader(http.StatusOK)
		}
	})

	mock.Server = httptest.NewServer(mux)
	return mock
}

func (m *MockAPIServer) AddEvent(event api.Event) {
	m.mu.Lock()
	m.events = append(m.events, event)
	m.mu.Unlock()
}

func (m *MockAPIServer) GetExecutions() []api.ExecutionResult {
	m.mu.RLock()
	defer m.mu.RUnlock()
	executions := make([]api.ExecutionResult, len(m.executions))
	copy(executions, m.executions)
	return executions
}

func (m *MockAPIServer) ClearExecutions() {
	m.mu.Lock()
	m.executions = []api.ExecutionResult{}
	m.mu.Unlock()
}

func TestEndToEnd_ScriptExecution(t *testing.T) {
	// Setup mock API server
	mockServer := NewMockAPIServer()
	defer mockServer.Close()

	// Create temporary test directory
	tmpDir := t.TempDir()

	// Create test script
	scriptPath := filepath.Join(tmpDir, "handle_alert.sh")
	scriptContent := `#!/bin/bash
echo "Handling alert: host=$REC_PARAM_HOST"
exit 0
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	// Setup configuration
	cfg := &config.Config{
		App: config.AppConfig{
			Name: "test-connector",
		},
		Rootly: config.RootlyConfig{
			APIURL:  mockServer.URL,
			APIPath: "/rec/v1",
			APIKey:  "test-key",
		},
		Poller: config.PollerConfig{
			PollingWaitIntervalMs: 100, // Fast polling for test
			VisibilityTimeoutSec:  30,
			MaxNumberOfMessages:   10,
			RetryOnError:          true,
			RetryBackoff:          "exponential",
			MaxRetries:            3,
		},
		Pool: config.PoolConfig{
			MaxNumberOfWorkers: 2,
			MinNumberOfWorkers: 1,
			QueueSize:          10,
		},
		Security: config.SecurityConfig{
			AllowedScriptPaths: []string{tmpDir},
			ScriptTimeout:      30,
		},
	}

	actions := []config.Action{
		{
			ID:         "handle_alert",
			Name:       "Handle Alert",
			Type:       "script",
			SourceType: "local",
			Script:     scriptPath,
			Trigger: config.TriggerConfig{
				EventType: "alert.created",
			},
			Parameters: map[string]string{
				"host": "{{ data.host }}", // Alert data is flat in event.Data
			},
			Timeout: 5,
		},
	}

	// Initialize components
	apiClient := api.NewClient(cfg.Rootly.APIURL, cfg.Rootly.APIPath, cfg.Rootly.APIKey, "test")
	scriptRunner := executor.NewScriptRunner(cfg.Security.AllowedScriptPaths, cfg.Security.GlobalEnv)
	httpExecutor := executor.NewHTTPExecutor()
	rep := reporter.New(apiClient)
	exec := executor.New(actions, scriptRunner, httpExecutor, rep)
	pool := worker.NewPool(&cfg.Pool, exec)
	poll := poller.New(apiClient, &cfg.Poller, pool)

	// Start components
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool.Start(ctx)
	go poll.Start(ctx)

	// Load and add test event from fixture
	fixture := loadEventFixture(t, "event_alert_for_script.json")
	require.Len(t, fixture.Events, 1, "Fixture should contain 1 event")
	mockServer.AddEvent(fixture.Events[0])

	// Wait for processing
	time.Sleep(1 * time.Second)

	// Verify execution was reported
	executions := mockServer.GetExecutions()
	require.GreaterOrEqual(t, len(executions), 1, "Expected at least one execution")

	// Find the completed execution (poller marks as "running" first, then executor reports "completed")
	var completedExecution *api.ExecutionResult
	for i := range executions {
		if executions[i].ExecutionStatus == "completed" {
			completedExecution = &executions[i]
			break
		}
	}
	require.NotNil(t, completedExecution, "Should have a completed execution")

	assert.Equal(t, "queue-123", completedExecution.DeliveryID)
	assert.Equal(t, "handle_alert", completedExecution.ExecutionActionName) // Action slug from config.id
	assert.Empty(t, completedExecution.ExecutionActionID)                   // No action UUID for alert.created events
	assert.Equal(t, "completed", completedExecution.ExecutionStatus)
	assert.Equal(t, 0, completedExecution.ExecutionExitCode)
	assert.Contains(t, completedExecution.ExecutionStdout, "host=prod-db-01")
	assert.NotEmpty(t, completedExecution.CompletedAt)
	assert.Empty(t, completedExecution.FailedAt)
}

func TestEndToEnd_HTTPAction(t *testing.T) {
	// Setup mock API server
	mockServer := NewMockAPIServer()
	defer mockServer.Close()

	// Setup mock webhook endpoint
	webhookCalled := false
	var webhookPayload map[string]interface{}
	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		webhookCalled = true
		json.NewDecoder(r.Body).Decode(&webhookPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer webhookServer.Close()

	cfg := &config.Config{
		App: config.AppConfig{
			Name: "test-connector",
		},
		Rootly: config.RootlyConfig{
			APIURL:  mockServer.URL,
			APIPath: "/rec/v1",
			APIKey:  "test-key",
		},
		Poller: config.PollerConfig{
			PollingWaitIntervalMs: 100,
			VisibilityTimeoutSec:  30,
			MaxNumberOfMessages:   10,
		},
		Pool: config.PoolConfig{
			MaxNumberOfWorkers: 2,
			MinNumberOfWorkers: 1,
			QueueSize:          10,
		},
	}

	actions := []config.Action{
		{
			ID:   "send_webhook",
			Name: "Send Webhook",
			Type: "http",
			Trigger: config.TriggerConfig{
				EventType: "incident.created",
			},
			HTTP: &config.HTTPAction{
				URL:    webhookServer.URL,
				Method: "POST",
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: `{"title": "{{ title }}"}`, // Liquid template syntax
			},
			Timeout: 5,
		},
	}

	// Initialize components
	apiClient := api.NewClient(cfg.Rootly.APIURL, cfg.Rootly.APIPath, cfg.Rootly.APIKey, "test")
	scriptRunner := executor.NewScriptRunner(nil, nil)
	httpExecutor := executor.NewHTTPExecutor()
	rep := reporter.New(apiClient)
	exec := executor.New(actions, scriptRunner, httpExecutor, rep)
	pool := worker.NewPool(&cfg.Pool, exec)
	poll := poller.New(apiClient, &cfg.Poller, pool)

	// Start components
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool.Start(ctx)
	go poll.Start(ctx)

	// Load and add test event from fixture
	fixture := loadEventFixture(t, "event_incident_for_http.json")
	require.Len(t, fixture.Events, 1, "Fixture should contain 1 event")
	mockServer.AddEvent(fixture.Events[0])

	// Wait for processing
	time.Sleep(1 * time.Second)

	// Verify webhook was called
	assert.True(t, webhookCalled, "Webhook should have been called")
	assert.Equal(t, "Database outage", webhookPayload["title"])

	// Verify execution was reported
	executions := mockServer.GetExecutions()
	require.GreaterOrEqual(t, len(executions), 1, "Expected at least one execution")

	// Find the completed execution
	var completedExecution *api.ExecutionResult
	for i := range executions {
		if executions[i].ExecutionStatus == "completed" {
			completedExecution = &executions[i]
			break
		}
	}
	require.NotNil(t, completedExecution, "Should have a completed execution")

	assert.Equal(t, "queue-789", completedExecution.DeliveryID)
	assert.Equal(t, "send_webhook", completedExecution.ExecutionActionName) // Action slug from config.id
	assert.Empty(t, completedExecution.ExecutionActionID)                   // No action UUID for incident.created events
	assert.Equal(t, "completed", completedExecution.ExecutionStatus)
	assert.Equal(t, 200, completedExecution.ExecutionExitCode)
}

func TestEndToEnd_NoMatchingAction(t *testing.T) {
	mockServer := NewMockAPIServer()
	defer mockServer.Close()

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "dummy.sh")
	err := os.WriteFile(scriptPath, []byte("#!/bin/bash\necho 'Should not run'\nexit 0"), 0755)
	require.NoError(t, err)

	cfg := &config.Config{
		Rootly: config.RootlyConfig{
			APIURL:  mockServer.URL,
			APIPath: "/rec/v1",
			APIKey:  "test-key",
		},
		Poller: config.PollerConfig{
			PollingWaitIntervalMs: 100,
			MaxNumberOfMessages:   10,
		},
		Pool: config.PoolConfig{
			MaxNumberOfWorkers: 2,
			MinNumberOfWorkers: 1,
			QueueSize:          10,
		},
		Security: config.SecurityConfig{
			AllowedScriptPaths: []string{tmpDir},
		},
	}

	// Action that doesn't match the event
	actions := []config.Action{
		{
			ID:         "wrong_action",
			Name:       "Wrong Action",
			Type:       "script",
			SourceType: "local",
			Script:     scriptPath,
			Trigger: config.TriggerConfig{
				EventType: "incident.created", // Won't match alert.created
			},
			Timeout: 5,
		},
	}

	apiClient := api.NewClient(cfg.Rootly.APIURL, cfg.Rootly.APIPath, cfg.Rootly.APIKey, "test")
	scriptRunner := executor.NewScriptRunner(nil, nil)
	httpExecutor := executor.NewHTTPExecutor()
	rep := reporter.New(apiClient)
	exec := executor.New(actions, scriptRunner, httpExecutor, rep)
	pool := worker.NewPool(&cfg.Pool, exec)
	poll := poller.New(apiClient, &cfg.Poller, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	pool.Start(ctx)
	go poll.Start(ctx)

	// Load and add event that won't match from fixture
	fixture := loadEventFixture(t, "event_alert_no_match.json")
	require.Len(t, fixture.Events, 1, "Fixture should contain 1 event")
	mockServer.AddEvent(fixture.Events[0])

	// Wait for processing
	time.Sleep(1 * time.Second)

	// Should report failure execution for no matching action
	executions := mockServer.GetExecutions()
	require.GreaterOrEqual(t, len(executions), 1, "Should report failure for no matching action")

	// Find the failed execution
	var failedExecution *api.ExecutionResult
	for i := range executions {
		if executions[i].ExecutionStatus == "failed" {
			failedExecution = &executions[i]
			break
		}
	}
	require.NotNil(t, failedExecution, "Should have a failed execution")

	assert.Equal(t, "failed", failedExecution.ExecutionStatus)
	assert.Equal(t, 1, failedExecution.ExecutionExitCode)
	assert.Contains(t, failedExecution.ExecutionStderr, "No action configured for event type")
}

func TestEndToEnd_ActionTriggeredEvents(t *testing.T) {
	mockServer := NewMockAPIServer()
	defer mockServer.Close()

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "action_handler.sh")
	scriptContent := `#!/bin/bash
echo "Action triggered: $REC_PARAM_ACTION_NAME"
exit 0
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	cfg := &config.Config{
		Rootly: config.RootlyConfig{
			APIURL:  mockServer.URL,
			APIPath: "/rec/v1",
			APIKey:  "test-key",
		},
		Poller: config.PollerConfig{
			PollingWaitIntervalMs: 100,
			MaxNumberOfMessages:   10,
		},
		Pool: config.PoolConfig{
			MaxNumberOfWorkers: 2,
			MinNumberOfWorkers: 1,
			QueueSize:          10,
		},
		Security: config.SecurityConfig{
			AllowedScriptPaths: []string{tmpDir},
		},
	}

	actions := []config.Action{
		{
			ID:         "restart_service",
			Name:       "Restart Service",
			Type:       "script",
			SourceType: "local",
			Trigger: config.TriggerConfig{
				EventType: "alert.action_triggered",
			},
			Script: scriptPath,
			Parameters: map[string]string{
				"action_name": "{{ action_name }}",
			},
			Timeout: 5,
		},
	}

	apiClient := api.NewClient(cfg.Rootly.APIURL, cfg.Rootly.APIPath, cfg.Rootly.APIKey, "test")
	scriptRunner := executor.NewScriptRunner(cfg.Security.AllowedScriptPaths, nil)
	httpExecutor := executor.NewHTTPExecutor()
	rep := reporter.New(apiClient)
	exec := executor.New(actions, scriptRunner, httpExecutor, rep)
	pool := worker.NewPool(&cfg.Pool, exec)
	poll := poller.New(apiClient, &cfg.Poller, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool.Start(ctx)
	go poll.Start(ctx)

	// Load and add action triggered event from fixture
	fixture := loadEventFixture(t, "event_alert_action_triggered.json")
	require.Len(t, fixture.Events, 1, "Fixture should contain 1 event")
	mockServer.AddEvent(fixture.Events[0])

	time.Sleep(1 * time.Second)

	executions := mockServer.GetExecutions()
	require.GreaterOrEqual(t, len(executions), 1)

	var completedExecution *api.ExecutionResult
	for i := range executions {
		if executions[i].ExecutionStatus == "completed" {
			completedExecution = &executions[i]
			break
		}
	}
	require.NotNil(t, completedExecution)

	assert.Equal(t, "completed", completedExecution.ExecutionStatus)
	assert.Equal(t, 0, completedExecution.ExecutionExitCode)
	assert.Contains(t, completedExecution.ExecutionStdout, "restart_service")
}

func TestEndToEnd_MultipleActions(t *testing.T) {
	mockServer := NewMockAPIServer()
	defer mockServer.Close()

	tmpDir := t.TempDir()

	// Create first script
	script1Path := filepath.Join(tmpDir, "script1.sh")
	err := os.WriteFile(script1Path, []byte("#!/bin/bash\necho 'Script 1 executed'\nexit 0"), 0755)
	require.NoError(t, err)

	// Create second script
	script2Path := filepath.Join(tmpDir, "script2.sh")
	err = os.WriteFile(script2Path, []byte("#!/bin/bash\necho 'Script 2 executed'\nexit 0"), 0755)
	require.NoError(t, err)

	cfg := &config.Config{
		Rootly: config.RootlyConfig{
			APIURL:  mockServer.URL,
			APIPath: "/rec/v1",
			APIKey:  "test-key",
		},
		Poller: config.PollerConfig{
			PollingWaitIntervalMs: 100,
			MaxNumberOfMessages:   10,
		},
		Pool: config.PoolConfig{
			MaxNumberOfWorkers: 2,
			MinNumberOfWorkers: 1,
			QueueSize:          10,
		},
		Security: config.SecurityConfig{
			AllowedScriptPaths: []string{tmpDir},
		},
	}

	// Multiple actions for same event type
	actions := []config.Action{
		{
			ID:         "action1",
			Name:       "Action 1",
			Type:       "script",
			SourceType: "local",
			Trigger: config.TriggerConfig{
				EventType: "alert.created",
			},
			Script:  script1Path,
			Timeout: 5,
		},
		{
			ID:         "action2",
			Name:       "Action 2",
			Type:       "script",
			SourceType: "local",
			Trigger: config.TriggerConfig{
				EventType: "alert.created",
			},
			Script:  script2Path,
			Timeout: 5,
		},
	}

	apiClient := api.NewClient(cfg.Rootly.APIURL, cfg.Rootly.APIPath, cfg.Rootly.APIKey, "test")
	scriptRunner := executor.NewScriptRunner(cfg.Security.AllowedScriptPaths, nil)
	httpExecutor := executor.NewHTTPExecutor()
	rep := reporter.New(apiClient)
	exec := executor.New(actions, scriptRunner, httpExecutor, rep)
	pool := worker.NewPool(&cfg.Pool, exec)
	poll := poller.New(apiClient, &cfg.Poller, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool.Start(ctx)
	go poll.Start(ctx)

	// Load and add simple event from fixture
	fixture := loadEventFixture(t, "event_alert_simple.json")
	require.Len(t, fixture.Events, 1, "Fixture should contain 1 event")
	mockServer.AddEvent(fixture.Events[0])

	time.Sleep(1 * time.Second)

	executions := mockServer.GetExecutions()
	// Should have executions for both actions
	assert.GreaterOrEqual(t, len(executions), 2, "Should execute both matching actions")
}

func TestEndToEnd_ScriptFailure(t *testing.T) {
	mockServer := NewMockAPIServer()
	defer mockServer.Close()

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "failing_script.sh")
	scriptContent := `#!/bin/bash
echo "Error message" >&2
exit 1
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	cfg := &config.Config{
		Rootly: config.RootlyConfig{
			APIURL:  mockServer.URL,
			APIPath: "/rec/v1",
			APIKey:  "test-key",
		},
		Poller: config.PollerConfig{
			PollingWaitIntervalMs: 100,
			MaxNumberOfMessages:   10,
		},
		Pool: config.PoolConfig{
			MaxNumberOfWorkers: 1,
			MinNumberOfWorkers: 1,
			QueueSize:          10,
		},
		Security: config.SecurityConfig{
			AllowedScriptPaths: []string{tmpDir},
		},
	}

	actions := []config.Action{
		{
			ID:         "failing_action",
			Name:       "Failing Action",
			Type:       "script",
			SourceType: "local",
			Trigger: config.TriggerConfig{
				EventType: "alert.created",
			},
			Script:  scriptPath,
			Timeout: 5,
		},
	}

	apiClient := api.NewClient(cfg.Rootly.APIURL, cfg.Rootly.APIPath, cfg.Rootly.APIKey, "test")
	scriptRunner := executor.NewScriptRunner(cfg.Security.AllowedScriptPaths, nil)
	httpExecutor := executor.NewHTTPExecutor()
	rep := reporter.New(apiClient)
	exec := executor.New(actions, scriptRunner, httpExecutor, rep)
	pool := worker.NewPool(&cfg.Pool, exec)
	poll := poller.New(apiClient, &cfg.Poller, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool.Start(ctx)
	go poll.Start(ctx)

	// Load and add failure event from fixture
	fixture := loadEventFixture(t, "event_alert_for_failure.json")
	require.Len(t, fixture.Events, 1, "Fixture should contain 1 event")
	mockServer.AddEvent(fixture.Events[0])

	time.Sleep(1 * time.Second)

	executions := mockServer.GetExecutions()
	require.GreaterOrEqual(t, len(executions), 1)

	var failedExecution *api.ExecutionResult
	for i := range executions {
		if executions[i].ExecutionStatus == "failed" {
			failedExecution = &executions[i]
			break
		}
	}
	require.NotNil(t, failedExecution, "Should have a failed execution")

	assert.Equal(t, "failed", failedExecution.ExecutionStatus)
	assert.Equal(t, 1, failedExecution.ExecutionExitCode)
	assert.Contains(t, failedExecution.ExecutionStderr, "Error message")
	assert.NotEmpty(t, failedExecution.FailedAt)
	assert.Empty(t, failedExecution.CompletedAt)
}

func TestEndToEnd_PythonScript(t *testing.T) {
	mockServer := NewMockAPIServer()
	defer mockServer.Close()

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "handler.py")
	scriptContent := `#!/usr/bin/env python3
import os
import sys

# Read parameters from environment
alert_id = os.getenv("REC_PARAM_ALERT_ID", "")
severity = os.getenv("REC_PARAM_SEVERITY", "")

print(f"Processing alert: {alert_id}, severity: {severity}")
sys.exit(0)
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	cfg := &config.Config{
		Rootly: config.RootlyConfig{
			APIURL:  mockServer.URL,
			APIPath: "/rec/v1",
			APIKey:  "test-key",
		},
		Poller: config.PollerConfig{
			PollingWaitIntervalMs: 100,
			MaxNumberOfMessages:   10,
		},
		Pool: config.PoolConfig{
			MaxNumberOfWorkers: 1,
			MinNumberOfWorkers: 1,
			QueueSize:          10,
		},
		Security: config.SecurityConfig{
			AllowedScriptPaths: []string{tmpDir},
		},
	}

	actions := []config.Action{
		{
			ID:         "python_handler",
			Name:       "Python Handler",
			Type:       "script",
			SourceType: "local",
			Trigger: config.TriggerConfig{
				EventType: "alert.created",
			},
			Script: scriptPath,
			Parameters: map[string]string{
				"alert_id": "{{ data.id }}",
				"severity": "{{ data.severity }}",
			},
			Timeout: 5,
		},
	}

	apiClient := api.NewClient(cfg.Rootly.APIURL, cfg.Rootly.APIPath, cfg.Rootly.APIKey, "test")
	scriptRunner := executor.NewScriptRunner(cfg.Security.AllowedScriptPaths, nil)
	httpExecutor := executor.NewHTTPExecutor()
	rep := reporter.New(apiClient)
	exec := executor.New(actions, scriptRunner, httpExecutor, rep)
	pool := worker.NewPool(&cfg.Pool, exec)
	poll := poller.New(apiClient, &cfg.Poller, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool.Start(ctx)
	go poll.Start(ctx)

	fixture := loadEventFixture(t, "event_alert_for_script.json")
	require.Len(t, fixture.Events, 1)
	mockServer.AddEvent(fixture.Events[0])

	time.Sleep(1 * time.Second)

	executions := mockServer.GetExecutions()
	require.GreaterOrEqual(t, len(executions), 1)

	var completedExecution *api.ExecutionResult
	for i := range executions {
		if executions[i].ExecutionStatus == "completed" {
			completedExecution = &executions[i]
			break
		}
	}
	require.NotNil(t, completedExecution)

	assert.Equal(t, "completed", completedExecution.ExecutionStatus)
	assert.Equal(t, 0, completedExecution.ExecutionExitCode)
	assert.Contains(t, completedExecution.ExecutionStdout, "Processing alert")
}

func TestEndToEnd_NodeJSScript(t *testing.T) {
	mockServer := NewMockAPIServer()
	defer mockServer.Close()

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "handler.js")
	scriptContent := `#!/usr/bin/env node
const alertId = process.env.REC_PARAM_ALERT_ID || '';
const severity = process.env.REC_PARAM_SEVERITY || '';

console.log('Alert:', alertId, 'Severity:', severity);
process.exit(0);
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	cfg := &config.Config{
		Rootly: config.RootlyConfig{
			APIURL:  mockServer.URL,
			APIPath: "/rec/v1",
			APIKey:  "test-key",
		},
		Poller: config.PollerConfig{
			PollingWaitIntervalMs: 100,
			MaxNumberOfMessages:   10,
		},
		Pool: config.PoolConfig{
			MaxNumberOfWorkers: 1,
			MinNumberOfWorkers: 1,
			QueueSize:          10,
		},
		Security: config.SecurityConfig{
			AllowedScriptPaths: []string{tmpDir},
		},
	}

	actions := []config.Action{
		{
			ID:         "nodejs_handler",
			Name:       "Node.js Handler",
			Type:       "script",
			SourceType: "local",
			Trigger: config.TriggerConfig{
				EventType: "alert.created",
			},
			Script: scriptPath,
			Parameters: map[string]string{
				"alert_id": "{{ data.id }}",
				"severity": "{{ data.severity }}",
			},
			Timeout: 5,
		},
	}

	apiClient := api.NewClient(cfg.Rootly.APIURL, cfg.Rootly.APIPath, cfg.Rootly.APIKey, "test")
	scriptRunner := executor.NewScriptRunner(cfg.Security.AllowedScriptPaths, nil)
	httpExecutor := executor.NewHTTPExecutor()
	rep := reporter.New(apiClient)
	exec := executor.New(actions, scriptRunner, httpExecutor, rep)
	pool := worker.NewPool(&cfg.Pool, exec)
	poll := poller.New(apiClient, &cfg.Poller, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool.Start(ctx)
	go poll.Start(ctx)

	fixture := loadEventFixture(t, "event_alert_for_script.json")
	require.Len(t, fixture.Events, 1)
	mockServer.AddEvent(fixture.Events[0])

	time.Sleep(1 * time.Second)

	executions := mockServer.GetExecutions()
	require.GreaterOrEqual(t, len(executions), 1)

	var completedExecution *api.ExecutionResult
	for i := range executions {
		if executions[i].ExecutionStatus == "completed" {
			completedExecution = &executions[i]
			break
		}
	}
	require.NotNil(t, completedExecution)

	assert.Equal(t, "completed", completedExecution.ExecutionStatus)
	assert.Equal(t, 0, completedExecution.ExecutionExitCode)
}

func TestEndToEnd_StandaloneActionTriggered(t *testing.T) {
	mockServer := NewMockAPIServer()
	defer mockServer.Close()

	webhookCalled := false
	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		webhookCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer webhookServer.Close()

	cfg := &config.Config{
		Rootly: config.RootlyConfig{
			APIURL:  mockServer.URL,
			APIPath: "/rec/v1",
			APIKey:  "test-key",
		},
		Poller: config.PollerConfig{
			PollingWaitIntervalMs: 100,
			MaxNumberOfMessages:   10,
		},
		Pool: config.PoolConfig{
			MaxNumberOfWorkers: 1,
			MinNumberOfWorkers: 1,
			QueueSize:          10,
		},
	}

	actions := []config.Action{
		{
			ID:   "clear_cache",
			Name: "Clear Cache",
			Type: "http",
			Trigger: config.TriggerConfig{
				EventType: "action.triggered",
			},
			HTTP: &config.HTTPAction{
				URL:    webhookServer.URL + "/cache/clear",
				Method: "POST",
			},
			Timeout: 5,
		},
	}

	apiClient := api.NewClient(cfg.Rootly.APIURL, cfg.Rootly.APIPath, cfg.Rootly.APIKey, "test")
	scriptRunner := executor.NewScriptRunner(nil, nil)
	httpExecutor := executor.NewHTTPExecutor()
	rep := reporter.New(apiClient)
	exec := executor.New(actions, scriptRunner, httpExecutor, rep)
	pool := worker.NewPool(&cfg.Pool, exec)
	poll := poller.New(apiClient, &cfg.Poller, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool.Start(ctx)
	go poll.Start(ctx)

	// Load standalone action triggered event
	fixture := loadEventFixture(t, "event_action_triggered_standalone.json")
	require.Len(t, fixture.Events, 1)
	mockServer.AddEvent(fixture.Events[0])

	time.Sleep(1 * time.Second)

	assert.True(t, webhookCalled, "Webhook should have been called")

	executions := mockServer.GetExecutions()
	require.GreaterOrEqual(t, len(executions), 1)

	var completedExecution *api.ExecutionResult
	for i := range executions {
		if executions[i].ExecutionStatus == "completed" {
			completedExecution = &executions[i]
			break
		}
	}
	require.NotNil(t, completedExecution)
	assert.Equal(t, "completed", completedExecution.ExecutionStatus)
}
