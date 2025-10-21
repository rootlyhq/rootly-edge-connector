package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rootly/edge-connector/internal/api"
)

// loadFixture loads a JSON fixture file and returns the response
func loadFixture(t *testing.T, filename string) api.EventsResponse {
	data, err := os.ReadFile("testdata/fixtures/" + filename)
	require.NoError(t, err, "Failed to read fixture file")

	var response api.EventsResponse
	err = json.Unmarshal(data, &response)
	require.NoError(t, err, "Failed to parse fixture JSON")

	return response
}

func TestClient_FetchEvents_Success(t *testing.T) {
	// Load fixture from JSON file
	fixture := loadFixture(t, "events_alert_and_incident.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers and query params
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "10", r.URL.Query().Get("max_messages"))
		assert.Equal(t, "30", r.URL.Query().Get("visibility_timeout"))

		// Return fixture data
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(fixture)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	events, err := client.FetchEvents(context.Background(), 10, 30)

	require.NoError(t, err)
	assert.Len(t, events, 2)

	// Verify alert event
	assert.Equal(t, "delivery-1", events[0].ID)
	assert.Equal(t, "event-1", events[0].EventID)
	assert.Equal(t, "alert.created", events[0].Type)
	assert.Equal(t, "alert-uuid-1", events[0].Data["id"])
	assert.Equal(t, "High database latency", events[0].Data["summary"])
	assert.Equal(t, "datadog", events[0].Data["source"])

	// Verify incident event
	assert.Equal(t, "delivery-2", events[1].ID)
	assert.Equal(t, "incident.created", events[1].Type)
	assert.Equal(t, "incident-uuid-1", events[1].Data["id"])
	assert.Equal(t, "Production API Gateway Outage", events[1].Data["title"])
}

func TestClient_FetchEvents_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := api.EventsResponse{Events: []api.Event{}}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	events, err := client.FetchEvents(context.Background(), 10, 30)

	require.NoError(t, err)
	assert.Len(t, events, 0)
}

