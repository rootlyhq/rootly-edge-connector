package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/rootly/edge-connector/internal/api"
	"github.com/rootly/edge-connector/internal/config"
)

func TestHTTPExecutor_AutoBuildBody(t *testing.T) {
	var receivedBody map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	executor := NewHTTPExecutor()
	action := &config.Action{
		Name: "test_http",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    server.URL,
			Method: "POST",
		},
		Timeout: 10,
		// No body template - should auto-build from params
	}

	event := api.Event{
		Data: map[string]interface{}{},
	}

	params := map[string]string{
		"message":  "Hello",
		"severity": "info",
		"count":    "42",
	}

	result := executor.Execute(context.Background(), action, event, params)

	assert.Equal(t, 200, result.ExitCode)
	assert.Nil(t, result.Error)
	assert.Equal(t, "Hello", receivedBody["message"])
	assert.Equal(t, "info", receivedBody["severity"])
	assert.Equal(t, "42", receivedBody["count"])
}

func TestHTTPExecutor_CustomBodyTemplate(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	executor := NewHTTPExecutor()
	action := &config.Action{
		Name: "test_http",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    server.URL,
			Method: "POST",
			Body:   `{"custom": "template", "message": "{{ message }}"}`,
		},
		Timeout: 10,
	}

	event := api.Event{
		Data: map[string]interface{}{
			"message": "test",
		},
	}

	params := map[string]string{
		"ignored": "value", // Should be ignored since custom body is used
	}

	result := executor.Execute(context.Background(), action, event, params)

	assert.Equal(t, 200, result.ExitCode)
	// Custom body should be used, not auto-built from params
	assert.Equal(t, "template", receivedBody["custom"])
	assert.Equal(t, "test", receivedBody["message"])
	assert.NotContains(t, receivedBody, "ignored")
}

func TestHTTPExecutor_EmptyParamsNoBody(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	executor := NewHTTPExecutor()
	action := &config.Action{
		Name: "test_http",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    server.URL,
			Method: "POST",
		},
		Timeout: 10,
	}

	event := api.Event{Data: map[string]interface{}{}}
	params := map[string]string{} // Empty params

	result := executor.Execute(context.Background(), action, event, params)

	assert.Equal(t, 200, result.ExitCode)
	// Should have no body
	assert.Empty(t, receivedBody)
}

// Advanced HTTP executor tests

func TestHTTPExecutor_DifferentHTTPMethods(t *testing.T) {
	tests := []struct {
		method string
	}{
		{"GET"},
		{"POST"},
		{"PUT"},
		{"PATCH"},
		{"DELETE"},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			var receivedMethod string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedMethod = r.Method
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			executor := NewHTTPExecutor()
			action := &config.Action{
				Type: "http",
				HTTP: &config.HTTPAction{
					URL:    server.URL,
					Method: tt.method,
				},
				Timeout: 10,
			}

			result := executor.Execute(context.Background(), action, api.Event{}, nil)

			assert.Equal(t, 200, result.ExitCode)
			assert.Equal(t, tt.method, receivedMethod)
		})
	}
}

func TestHTTPExecutor_HTTPStatusCodes(t *testing.T) {
	tests := []struct {
		statusCode   int
		expectError  bool
		expectedExit int
	}{
		{200, false, 200},
		{201, false, 201},
		{204, false, 204},
		{400, true, 400},
		{401, true, 401},
		{403, true, 403},
		{404, true, 404},
		{500, true, 500},
		{502, true, 502},
		{503, true, 503},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.statusCode), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			executor := NewHTTPExecutor()
			action := &config.Action{
				Type: "http",
				HTTP: &config.HTTPAction{
					URL:    server.URL,
					Method: "GET",
				},
				Timeout: 10,
			}

			result := executor.Execute(context.Background(), action, api.Event{}, nil)

			assert.Equal(t, tt.expectedExit, result.ExitCode)
			if tt.expectError {
				assert.NotNil(t, result.Error)
				assert.Contains(t, result.Error.Error(), fmt.Sprintf("HTTP %d", tt.statusCode))
			} else {
				assert.Nil(t, result.Error)
			}
		})
	}
}

func TestHTTPExecutor_URLTemplate(t *testing.T) {
	var receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	executor := NewHTTPExecutor()
	action := &config.Action{
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    server.URL + "/api/{{ service_name }}/restart",
			Method: "POST",
		},
		Timeout: 10,
	}

	event := api.Event{
		Data: map[string]interface{}{
			"service_name": "database",
		},
	}

	result := executor.Execute(context.Background(), action, event, nil)

	assert.Equal(t, 200, result.ExitCode)
	assert.Equal(t, "/api/database/restart", receivedPath)
}

