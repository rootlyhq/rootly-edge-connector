package metrics_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rootly/edge-connector/internal/config"
	"github.com/rootly/edge-connector/internal/metrics"
)

func TestInitMetrics(t *testing.T) {
	// Test that InitMetrics can be called without panicking

	customLabels := map[string]string{
		"connector_id": "test-connector",
		"environment":  "test",
		"region":       "us-west-2",
	}

	// Call InitMetrics
	metrics.InitMetrics(customLabels)

	// Verify metrics are initialized (not nil)
	assert.NotNil(t, metrics.EventsPolled)
	assert.NotNil(t, metrics.EventsReceived)
	assert.NotNil(t, metrics.ActionsExecuted)
	assert.NotNil(t, metrics.ActionExecutionDuration)
	assert.NotNil(t, metrics.DeliveriesMarkedRunning)
	assert.NotNil(t, metrics.EventsRunning)
	assert.NotNil(t, metrics.WorkerPoolSize)
	assert.NotNil(t, metrics.WorkerPoolQueueSize)
	assert.NotNil(t, metrics.HTTPRequestsTotal)
	assert.NotNil(t, metrics.HTTPRequestDuration)
	assert.NotNil(t, metrics.GitPullsTotal)
	assert.NotNil(t, metrics.GitPullDuration)

	// Calling InitMetrics again should be safe (due to sync.Once)
	metrics.InitMetrics(map[string]string{"foo": "bar"})
	assert.NotNil(t, metrics.EventsPolled) // Should still be the original
}

func TestInitMetrics_EmptyLabels(t *testing.T) {
	// Test with empty labels
	metrics.InitMetrics(nil)

	// Should still initialize metrics
	assert.NotNil(t, metrics.EventsPolled)
}

func TestNewServer(t *testing.T) {
	cfg := &config.MetricsConfig{
		Enabled: true,
		Port:    9091,
		Path:    "/metrics",
	}

	server := metrics.NewServer(cfg)
	require.NotNil(t, server)
}

func TestServer_StartAndShutdown(t *testing.T) {
	cfg := &config.MetricsConfig{
		Enabled: true,
		Port:    19090, // Use high port to avoid conflicts
		Path:    "/metrics",
	}

	server := metrics.NewServer(cfg)
	require.NotNil(t, server)

	// Start server
	err := server.Start()
	require.NoError(t, err)

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Verify server is running by making a request
	resp, err := http.Get("http://localhost:19090/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Read response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Should contain Prometheus metrics
	assert.Contains(t, string(body), "# HELP")
	assert.Contains(t, string(body), "# TYPE")

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Shutdown(ctx)
	assert.NoError(t, err)

	// Give server time to shut down
	time.Sleep(100 * time.Millisecond)

	// Verify server is stopped
	_, err = http.Get("http://localhost:19090/metrics")
	assert.Error(t, err) // Should fail because server is stopped
}

func TestRecordActionExecution(t *testing.T) {
	// Initialize metrics first
	metrics.InitMetrics(map[string]string{"test": "true"})

	// Record some action executions
	metrics.RecordActionExecution("test_action", "script", "completed", 1500*time.Millisecond)
	metrics.RecordActionExecution("test_action", "script", "failed", 500*time.Millisecond)
	metrics.RecordActionExecution("http_action", "http", "completed", 200*time.Millisecond)

	// Verify metrics were recorded by gathering them
	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)

	// Find our action metrics
	var foundCounter, foundHistogram bool
	for _, mf := range metricFamilies {
		if mf.GetName() == "rec_actions_executed_total" {
			foundCounter = true
			assert.GreaterOrEqual(t, len(mf.GetMetric()), 3) // At least 3 label combinations
		}
		if mf.GetName() == "rec_action_execution_duration_seconds" {
			foundHistogram = true
			assert.GreaterOrEqual(t, len(mf.GetMetric()), 2) // At least 2 label combinations
		}
	}

	assert.True(t, foundCounter, "Should have recorded action execution counter")
	assert.True(t, foundHistogram, "Should have recorded action execution duration")
}