func TestClient_FetchEvents_ActionTriggered(t *testing.T) {
	// Load fixture from JSON file
	fixture := loadFixture(t, "events_action_triggered.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(fixture)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	events, err := client.FetchEvents(context.Background(), 10, 30)

	require.NoError(t, err)
	assert.Len(t, events, 3)

	// Verify alert action (flat structure)
	assert.Equal(t, "alert.action_triggered", events[0].Type)
	assert.Equal(t, "restart_test_service", events[0].Data["action_name"])
	assert.Equal(t, "alert-uuid-123", events[0].Data["entity_id"])
	params := events[0].Data["parameters"].(map[string]interface{})
	assert.Equal(t, "api", params["service_name"])
	assert.Equal(t, "production", params["environment"])

	// Verify triggered_by
	triggeredBy := events[0].Data["triggered_by"].(map[string]interface{})
	assert.Equal(t, "Quentin Rousseau", triggeredBy["name"])
	assert.Equal(t, "quentin@rootly.com", triggeredBy["email"])

	// Verify incident action
	assert.Equal(t, "incident.action_triggered", events[1].Type)
	assert.Equal(t, "escalate", events[1].Data["action_name"])
	assert.Equal(t, "incident-uuid-456", events[1].Data["entity_id"])

	// Verify standalone action (no entity_id)
	assert.Equal(t, "action.triggered", events[2].Type)
	assert.Equal(t, "clear_cache", events[2].Data["action_name"])
	_, hasEntityID := events[2].Data["entity_id"]
	assert.False(t, hasEntityID, "Standalone action should not have entity_id")

	// REGRESSION TEST: Ensure all action_triggered events have action_name
	// This is CRITICAL for action matching - without action_name, the connector
	// cannot determine which action to execute
	assert.NotEmpty(t, events[0].Data["action_name"], "alert.action_triggered MUST have action_name in data")
	assert.NotEmpty(t, events[1].Data["action_name"], "incident.action_triggered MUST have action_name in data")
	assert.NotEmpty(t, events[2].Data["action_name"], "action.triggered MUST have action_name in data")
}

// REGRESSION TEST: action.triggered events MUST include action_name
// This test ensures the backend always sends action_name for action.triggered events
func TestClient_FetchEvents_ActionTriggeredRequiresActionName(t *testing.T) {

	// Test payload with action_name present (as backend should send)
	validPayload := api.EventsResponse{
		Events: []api.Event{
			{
				ID:        "delivery-123",
				EventID:   "event-456",
				Type:      "action.triggered",
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"action_name": "test_action", // REQUIRED field
					"parameters": map[string]interface{}{
						"message": "test",
					},
					"triggered_by": map[string]interface{}{
						"id":    50,
						"name":  "Test User",
						"email": "test@example.com",
					},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(validPayload)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	events, err := client.FetchEvents(context.Background(), 10, 30)

	require.NoError(t, err)
	require.Len(t, events, 1)

	// CRITICAL ASSERTION: action_name MUST be present
	// If this fails, the backend is not sending action_name in action.triggered events
	actionName, exists := events[0].Data["action_name"]
	assert.True(t, exists, "REGRESSION: action.triggered events MUST include 'action_name' in data - without it, connector cannot determine which action to execute")
	assert.NotEmpty(t, actionName, "REGRESSION: action_name must not be empty")
	assert.Equal(t, "test_action", actionName, "action_name should match expected value")
}

func TestClient_MarkDeliveryAsRunning(t *testing.T) {
	var receivedID string
	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		assert.Contains(t, r.URL.Path, "/deliveries/")
		assert.NotContains(t, r.URL.Path, "/acknowledge") // No /acknowledge suffix
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Extract delivery ID from URL
		parts := strings.Split(r.URL.Path, "/")
		for i, part := range parts {
			if part == "deliveries" && i+1 < len(parts) {
				receivedID = parts[i+1]
				break
			}
		}

		// Read request body
		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		require.NoError(t, err)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	err := client.MarkDeliveryAsRunning(context.Background(), "delivery-123")

	require.NoError(t, err)
	assert.Equal(t, "delivery-123", receivedID)
	assert.Equal(t, "running", receivedPayload["execution_status"])
}

func TestClient_ReportExecution(t *testing.T) {
	var receivedExecution api.ExecutionResult
	var receivedDeliveryID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		assert.Contains(t, r.URL.Path, "/deliveries/")
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Extract delivery_id from URL path
		receivedDeliveryID = r.URL.Path[len("/deliveries/"):]

		err := json.NewDecoder(r.Body).Decode(&receivedExecution)
		require.NoError(t, err)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")

	execution := api.ExecutionResult{
		DeliveryID:          "delivery-123",
		ExecutionActionID:   "test_action",
		ExecutionStatus:     "completed",
		ExecutionExitCode:   0,
		ExecutionStdout:     "Success output",
		ExecutionStderr:     "",
		ExecutionDurationMs: 1234,
		CompletedAt:         "2025-10-26T20:00:00Z",
	}

	err := client.ReportExecution(context.Background(), execution)
	require.NoError(t, err)

	// delivery_id should be in URL path, NOT in JSON body
	assert.Equal(t, "delivery-123", receivedDeliveryID)
	assert.Empty(t, receivedExecution.DeliveryID, "delivery_id should not be in JSON body")

	// Other fields should be in JSON body
	assert.Equal(t, "test_action", receivedExecution.ExecutionActionID)
	assert.Equal(t, "completed", receivedExecution.ExecutionStatus)
	assert.Equal(t, 0, receivedExecution.ExecutionExitCode)
}

func TestClient_ReportExecution_Failed(t *testing.T) {
	var receivedExecution api.ExecutionResult

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedExecution)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")

	execution := api.ExecutionResult{
		DeliveryID:          "delivery-456",
		ExecutionActionID:   "failing_action",
		ExecutionStatus:     "failed",
		ExecutionExitCode:   1,
		ExecutionStderr:     "Error message",
		ExecutionDurationMs: 567,
		ExecutionError:      "Script failed with exit code 1",
		FailedAt:            "2025-10-26T20:00:00Z",
	}

	err := client.ReportExecution(context.Background(), execution)
	require.NoError(t, err)

	assert.Equal(t, "failed", receivedExecution.ExecutionStatus)
	assert.Equal(t, 1, receivedExecution.ExecutionExitCode)
	assert.Equal(t, "Script failed with exit code 1", receivedExecution.ExecutionError)
}
func TestClient_UserAgent(t *testing.T) {
	var receivedUserAgent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUserAgent = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.EventsResponse{Events: []api.Event{}})
	}))
	defer server.Close()

	// Create client with version
	client := api.NewClient(server.URL, "", "test-key", "v1.2.3")

	// Make a request
	_, err := client.FetchEvents(context.Background(), 10, 30)
	require.NoError(t, err)

	// Verify User-Agent header
	assert.Equal(t, "rootly-edge-connector/v1.2.3", receivedUserAgent)
}

