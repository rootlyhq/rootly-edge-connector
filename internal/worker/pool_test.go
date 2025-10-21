package worker_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rootly/edge-connector/internal/api"
	"github.com/rootly/edge-connector/internal/config"
	"github.com/rootly/edge-connector/internal/worker"
)

// mockExecutor implements worker.Executor for testing
type mockExecutor struct {
	mu       sync.Mutex
	executed []api.Event
	delay    time.Duration
}

func (m *mockExecutor) Execute(ctx context.Context, event api.Event) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executed = append(m.executed, event)
}

func (m *mockExecutor) GetExecuted() []api.Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]api.Event{}, m.executed...)
}

func (m *mockExecutor) ExecutedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.executed)
}

func TestNewPool(t *testing.T) {
	cfg := &config.PoolConfig{
		MaxNumberOfWorkers: 10,
		MinNumberOfWorkers: 2,
		QueueSize:          100,
	}

	executor := &mockExecutor{}
	pool := worker.NewPool(cfg, executor)

	require.NotNil(t, pool)
	assert.Equal(t, 100, pool.QueueCapacity())
	assert.Equal(t, 0, pool.QueueSize())
}

func TestNewPool_DefaultQueueSize(t *testing.T) {
	cfg := &config.PoolConfig{
		MaxNumberOfWorkers: 10,
		MinNumberOfWorkers: 2,
		QueueSize:          0, // Should use default
	}

	executor := &mockExecutor{}
	pool := worker.NewPool(cfg, executor)

	require.NotNil(t, pool)
	assert.Equal(t, 1000, pool.QueueCapacity(), "Should use default queue size of 1000")
}

func TestPool_Submit(t *testing.T) {
	cfg := &config.PoolConfig{
		MaxNumberOfWorkers: 5,
		MinNumberOfWorkers: 1,
		QueueSize:          10,
	}

	executor := &mockExecutor{}
	pool := worker.NewPool(cfg, executor)

	event := api.Event{
		ID:      "delivery-1",
		EventID: "event-1",
		Type:    "test.event",
	}

	pool.Submit(event)

	assert.Equal(t, 1, pool.QueueSize())
}

func TestPool_SubmitMultiple(t *testing.T) {
	cfg := &config.PoolConfig{
		MaxNumberOfWorkers: 5,
		MinNumberOfWorkers: 1,
		QueueSize:          10,
	}

	executor := &mockExecutor{}
	pool := worker.NewPool(cfg, executor)

	// Submit multiple events
	for i := 0; i < 5; i++ {
		event := api.Event{
			ID:      "delivery-" + string(rune(i)),
			EventID: "event-" + string(rune(i)),
			Type:    "test.event",
		}
		pool.Submit(event)
	}

	assert.Equal(t, 5, pool.QueueSize())
}

func TestPool_SubmitDropsWhenFull(t *testing.T) {
	cfg := &config.PoolConfig{
		MaxNumberOfWorkers: 5,
		MinNumberOfWorkers: 1,
		QueueSize:          3, // Small queue
	}

	executor := &mockExecutor{}
	pool := worker.NewPool(cfg, executor)

	// Fill the queue
	for i := 0; i < 3; i++ {
		event := api.Event{
			ID:      "delivery-" + string(rune(i)),
			EventID: "event-" + string(rune(i)),
			Type:    "test.event",
		}
		pool.Submit(event)
	}

	assert.Equal(t, 3, pool.QueueSize())

	// Try to add one more (should be dropped)
	extraEvent := api.Event{
		ID:      "delivery-extra",
		EventID: "event-extra",
		Type:    "test.event",
	}
	pool.Submit(extraEvent)

	// Queue should still be 3
	assert.Equal(t, 3, pool.QueueSize())
}