func TestRecordHTTPRequest(t *testing.T) {
	// Initialize metrics first
	metrics.InitMetrics(map[string]string{"test": "true"})

	// Record some HTTP requests
	metrics.RecordHTTPRequest("POST", 200, 150*time.Millisecond)
	metrics.RecordHTTPRequest("GET", 404, 50*time.Millisecond)
	metrics.RecordHTTPRequest("POST", 500, 1000*time.Millisecond)

	// Verify metrics were recorded
	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)

	var foundCounter, foundHistogram bool
	for _, mf := range metricFamilies {
		if mf.GetName() == "rec_http_requests_total" {
			foundCounter = true
			assert.GreaterOrEqual(t, len(mf.GetMetric()), 3) // At least 3 label combinations
		}
		if mf.GetName() == "rec_http_request_duration_seconds" {
			foundHistogram = true
		}
	}

	assert.True(t, foundCounter, "Should have recorded HTTP request counter")
	assert.True(t, foundHistogram, "Should have recorded HTTP request duration")
}

func TestRecordGitPull(t *testing.T) {
	// Initialize metrics first
	metrics.InitMetrics(map[string]string{"test": "true"})

	// Record some git pulls
	metrics.RecordGitPull("github.com/example/repo", "success", 2500*time.Millisecond)
	metrics.RecordGitPull("github.com/example/repo", "error", 100*time.Millisecond)
	metrics.RecordGitPull("github.com/other/repo", "success", 3000*time.Millisecond)

	// Verify metrics were recorded
	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)

	var foundCounter, foundHistogram bool
	for _, mf := range metricFamilies {
		if mf.GetName() == "rec_git_pulls_total" {
			foundCounter = true
			assert.GreaterOrEqual(t, len(mf.GetMetric()), 2) // At least 2 repositories
		}
		if mf.GetName() == "rec_git_pull_duration_seconds" {
			foundHistogram = true
		}
	}

	assert.True(t, foundCounter, "Should have recorded git pull counter")
	assert.True(t, foundHistogram, "Should have recorded git pull duration")
}

func TestRecordMetrics_WhenNotInitialized(t *testing.T) {
	// Reset metrics to nil to test behavior when metrics are disabled
	// Note: We can't actually reset them due to sync.Once, but we can test
	// that calling the record functions doesn't panic

	// These should not panic even if metrics are nil
	assert.NotPanics(t, func() {
		metrics.RecordActionExecution("test", "script", "completed", time.Second)
	})

	assert.NotPanics(t, func() {
		metrics.RecordHTTPRequest("GET", 200, time.Second)
	})

	assert.NotPanics(t, func() {
		metrics.RecordGitPull("repo", "success", time.Second)
	})
}

func TestServer_MetricsEndpoint(t *testing.T) {
	// Initialize metrics (may already be initialized from other tests, that's ok)
	metrics.InitMetrics(map[string]string{
		"connector_id": "test-metrics",
		"environment":  "test",
	})

	// Record some metrics
	metrics.RecordActionExecution("test_action", "script", "completed", 100*time.Millisecond)
	metrics.RecordHTTPRequest("POST", 200, 50*time.Millisecond)

	cfg := &config.MetricsConfig{
		Enabled: true,
		Port:    19091,
		Path:    "/custom-metrics",
	}

	server := metrics.NewServer(cfg)
	require.NotNil(t, server)

	// Start server
	err := server.Start()
	require.NoError(t, err)

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test the custom path
	resp, err := http.Get("http://localhost:19091/custom-metrics")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	bodyStr := string(body)

	// Verify it contains Prometheus format
	assert.Contains(t, bodyStr, "# HELP")
	assert.Contains(t, bodyStr, "# TYPE")

	// Verify it contains our recorded metrics
	assert.Contains(t, bodyStr, "rec_actions_executed_total")
	assert.Contains(t, bodyStr, "rec_http_requests_total")

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}

func TestServer_NotFoundOnWrongPath(t *testing.T) {
	cfg := &config.MetricsConfig{
		Enabled: true,
		Port:    19092,
		Path:    "/metrics",
	}

	server := metrics.NewServer(cfg)
	err := server.Start()
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// Request wrong path
	resp, err := http.Get("http://localhost:19092/wrong-path")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}