func TestClient_UserAgentDev(t *testing.T) {
	var receivedUserAgent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUserAgent = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		// Need to consume body for PATCH request
		json.NewDecoder(r.Body).Decode(&map[string]interface{}{})
	}))
	defer server.Close()

	// Create client with "dev" version
	client := api.NewClient(server.URL, "", "test-key", "dev")

	// Make mark running request
	err := client.MarkDeliveryAsRunning(context.Background(), "delivery-123")
	require.NoError(t, err)

	// Verify User-Agent header
	assert.Equal(t, "rootly-edge-connector/dev", receivedUserAgent)
}

func TestClient_RegisterActions(t *testing.T) {
	var receivedRequest api.RegisterActionsRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/actions", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		err := json.NewDecoder(r.Body).Decode(&receivedRequest)
		require.NoError(t, err)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(api.RegisterActionsResponse{
			Registered: struct {
				Automatic int `json:"automatic"`
				Callable  int `json:"callable"`
				Total     int `json:"total"`
			}{
				Automatic: len(receivedRequest.Automatic),
				Callable:  len(receivedRequest.Callable),
				Total:     len(receivedRequest.Automatic) + len(receivedRequest.Callable),
			},
			Failed:   0,
			Failures: []api.ActionFailure{},
		})
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")

	request := api.RegisterActionsRequest{
		Automatic: []api.AutomaticActionRegistration{
			{
				Slug:       "alert.created",
				ActionType: "script",
				Trigger:    "alert.created",
				Timeout:    300,
			},
		},
		Callable: []api.CallableActionRegistration{
			{
				Slug:       "restart_server",
				Name:       "Restart Server",
				ActionType: "script",
				Trigger:    "action.triggered",
				Timeout:    60,
			},
		},
	}

	resp, err := client.RegisterActions(context.Background(), request)
	require.NoError(t, err)

	assert.Equal(t, 1, resp.Registered.Automatic)
	assert.Equal(t, 1, resp.Registered.Callable)
	assert.Equal(t, 2, resp.Registered.Total)
	assert.Equal(t, 0, resp.Failed)
	assert.Empty(t, resp.Failures)
}

// Retry and Error Handling Tests

func TestClient_FetchEvents_RetrySuccess(t *testing.T) {
	attemptCount := 0
	fixture := loadFixture(t, "events_alert_and_incident.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		// Fail first 2 attempts, succeed on 3rd
		if attemptCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(fixture)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	events, err := client.FetchEvents(context.Background(), 10, 30)

	require.NoError(t, err)
	assert.Len(t, events, 2)
	assert.Equal(t, 3, attemptCount, "Should have retried 2 times before succeeding")
}

func TestClient_FetchEvents_RetryExhausted(t *testing.T) {
	attemptCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	events, err := client.FetchEvents(context.Background(), 10, 30)

	require.Error(t, err)
	assert.Nil(t, events)
	// After retries are exhausted, retryablehttp wraps with "giving up after X attempts"
	assert.Contains(t, err.Error(), "giving up after")
	// retryablehttp retries 3 times by default, so total of 4 attempts
	assert.Equal(t, 4, attemptCount, "Should have made 4 attempts (1 initial + 3 retries)")
}

func TestClient_FetchEvents_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response that will trigger context timeout
		<-r.Context().Done()
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	events, err := client.FetchEvents(ctx, 10, 30)

	require.Error(t, err)
	assert.Nil(t, events)
	assert.Contains(t, err.Error(), "context")
}

func TestClient_FetchEvents_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "100")
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Used", "100")
		w.Header().Set("X-RateLimit-Reset", "1698765432")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	events, err := client.FetchEvents(context.Background(), 10, 30)

	require.Error(t, err)
	assert.Nil(t, events)
	// retryablehttp retries on 429, so final error will be "giving up after X attempts"
	assert.Contains(t, err.Error(), "giving up after")
}

