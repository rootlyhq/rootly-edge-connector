package worker

import (
	"context"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/rootly/edge-connector/internal/api"
	"github.com/rootly/edge-connector/internal/config"
	"github.com/rootly/edge-connector/internal/metrics"
)

// Executor interface for processing events
type Executor interface {
	Execute(ctx context.Context, event api.Event)
}

// Pool manages a pool of workers for processing events
type Pool struct {
	queue      chan api.Event
	ctx        context.Context
	cancel     context.CancelFunc
	executor   Executor
	wg         sync.WaitGroup
	maxWorkers int
	minWorkers int
}

// NewPool creates a new worker pool
func NewPool(cfg *config.PoolConfig, executor Executor) *Pool {
	queueSize := cfg.QueueSize
	if queueSize == 0 {
		queueSize = 1000 // Default queue size
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Pool{
		maxWorkers: cfg.MaxNumberOfWorkers,
		minWorkers: cfg.MinNumberOfWorkers,
		queue:      make(chan api.Event, queueSize),
		executor:   executor,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start starts the worker pool with minimum number of workers
func (p *Pool) Start(ctx context.Context) {
	log.WithFields(log.Fields{
		"min_workers": p.minWorkers,
		"max_workers": p.maxWorkers,
		"queue_size":  cap(p.queue),
	}).Info("Starting worker pool")

	// Set initial pool size metric
	if metrics.WorkerPoolSize != nil {
		metrics.WorkerPoolSize.Set(float64(p.minWorkers))
	}

	// Start minimum number of workers
	for i := 0; i < p.minWorkers; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i+1)
	}
}

// Submit submits an event to the worker pool for processing
func (p *Pool) Submit(event api.Event) {
	select {
	case p.queue <- event:
		// Update queue size metric
		if metrics.WorkerPoolQueueSize != nil {
			metrics.WorkerPoolQueueSize.Set(float64(len(p.queue)))
		}
		log.WithFields(log.Fields{
			"delivery_id": event.ID,
			"event_id":    event.EventID,
			"event_type":  event.Type,
		}).Debug("Event submitted to worker pool")
	default:
		log.WithFields(log.Fields{
			"delivery_id": event.ID,
			"event_id":    event.EventID,
			"event_type":  event.Type,
		}).Warn("Worker pool queue is full, dropping event")
	}
}

// worker is a worker goroutine that processes events from the queue
func (p *Pool) worker(ctx context.Context, workerID int) {
	defer p.wg.Done()

	log.WithField("worker_id", workerID).Debug("Worker started")

	for {
		select {
		case <-ctx.Done():
			log.WithField("worker_id", workerID).Debug("Worker stopped")
			return
		case event, ok := <-p.queue:
			if !ok {
				log.WithField("worker_id", workerID).Debug("Queue closed, worker exiting")
				return
			}
			log.WithFields(log.Fields{
				"worker_id":   workerID,
				"delivery_id": event.ID,
				"event_id":    event.EventID,
				"event_type":  event.Type,
			}).Debug("Worker processing event")
			p.executor.Execute(ctx, event)
		}
	}
}

// Shutdown gracefully shuts down the worker pool
func (p *Pool) Shutdown() {
	log.Info("Shutting down worker pool")
	close(p.queue)
	p.wg.Wait()
	log.Info("Worker pool shut down complete")
}

// QueueSize returns the current number of events in the queue
func (p *Pool) QueueSize() int {
	return len(p.queue)
}

// QueueCapacity returns the maximum capacity of the queue
func (p *Pool) QueueCapacity() int {
	return cap(p.queue)
}