func TestMetrics_CounterIncrement(t *testing.T) {
	metrics.InitMetrics(nil)

	// Get initial value (may not be 0 due to other tests)
	// Just verify we can increment without panic
	assert.NotPanics(t, func() {
		metrics.EventsReceived.Inc()
		metrics.EventsReceived.Add(5)
	})
}

func TestMetrics_GaugeSetAndInc(t *testing.T) {
	metrics.InitMetrics(nil)

	assert.NotPanics(t, func() {
		metrics.EventsRunning.Set(10)
		metrics.EventsRunning.Inc()
		metrics.EventsRunning.Dec()

		metrics.WorkerPoolSize.Set(5)
		metrics.WorkerPoolQueueSize.Set(20)
	})
}

func TestMetrics_CounterVecWithLabels(t *testing.T) {
	metrics.InitMetrics(nil)

	assert.NotPanics(t, func() {
		// Test all CounterVec metrics
		metrics.EventsPolled.WithLabelValues("success").Inc()
		metrics.EventsPolled.WithLabelValues("error").Inc()

		metrics.DeliveriesMarkedRunning.WithLabelValues("success").Inc()
		metrics.DeliveriesMarkedRunning.WithLabelValues("error").Inc()

		metrics.ActionsExecuted.WithLabelValues("action1", "script", "completed").Inc()
		metrics.ActionsExecuted.WithLabelValues("action2", "http", "failed").Inc()

		metrics.HTTPRequestsTotal.WithLabelValues("POST", "200").Inc()
		metrics.HTTPRequestsTotal.WithLabelValues("GET", "404").Inc()

		metrics.GitPullsTotal.WithLabelValues("repo1", "success").Inc()
		metrics.GitPullsTotal.WithLabelValues("repo2", "error").Inc()
	})
}

func TestMetrics_HistogramObserve(t *testing.T) {
	metrics.InitMetrics(nil)

	assert.NotPanics(t, func() {
		// Test all HistogramVec metrics
		metrics.ActionExecutionDuration.WithLabelValues("action1", "script").Observe(1.5)
		metrics.ActionExecutionDuration.WithLabelValues("action2", "http").Observe(0.25)

		metrics.HTTPRequestDuration.WithLabelValues("POST").Observe(0.1)
		metrics.HTTPRequestDuration.WithLabelValues("GET").Observe(0.05)

		metrics.GitPullDuration.WithLabelValues("repo1").Observe(2.5)
		metrics.GitPullDuration.WithLabelValues("repo2").Observe(3.0)
	})
}

func TestMetrics_AllMetricsPresent(t *testing.T) {
	metrics.InitMetrics(map[string]string{"test": "all-metrics"})

	// Trigger all metrics
	metrics.EventsPolled.WithLabelValues("success").Inc()
	metrics.EventsReceived.Inc()
	metrics.ActionsExecuted.WithLabelValues("test", "script", "completed").Inc()
	metrics.ActionExecutionDuration.WithLabelValues("test", "script").Observe(1.0)
	metrics.DeliveriesMarkedRunning.WithLabelValues("success").Inc()
	metrics.EventsRunning.Set(5)
	metrics.WorkerPoolSize.Set(3)
	metrics.WorkerPoolQueueSize.Set(10)
	metrics.HTTPRequestsTotal.WithLabelValues("POST", "200").Inc()
	metrics.HTTPRequestDuration.WithLabelValues("POST").Observe(0.5)
	metrics.GitPullsTotal.WithLabelValues("repo", "success").Inc()
	metrics.GitPullDuration.WithLabelValues("repo").Observe(2.0)

	// Gather metrics
	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)

	// Verify all expected metrics are present
	expectedMetrics := []string{
		"rec_events_polled_total",
		"rec_events_received_total",
		"rec_actions_executed_total",
		"rec_action_execution_duration_seconds",
		"rec_deliveries_marked_running_total",
		"rec_events_running",
		"rec_worker_pool_size",
		"rec_worker_pool_queue_size",
		"rec_http_requests_total",
		"rec_http_request_duration_seconds",
		"rec_git_pulls_total",
		"rec_git_pull_duration_seconds",
	}

	foundMetrics := make(map[string]bool)
	for _, mf := range metricFamilies {
		foundMetrics[mf.GetName()] = true
	}

	for _, expectedMetric := range expectedMetrics {
		assert.True(t, foundMetrics[expectedMetric], "Metric %s should be present", expectedMetric)
	}
}