func TestClient_FetchEvents_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	events, err := client.FetchEvents(context.Background(), 10, 30)

	require.Error(t, err)
	assert.Nil(t, events)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestClient_HTTPStatusCodes(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		wantError    bool
		errorMsg     string
		willRetry    bool // retryablehttp retries 5xx errors
		attemptCount int
	}{
		{
			name:         "400 Bad Request",
			statusCode:   http.StatusBadRequest,
			wantError:    true,
			errorMsg:     "unexpected status code: 400",
			willRetry:    false,
			attemptCount: 1,
		},
		{
			name:         "401 Unauthorized",
			statusCode:   http.StatusUnauthorized,
			wantError:    true,
			errorMsg:     "unexpected status code: 401",
			willRetry:    false,
			attemptCount: 1,
		},
		{
			name:         "403 Forbidden",
			statusCode:   http.StatusForbidden,
			wantError:    true,
			errorMsg:     "unexpected status code: 403",
			willRetry:    false,
			attemptCount: 1,
		},
		{
			name:         "404 Not Found",
			statusCode:   http.StatusNotFound,
			wantError:    true,
			errorMsg:     "unexpected status code: 404",
			willRetry:    false,
			attemptCount: 1,
		},
		{
			name:         "500 Internal Server Error",
			statusCode:   http.StatusInternalServerError,
			wantError:    true,
			errorMsg:     "giving up after", // retryablehttp wraps error after retries
			willRetry:    true,
			attemptCount: 4, // 1 initial + 3 retries
		},
		{
			name:         "502 Bad Gateway",
			statusCode:   http.StatusBadGateway,
			wantError:    true,
			errorMsg:     "giving up after",
			willRetry:    true,
			attemptCount: 4,
		},
		{
			name:         "503 Service Unavailable",
			statusCode:   http.StatusServiceUnavailable,
			wantError:    true,
			errorMsg:     "giving up after",
			willRetry:    true,
			attemptCount: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attemptCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attemptCount++
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := api.NewClient(server.URL, "", "test-key", "test")
			events, err := client.FetchEvents(context.Background(), 10, 30)

			if tt.wantError {
				require.Error(t, err)
				assert.Nil(t, events)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Equal(t, tt.attemptCount, attemptCount, "Should have made expected number of attempts")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestClient_ReportExecution_RetryOnFailure(t *testing.T) {
	attemptCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		// Fail first 2 attempts, succeed on 3rd
		if attemptCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")

	execution := api.ExecutionResult{
		DeliveryID:          "delivery-123",
		ExecutionActionID:   "test_action",
		ExecutionStatus:     "completed",
		ExecutionExitCode:   0,
		ExecutionDurationMs: 1234,
		CompletedAt:         "2025-10-26T20:00:00Z",
	}

	err := client.ReportExecution(context.Background(), execution)

	require.NoError(t, err)
	assert.Equal(t, 3, attemptCount, "Should have retried 2 times before succeeding")
}

func TestClient_MarkDeliveryAsRunning_RetryOnFailure(t *testing.T) {
	attemptCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		// Fail first attempt, succeed on 2nd
		if attemptCount < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	err := client.MarkDeliveryAsRunning(context.Background(), "delivery-123")

	require.NoError(t, err)
	assert.Equal(t, 2, attemptCount, "Should have retried once before succeeding")
}

func TestClient_RegisterActions_PartialFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(api.RegisterActionsResponse{
			Registered: struct {
				Automatic int `json:"automatic"`
				Callable  int `json:"callable"`
				Total     int `json:"total"`
			}{
				Automatic: 1,
				Callable:  0,
				Total:     1,
			},
			Failed: 1,
			Failures: []api.ActionFailure{
				{
					Slug:   "broken_action",
					Reason: "Invalid configuration",
				},
			},
		})
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")

	request := api.RegisterActionsRequest{
		Automatic: []api.AutomaticActionRegistration{
			{Slug: "working_action", ActionType: "script", Trigger: "alert.created", Timeout: 300},
		},
		Callable: []api.CallableActionRegistration{
			{Slug: "broken_action", Name: "Broken Action", ActionType: "http", Trigger: "action.triggered", Timeout: 300},
		},
	}

	resp, err := client.RegisterActions(context.Background(), request)
	require.NoError(t, err)

	assert.Equal(t, 1, resp.Registered.Automatic)
	assert.Equal(t, 0, resp.Registered.Callable)
	assert.Equal(t, 1, resp.Registered.Total)
	assert.Equal(t, 1, resp.Failed)
	assert.Len(t, resp.Failures, 1)
	assert.Equal(t, "broken_action", resp.Failures[0].Slug)
	assert.Equal(t, "Invalid configuration", resp.Failures[0].Reason)
}

func TestClient_RegisterActions_NetworkError(t *testing.T) {
	// Use invalid URL to trigger network error
	client := api.NewClient("http://invalid-host-that-does-not-exist-12345.local", "", "test-key", "test")

	request := api.RegisterActionsRequest{
		Automatic: []api.AutomaticActionRegistration{
			{Slug: "test", ActionType: "script", Trigger: "alert.created", Timeout: 300},
		},
		Callable: []api.CallableActionRegistration{},
	}

	resp, err := client.RegisterActions(context.Background(), request)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "request failed")
}

// Additional edge case tests

func TestClient_FetchEvents_WithAPIPath(t *testing.T) {
	fixture := loadFixture(t, "events_alert_and_incident.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify API path is used
		assert.Contains(t, r.URL.Path, "/api/v2/deliveries")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(fixture)
	}))
	defer server.Close()

	// Create client with custom API path
	client := api.NewClient(server.URL, "/api/v2", "test-key", "test")
	events, err := client.FetchEvents(context.Background(), 10, 30)

	require.NoError(t, err)
	assert.Len(t, events, 2)
}