func TestHTTPExecutor_HeaderTemplates(t *testing.T) {
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	executor := NewHTTPExecutor()
	action := &config.Action{
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    server.URL,
			Method: "POST",
			Headers: map[string]string{
				"X-Alert-Source":   "{{ source }}",
				"X-Alert-Severity": "{{ severity }}",
				"Authorization":    "Bearer {{ env.API_TOKEN }}",
			},
		},
		Timeout: 10,
	}

	// Set environment variable for test
	os.Setenv("API_TOKEN", "test-token-123")
	defer os.Unsetenv("API_TOKEN")

	event := api.Event{
		Data: map[string]interface{}{
			"source":   "datadog",
			"severity": "critical",
		},
	}

	result := executor.Execute(context.Background(), action, event, nil)

	assert.Equal(t, 200, result.ExitCode)
	assert.Equal(t, "datadog", receivedHeaders.Get("X-Alert-Source"))
	assert.Equal(t, "critical", receivedHeaders.Get("X-Alert-Severity"))
	assert.Equal(t, "Bearer test-token-123", receivedHeaders.Get("Authorization"))
}

func TestHTTPExecutor_QueryParameters(t *testing.T) {
	var receivedQuery url.Values

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.Query()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	executor := NewHTTPExecutor()
	action := &config.Action{
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    server.URL,
			Method: "GET",
			Params: map[string]string{
				"service":     "{{ service }}",
				"environment": "{{ environment }}",
				"severity":    "{{ severity.slug }}",
			},
		},
		Timeout: 10,
	}

	event := api.Event{
		Data: map[string]interface{}{
			"service":     "api-gateway",
			"environment": "production",
			"severity": map[string]interface{}{
				"slug": "sev1",
			},
		},
	}

	result := executor.Execute(context.Background(), action, event, nil)

	assert.Equal(t, 200, result.ExitCode)
	assert.Equal(t, "api-gateway", receivedQuery.Get("service"))
	assert.Equal(t, "production", receivedQuery.Get("environment"))
	assert.Equal(t, "sev1", receivedQuery.Get("severity"))
}

func TestHTTPExecutor_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow endpoint
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	executor := NewHTTPExecutor()
	action := &config.Action{
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    server.URL,
			Method: "GET",
		},
		Timeout: 1, // 1 second timeout
	}

	result := executor.Execute(context.Background(), action, api.Event{}, nil)

	assert.Equal(t, 1, result.ExitCode)
	assert.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "context deadline exceeded")
}

func TestHTTPExecutor_NetworkError(t *testing.T) {
	executor := NewHTTPExecutor()
	action := &config.Action{
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "http://invalid-host-that-does-not-exist-12345.local",
			Method: "GET",
		},
		Timeout: 5,
	}

	result := executor.Execute(context.Background(), action, api.Event{}, nil)

	assert.Equal(t, 1, result.ExitCode)
	assert.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "HTTP request failed")
}

func TestHTTPExecutor_MissingHTTPConfig(t *testing.T) {
	executor := NewHTTPExecutor()
	action := &config.Action{
		Type: "http",
		HTTP: nil, // Missing config
	}

	result := executor.Execute(context.Background(), action, api.Event{}, nil)

	assert.Equal(t, 1, result.ExitCode)
	assert.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "HTTP action configuration is missing")
}

func TestHTTPExecutor_InvalidURLTemplate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	executor := NewHTTPExecutor()
	action := &config.Action{
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    server.URL + "/{{ field | unknown_filter }}",
			Method: "GET",
		},
		Timeout: 10,
	}

	event := api.Event{Data: map[string]interface{}{}}

	result := executor.Execute(context.Background(), action, event, nil)

	assert.Equal(t, 1, result.ExitCode)
	assert.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "failed to render URL")
}

func TestHTTPExecutor_InvalidBodyTemplate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	executor := NewHTTPExecutor()
	action := &config.Action{
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    server.URL,
			Method: "POST",
			Body:   "{{ field | unknown_filter }}",
		},
		Timeout: 10,
	}

	event := api.Event{Data: map[string]interface{}{}}

	result := executor.Execute(context.Background(), action, event, nil)

	assert.Equal(t, 1, result.ExitCode)
	assert.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "failed to render body")
}