func TestServer_ShutdownWithShortTimeout(t *testing.T) {
	cfg := &config.MetricsConfig{
		Enabled: true,
		Port:    19093,
		Path:    "/metrics",
	}

	server := metrics.NewServer(cfg)
	err := server.Start()
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	// Shutdown with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = server.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestMetrics_PrometheusFormat(t *testing.T) {
	metrics.InitMetrics(map[string]string{
		"connector_id": "format-test",
	})

	// Record a metric
	metrics.RecordActionExecution("format_test", "script", "completed", 100*time.Millisecond)

	// Start a temporary server
	cfg := &config.MetricsConfig{
		Enabled: true,
		Port:    19094,
		Path:    "/metrics",
	}

	server := metrics.NewServer(cfg)
	server.Start()
	time.Sleep(100 * time.Millisecond)

	// Fetch metrics
	resp, err := http.Get("http://localhost:19094/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	lines := strings.Split(string(body), "\n")

	// Verify Prometheus format
	var hasHelp, hasType, hasMetric bool
	for _, line := range lines {
		if strings.HasPrefix(line, "# HELP rec_") {
			hasHelp = true
		}
		if strings.HasPrefix(line, "# TYPE rec_") {
			hasType = true
		}
		if strings.HasPrefix(line, "rec_") && !strings.HasPrefix(line, "# ") {
			hasMetric = true
		}
	}

	assert.True(t, hasHelp, "Should have HELP comments")
	assert.True(t, hasType, "Should have TYPE comments")
	assert.True(t, hasMetric, "Should have actual metric values")

	// Cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}

// Edge case tests for improved coverage

func TestRecordActionExecution_MetricsDisabled(t *testing.T) {
	// Test recording when metrics are not initialized (nil)
	// Save current metrics
	oldActionsExecuted := metrics.ActionsExecuted
	oldActionExecutionDuration := metrics.ActionExecutionDuration

	// Set to nil to simulate disabled metrics
	metrics.ActionsExecuted = nil
	metrics.ActionExecutionDuration = nil

	// Should not panic when metrics are disabled
	metrics.RecordActionExecution("test_action", "script", "completed", 1*time.Second)

	// Restore metrics
	metrics.ActionsExecuted = oldActionsExecuted
	metrics.ActionExecutionDuration = oldActionExecutionDuration
}

func TestRecordHTTPRequest_MetricsDisabled(t *testing.T) {
	// Test recording when metrics are not initialized (nil)
	oldHTTPRequestsTotal := metrics.HTTPRequestsTotal
	oldHTTPRequestDuration := metrics.HTTPRequestDuration

	// Set to nil to simulate disabled metrics
	metrics.HTTPRequestsTotal = nil
	metrics.HTTPRequestDuration = nil

	// Should not panic when metrics are disabled
	metrics.RecordHTTPRequest("GET", 200, 100*time.Millisecond)

	// Restore metrics
	metrics.HTTPRequestsTotal = oldHTTPRequestsTotal
	metrics.HTTPRequestDuration = oldHTTPRequestDuration
}

func TestRecordGitPull_MetricsDisabled(t *testing.T) {
	// Test recording when metrics are not initialized (nil)
	oldGitPullsTotal := metrics.GitPullsTotal
	oldGitPullDuration := metrics.GitPullDuration

	// Set to nil to simulate disabled metrics
	metrics.GitPullsTotal = nil
	metrics.GitPullDuration = nil

	// Should not panic when metrics are disabled
	metrics.RecordGitPull("test-repo", "success", 500*time.Millisecond)

	// Restore metrics
	metrics.GitPullsTotal = oldGitPullsTotal
	metrics.GitPullDuration = oldGitPullDuration
}

func TestServer_StartError(t *testing.T) {
	// Test server start with port already in use
	cfg1 := &config.MetricsConfig{
		Enabled: true,
		Port:    29091,
		Path:    "/metrics",
	}

	server1 := metrics.NewServer(cfg1)
	require.NotNil(t, server1)

	// Start first server
	err := server1.Start()
	require.NoError(t, err)

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Try to start second server on same port (should log error but not panic)
	cfg2 := &config.MetricsConfig{
		Enabled: true,
		Port:    29091, // Same port
		Path:    "/metrics",
	}

	server2 := metrics.NewServer(cfg2)
	require.NotNil(t, server2)

	// Starting should not return error (error is logged in goroutine)
	err = server2.Start()
	assert.NoError(t, err)

	// Give time for error to be logged
	time.Sleep(200 * time.Millisecond)

	// Cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	server1.Shutdown(ctx)
	server2.Shutdown(ctx)
}

func TestRecordActionExecution_ConcurrentCalls(t *testing.T) {
	// Test concurrent metric recording (race condition check)
	metrics.InitMetrics(map[string]string{"test": "concurrent"})

	// Use WaitGroup to synchronize goroutines
	var wg sync.WaitGroup
	numGoroutines := 10
	callsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < callsPerGoroutine; j++ {
				metrics.RecordActionExecution("test_action", "script", "completed", time.Duration(j)*time.Millisecond)
			}
		}()
	}

	wg.Wait()

	// Verify metrics were recorded (should not panic)
	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)

	// Find action metrics
	var found bool
	for _, mf := range metricFamilies {
		if mf.GetName() == "rec_actions_executed_total" {
			found = true
			break
		}
	}

	assert.True(t, found, "Should have recorded action metrics concurrently")
}

