package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/natefinch/lumberjack"
	log "github.com/sirupsen/logrus"

	"github.com/rootly/edge-connector/internal/api"
	"github.com/rootly/edge-connector/internal/config"
	"github.com/rootly/edge-connector/internal/executor"
	"github.com/rootly/edge-connector/internal/metrics"
	"github.com/rootly/edge-connector/internal/poller"
	"github.com/rootly/edge-connector/internal/reporter"
	"github.com/rootly/edge-connector/internal/worker"
	"github.com/rootly/edge-connector/pkg/git"
)

// version is set via ldflags during build
var version = "dev"

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "config.yml", "Path to configuration file")
	actionsPath := flag.String("actions", "actions.yml", "Path to actions configuration")
	showVersion := flag.Bool("version", false, "Show version and exit")
	validateOnly := flag.Bool("validate", false, "Validate configuration and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Rootly Edge Connector v%s\n", version)
		os.Exit(0)
	}

	// Validate mode: check config and actions, then exit
	if *validateOnly {
		exitCode := validateConfig(*configPath, *actionsPath)
		os.Exit(exitCode)
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := initLogger(&cfg.Logging); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	log.WithFields(log.Fields{
		"version":  version,
		"app_name": cfg.App.Name,
	}).Info("Starting Rootly Edge Connector")

	// Load actions configuration
	actionsConfig, err := config.LoadActions(*actionsPath)
	if err != nil {
		log.WithError(err).Fatal("Failed to load actions")
	}

	log.WithField("action_count", len(actionsConfig.Actions)).Info("Loaded actions configuration")

	// Initialize and start Prometheus metrics server if enabled
	var metricsServer *metrics.Server
	if cfg.Metrics.Enabled {
		// Initialize Prometheus metrics with custom labels (if any)
		metrics.InitMetrics(cfg.Metrics.Labels)

		metricsServer = metrics.NewServer(&cfg.Metrics)
		if err := metricsServer.Start(); err != nil {
			log.WithError(err).Fatal("Failed to start metrics server")
		}
		log.WithFields(log.Fields{
			"port":   cfg.Metrics.Port,
			"path":   cfg.Metrics.Path,
			"labels": cfg.Metrics.Labels,
		}).Info("Prometheus metrics server started")
	} else {
		log.Debug("Prometheus metrics disabled")
	}

	// Initialize API client with version for User-Agent header
	apiClient := api.NewClient(cfg.Rootly.APIURL, cfg.Rootly.APIPath, cfg.Rootly.APIKey, version)

	// Register all actions with backend (automatic + callable)
	registrationRequest := api.ConvertActionsToRegistrations(actionsConfig.Actions)

	if len(registrationRequest.Automatic) > 0 || len(registrationRequest.Callable) > 0 {
		log.WithFields(log.Fields{
			"automatic_count": len(registrationRequest.Automatic),
			"callable_count":  len(registrationRequest.Callable),
		}).Info("Registering actions with backend")

		resp, err := apiClient.RegisterActions(context.Background(), registrationRequest)
		if err != nil {
			log.WithError(err).Warn("Failed to register actions with backend (continuing anyway)")
		} else {
			log.WithFields(log.Fields{
				"automatic": resp.Registered.Automatic,
				"callable":  resp.Registered.Callable,
				"total":     resp.Registered.Total,
				"failed":    resp.Failed,
			}).Info("Successfully registered actions")
			if resp.Failed > 0 {
				for _, failure := range resp.Failures {
					log.WithFields(log.Fields{
						"action_slug": failure.Slug,
						"reason":      failure.Reason,
					}).Warn("Failed to register action")
				}
			}
		}
	} else {
		log.Debug("No actions to register")
	}

	// Initialize Git repository manager for git-based actions
	gitManager := git.NewManager("/tmp/rec-repos")

	// Pre-download git repositories
	for i := range actionsConfig.Actions {
		action := &actionsConfig.Actions[i]
		if action.SourceType == "git" && action.GitOptions != nil {
			log.WithField("repo_url", action.GitOptions.URL).Info("Downloading Git repository")
			if _, err := gitManager.Download(action.GitOptions); err != nil {
				log.WithError(err).WithFields(log.Fields{
					"repo_url":    action.GitOptions.URL,
					"action_id":   action.ID,
					"action_name": action.Name,
				}).Error("Failed to download Git repository - action will be skipped")
				continue // Skip this action but continue with others
			}

			// Get script path from repository
			scriptPath, err := gitManager.GetScriptPath(action.GitOptions.URL, action.Script)
			if err != nil {
				log.WithError(err).WithFields(log.Fields{
					"script":      action.Script,
					"repo_url":    action.GitOptions.URL,
					"action_id":   action.ID,
					"action_name": action.Name,
				}).Error("Failed to get script path from repository - action will be skipped")
				continue // Skip this action but continue with others
			}

			// Update action to use local script path
			action.Script = scriptPath
			log.WithFields(log.Fields{
				"action_id":   action.ID,
				"action_name": action.Name,
				"script_path": scriptPath,
			}).Debug("Updated action script path from Git repository")
		}
	}

	// Initialize script runner with git manager for repository locking
	scriptRunner := executor.NewScriptRunner(
		cfg.Security.AllowedScriptPaths,
		cfg.Security.GlobalEnv,
	)
	scriptRunner.SetGitManager(gitManager)

	// Initialize HTTP executor
	httpExecutor := executor.NewHTTPExecutor()

	// Initialize reporter
	rep := reporter.New(apiClient)

	// Initialize executor
	exec := executor.New(actionsConfig.Actions, scriptRunner, httpExecutor, rep)

	// Initialize worker pool
	pool := worker.NewPool(&cfg.Pool, exec)

	// Initialize poller
	poll := poller.New(apiClient, &cfg.Poller, pool)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start Git repository periodic pull (if any git-based actions exist)
	hasGitActions := false
	for _, action := range actionsConfig.Actions {
		if action.SourceType == "git" {
			hasGitActions = true
			break
		}
	}
	if hasGitActions {
		go gitManager.StartPeriodicPull(ctx)
		log.Info("Started periodic Git repository pull")
	}

	// Start worker pool
	pool.Start(ctx)

	// Start poller in goroutine
	go func() {
		if err := poll.Start(ctx); err != nil && err != context.Canceled {
			log.WithError(err).Error("Poller stopped unexpectedly")
		}
	}()

	log.Info("Rootly Edge Connector started successfully")

	// Wait for shutdown signal
	sig := <-sigCh
	log.WithField("signal", sig).Info("Received shutdown signal")

	// Cancel context and shutdown gracefully
	cancel()

	log.Info("Shutting down worker pool...")
	pool.Shutdown()

	// Shutdown metrics server if enabled
	if metricsServer != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := metricsServer.Shutdown(shutdownCtx); err != nil {
			log.WithError(err).Error("Error shutting down metrics server")
		}
	}

	log.Info("Shutdown complete")
}