func TestHTTPExecutor_InvalidHeaderTemplate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	executor := NewHTTPExecutor()
	action := &config.Action{
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    server.URL,
			Method: "GET",
			Headers: map[string]string{
				"X-Custom": "{{ field | unknown_filter }}",
			},
		},
		Timeout: 10,
	}

	event := api.Event{Data: map[string]interface{}{}}

	result := executor.Execute(context.Background(), action, event, nil)

	assert.Equal(t, 1, result.ExitCode)
	assert.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "failed to render header")
}

func TestHTTPExecutor_InvalidParamTemplate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	executor := NewHTTPExecutor()
	action := &config.Action{
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    server.URL,
			Method: "GET",
			Params: map[string]string{
				"param1": "{{ field | unknown_filter }}",
			},
		},
		Timeout: 10,
	}

	event := api.Event{Data: map[string]interface{}{}}

	result := executor.Execute(context.Background(), action, event, nil)

	assert.Equal(t, 1, result.ExitCode)
	assert.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "failed to render param")
}

func TestHTTPExecutor_ComplexBodyTemplate(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	executor := NewHTTPExecutor()
	action := &config.Action{
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    server.URL,
			Method: "POST",
			Body: `{
				"alert_id": "{{ id }}",
				"source": "{{ source }}",
				"severity": "{{ severity.slug }}",
				"title": "{{ title }}"
			}`,
		},
		Timeout: 10,
	}

	event := api.Event{
		Data: map[string]interface{}{
			"id":     "alert-123",
			"source": "datadog",
			"title":  "High CPU Usage",
			"severity": map[string]interface{}{
				"slug": "critical",
			},
		},
	}

	result := executor.Execute(context.Background(), action, event, nil)

	assert.Equal(t, 200, result.ExitCode)
	assert.Equal(t, "alert-123", receivedBody["alert_id"])
	assert.Equal(t, "datadog", receivedBody["source"])
	assert.Equal(t, "critical", receivedBody["severity"])
	assert.Equal(t, "High CPU Usage", receivedBody["title"])
}

func TestHTTPExecutor_ResponseWithBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success", "message": "Action completed"}`))
	}))
	defer server.Close()

	executor := NewHTTPExecutor()
	action := &config.Action{
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    server.URL,
			Method: "GET",
		},
		Timeout: 10,
	}

	result := executor.Execute(context.Background(), action, api.Event{}, nil)

	assert.Equal(t, 200, result.ExitCode)
	assert.Contains(t, result.Stdout, "status")
	assert.Contains(t, result.Stdout, "success")
}

func TestHTTPExecutor_DefaultMethodPOST(t *testing.T) {
	var receivedMethod string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	executor := NewHTTPExecutor()
	action := &config.Action{
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    server.URL,
			Method: "", // Empty, should default to POST
		},
		Timeout: 10,
	}

	result := executor.Execute(context.Background(), action, api.Event{}, nil)

	assert.Equal(t, 200, result.ExitCode)
	assert.Equal(t, "POST", receivedMethod)
}

func TestHTTPExecutor_AuthHeaders(t *testing.T) {
	var receivedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	executor := NewHTTPExecutor()
	action := &config.Action{
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    server.URL,
			Method: "GET",
			Headers: map[string]string{
				"Authorization": "Bearer secret-token",
			},
		},
		Timeout: 10,
	}

	result := executor.Execute(context.Background(), action, api.Event{}, nil)

	assert.Equal(t, 200, result.ExitCode)
	assert.Equal(t, "Bearer secret-token", receivedAuth)
}

func TestHTTPExecutor_ActionTriggeredWithParameters(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	executor := NewHTTPExecutor()
	action := &config.Action{
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    server.URL,
			Method: "POST",
			Body: `{
				"message": "{{ parameters.message }}",
				"severity": "{{ parameters.severity }}",
				"triggered_by": "{{ triggered_by.email }}"
			}`,
		},
		Timeout: 10,
	}

	// Simulate action.triggered event payload
	event := api.Event{
		Data: map[string]interface{}{
			"action_name": "test_manual_action_http",
			"parameters": map[string]interface{}{
				"message":  "BOOP",
				"severity": "info",
			},
			"triggered_by": map[string]interface{}{
				"id":    50,
				"name":  "Quentin Rousseau",
				"email": "quentin@rootly.com",
			},
		},
	}

	result := executor.Execute(context.Background(), action, event, nil)

	assert.Equal(t, 200, result.ExitCode)
	assert.Nil(t, result.Error)
	assert.Equal(t, "BOOP", receivedBody["message"])
	assert.Equal(t, "info", receivedBody["severity"])
	assert.Equal(t, "quentin@rootly.com", receivedBody["triggered_by"])
}