func TestPool_StartAndProcessEvents(t *testing.T) {
	cfg := &config.PoolConfig{
		MaxNumberOfWorkers: 5,
		MinNumberOfWorkers: 2,
		QueueSize:          10,
	}

	executor := &mockExecutor{}
	pool := worker.NewPool(cfg, executor)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the pool
	pool.Start(ctx)

	// Submit events
	events := []api.Event{
		{ID: "delivery-1", EventID: "event-1", Type: "test.event.1"},
		{ID: "delivery-2", EventID: "event-2", Type: "test.event.2"},
		{ID: "delivery-3", EventID: "event-3", Type: "test.event.3"},
	}

	for _, event := range events {
		pool.Submit(event)
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Shutdown pool
	pool.Shutdown()

	// Verify all events were processed
	executed := executor.GetExecuted()
	assert.Equal(t, 3, len(executed))

	// Extract IDs (order not guaranteed due to concurrent workers)
	executedIDs := make([]string, 0, len(executed))
	for _, event := range executed {
		executedIDs = append(executedIDs, event.ID)
	}

	// Verify all expected IDs are present (order-independent)
	assert.ElementsMatch(t, []string{"delivery-1", "delivery-2", "delivery-3"}, executedIDs)
}

func TestPool_Shutdown(t *testing.T) {
	cfg := &config.PoolConfig{
		MaxNumberOfWorkers: 5,
		MinNumberOfWorkers: 2,
		QueueSize:          10,
	}

	executor := &mockExecutor{}
	pool := worker.NewPool(cfg, executor)

	ctx := context.Background()
	pool.Start(ctx)

	// Submit some events
	for i := 0; i < 5; i++ {
		event := api.Event{
			ID:      "delivery-" + string(rune(i)),
			EventID: "event-" + string(rune(i)),
			Type:    "test.event",
		}
		pool.Submit(event)
	}

	// Shutdown should wait for all events to be processed
	pool.Shutdown()

	// Verify all events were processed
	executed := executor.GetExecuted()
	assert.Equal(t, 5, len(executed))
}

func TestPool_QueueSizeAndCapacity(t *testing.T) {
	cfg := &config.PoolConfig{
		MaxNumberOfWorkers: 5,
		MinNumberOfWorkers: 1,
		QueueSize:          20,
	}

	executor := &mockExecutor{}
	pool := worker.NewPool(cfg, executor)

	assert.Equal(t, 20, pool.QueueCapacity())
	assert.Equal(t, 0, pool.QueueSize())

	// Add some events
	for i := 0; i < 5; i++ {
		event := api.Event{
			ID:      "delivery-" + string(rune(i)),
			EventID: "event-" + string(rune(i)),
			Type:    "test.event",
		}
		pool.Submit(event)
	}

	assert.Equal(t, 5, pool.QueueSize())
	assert.Equal(t, 20, pool.QueueCapacity())
}

func TestPool_WorkerStopsOnContextCancel(t *testing.T) {
	cfg := &config.PoolConfig{
		MaxNumberOfWorkers: 5,
		MinNumberOfWorkers: 2,
		QueueSize:          10,
	}

	executor := &mockExecutor{
		delay: 50 * time.Millisecond, // Add delay to processing
	}
	pool := worker.NewPool(cfg, executor)

	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)

	// Submit an event
	event := api.Event{
		ID:      "delivery-1",
		EventID: "event-1",
		Type:    "test.event",
	}
	pool.Submit(event)

	// Cancel context before event can be processed
	cancel()

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Shutdown
	pool.Shutdown()

	// Event should still be processed or the worker should exit gracefully
	// (This tests that context cancellation doesn't cause panics)
}

func TestPool_ConcurrentSubmit(t *testing.T) {
	cfg := &config.PoolConfig{
		MaxNumberOfWorkers: 10,
		MinNumberOfWorkers: 3,
		QueueSize:          100,
	}

	executor := &mockExecutor{}
	pool := worker.NewPool(cfg, executor)

	ctx := context.Background()
	pool.Start(ctx)

	// Concurrently submit events
	var wg sync.WaitGroup
	numGoroutines := 10
	eventsPerGoroutine := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				event := api.Event{
					ID:      "delivery-" + string(rune(goroutineID*100+j)),
					EventID: "event-" + string(rune(goroutineID*100+j)),
					Type:    "test.event",
				}
				pool.Submit(event)
			}
		}(i)
	}

	wg.Wait()

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Shutdown
	pool.Shutdown()

	// Verify all events were processed
	executed := executor.GetExecuted()
	assert.Equal(t, numGoroutines*eventsPerGoroutine, len(executed))
}

