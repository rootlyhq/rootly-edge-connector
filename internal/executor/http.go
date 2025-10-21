package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

// HTTPExecutor handles HTTP action execution
type HTTPExecutor struct {
	client *http.Client
}

// HTTPResponse represents an HTTP response
type HTTPResponse struct {
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	Duration   int64             `json:"duration_ms"`
	StatusCode int               `json:"status_code"`
}

// NewHTTPExecutor creates a new HTTP executor
func NewHTTPExecutor() *HTTPExecutor {
	return &HTTPExecutor{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Execute executes an HTTP action
func (h *HTTPExecutor) Execute(ctx context.Context, action *config.Action, event api.Event, params map[string]string) reporter.ScriptResult {
	start := time.Now()

	if action.HTTP == nil {
		return reporter.ScriptResult{
			ExitCode:   1,
			DurationMs: 0,
			Error:      fmt.Errorf("HTTP action configuration is missing"),
		}
	}

	// Render URL with template variables
	renderedURL, err := h.renderTemplate(action.HTTP.URL, event)
	if err != nil {
		return reporter.ScriptResult{
			ExitCode:   1,
			DurationMs: time.Since(start).Milliseconds(),
			Error:      fmt.Errorf("failed to render URL: %w", err),
		}
	}

	// Parse and add query parameters
	parsedURL, err := url.Parse(renderedURL)
	if err != nil {
		return reporter.ScriptResult{
			ExitCode:   1,
			DurationMs: time.Since(start).Milliseconds(),
			Error:      fmt.Errorf("invalid URL: %w", err),
		}
	}

	query := parsedURL.Query()
	for key, valueTemplate := range action.HTTP.Params {
		value, err := h.renderTemplate(valueTemplate, event)
		if err != nil {
			return reporter.ScriptResult{
				ExitCode:   1,
				DurationMs: time.Since(start).Milliseconds(),
				Error:      fmt.Errorf("failed to render param %s: %w", key, err),
			}
		}
		query.Set(key, value)
	}
	parsedURL.RawQuery = query.Encode()

	// Render request body
	var bodyReader io.Reader
	var bodyContent string
	if action.HTTP.Body != "" {
		// Use custom body template if provided
		renderedBody, err := h.renderTemplate(action.HTTP.Body, event)
		if err != nil {
			return reporter.ScriptResult{
				ExitCode:   1,
				DurationMs: time.Since(start).Milliseconds(),
				Error:      fmt.Errorf("failed to render body: %w", err),
			}
		}
		bodyContent = renderedBody
		bodyReader = strings.NewReader(renderedBody)
		log.WithField("body_length", len(renderedBody)).Trace("Rendered HTTP request body")
	} else if len(params) > 0 {
		// Auto-build JSON body from parameters if no custom body template
		bodyJSON, err := json.Marshal(params)
		if err != nil {
			return reporter.ScriptResult{
				ExitCode:   1,
				DurationMs: time.Since(start).Milliseconds(),
				Error:      fmt.Errorf("failed to marshal parameters to JSON: %w", err),
			}
		}
		bodyContent = string(bodyJSON)
		bodyReader = bytes.NewReader(bodyJSON)
		log.WithField("auto_body", string(bodyJSON)).Debug("Auto-built HTTP body from parameters")
	}

	// Create HTTP request
	method := action.HTTP.Method
	if method == "" {
		method = "POST"
	}

	// Apply timeout from action config
	timeout := time.Duration(action.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctxWithTimeout, method, parsedURL.String(), bodyReader)
	if err != nil {
		return reporter.ScriptResult{
			ExitCode:   1,
			DurationMs: time.Since(start).Milliseconds(),
			Error:      fmt.Errorf("failed to create request: %w", err),
		}
	}

	// Add headers
	for key, valueTemplate := range action.HTTP.Headers {
		value, err := h.renderTemplate(valueTemplate, event)
		if err != nil {
			return reporter.ScriptResult{
				ExitCode:   1,
				DurationMs: time.Since(start).Milliseconds(),
				Error:      fmt.Errorf("failed to render header %s: %w", key, err),
			}
		}
		req.Header.Set(key, value)
	}

	// Log all request headers at TRACE level
	headerMap := make(map[string]string)
	for key := range req.Header {
		headerMap[key] = req.Header.Get(key)
	}
	log.WithField("headers", headerMap).Trace("HTTP request headers")

	log.WithFields(log.Fields{
		"method":       method,
		"url":          parsedURL.String(),
		"timeout":      timeout,
		"has_body":     bodyReader != nil,
		"body_preview": truncateString(bodyContent, 100),
	}).Info("Executing HTTP request")

	// Execute HTTP request
	log.Debug("Sending HTTP request...")
	resp, err := h.client.Do(req)
	duration := time.Since(start)

	if err != nil {
		log.WithError(err).Error("HTTP request failed")
		metrics.RecordHTTPRequest(method, 0, duration)
		return reporter.ScriptResult{
			ExitCode:   1,
			Stderr:     err.Error(),
			DurationMs: duration.Milliseconds(),
			Error:      fmt.Errorf("HTTP request failed: %w", err),
		}
	}
	defer resp.Body.Close()

	log.WithFields(log.Fields{
		"status_code": resp.StatusCode,
		"duration_ms": duration.Milliseconds(),
	}).Debug("HTTP request completed, reading response body...")

	// Record HTTP metrics
	metrics.RecordHTTPRequest(method, resp.StatusCode, duration)

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.WithError(err).Error("Failed to read HTTP response body")
		return reporter.ScriptResult{
			ExitCode:   1,
			DurationMs: time.Since(start).Milliseconds(),
			Error:      fmt.Errorf("failed to read response: %w", err),
		}
	}

	log.WithFields(log.Fields{
		"body_length": len(respBody),
		"status_code": resp.StatusCode,
	}).Debug("HTTP response body read successfully")

	// Build response
	httpResp := HTTPResponse{
		StatusCode: resp.StatusCode,
		Headers:    make(map[string]string),
		Body:       string(respBody),
		Duration:   time.Since(start).Milliseconds(),
	}

	// Capture important headers
	for _, header := range []string{"Content-Type", "X-Request-Id", "Location"} {
		if val := resp.Header.Get(header); val != "" {
			httpResp.Headers[header] = val
		}
	}

	// Log response body at TRACE level
	log.WithField("response_body", string(respBody)).Trace("HTTP response body")

	// Marshal HTTP response as JSON for stdout
	respJSON, err := json.MarshalIndent(httpResp, "", "  ")
	if err != nil {
		respJSON = []byte(fmt.Sprintf(`{"error": "failed to marshal response: %v"}`, err))
	}

	result := reporter.ScriptResult{
		Stdout:     string(respJSON),
		DurationMs: time.Since(start).Milliseconds(),
		ExitCode:   resp.StatusCode,
	}

	// Consider 2xx status codes as success
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Error = nil
		log.WithFields(log.Fields{
			"status_code": resp.StatusCode,
			"duration_ms": result.DurationMs,
		}).Info("HTTP request completed successfully")
		log.Debug("Returning success result to executor")
	} else {
		result.Error = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
		log.WithFields(log.Fields{
			"status_code": resp.StatusCode,
			"error":       string(respBody),
		}).Error("HTTP request failed")
		log.Debug("Returning error result to executor")
	}

	return result
}

// renderTemplate renders a template string with event data using Liquid templates
// This provides consistent syntax with script actions: {{ field }} instead of {{ .field }}
func (h *HTTPExecutor) renderTemplate(tmplStr string, event api.Event) (string, error) {
	// Create Liquid engine
	engine := liquid.NewEngine()

	// Prepare template context with event data + env variables
	context := h.prepareTemplateContext(tmplStr, event)

	// Render template
	result, err := engine.ParseAndRenderString(tmplStr, context)
	if err != nil {
		return "", fmt.Errorf("template rendering failed: %w", err)
	}

	return result, nil
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// prepareTemplateContext creates the template context with event data and environment variables
func (h *HTTPExecutor) prepareTemplateContext(tmplStr string, event api.Event) map[string]interface{} {
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
