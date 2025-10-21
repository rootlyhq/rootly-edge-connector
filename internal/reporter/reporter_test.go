package reporter_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rootly/edge-connector/internal/api"
	"github.com/rootly/edge-connector/internal/reporter"
)

func TestReporter_Report_Success(t *testing.T) {
	var receivedExecution api.ExecutionResult
	var receivedDeliveryID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		assert.Contains(t, r.URL.Path, "/deliveries/")

		// Extract delivery ID from URL
		parts := strings.Split(r.URL.Path, "/")
		for i, part := range parts {
			if part == "deliveries" && i+1 < len(parts) {
				receivedDeliveryID = parts[i+1]
				break
			}
		}

		err := json.NewDecoder(r.Body).Decode(&receivedExecution)
		require.NoError(t, err)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	rep := reporter.New(client)

	result := reporter.ScriptResult{
		Error:      nil,
		Stdout:     "Success output",
		Stderr:     "",
		DurationMs: 1234,
		ExitCode:   0,
	}

	err := rep.Report(context.Background(), "delivery-123", "test_action", "", result)
	require.NoError(t, err)

	// Verify the execution result
	assert.Equal(t, "delivery-123", receivedDeliveryID)
	assert.Equal(t, "completed", receivedExecution.ExecutionStatus)
	assert.Equal(t, 0, receivedExecution.ExecutionExitCode)
	assert.Equal(t, "Success output", receivedExecution.ExecutionStdout)
	assert.Equal(t, "", receivedExecution.ExecutionStderr)
	assert.Equal(t, int64(1234), receivedExecution.ExecutionDurationMs)
	assert.Equal(t, "", receivedExecution.ExecutionError)
	assert.Equal(t, "test_action", receivedExecution.ExecutionActionName) // Action slug
	assert.Equal(t, "", receivedExecution.ExecutionActionID)              // No UUID in test
	assert.NotEmpty(t, receivedExecution.CompletedAt)
	assert.Empty(t, receivedExecution.FailedAt)
}

func TestReporter_Report_FailureWithError(t *testing.T) {
	var receivedExecution api.ExecutionResult

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedExecution)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	rep := reporter.New(client)

	result := reporter.ScriptResult{
		Error:      errors.New("script execution failed"),
		Stdout:     "Partial output",
		Stderr:     "Error output",
		DurationMs: 567,
		ExitCode:   1,
	}

	err := rep.Report(context.Background(), "delivery-456", "failing_action", "", result)
	require.NoError(t, err)

	// Verify the execution result
	assert.Equal(t, "failed", receivedExecution.ExecutionStatus)
	assert.Equal(t, 1, receivedExecution.ExecutionExitCode)
	assert.Equal(t, "Partial output", receivedExecution.ExecutionStdout)
	assert.Equal(t, "Error output", receivedExecution.ExecutionStderr)
	assert.Equal(t, int64(567), receivedExecution.ExecutionDurationMs)
	assert.Equal(t, "script execution failed", receivedExecution.ExecutionError)
	assert.Equal(t, "failing_action", receivedExecution.ExecutionActionName)
	assert.Empty(t, receivedExecution.CompletedAt)
	assert.NotEmpty(t, receivedExecution.FailedAt)
}

func TestReporter_Report_FailureWithNonZeroExitCode(t *testing.T) {
	var receivedExecution api.ExecutionResult

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedExecution)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	rep := reporter.New(client)

	// Non-zero exit code without explicit error
	result := reporter.ScriptResult{
		Error:      nil,
		Stdout:     "",
		Stderr:     "Command not found",
		DurationMs: 100,
		ExitCode:   127,
	}

	err := rep.Report(context.Background(), "delivery-789", "broken_action", "", result)
	require.NoError(t, err)

	// Verify the execution is marked as failed
	assert.Equal(t, "failed", receivedExecution.ExecutionStatus)
	assert.Equal(t, 127, receivedExecution.ExecutionExitCode)
	assert.Equal(t, "", receivedExecution.ExecutionStdout)
	assert.Equal(t, "Command not found", receivedExecution.ExecutionStderr)
	assert.Equal(t, "", receivedExecution.ExecutionError) // No explicit error
	assert.Empty(t, receivedExecution.CompletedAt)
	assert.NotEmpty(t, receivedExecution.FailedAt)
}

func TestReporter_Report_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal server error"}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	rep := reporter.New(client)

	result := reporter.ScriptResult{
		Error:      nil,
		Stdout:     "Output",
		Stderr:     "",
		DurationMs: 100,
		ExitCode:   0,
	}

	err := rep.Report(context.Background(), "delivery-999", "test_action", "", result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to report execution")
}

func TestReporter_Report_TimestampFormat(t *testing.T) {
	var receivedExecution api.ExecutionResult

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedExecution)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	rep := reporter.New(client)

	result := reporter.ScriptResult{
		Error:      nil,
		Stdout:     "Output",
		Stderr:     "",
		DurationMs: 100,
		ExitCode:   0,
	}

	err := rep.Report(context.Background(), "delivery-timestamp", "test_action", "", result)
	require.NoError(t, err)

	// Verify timestamp is in RFC3339 format
	assert.NotEmpty(t, receivedExecution.CompletedAt)
	_, parseErr := time.Parse(time.RFC3339, receivedExecution.CompletedAt)
	assert.NoError(t, parseErr, "CompletedAt should be in RFC3339 format")
}

func TestReporter_Report_AllFields(t *testing.T) {
	var receivedExecution api.ExecutionResult

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedExecution)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	rep := reporter.New(client)

	result := reporter.ScriptResult{
		Error:      errors.New("test error"),
		Stdout:     "stdout content",
		Stderr:     "stderr content",
		DurationMs: 9999,
		ExitCode:   42,
	}

	err := rep.Report(context.Background(), "delivery-all", "complete_action", "", result)
	require.NoError(t, err)

	// Verify all fields are populated correctly
	assert.Equal(t, "failed", receivedExecution.ExecutionStatus)
	assert.Equal(t, 42, receivedExecution.ExecutionExitCode)
	assert.Equal(t, "stdout content", receivedExecution.ExecutionStdout)
	assert.Equal(t, "stderr content", receivedExecution.ExecutionStderr)
	assert.Equal(t, int64(9999), receivedExecution.ExecutionDurationMs)
	assert.Equal(t, "test error", receivedExecution.ExecutionError)
	assert.Equal(t, "complete_action", receivedExecution.ExecutionActionName)
	assert.Empty(t, receivedExecution.CompletedAt)
	assert.NotEmpty(t, receivedExecution.FailedAt)
}

func TestReporter_New(t *testing.T) {
	client := api.NewClient("http://test.com", "", "test-key", "test")
	rep := reporter.New(client)

	assert.NotNil(t, rep)
}