func TestPool_ProcessingWithMultipleWorkers(t *testing.T) {
	cfg := &config.PoolConfig{
		MaxNumberOfWorkers: 10,
		MinNumberOfWorkers: 5, // Multiple workers
		QueueSize:          50,
	}

	executor := &mockExecutor{
		delay: 10 * time.Millisecond, // Small delay to simulate work
	}
	pool := worker.NewPool(cfg, executor)

	ctx := context.Background()
	pool.Start(ctx)

	// Submit many events
	numEvents := 20
	for i := 0; i < numEvents; i++ {
		event := api.Event{
			ID:      "delivery-" + string(rune(i)),
			EventID: "event-" + string(rune(i)),
			Type:    "test.event",
		}
		pool.Submit(event)
	}

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	// Shutdown
	pool.Shutdown()

	// Verify all events were processed
	executed := executor.GetExecuted()
	assert.Equal(t, numEvents, len(executed))
}

// Edge case tests for improved coverage

func TestPool_StartWithZeroWorkers(t *testing.T) {
	// Test edge case: Start with 0 minimum workers
	cfg := &config.PoolConfig{
		MaxNumberOfWorkers: 5,
		MinNumberOfWorkers: 0, // Zero workers
		QueueSize:          10,
	}

	executor := &mockExecutor{}
	pool := worker.NewPool(cfg, executor)

	ctx := context.Background()
	pool.Start(ctx)

	// Submit an event
	event := api.Event{
		ID:      "delivery-1",
		EventID: "event-1",
		Type:    "test.event",
	}
	pool.Submit(event)

	// With zero workers, the event should remain in queue
	assert.Equal(t, 1, pool.QueueSize())

	// Shutdown should be graceful even with no workers
	pool.Shutdown()

	// Event should not be processed since there were no workers
	executed := executor.GetExecuted()
	assert.Equal(t, 0, len(executed))
}

func TestPool_SubmitWithCanceledContext(t *testing.T) {
	// Test submitting events after context is canceled
	cfg := &config.PoolConfig{
		MaxNumberOfWorkers: 5,
		MinNumberOfWorkers: 2,
		QueueSize:          10,
	}

	executor := &mockExecutor{}
	pool := worker.NewPool(cfg, executor)

	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)

	// Cancel context immediately
	cancel()

	// Wait for workers to stop
	time.Sleep(50 * time.Millisecond)

	// Submit event after context cancellation
	event := api.Event{
		ID:      "delivery-1",
		EventID: "event-1",
		Type:    "test.event",
	}
	pool.Submit(event)

	// Event should be queued but workers are stopped
	assert.Equal(t, 1, pool.QueueSize())

	// Shutdown should be graceful
	pool.Shutdown()
}

func TestPool_QueueOverflowWithBackpressure(t *testing.T) {
	// Test queue overflow with slow executor
	cfg := &config.PoolConfig{
		MaxNumberOfWorkers: 2,
		MinNumberOfWorkers: 1,
		QueueSize:          5, // Small queue
	}

	executor := &mockExecutor{
		delay: 200 * time.Millisecond, // Very slow processing
	}
	pool := worker.NewPool(cfg, executor)

	ctx := context.Background()
	pool.Start(ctx)

	// Submit more events than queue can hold
	submitted := 0
	for i := 0; i < 10; i++ {
		event := api.Event{
			ID:      "delivery-" + string(rune(i)),
			EventID: "event-" + string(rune(i)),
			Type:    "test.event",
		}
		queueSizeBefore := pool.QueueSize()
		pool.Submit(event)
		queueSizeAfter := pool.QueueSize()

		// Count successfully submitted events
		if queueSizeAfter > queueSizeBefore {
			submitted++
		}
	}

	// Should have dropped some events due to full queue
	assert.LessOrEqual(t, pool.QueueSize(), pool.QueueCapacity())
	assert.Less(t, submitted, 10, "Some events should have been dropped")

	// Shutdown
	pool.Shutdown()
}

func TestPool_ShutdownDuringActiveExecution(t *testing.T) {
	// Test graceful shutdown while events are actively processing
	cfg := &config.PoolConfig{
		MaxNumberOfWorkers: 3,
		MinNumberOfWorkers: 2,
		QueueSize:          10,
	}

	executor := &mockExecutor{
		delay: 100 * time.Millisecond, // Moderate delay
	}
	pool := worker.NewPool(cfg, executor)

	ctx := context.Background()
	pool.Start(ctx)

	// Submit several events
	numEvents := 5
	for i := 0; i < numEvents; i++ {
		event := api.Event{
			ID:      "delivery-" + string(rune(i)),
			EventID: "event-" + string(rune(i)),
			Type:    "test.event",
		}
		pool.Submit(event)
	}

	// Wait a bit for some processing to start
	time.Sleep(50 * time.Millisecond)

	// Shutdown should wait for all in-flight events to complete
	pool.Shutdown()

	// All events should have been processed
	executed := executor.GetExecuted()
	assert.Equal(t, numEvents, len(executed), "All events should be processed before shutdown completes")
}

