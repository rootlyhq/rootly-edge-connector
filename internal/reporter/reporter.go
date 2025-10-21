package reporter

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/rootly/edge-connector/internal/api"
)

const (
	executionStatusCompleted = "completed"
	executionStatusFailed    = "failed"
)

// ScriptResult represents the result of a script execution
type ScriptResult struct {
	Error      error
	Stdout     string
	Stderr     string
	DurationMs int64
	ExitCode   int
}

// Reporter reports execution results back to the Rootly API
type Reporter struct {
	client *api.Client
}

// New creates a new reporter
func New(client *api.Client) *Reporter {
	return &Reporter{
		client: client,
	}
}

// Report reports an execution result to the Rootly API
func (r *Reporter) Report(ctx context.Context, deliveryID, actionName, actionUUID string, result ScriptResult) error {
	executionStatus := executionStatusCompleted
	errorMsg := ""
	timestamp := time.Now().UTC().Format(time.RFC3339) // ISO 8601 format

	// Determine execution status based on Error field and ExitCode
	// For HTTP actions: ExitCode is HTTP status code (200, 404, 500, etc.)
	// For Script actions: ExitCode is shell exit code (0 = success, 1-255 = error)
	if result.Error != nil {
		executionStatus = executionStatusFailed
		errorMsg = result.Error.Error()
	} else if result.ExitCode != 0 && !(result.ExitCode >= 200 && result.ExitCode < 300) {
		// Non-zero exit code, but not HTTP 2xx (which is success)
		// This handles edge cases where Error is nil but script failed
		executionStatus = executionStatusFailed
	}

	execution := api.ExecutionResult{
		DeliveryID:          deliveryID,
		ExecutionStatus:     executionStatus,
		ExecutionExitCode:   result.ExitCode,
		ExecutionStdout:     result.Stdout,
		ExecutionStderr:     result.Stderr,
		ExecutionDurationMs: result.DurationMs,
		ExecutionError:      errorMsg,
		ExecutionActionName: actionName, // Action slug from config (e.g., "test_manual_action_http")
		ExecutionActionID:   actionUUID, // Action UUID from event (e.g., "01939a0e-...", empty for non-action events)
	}

	// Set appropriate timestamp based on status
	if executionStatus == executionStatusCompleted {
		execution.CompletedAt = timestamp
	} else if executionStatus == executionStatusFailed {
		execution.FailedAt = timestamp
	}

	if err := r.client.ReportExecution(ctx, execution); err != nil {
		return fmt.Errorf("failed to report execution: %w", err)
	}

	log.WithFields(log.Fields{
		"delivery_id":      deliveryID,
		"action_name":      actionName,
		"execution_status": executionStatus,
		"exit_code":        result.ExitCode,
		"duration_ms":      result.DurationMs,
	}).Info("Execution result reported")

	return nil
}