// validateConfig validates configuration files and outputs a nice summary
// Returns exit code: 0 for success, 1 for validation errors
func validateConfig(configPath, actionsPath string) int {
	fmt.Printf("🔍 Validating Rootly Edge Connector Configuration\n\n")

	hasErrors := false

	// Validate main configuration
	fmt.Printf("📋 Checking main configuration: %s\n", configPath)
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("❌ Configuration validation FAILED:\n")
		fmt.Printf("   %v\n\n", err)
		hasErrors = true
	} else {
		fmt.Printf("✅ Main configuration is valid\n")
		fmt.Printf("   App Name: %s\n", cfg.App.Name)
		fmt.Printf("   API URL: %s\n", cfg.Rootly.APIURL)
		fmt.Printf("   Log Level: %s\n", cfg.Logging.Level)
		fmt.Printf("   Max Workers: %d\n", cfg.Pool.MaxNumberOfWorkers)
		fmt.Printf("   Metrics Enabled: %v\n\n", cfg.Metrics.Enabled)
	}

	// Validate actions configuration
	fmt.Printf("🎯 Checking actions configuration: %s\n", actionsPath)
	actionsConfig, err := config.LoadActions(actionsPath)
	if err != nil {
		fmt.Printf("❌ Actions validation FAILED:\n")
		fmt.Printf("   %v\n\n", err)
		hasErrors = true
	} else {
		fmt.Printf("✅ Actions configuration is valid\n")
		fmt.Printf("   Total actions: %d\n\n", len(actionsConfig.Actions))

		// Show action summary
		fmt.Printf("📊 Action Summary:\n")
		fmt.Printf("┌─────────────────────────────────┬──────────┬───────────┬─────────────────────────────┐\n")
		fmt.Printf("│ %-31s │ %-8s │ %-9s │ %-27s │\n", "ID", "Type", "Source", "Trigger")
		fmt.Printf("├─────────────────────────────────┼──────────┼───────────┼─────────────────────────────┤\n")

		for _, action := range actionsConfig.Actions {
			id := action.ID
			if len(id) > 31 {
				id = id[:28] + "..."
			}

			actionType := action.Type
			sourceType := action.SourceType
			if sourceType == "" {
				sourceType = "local"
			}

			triggers := action.Trigger.GetEventTypes()
			triggerStr := ""
			if len(triggers) > 0 {
				triggerStr = triggers[0]
				if len(triggerStr) > 27 {
					triggerStr = triggerStr[:24] + "..."
				}
				if len(triggers) > 1 {
					triggerStr += fmt.Sprintf(" +%d", len(triggers)-1)
				}
			}

			fmt.Printf("│ %-31s │ %-8s │ %-9s │ %-27s │\n", id, actionType, sourceType, triggerStr)
		}
		fmt.Printf("└─────────────────────────────────┴──────────┴───────────┴─────────────────────────────┘\n\n")

		// Show parameter definitions summary
		callableCount := 0
		for _, action := range actionsConfig.Actions {
			if len(action.ParameterDefinitions) > 0 {
				callableCount++
			}
		}

		if callableCount > 0 {
			fmt.Printf("📞 Callable Actions (with parameter_definitions): %d\n", callableCount)
			for _, action := range actionsConfig.Actions {
				if len(action.ParameterDefinitions) > 0 {
					fmt.Printf("   • %s (%d parameters)\n", action.ID, len(action.ParameterDefinitions))
				}
			}
			fmt.Printf("\n")
		}
	}

	// Final result
	if hasErrors {
		fmt.Printf("❌ Validation FAILED - Please fix the errors above\n")
		return 1
	}

	fmt.Printf("✅ All validations passed! Configuration is ready to use.\n")
	fmt.Printf("\nTo start the connector:\n")
	fmt.Printf("  ./bin/rootly-edge-connector -config %s -actions %s\n", configPath, actionsPath)
	return 0
}

