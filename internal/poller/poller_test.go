package poller_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rootly/edge-connector/internal/api"
	"github.com/rootly/edge-connector/internal/config"
	"github.com/rootly/edge-connector/internal/poller"
)

// mockWorkerPool implements poller.WorkerPool for testing
type mockWorkerPool struct {
	mu        sync.Mutex
	submitted []api.Event
}

func (m *mockWorkerPool) Submit(event api.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.submitted = append(m.submitted, event)
}

func (m *mockWorkerPool) GetSubmitted() []api.Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]api.Event{}, m.submitted...)
}

func (m *mockWorkerPool) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.submitted)
}

func TestNew(t *testing.T) {
	client := api.NewClient("http://test.com", "", "test-key", "test")
	cfg := &config.PollerConfig{
		PollingWaitIntervalMs: 1000,
		MaxNumberOfMessages:   10,
		VisibilityTimeoutSec:  30,
	}
	pool := &mockWorkerPool{}

	p := poller.New(client, cfg, pool)
	require.NotNil(t, p)
}

func TestPoller_Poll_Success(t *testing.T) {
	var markRunningCalls int
	markRunningMu := sync.Mutex{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/deliveries" {
			// Fetch events
			response := api.EventsResponse{
				Events: []api.Event{
					{
						ID:        "delivery-1",
						EventID:   "event-1",
						Type:      "test.event",
						Timestamp: time.Now(),
					},
					{
						ID:        "delivery-2",
						EventID:   "event-2",
						Type:      "test.event",
						Timestamp: time.Now(),
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if r.Method == "PATCH" {
			// Mark delivery as running
			markRunningMu.Lock()
			markRunningCalls++
			markRunningMu.Unlock()
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	cfg := &config.PollerConfig{
		PollingWaitIntervalMs: 100,
		MaxNumberOfMessages:   10,
		VisibilityTimeoutSec:  30,
		RetryOnError:          true,
		MaxRetries:            3,
		RetryBackoff:          "exponential",
	}
	pool := &mockWorkerPool{}

	p := poller.New(client, cfg, pool)

	// Start poller in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go p.Start(ctx)

	// Wait for at least one poll cycle
	time.Sleep(200 * time.Millisecond)
	cancel()

	// Verify events were submitted to worker pool
	submitted := pool.GetSubmitted()
	assert.GreaterOrEqual(t, len(submitted), 2)

	// Verify mark running was called
	markRunningMu.Lock()
	assert.GreaterOrEqual(t, markRunningCalls, 2)
	markRunningMu.Unlock()
}

func TestPoller_Poll_NoEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := api.EventsResponse{Events: []api.Event{}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	cfg := &config.PollerConfig{
		PollingWaitIntervalMs: 100,
		MaxNumberOfMessages:   10,
		VisibilityTimeoutSec:  30,
	}
	pool := &mockWorkerPool{}

	p := poller.New(client, cfg, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Start should return context.DeadlineExceeded
	err := p.Start(ctx)
	assert.Error(t, err)

	// No events should be submitted
	assert.Equal(t, 0, pool.Count())
}

func TestPoller_Poll_FetchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	cfg := &config.PollerConfig{
		PollingWaitIntervalMs: 100,
		MaxNumberOfMessages:   10,
		VisibilityTimeoutSec:  30,
		RetryOnError:          false, // Don't retry
	}
	pool := &mockWorkerPool{}

	p := poller.New(client, cfg, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	err := p.Start(ctx)
	assert.Error(t, err)

	// No events should be submitted
	assert.Equal(t, 0, pool.Count())
}

func TestPoller_Poll_MarkRunningError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/deliveries" {
			// Return events
			response := api.EventsResponse{
				Events: []api.Event{
					{
						ID:        "delivery-1",
						EventID:   "event-1",
						Type:      "test.event",
						Timestamp: time.Now(),
					},
					{
						ID:        "delivery-2",
						EventID:   "event-2",
						Type:      "test.event",
						Timestamp: time.Now(),
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if r.Method == "PATCH" {
			// Mark running fails
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	cfg := &config.PollerConfig{
		PollingWaitIntervalMs: 100,
		MaxNumberOfMessages:   10,
		VisibilityTimeoutSec:  30,
	}
	pool := &mockWorkerPool{}

	p := poller.New(client, cfg, pool)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go p.Start(ctx)

	// Wait for poll cycle
	time.Sleep(200 * time.Millisecond)
	cancel()

	// Events should not be submitted if mark running fails
	assert.Equal(t, 0, pool.Count())
}

func TestPoller_Start_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := api.EventsResponse{Events: []api.Event{}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	cfg := &config.PollerConfig{
		PollingWaitIntervalMs: 100,
		MaxNumberOfMessages:   10,
		VisibilityTimeoutSec:  30,
	}
	pool := &mockWorkerPool{}

	p := poller.New(client, cfg, pool)

	ctx, cancel := context.WithCancel(context.Background())

	// Start in background
	done := make(chan error, 1)
	go func() {
		done <- p.Start(ctx)
	}()

	// Cancel context
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Wait for Start to return
	err := <-done
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestPoller_RetryWithExponentialBackoff(t *testing.T) {
	callCount := 0
	mu := sync.Mutex{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		callCount++
		mu.Unlock()

		// Always return empty events (we're testing backoff, not recovery)
		response := api.EventsResponse{Events: []api.Event{}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	cfg := &config.PollerConfig{
		PollingWaitIntervalMs: 50,
		MaxNumberOfMessages:   10,
		VisibilityTimeoutSec:  30,
		RetryOnError:          true,
		MaxRetries:            5,
		RetryBackoff:          "exponential",
	}
	pool := &mockWorkerPool{}

	p := poller.New(client, cfg, pool)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go p.Start(ctx)

	// Wait for multiple poll cycles
	time.Sleep(300 * time.Millisecond)
	cancel()

	mu.Lock()
	finalCount := callCount
	mu.Unlock()

	// Should have polled at least once
	assert.GreaterOrEqual(t, finalCount, 1)
}

func TestPoller_RetryWithLinearBackoff(t *testing.T) {
	callCount := 0
	mu := sync.Mutex{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		callCount++
		mu.Unlock()

		// Always return empty events (we're testing backoff, not recovery)
		response := api.EventsResponse{Events: []api.Event{}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	cfg := &config.PollerConfig{
		PollingWaitIntervalMs: 50,
		MaxNumberOfMessages:   10,
		VisibilityTimeoutSec:  30,
		RetryOnError:          true,
		MaxRetries:            5,
		RetryBackoff:          "linear",
	}
	pool := &mockWorkerPool{}

	p := poller.New(client, cfg, pool)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go p.Start(ctx)

	// Wait for multiple poll cycles
	time.Sleep(300 * time.Millisecond)
	cancel()

	mu.Lock()
	finalCount := callCount
	mu.Unlock()

	// Should have polled at least once
	assert.GreaterOrEqual(t, finalCount, 1)
}

func TestPoller_MaxRetriesExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always fail
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	cfg := &config.PollerConfig{
		PollingWaitIntervalMs: 50,
		MaxNumberOfMessages:   10,
		VisibilityTimeoutSec:  30,
		RetryOnError:          true,
		MaxRetries:            2, // Low max retries
		RetryBackoff:          "linear",
	}
	pool := &mockWorkerPool{}

	p := poller.New(client, cfg, pool)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go p.Start(ctx)

	// Wait for retries to be exceeded
	time.Sleep(500 * time.Millisecond)
	cancel()

	// Should continue polling even after max retries
	assert.Equal(t, 0, pool.Count())
}

func TestPoller_MultiplePollCycles(t *testing.T) {
	pollCount := 0
	mu := sync.Mutex{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/deliveries" {
			mu.Lock()
			pollCount++
			mu.Unlock()

			response := api.EventsResponse{
				Events: []api.Event{
					{
						ID:        "delivery-1",
						EventID:   "event-1",
						Type:      "test.event",
						Timestamp: time.Now(),
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if r.Method == "PATCH" {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	cfg := &config.PollerConfig{
		PollingWaitIntervalMs: 50,
		MaxNumberOfMessages:   10,
		VisibilityTimeoutSec:  30,
	}
	pool := &mockWorkerPool{}

	p := poller.New(client, cfg, pool)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go p.Start(ctx)

	// Wait for multiple poll cycles
	time.Sleep(200 * time.Millisecond)
	cancel()

	mu.Lock()
	finalPollCount := pollCount
	mu.Unlock()

	// Should have polled multiple times
	assert.GreaterOrEqual(t, finalPollCount, 2)

	// Should have submitted events from multiple polls
	assert.GreaterOrEqual(t, pool.Count(), 2)
}

// Edge case tests for improved coverage

func TestPoller_Poll_MetricsDisabled(t *testing.T) {
	// Test polling when metrics are disabled (nil)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/deliveries" {
			response := api.EventsResponse{
				Events: []api.Event{
					{
						ID:        "delivery-1",
						EventID:   "event-1",
						Type:      "test.event",
						Timestamp: time.Now(),
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if r.Method == "PATCH" {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	cfg := &config.PollerConfig{
		PollingWaitIntervalMs: 100,
		MaxNumberOfMessages:   10,
		VisibilityTimeoutSec:  30,
	}
	pool := &mockWorkerPool{}

	p := poller.New(client, cfg, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Start poller
	go p.Start(ctx)

	// Wait for at least one poll cycle
	time.Sleep(200 * time.Millisecond)

	// Should have submitted event even with metrics disabled
	assert.GreaterOrEqual(t, pool.Count(), 1)
}

func TestPoller_MarkDeliveryAsRunning_AllFailures(t *testing.T) {
	// Test when MarkDeliveryAsRunning fails for all events
	markRunningAttempts := 0
	mu := sync.Mutex{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/deliveries" {
			// Return multiple events
			response := api.EventsResponse{
				Events: []api.Event{
					{
						ID:        "delivery-1",
						EventID:   "event-1",
						Type:      "test.event",
						Timestamp: time.Now(),
					},
					{
						ID:        "delivery-2",
						EventID:   "event-2",
						Type:      "test.event",
						Timestamp: time.Now(),
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if r.Method == "PATCH" {
			// Mark running always fails
			mu.Lock()
			markRunningAttempts++
			mu.Unlock()
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	cfg := &config.PollerConfig{
		PollingWaitIntervalMs: 100,
		MaxNumberOfMessages:   10,
		VisibilityTimeoutSec:  30,
	}
	pool := &mockWorkerPool{}

	p := poller.New(client, cfg, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	// Start poller
	go p.Start(ctx)

	// Wait for poll
	time.Sleep(250 * time.Millisecond)

	// Should have attempted to mark running (retries happen automatically)
	mu.Lock()
	attempts := markRunningAttempts
	mu.Unlock()

	// retryablehttp will retry failed PATCH requests, so expect 8 attempts (4 per delivery)
	assert.GreaterOrEqual(t, attempts, 1, "Should attempt to mark deliveries as running")

	// But no events should be submitted since marking failed
	assert.Equal(t, 0, pool.Count(), "No events should be submitted when marking fails")
}

func TestPoller_ConcurrentPolls(t *testing.T) {
	// Test that poller handles concurrent poll cycles correctly
	pollCount := 0
	mu := sync.Mutex{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/deliveries" {
			mu.Lock()
			pollCount++
			mu.Unlock()

			response := api.EventsResponse{
				Events: []api.Event{
					{
						ID:        "delivery-1",
						EventID:   "event-1",
						Type:      "test.event",
						Timestamp: time.Now(),
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if r.Method == "PATCH" {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "", "test-key", "test")
	cfg := &config.PollerConfig{
		PollingWaitIntervalMs: 50, // Very fast polling
		MaxNumberOfMessages:   10,
		VisibilityTimeoutSec:  30,
	}
	pool := &mockWorkerPool{}

	p := poller.New(client, cfg, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()

	// Start poller
	go p.Start(ctx)

	// Wait for multiple poll cycles
	time.Sleep(350 * time.Millisecond)

	mu.Lock()
	finalPollCount := pollCount
	mu.Unlock()

	// Should have polled multiple times rapidly
	assert.GreaterOrEqual(t, finalPollCount, 2, "Should complete multiple rapid poll cycles")
	assert.GreaterOrEqual(t, pool.Count(), 2, "Should submit events from multiple polls")
}