func TestMetrics_WithCustomLabels(t *testing.T) {
	// Test that metrics work correctly when initialized with comprehensive custom labels
	customLabels := map[string]string{
		"connector_id": "test-123",
		"environment":  "production",
		"region":       "us-east-1",
		"datacenter":   "dc1",
		"cluster":      "main",
	}

	// Initialize with many labels
	metrics.InitMetrics(customLabels)

	// Record various metrics - should not panic or error
	metrics.RecordActionExecution("test", "script", "completed", 1*time.Second)
	metrics.RecordHTTPRequest("POST", 201, 200*time.Millisecond)
	metrics.RecordGitPull("github.com/test/repo", "success", 3*time.Second)

	// Verify metrics exist and can be gathered
	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)

	// Verify at least some metrics are present
	var foundAction, foundHTTP, foundGit bool
	for _, mf := range metricFamilies {
		switch mf.GetName() {
		case "rec_actions_executed_total":
			foundAction = true
		case "rec_http_requests_total":
			foundHTTP = true
		case "rec_git_pulls_total":
			foundGit = true
		}
	}

	assert.True(t, foundAction, "Should have action metrics")
	assert.True(t, foundHTTP, "Should have HTTP metrics")
	assert.True(t, foundGit, "Should have git metrics")
}

func TestRecordMultipleMetricTypes(t *testing.T) {
	// Test recording all metric types in sequence
	metrics.InitMetrics(map[string]string{"test": "multiple"})

	// Record action execution metrics
	metrics.RecordActionExecution("action1", "script", "completed", 100*time.Millisecond)
	metrics.RecordActionExecution("action2", "http", "failed", 50*time.Millisecond)

	// Record HTTP request metrics
	metrics.RecordHTTPRequest("GET", 200, 10*time.Millisecond)
	metrics.RecordHTTPRequest("POST", 500, 200*time.Millisecond)

	// Record git pull metrics
	metrics.RecordGitPull("repo1", "success", 1*time.Second)
	metrics.RecordGitPull("repo2", "failure", 500*time.Millisecond)

	// Gather and verify all metrics exist
	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)

	metricNames := make(map[string]bool)
	for _, mf := range metricFamilies {
		metricNames[mf.GetName()] = true
	}

	// Verify all expected metrics are present
	assert.True(t, metricNames["rec_actions_executed_total"])
	assert.True(t, metricNames["rec_action_execution_duration_seconds"])
	assert.True(t, metricNames["rec_http_requests_total"])
	assert.True(t, metricNames["rec_http_request_duration_seconds"])
	assert.True(t, metricNames["rec_git_pulls_total"])
	assert.True(t, metricNames["rec_git_pull_duration_seconds"])
}