func TestHTTPExecutor_AlertActionTriggeredWithEntity(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	executor := NewHTTPExecutor()
	action := &config.Action{
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    server.URL,
			Method: "POST",
			Body: `{
				"service_name": "{{ parameters.service_name }}",
				"environment": "{{ parameters.environment }}",
				"entity_id": "{{ entity_id }}",
				"triggered_by": "{{ triggered_by.email }}"
			}`,
		},
		Timeout: 10,
	}

	// Simulate alert.action_triggered event payload
	event := api.Event{
		Data: map[string]interface{}{
			"action_name": "restart_service",
			"entity_id":   "alert-uuid-123",
			"parameters": map[string]interface{}{
				"service_name": "api-gateway",
				"environment":  "production",
			},
			"triggered_by": map[string]interface{}{
				"id":    50,
				"name":  "Quentin Rousseau",
				"email": "quentin@rootly.com",
			},
		},
	}

	result := executor.Execute(context.Background(), action, event, nil)

	assert.Equal(t, 200, result.ExitCode)
	assert.Nil(t, result.Error)
	assert.Equal(t, "api-gateway", receivedBody["service_name"])
	assert.Equal(t, "production", receivedBody["environment"])
	assert.Equal(t, "alert-uuid-123", receivedBody["entity_id"])
	assert.Equal(t, "quentin@rootly.com", receivedBody["triggered_by"])
}

// Edge case tests for improved coverage

func TestHTTPExecutor_DefaultTimeout(t *testing.T) {
	// Test default timeout when Timeout=0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	executor := NewHTTPExecutor()

	action := &config.Action{
		ID:   "timeout_test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    server.URL,
			Method: "GET",
		},
		Timeout: 0, // Should use default 30s
	}

	event := api.Event{
		Data: map[string]interface{}{},
	}

	result := executor.Execute(context.Background(), action, event, nil)

	assert.Nil(t, result.Error)
	assert.Equal(t, 200, result.ExitCode)
}

func TestHTTPExecutor_TemplateWithActionMetadata(t *testing.T) {
	// Test HTTP templates with action metadata
	var receivedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	executor := NewHTTPExecutor()

	action := &config.Action{
		ID:   "action_meta_test",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    server.URL,
			Method: "POST",
			Headers: map[string]string{
				"X-Action-ID":   "{{ action.id }}",
				"X-Action-Name": "{{ action.name }}",
				"X-Action-Slug": "{{ action.slug }}",
			},
		},
	}

	event := api.Event{
		Data: map[string]interface{}{},
		Action: &api.ActionMetadata{
			ID:   "action-uuid-456",
			Name: "Restart Service",
			Slug: "restart_service",
		},
	}

	result := executor.Execute(context.Background(), action, event, nil)

	assert.Nil(t, result.Error)
	assert.Equal(t, "action-uuid-456", receivedHeaders.Get("X-Action-ID"))
	assert.Equal(t, "Restart Service", receivedHeaders.Get("X-Action-Name"))
	assert.Equal(t, "restart_service", receivedHeaders.Get("X-Action-Slug"))
}

func TestHTTPExecutor_InvalidURLAfterRender(t *testing.T) {
	// Test URL parse error
	executor := NewHTTPExecutor()

	action := &config.Action{
		ID:   "invalid_url",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "://invalid-scheme",
			Method: "GET",
		},
	}

	event := api.Event{
		Data: map[string]interface{}{},
	}

	result := executor.Execute(context.Background(), action, event, nil)

	assert.Equal(t, 1, result.ExitCode)
	assert.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "invalid URL")
}

func TestHTTPExecutor_RequestCreationError(t *testing.T) {
	// Test request creation error
	executor := NewHTTPExecutor()

	action := &config.Action{
		ID:   "bad_request",
		Type: "http",
		HTTP: &config.HTTPAction{
			URL:    "http://example.com",
			Method: "INVALID\nMETHOD",
		},
	}

	event := api.Event{
		Data: map[string]interface{}{},
	}

	result := executor.Execute(context.Background(), action, event, nil)

	assert.Equal(t, 1, result.ExitCode)
	assert.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "failed to create request")
}