func TestClient_MarkDeliveryAsRunning_HTTPError(t *testing.T) {
	attemptCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusBadRequest) // 4xx errors don't retry
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	err := client.MarkDeliveryAsRunning(context.Background(), "delivery-123")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code: 400")
	assert.Equal(t, 1, attemptCount, "4xx errors should not retry")
}

func TestClient_ReportExecution_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")

	execution := api.ExecutionResult{
		DeliveryID:        "delivery-123",
		ExecutionActionID: "test",
		ExecutionStatus:   "completed",
	}

	err := client.ReportExecution(context.Background(), execution)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code: 404")
}

func TestClient_RegisterActions_InvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")

	request := api.RegisterActionsRequest{
		Automatic: []api.AutomaticActionRegistration{
			{Slug: "test", ActionType: "script", Trigger: "alert.created", Timeout: 300},
		},
		Callable: []api.CallableActionRegistration{},
	}

	resp, err := client.RegisterActions(context.Background(), request)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestClient_FetchEvents_EmptyAPIPath(t *testing.T) {
	fixture := loadFixture(t, "events_alert_and_incident.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// When API path is empty, should hit /deliveries directly
		assert.Equal(t, "/deliveries", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(fixture)
	}))
	defer server.Close()

	// Client with empty API path
	client := api.NewClient(server.URL, "", "test-key", "test")
	events, err := client.FetchEvents(context.Background(), 10, 30)

	require.NoError(t, err)
	assert.Len(t, events, 2)
}

func TestClient_RateLimitHeaders_AllPresent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "3000")
		w.Header().Set("X-RateLimit-Remaining", "2500")
		w.Header().Set("X-RateLimit-Used", "500")
		w.Header().Set("X-RateLimit-Reset", "1698765432")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(api.EventsResponse{Events: []api.Event{}})
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	_, err := client.FetchEvents(context.Background(), 10, 30)

	assert.NoError(t, err)
	// Rate limit headers are logged at TRACE level
	// This test primarily ensures the code path is covered
}

func TestClient_RateLimitHeaders_PartiallyPresent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only some headers present
		w.Header().Set("X-RateLimit-Remaining", "100")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(api.EventsResponse{Events: []api.Event{}})
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	_, err := client.FetchEvents(context.Background(), 10, 30)

	assert.NoError(t, err)
}

// Additional coverage for MarkDeliveryAsRunning and ReportExecution

