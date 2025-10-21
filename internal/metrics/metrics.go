package metrics

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"

	"github.com/rootly/edge-connector/internal/config"
)

var (
	// Events metrics
	EventsPolled *prometheus.CounterVec

	EventsReceived prometheus.Counter

	// Action execution metrics
	ActionsExecuted *prometheus.CounterVec

	ActionExecutionDuration *prometheus.HistogramVec

	// Delivery status metrics
	DeliveriesMarkedRunning *prometheus.CounterVec

	EventsRunning prometheus.Gauge

	// Worker pool metrics
	WorkerPoolSize prometheus.Gauge

	WorkerPoolQueueSize prometheus.Gauge

	// HTTP client metrics (for HTTP actions)
	HTTPRequestsTotal *prometheus.CounterVec

	HTTPRequestDuration *prometheus.HistogramVec

	// Git repository metrics
	GitPullsTotal *prometheus.CounterVec

	GitPullDuration *prometheus.HistogramVec

	// Ensure metrics are only initialized once
	once sync.Once
)

// InitMetrics initializes all Prometheus metrics with optional custom labels
// This must be called before the metrics server starts
func InitMetrics(customLabels map[string]string) {
	once.Do(func() {
		constLabels := prometheus.Labels(customLabels)

		EventsPolled = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "rec_events_polled_total",
				Help:        "Total number of events polled from Rootly API",
				ConstLabels: constLabels,
			},
			[]string{"status"}, // success, error
		)

		EventsReceived = prometheus.NewCounter(
			prometheus.CounterOpts{
				Name:        "rec_events_received_total",
				Help:        "Total number of events received from Rootly",
				ConstLabels: constLabels,
			},
		)

		ActionsExecuted = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "rec_actions_executed_total",
				Help:        "Total number of actions executed",
				ConstLabels: constLabels,
			},
			[]string{"action_name", "action_type", "status"}, // status: completed, failed
		)

		ActionExecutionDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:        "rec_action_execution_duration_seconds",
				Help:        "Duration of action execution in seconds",
				Buckets:     prometheus.DefBuckets, // 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10
				ConstLabels: constLabels,
			},
			[]string{"action_name", "action_type"},
		)

		DeliveriesMarkedRunning = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "rec_deliveries_marked_running_total",
				Help:        "Total number of deliveries marked as running",
				ConstLabels: constLabels,
			},
			[]string{"status"}, // success, error
		)

		EventsRunning = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name:        "rec_events_running",
				Help:        "Number of events currently running (being executed)",
				ConstLabels: constLabels,
			},
		)

		WorkerPoolSize = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name:        "rec_worker_pool_size",
				Help:        "Current number of worker goroutines",
				ConstLabels: constLabels,
			},
		)

		WorkerPoolQueueSize = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name:        "rec_worker_pool_queue_size",
				Help:        "Current number of jobs in worker pool queue",
				ConstLabels: constLabels,
			},
		)

		HTTPRequestsTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "rec_http_requests_total",
				Help:        "Total HTTP requests made by HTTP actions",
				ConstLabels: constLabels,
			},
			[]string{"method", "status_code"},
		)

		HTTPRequestDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:        "rec_http_request_duration_seconds",
				Help:        "HTTP request duration in seconds",
				Buckets:     prometheus.DefBuckets,
				ConstLabels: constLabels,
			},
			[]string{"method"},
		)

		GitPullsTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "rec_git_pulls_total",
				Help:        "Total number of git pulls executed",
				ConstLabels: constLabels,
			},
			[]string{"repository", "status"}, // success, error
		)

		GitPullDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:        "rec_git_pull_duration_seconds",
				Help:        "Git pull duration in seconds",
				Buckets:     prometheus.DefBuckets,
				ConstLabels: constLabels,
			},
			[]string{"repository"},
		)

		// Register all metrics with Prometheus default registry
		prometheus.MustRegister(EventsPolled)
		prometheus.MustRegister(EventsReceived)
		prometheus.MustRegister(ActionsExecuted)
		prometheus.MustRegister(ActionExecutionDuration)
		prometheus.MustRegister(DeliveriesMarkedRunning)
		prometheus.MustRegister(EventsRunning)
		prometheus.MustRegister(WorkerPoolSize)
		prometheus.MustRegister(WorkerPoolQueueSize)
		prometheus.MustRegister(HTTPRequestsTotal)
		prometheus.MustRegister(HTTPRequestDuration)
		prometheus.MustRegister(GitPullsTotal)
		prometheus.MustRegister(GitPullDuration)

		log.WithField("custom_labels", customLabels).Debug("Initialized Prometheus metrics with custom labels")
	})
}

// Server represents the Prometheus metrics HTTP server
type Server struct {
	server *http.Server
}

// NewServer creates a new metrics server
func NewServer(cfg *config.MetricsConfig) *Server {
	mux := http.NewServeMux()
	mux.Handle(cfg.Path, promhttp.Handler())

	return &Server{
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Port),
			Handler: mux,
		},
	}
}

// Start starts the metrics server in a goroutine
func (s *Server) Start() error {
	log.WithField("address", s.server.Addr).Info("Starting Prometheus metrics server")

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Error("Metrics server error")
		}
	}()

	return nil
}

// Shutdown gracefully stops the metrics server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Info("Shutting down metrics server")
	return s.server.Shutdown(ctx)
}

// RecordActionExecution records metrics for an action execution
func RecordActionExecution(actionName, actionType, status string, duration time.Duration) {
	if ActionsExecuted == nil || ActionExecutionDuration == nil {
		return // Metrics not initialized (disabled)
	}
	ActionsExecuted.WithLabelValues(actionName, actionType, status).Inc()
	ActionExecutionDuration.WithLabelValues(actionName, actionType).Observe(duration.Seconds())
}

// RecordHTTPRequest records metrics for an HTTP request
func RecordHTTPRequest(method string, statusCode int, duration time.Duration) {
	if HTTPRequestsTotal == nil || HTTPRequestDuration == nil {
		return // Metrics not initialized (disabled)
	}
	HTTPRequestsTotal.WithLabelValues(method, fmt.Sprintf("%d", statusCode)).Inc()
	HTTPRequestDuration.WithLabelValues(method).Observe(duration.Seconds())
}

// RecordGitPull records metrics for a git pull operation
func RecordGitPull(repository, status string, duration time.Duration) {
	if GitPullsTotal == nil || GitPullDuration == nil {
		return // Metrics not initialized (disabled)
	}
	GitPullsTotal.WithLabelValues(repository, status).Inc()
	GitPullDuration.WithLabelValues(repository).Observe(duration.Seconds())
}
