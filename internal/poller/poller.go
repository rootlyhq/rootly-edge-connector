package poller

import (
	"context"
	"fmt"
	"math"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/rootly/edge-connector/internal/api"
	"github.com/rootly/edge-connector/internal/config"
	"github.com/rootly/edge-connector/internal/metrics"
)

// WorkerPool interface for submitting events
type WorkerPool interface {
	Submit(event api.Event)
}

// Poller manages polling events from the Rootly API
type Poller struct {
	client     *api.Client
	config     *config.PollerConfig
	workerPool WorkerPool
	retryCount int
}

// New creates a new poller
func New(client *api.Client, cfg *config.PollerConfig, pool WorkerPool) *Poller {
	return &Poller{
		client:     client,
		config:     cfg,
		workerPool: pool,
		retryCount: 0,
	}
}

// Start starts the polling loop
func (p *Poller) Start(ctx context.Context) error {
	log.WithFields(log.Fields{
		"polling_interval_ms": p.config.PollingWaitIntervalMs,
		"max_messages":        p.config.MaxNumberOfMessages,
		"visibility_timeout":  p.config.VisibilityTimeoutSec,
	}).Info("Starting poller")

	ticker := time.NewTicker(time.Duration(p.config.PollingWaitIntervalMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("Poller stopped")
			return ctx.Err()
		case <-ticker.C:
			if err := p.poll(ctx); err != nil {
				log.WithError(err).Error("Polling error")
				if p.config.RetryOnError {
					p.handleError(err, ticker)
				}
			} else {
				// Reset retry count on success
				p.retryCount = 0
			}
		}
	}
}

// poll fetches and processes events from the API
func (p *Poller) poll(ctx context.Context) error {
	// Fetch events from Rootly API
	events, err := p.client.FetchEvents(ctx, p.config.MaxNumberOfMessages, p.config.VisibilityTimeoutSec)
	if err != nil {
		if metrics.EventsPolled != nil {
			metrics.EventsPolled.WithLabelValues("error").Inc()
		}
		return fmt.Errorf("failed to fetch events: %w", err)
	}

	if metrics.EventsPolled != nil {
		metrics.EventsPolled.WithLabelValues("success").Inc()
	}

	if len(events) == 0 {
		log.Debug("No events to process")
		return nil
	}

	// Record received events
	if metrics.EventsReceived != nil {
		metrics.EventsReceived.Add(float64(len(events)))
	}

	log.WithField("event_count", len(events)).Info("Fetched events from API")

	// Process each event
	for _, event := range events {
		// Mark delivery as running immediately (claims it for execution)
		if err := p.client.MarkDeliveryAsRunning(ctx, event.ID); err != nil {
			if metrics.DeliveriesMarkedRunning != nil {
				metrics.DeliveriesMarkedRunning.WithLabelValues("error").Inc()
			}
			log.WithFields(log.Fields{
				"delivery_id": event.ID,
				"event_id":    event.EventID,
			}).WithError(err).Error("Failed to mark delivery as running")
			continue
		}

		if metrics.DeliveriesMarkedRunning != nil {
			metrics.DeliveriesMarkedRunning.WithLabelValues("success").Inc()
		}

		log.WithFields(log.Fields{
			"delivery_id": event.ID,
			"event_id":    event.EventID,
			"event_type":  event.Type,
		}).Debug("Delivery marked as running")

		// Submit event to worker pool for processing
		p.workerPool.Submit(event)
	}

	return nil
}

// handleError implements retry logic with backoff
func (p *Poller) handleError(err error, ticker *time.Ticker) {
	p.retryCount++

	if p.retryCount > p.config.MaxRetries {
		log.WithFields(log.Fields{
			"retry_count": p.retryCount,
			"max_retries": p.config.MaxRetries,
		}).Error("Max retries exceeded, resetting retry count")
		p.retryCount = 0
		return
	}

	var backoffDuration time.Duration
	if p.config.RetryBackoff == "exponential" {
		// Exponential backoff: 2^retry * polling_interval
		multiplier := math.Pow(2, float64(p.retryCount))
		backoffDuration = time.Duration(float64(p.config.PollingWaitIntervalMs)*multiplier) * time.Millisecond
	} else {
		// Linear backoff: retry * polling_interval
		backoffDuration = time.Duration(p.config.PollingWaitIntervalMs*p.retryCount) * time.Millisecond
	}

	// Cap backoff at 5 minutes
	maxBackoff := 5 * time.Minute
	if backoffDuration > maxBackoff {
		backoffDuration = maxBackoff
	}

	log.WithFields(log.Fields{
		"retry_count":      p.retryCount,
		"backoff_duration": backoffDuration,
		"backoff_strategy": p.config.RetryBackoff,
		"error":            err.Error(),
	}).Warn("Backing off before next poll")

	// Reset ticker with new interval
	ticker.Reset(backoffDuration)
}