// initLogger initializes logrus with configuration including log rotation
func initLogger(cfg *config.LoggingConfig) error {
	// Set log level
	level, err := log.ParseLevel(cfg.Level)
	if err != nil {
		return fmt.Errorf("invalid log level %q: %w", cfg.Level, err)
	}
	log.SetLevel(level)

	// Set log format
	switch cfg.Format {
	case "json":
		log.SetFormatter(&log.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	case "text":
		log.SetFormatter(&log.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	case "colored":
		log.SetFormatter(&log.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
			ForceColors:     true, // Force colors even when not a TTY (for Docker logs)
		})
	default:
		return fmt.Errorf("invalid log format %q", cfg.Format)
	}

	// Set log output with rotation
	var writer io.Writer
	if cfg.Output == "stdout" || cfg.Output == "" {
		writer = os.Stdout
	} else {
		// Use lumberjack for log rotation
		writer = &lumberjack.Logger{
			Filename:   cfg.Output,
			MaxSize:    cfg.MaxSizeMB,  // megabytes
			MaxBackups: cfg.MaxBackups, // number of backups
			MaxAge:     cfg.MaxAgeDays, // days
			Compress:   cfg.Compress,   // compress rotated files
			LocalTime:  cfg.LocalTime,  // use local time for filenames
		}
	}
	log.SetOutput(writer)

	return nil
}