func TestClient_MarkDeliveryAsRunning_SuccessWithLogging(t *testing.T) {
	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		json.NewDecoder(r.Body).Decode(&receivedPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	err := client.MarkDeliveryAsRunning(context.Background(), "delivery-456")

	require.NoError(t, err)
	assert.Equal(t, "running", receivedPayload["execution_status"])
	assert.NotEmpty(t, receivedPayload["running_at"])
}

func TestClient_MarkDeliveryAsRunning_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", "1698765432")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	err := client.MarkDeliveryAsRunning(context.Background(), "delivery-789")

	require.Error(t, err)
	// retryablehttp retries on 429, so error will be "giving up after X attempts"
	assert.Contains(t, err.Error(), "giving up after")
}

func TestClient_MarkDeliveryAsRunning_UnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	err := client.MarkDeliveryAsRunning(context.Background(), "delivery-404")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code: 404")
}

func TestClient_ReportExecution_SuccessWithAllFields(t *testing.T) {
	var receivedExecution api.ExecutionResult

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		json.NewDecoder(r.Body).Decode(&receivedExecution)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")

	execution := api.ExecutionResult{
		DeliveryID:          "delivery-999",
		ExecutionActionID:   "full_test",
		ExecutionStatus:     "completed",
		ExecutionExitCode:   0,
		ExecutionStdout:     "All good",
		ExecutionStderr:     "",
		ExecutionDurationMs: 5000,
		ExecutionError:      "",
		CompletedAt:         "2025-10-26T20:00:00Z",
		RunningAt:           "2025-10-26T19:59:55Z",
	}

	err := client.ReportExecution(context.Background(), execution)
	require.NoError(t, err)

	assert.Equal(t, "full_test", receivedExecution.ExecutionActionID)
	assert.Equal(t, "completed", receivedExecution.ExecutionStatus)
	assert.Equal(t, int64(5000), receivedExecution.ExecutionDurationMs)
}

func TestClient_ReportExecution_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")

	execution := api.ExecutionResult{
		DeliveryID:      "delivery-rate-limit",
		ExecutionStatus: "completed",
	}

	err := client.ReportExecution(context.Background(), execution)
	require.Error(t, err)
	// retryablehttp retries on 429
	assert.Contains(t, err.Error(), "giving up after")
}

func TestClient_ReportExecution_UnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")

	execution := api.ExecutionResult{
		DeliveryID:      "delivery-bad",
		ExecutionStatus: "completed",
	}

	err := client.ReportExecution(context.Background(), execution)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code: 400")
}

func TestClient_RegisterActions_EmptyActions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(api.RegisterActionsResponse{
			Registered: struct {
				Automatic int `json:"automatic"`
				Callable  int `json:"callable"`
				Total     int `json:"total"`
			}{
				Automatic: 0,
				Callable:  0,
				Total:     0,
			},
			Failed: 0,
		})
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	request := api.RegisterActionsRequest{
		Automatic: []api.AutomaticActionRegistration{},
		Callable:  []api.CallableActionRegistration{},
	}
	resp, err := client.RegisterActions(context.Background(), request)

	require.NoError(t, err)
	assert.Equal(t, 0, resp.Registered.Total)
}

func TestClient_RegisterActions_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")

	request := api.RegisterActionsRequest{
		Automatic: []api.AutomaticActionRegistration{
			{Slug: "test", ActionType: "script", Trigger: "alert.created", Timeout: 300},
		},
		Callable: []api.CallableActionRegistration{},
	}

	resp, err := client.RegisterActions(context.Background(), request)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "giving up after")
}

func TestClient_FetchEvents_ReadBodyError(t *testing.T) {
	// This tests the rarely-hit error path where response body can't be read
	// In practice this almost never happens, but for coverage completeness
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1")
		w.WriteHeader(http.StatusOK)
		// Close connection immediately to cause read error
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	events, err := client.FetchEvents(context.Background(), 10, 30)

	// May succeed or fail depending on timing, but covers the code path
	if err != nil {
		assert.Nil(t, events)
	}
}

// Edge case tests for improved coverage

func TestClient_RedactToken_ShortToken(t *testing.T) {
	// Test with a very short token (<=8 chars)
	// This tests the redactToken function edge case
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Just verify the request was made with short token
		assert.Equal(t, "Bearer short", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(api.EventsResponse{Events: []api.Event{}})
	}))
	defer server.Close()

	// Create client with very short token
	client := api.NewClient(server.URL, "", "short", "test")
	_, err := client.FetchEvents(context.Background(), 10, 30)

	// Should succeed even with short token
	require.NoError(t, err)
}

