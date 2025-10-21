package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	log "github.com/sirupsen/logrus"
)

// Client represents a Rootly API client
type Client struct {
	httpClient *retryablehttp.Client
	baseURL    string
	apiPath    string
	apiKey     string
	userAgent  string
}

// NewClient creates a new Rootly API client
func NewClient(baseURL, apiPath, apiKey, version string) *Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.RetryWaitMin = 1 * time.Second
	retryClient.RetryWaitMax = 10 * time.Second
	retryClient.Logger = nil // Disable retryablehttp logging, we'll use our own

	return &Client{
		baseURL:    baseURL,
		apiPath:    apiPath,
		apiKey:     apiKey,
		httpClient: retryClient,
		userAgent:  fmt.Sprintf("rootly-edge-connector/%s", version),
	}
}

// FetchEvents fetches events from the Rootly API
func (c *Client) FetchEvents(ctx context.Context, maxMessages, visibilityTimeout int) ([]Event, error) {
	url := fmt.Sprintf("%s%s/deliveries?max_messages=%d&visibility_timeout=%d",
		c.baseURL, c.apiPath, maxMessages, visibilityTimeout)

	req, err := retryablehttp.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	// Log HTTP request at DEBUG level
	log.WithFields(log.Fields{
		"method": "GET",
		"url":    url,
		"headers": map[string]string{
			"Authorization": redactToken(c.apiKey),
			"Content-Type":  "application/json",
			"User-Agent":    c.userAgent,
		},
	}).Debug("HTTP request")

	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		log.WithFields(log.Fields{
			"method":   "GET",
			"url":      url,
			"error":    err.Error(),
			"duration": duration.String(),
		}).Error("HTTP request failed")
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Log HTTP response at DEBUG level
	log.WithFields(log.Fields{
		"method":   "GET",
		"url":      url,
		"status":   resp.StatusCode,
		"duration": duration.String(),
	}).Debug("HTTP response")

	// Log rate limit headers from Rootly API
	c.logRateLimitHeaders(resp)

	// Handle rate limiting (429 Too Many Requests)
	if resp.StatusCode == http.StatusTooManyRequests {
		remaining := resp.Header.Get("X-RateLimit-Remaining")
		reset := resp.Header.Get("X-RateLimit-Reset")
		log.WithFields(log.Fields{
			"remaining": remaining,
			"reset":     reset,
		}).Warn("Rate limit exceeded")
		return nil, fmt.Errorf("rate limit exceeded (remaining: %s, resets at: %s)", remaining, reset)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Log response body at TRACE level (pretty-printed JSON)
	if log.IsLevelEnabled(log.TraceLevel) {
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, bodyBytes, "", "  "); err == nil {
			log.WithFields(log.Fields{
				"method": "GET",
				"url":    url,
			}).Tracef("HTTP response body:\n%s", prettyJSON.String())
		} else {
			log.WithFields(log.Fields{
				"method": "GET",
				"url":    url,
				"body":   string(bodyBytes),
			}).Trace("HTTP response body")
		}
	}

	// Decode response
	var response EventsResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	log.WithFields(log.Fields{
		"event_count": len(response.Events),
	}).Debug("Fetched events from API")

	return response.Events, nil
}

// logRateLimitHeaders logs Rootly API rate limit headers for monitoring
func (c *Client) logRateLimitHeaders(resp *http.Response) {
	limit := resp.Header.Get("X-RateLimit-Limit")
	remaining := resp.Header.Get("X-RateLimit-Remaining")
	used := resp.Header.Get("X-RateLimit-Used")
	reset := resp.Header.Get("X-RateLimit-Reset")

	if limit != "" || remaining != "" {
		log.WithFields(log.Fields{
			"limit":     limit,
			"remaining": remaining,
			"used":      used,
			"reset":     reset,
		}).Trace("Rate limit status")
	}
}