func TestPool_ContextCancellationDuringExecution(t *testing.T) {
	// Test context cancellation while events are being executed
	cfg := &config.PoolConfig{
		MaxNumberOfWorkers: 3,
		MinNumberOfWorkers: 2,
		QueueSize:          10,
	}

	executor := &mockExecutor{
		delay: 150 * time.Millisecond, // Long enough to cancel during execution
	}
	pool := worker.NewPool(cfg, executor)

	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)

	// Submit events
	for i := 0; i < 5; i++ {
		event := api.Event{
			ID:      "delivery-" + string(rune(i)),
			EventID: "event-" + string(rune(i)),
			Type:    "test.event",
		}
		pool.Submit(event)
	}

	// Wait for processing to start
	time.Sleep(50 * time.Millisecond)

	// Cancel context while events are being processed
	cancel()

	// Wait a bit for workers to receive cancellation
	time.Sleep(100 * time.Millisecond)

	// Shutdown should be graceful
	pool.Shutdown()

	// Some events should have been processed before cancellation
	executed := executor.GetExecuted()
	assert.GreaterOrEqual(t, len(executed), 1, "At least some events should be processed")
}

func TestPool_EmptyQueueShutdown(t *testing.T) {
	// Test shutdown with empty queue
	cfg := &config.PoolConfig{
		MaxNumberOfWorkers: 5,
		MinNumberOfWorkers: 2,
		QueueSize:          10,
	}

	executor := &mockExecutor{}
	pool := worker.NewPool(cfg, executor)

	ctx := context.Background()
	pool.Start(ctx)

	// Don't submit any events

	// Immediate shutdown with empty queue
	pool.Shutdown()

	// No events processed
	executed := executor.GetExecuted()
	assert.Equal(t, 0, len(executed))
}

func TestPool_RapidSubmitAndShutdown(t *testing.T) {
	// Test rapid submission followed by immediate shutdown
	cfg := &config.PoolConfig{
		MaxNumberOfWorkers: 5,
		MinNumberOfWorkers: 3,
		QueueSize:          50,
	}

	executor := &mockExecutor{
		delay: 10 * time.Millisecond,
	}
	pool := worker.NewPool(cfg, executor)

	ctx := context.Background()
	pool.Start(ctx)

	// Rapidly submit many events
	numEvents := 30
	for i := 0; i < numEvents; i++ {
		event := api.Event{
			ID:      "delivery-" + string(rune(i)),
			EventID: "event-" + string(rune(i)),
			Type:    "test.event",
		}
		pool.Submit(event)
	}

	// Immediate shutdown (don't wait)
	pool.Shutdown()

	// All queued events should still be processed
	executed := executor.GetExecuted()
	assert.Equal(t, numEvents, len(executed), "All submitted events should be processed even with immediate shutdown")
}

func TestPool_SingleWorkerProcessing(t *testing.T) {
	// Test with exactly one worker (edge case for concurrency)
	cfg := &config.PoolConfig{
		MaxNumberOfWorkers: 1,
		MinNumberOfWorkers: 1, // Single worker
		QueueSize:          10,
	}

	executor := &mockExecutor{}
	pool := worker.NewPool(cfg, executor)

	ctx := context.Background()
	pool.Start(ctx)

	// Submit sequential events
	events := []api.Event{
		{ID: "delivery-1", EventID: "event-1", Type: "test.1"},
		{ID: "delivery-2", EventID: "event-2", Type: "test.2"},
		{ID: "delivery-3", EventID: "event-3", Type: "test.3"},
	}

	for _, event := range events {
		pool.Submit(event)
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Shutdown
	pool.Shutdown()

	// Verify all events processed in order (single worker = sequential)
	executed := executor.GetExecuted()
	assert.Equal(t, 3, len(executed))

	// With single worker, events should be processed in order
	assert.Equal(t, "delivery-1", executed[0].ID)
	assert.Equal(t, "delivery-2", executed[1].ID)
	assert.Equal(t, "delivery-3", executed[2].ID)
}