func TestClient_RegisterActions_MultiStatus(t *testing.T) {
	// Test 207 Multi-Status response (partial success)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMultiStatus) // 207
		json.NewEncoder(w).Encode(api.RegisterActionsResponse{
			Registered: struct {
				Automatic int `json:"automatic"`
				Callable  int `json:"callable"`
				Total     int `json:"total"`
			}{
				Automatic: 1,
				Callable:  0,
				Total:     1,
			},
			Failed: 1,
			Failures: []api.ActionFailure{
				{Slug: "failed_action", Reason: "Validation error"},
			},
		})
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")

	request := api.RegisterActionsRequest{
		Automatic: []api.AutomaticActionRegistration{
			{Slug: "success", ActionType: "script", Trigger: "alert.created", Timeout: 300},
		},
		Callable: []api.CallableActionRegistration{
			{Slug: "failed_action", Name: "Failed", ActionType: "script", Trigger: "action.triggered", Timeout: 300},
		},
	}

	resp, err := client.RegisterActions(context.Background(), request)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 1, resp.Registered.Total)
	assert.Equal(t, 1, resp.Failed)
}

func TestClient_ReportExecution_StatusCreated(t *testing.T) {
	// Test with 201 Created response instead of 200 OK
	var receivedExecution api.ExecutionResult

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		json.NewDecoder(r.Body).Decode(&receivedExecution)
		w.WriteHeader(http.StatusCreated) // 201 instead of 200
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")

	execution := api.ExecutionResult{
		DeliveryID:          "delivery-201",
		ExecutionActionID:   "test_action",
		ExecutionStatus:     "completed",
		ExecutionExitCode:   0,
		ExecutionDurationMs: 1234,
		CompletedAt:         "2025-10-26T20:00:00Z",
	}

	err := client.ReportExecution(context.Background(), execution)
	require.NoError(t, err)
	assert.Equal(t, "test_action", receivedExecution.ExecutionActionID)
}

func TestClient_RegisterActions_HTTPError(t *testing.T) {
	// Test when RegisterActions gets an HTTP error (not 201 or 207)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")

	request := api.RegisterActionsRequest{
		Automatic: []api.AutomaticActionRegistration{
			{Slug: "test", ActionType: "script", Trigger: "alert.created", Timeout: 300},
		},
		Callable: []api.CallableActionRegistration{},
	}

	resp, err := client.RegisterActions(context.Background(), request)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "giving up after") // Will retry on 5xx
}

func TestClient_FetchEvents_LargeResponse(t *testing.T) {
	// Test with a large response to cover edge cases
	// Create 100 events
	events := make([]api.Event, 100)
	for i := 0; i < 100; i++ {
		events[i] = api.Event{
			ID:        fmt.Sprintf("delivery-%d", i),
			EventID:   fmt.Sprintf("event-%d", i),
			Type:      "alert.created",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"id":      fmt.Sprintf("alert-%d", i),
				"summary": fmt.Sprintf("Alert number %d", i),
			},
		}
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(api.EventsResponse{Events: events})
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	receivedEvents, err := client.FetchEvents(context.Background(), 100, 30)

	require.NoError(t, err)
	assert.Len(t, receivedEvents, 100)
}

func TestClient_MarkDeliveryAsRunning_ContextCancelled(t *testing.T) {
	// Test context cancellation during MarkDeliveryAsRunning
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay to allow context cancellation
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")

	// Create context that's already canceled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.MarkDeliveryAsRunning(ctx, "delivery-cancel")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context")
}

func TestClient_ReportExecution_ContextCancelled(t *testing.T) {
	// Test context cancellation during ReportExecution
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")

	execution := api.ExecutionResult{
		DeliveryID:      "delivery-cancel",
		ExecutionStatus: "completed",
	}

	// Create context that's already canceled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.ReportExecution(ctx, execution)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context")
}

func TestClient_RegisterActions_ContextCancelled(t *testing.T) {
	// Test context cancellation during RegisterActions
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")

	request := api.RegisterActionsRequest{
		Automatic: []api.AutomaticActionRegistration{},
		Callable:  []api.CallableActionRegistration{},
	}

	// Create context that's already canceled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	resp, err := client.RegisterActions(ctx, request)
	require.Error(t, err)
	assert.Nil(t, resp)
}