// RegisterActions registers callable actions with the backend
func (c *Client) RegisterActions(ctx context.Context, request RegisterActionsRequest) (*RegisterActionsResponse, error) {
	url := fmt.Sprintf("%s%s/actions", c.baseURL, c.apiPath)

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal actions: %w", err)
	}

	req, err := retryablehttp.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	// Log HTTP request at DEBUG level
	log.WithFields(log.Fields{
		"method":          "POST",
		"url":             url,
		"automatic_count": len(request.Automatic),
		"callable_count":  len(request.Callable),
	}).Debug("HTTP request")

	// Log full request body at TRACE level (pretty-printed JSON)
	if log.IsLevelEnabled(log.TraceLevel) {
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, body, "", "  "); err == nil {
			log.WithFields(log.Fields{
				"method": "POST",
				"url":    url,
			}).Tracef("HTTP request body:\n%s", prettyJSON.String())
		}
	}

	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		log.WithFields(log.Fields{
			"method":   "POST",
			"url":      url,
			"error":    err.Error(),
			"duration": duration.String(),
		}).Error("HTTP request failed")
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Log HTTP response at DEBUG level
	log.WithFields(log.Fields{
		"method":   "POST",
		"url":      url,
		"status":   resp.StatusCode,
		"duration": duration.String(),
	}).Debug("HTTP response")

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Log response body at TRACE level (pretty-printed JSON)
	if log.IsLevelEnabled(log.TraceLevel) {
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, bodyBytes, "", "  "); err == nil {
			log.WithFields(log.Fields{
				"method": "POST",
				"url":    url,
			}).Tracef("HTTP response body:\n%s", prettyJSON.String())
		}
	}

	// Log rate limit headers
	c.logRateLimitHeaders(resp)

	// Handle different status codes
	if resp.StatusCode == http.StatusTooManyRequests {
		remaining := resp.Header.Get("X-RateLimit-Remaining")
		reset := resp.Header.Get("X-RateLimit-Reset")
		return nil, fmt.Errorf("rate limit exceeded (remaining: %s, resets at: %s)", remaining, reset)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusMultiStatus {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Decode response
	var response RegisterActionsResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// redactToken redacts an API token for safe logging (shows last 8 chars)
func redactToken(token string) string {
	if len(token) <= 8 {
		return "***"
	}
	return "***" + token[len(token)-8:]
}

// MarkDeliveryAsRunning marks a delivery as running to claim it for execution
// Uses PATCH /rec/v1/deliveries/{id} with execution_status: running
func (c *Client) MarkDeliveryAsRunning(ctx context.Context, deliveryID string) error {
	// IMPORTANT: Use delivery_id - this is the `id` field from deliveries response
	url := fmt.Sprintf("%s%s/deliveries/%s", c.baseURL, c.apiPath, deliveryID)

	// Minimal payload: set execution_status to running with timestamp
	ackPayload := map[string]interface{}{
		"execution_status": "running",
		"running_at":       time.Now().UTC().Format(time.RFC3339),
	}

	body, err := json.Marshal(ackPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal acknowledge payload: %w", err)
	}

	req, err := retryablehttp.NewRequestWithContext(ctx, "PATCH", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	// Log HTTP request at DEBUG level
	log.WithFields(log.Fields{
		"method":      "PATCH",
		"url":         url,
		"delivery_id": deliveryID,
	}).Debug("HTTP request")

	// Log request body at TRACE level (pretty-printed JSON)
	if log.IsLevelEnabled(log.TraceLevel) {
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, body, "", "  "); err == nil {
			log.WithFields(log.Fields{
				"method": "PATCH",
				"url":    url,
			}).Tracef("HTTP request body:\n%s", prettyJSON.String())
		}
	}

	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		log.WithFields(log.Fields{
			"method":   "PATCH",
			"url":      url,
			"error":    err.Error(),
			"duration": duration.String(),
		}).Error("HTTP request failed")
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Log HTTP response at DEBUG level
	log.WithFields(log.Fields{
		"method":   "PATCH",
		"url":      url,
		"status":   resp.StatusCode,
		"duration": duration.String(),
	}).Debug("HTTP response")

	// Log response body at TRACE level (pretty-printed if JSON)
	if log.IsLevelEnabled(log.TraceLevel) {
		respBody := new(bytes.Buffer)
		if _, err := respBody.ReadFrom(resp.Body); err != nil {
			log.WithError(err).Warn("Failed to read response body for trace logging")
		} else {
			bodyStr := respBody.String()
			var prettyJSON bytes.Buffer
			if err := json.Indent(&prettyJSON, []byte(bodyStr), "", "  "); err == nil {
				log.WithFields(log.Fields{
					"method": "PATCH",
					"url":    url,
				}).Tracef("HTTP response body:\n%s", prettyJSON.String())
			} else {
				log.WithFields(log.Fields{
					"method": "PATCH",
					"url":    url,
					"body":   bodyStr,
				}).Trace("HTTP response body")
			}
		}
	}

	// Log rate limit headers
	c.logRateLimitHeaders(resp)

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		remaining := resp.Header.Get("X-RateLimit-Remaining")
		reset := resp.Header.Get("X-RateLimit-Reset")
		return fmt.Errorf("rate limit exceeded (remaining: %s, resets at: %s)", remaining, reset)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	log.WithFields(log.Fields{
		"delivery_id": deliveryID,
	}).Debug("Marked delivery as running")

	return nil
}

// ReportExecution reports execution results to the Rootly API
// Uses PATCH /rec/v1/deliveries/:id which also auto-acknowledges the delivery
func (c *Client) ReportExecution(ctx context.Context, execution ExecutionResult) error {
	url := fmt.Sprintf("%s%s/deliveries/%s", c.baseURL, c.apiPath, execution.DeliveryID)

	body, err := json.Marshal(execution)
	if err != nil {
		return fmt.Errorf("failed to marshal execution result: %w", err)
	}

	req, err := retryablehttp.NewRequestWithContext(ctx, "PATCH", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	// Log HTTP request at DEBUG level (with request body)
	log.WithFields(log.Fields{
		"method":           "PATCH",
		"url":              url,
		"delivery_id":      execution.DeliveryID,
		"action_name":      execution.ExecutionActionID,
		"execution_status": execution.ExecutionStatus,
		"body_size":        len(body),
	}).Debug("HTTP request")

	// Log full request body at TRACE level (pretty-printed JSON)
	if log.IsLevelEnabled(log.TraceLevel) {
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, body, "", "  "); err == nil {
			log.WithFields(log.Fields{
				"method": "PATCH",
				"url":    url,
			}).Tracef("HTTP request body:\n%s", prettyJSON.String())
		} else {
			log.WithFields(log.Fields{
				"method": "PATCH",
				"url":    url,
				"body":   string(body),
			}).Trace("HTTP request body")
		}
	}

	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		log.WithFields(log.Fields{
			"method":   "PATCH",
			"url":      url,
			"error":    err.Error(),
			"duration": duration.String(),
		}).Error("HTTP request failed")
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Log HTTP response at DEBUG level
	log.WithFields(log.Fields{
		"method":   "PATCH",
		"url":      url,
		"status":   resp.StatusCode,
		"duration": duration.String(),
	}).Debug("HTTP response")

	// Log response body at TRACE level (pretty-printed if JSON)
	if log.IsLevelEnabled(log.TraceLevel) {
		respBody := new(bytes.Buffer)
		if _, err := respBody.ReadFrom(resp.Body); err != nil {
			log.WithError(err).Warn("Failed to read response body for trace logging")
		} else {
			bodyStr := respBody.String()
			var prettyJSON bytes.Buffer
			if err := json.Indent(&prettyJSON, []byte(bodyStr), "", "  "); err == nil {
				log.WithFields(log.Fields{
					"method": "PATCH",
					"url":    url,
				}).Tracef("HTTP response body:\n%s", prettyJSON.String())
			} else {
				log.WithFields(log.Fields{
					"method": "PATCH",
					"url":    url,
					"body":   bodyStr,
				}).Trace("HTTP response body")
			}
		}
		// Note: Response body already consumed, but we don't need it
	}

	// Log rate limit headers
	c.logRateLimitHeaders(resp)

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		remaining := resp.Header.Get("X-RateLimit-Remaining")
		reset := resp.Header.Get("X-RateLimit-Reset")
		return fmt.Errorf("rate limit exceeded (remaining: %s, resets at: %s)", remaining, reset)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	log.WithFields(log.Fields{
		"delivery_id":      execution.DeliveryID,
		"action_name":      execution.ExecutionActionID,
		"execution_status": execution.ExecutionStatus,
		"exit_code":        execution.ExecutionExitCode,
		"duration_ms":      execution.ExecutionDurationMs,
	}).Info("Reported execution result")

	return nil
}
