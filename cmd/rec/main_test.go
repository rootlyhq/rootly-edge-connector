package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rootly/edge-connector/internal/config"
)

// Test validateConfig function

func TestValidateConfig_Success(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := validateConfig(
		"testdata/fixtures/simple_valid_config.yml",
		"testdata/fixtures/simple_valid_actions.yml",
	)

	w.Close()
	os.Stdout = old
	output, _ := io.ReadAll(r)

	assert.Equal(t, 0, exitCode)
	assert.Contains(t, string(output), "✅ All validations passed")
	assert.Contains(t, string(output), "test-connector")
	assert.Contains(t, string(output), "alert.created")
	assert.Contains(t, string(output), "📞 Callable Actions")
	assert.Contains(t, string(output), "restart_service")
	assert.Contains(t, string(output), "2 parameters")
}

func TestValidateConfig_ConfigNotFound(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := validateConfig("/nonexistent/config.yml", "/nonexistent/actions.yml")

	w.Close()
	os.Stdout = old
	output, _ := io.ReadAll(r)

	assert.Equal(t, 1, exitCode)
	assert.Contains(t, string(output), "❌")
	assert.Contains(t, string(output), "FAILED")
}

func TestValidateConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()

	// Create syntactically invalid YAML
	configPath := filepath.Join(tmpDir, "bad.yml")
	err := os.WriteFile(configPath, []byte("invalid: yaml: syntax: [[["), 0644)
	require.NoError(t, err)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := validateConfig(configPath, "../../actions.example.dev.yml")

	w.Close()
	os.Stdout = old
	output, _ := io.ReadAll(r)

	assert.Equal(t, 1, exitCode)
	assert.Contains(t, string(output), "❌")
}

func TestValidateConfig_ActionsNotFound(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := validateConfig("testdata/fixtures/simple_valid_config.yml", "/nonexistent/actions.yml")

	w.Close()
	os.Stdout = old
	output, _ := io.ReadAll(r)

	assert.Equal(t, 1, exitCode)
	assert.Contains(t, string(output), "❌")
}

func TestValidateConfig_ConfigAndActionsNotFound(t *testing.T) {
	// Test when both config and actions not found
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := validateConfig("/nonexistent/config.yml", "/nonexistent/actions.yml")

	w.Close()
	os.Stdout = old
	output, _ := io.ReadAll(r)

	assert.Equal(t, 1, exitCode)
	assert.Contains(t, string(output), "❌")
	assert.Contains(t, string(output), "FAILED")
}

// Test initLogger function - comprehensive coverage of all code paths

func TestInitLogger_JSONFormat(t *testing.T) {
	cfg := &config.LoggingConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}

	err := initLogger(cfg)
	require.NoError(t, err)

	_, ok := log.StandardLogger().Formatter.(*log.JSONFormatter)
	assert.True(t, ok, "Should use JSON formatter")
	assert.Equal(t, log.InfoLevel, log.GetLevel())
}

func TestInitLogger_TextFormat(t *testing.T) {
	cfg := &config.LoggingConfig{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	}

	err := initLogger(cfg)
	require.NoError(t, err)

	formatter, ok := log.StandardLogger().Formatter.(*log.TextFormatter)
	assert.True(t, ok, "Should use Text formatter")
	assert.False(t, formatter.ForceColors, "Text format should not force colors")
	assert.Equal(t, log.DebugLevel, log.GetLevel())
}

func TestInitLogger_ColoredFormat(t *testing.T) {
	cfg := &config.LoggingConfig{
		Level:  "warn",
		Format: "colored",
		Output: "stdout",
	}

	err := initLogger(cfg)
	require.NoError(t, err)

	formatter, ok := log.StandardLogger().Formatter.(*log.TextFormatter)
	assert.True(t, ok, "Colored format should use Text formatter")
	assert.True(t, formatter.ForceColors, "Colored format should force colors")
	assert.Equal(t, log.WarnLevel, log.GetLevel())
}

func TestInitLogger_InvalidLevel(t *testing.T) {
	cfg := &config.LoggingConfig{
		Level:  "invalid_level",
		Format: "json",
	}

	err := initLogger(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid log level")
	assert.Contains(t, err.Error(), "invalid_level")
}

func TestInitLogger_InvalidFormat(t *testing.T) {
	cfg := &config.LoggingConfig{
		Level:  "info",
		Format: "invalid_format",
	}

	err := initLogger(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid log format")
	assert.Contains(t, err.Error(), "invalid_format")
}

func TestInitLogger_FileOutputWithRotation(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "app.log")

	cfg := &config.LoggingConfig{
		Level:      "info",
		Format:     "json",
		Output:     logFile,
		MaxSizeMB:  100,
		MaxBackups: 5,
		MaxAgeDays: 30,
		Compress:   true,
		LocalTime:  false,
	}

	// Save original output to restore later
	originalOutput := log.StandardLogger().Out
	defer func() {
		// Reset to original output to close lumberjack logger (fixes Windows cleanup)
		log.SetOutput(originalOutput)
	}()

	err := initLogger(cfg)
	require.NoError(t, err)

	// Write a log message
	log.Info("Test log message")
	log.Debug("Debug message")

	// Verify file was created
	assert.FileExists(t, logFile)

	// Reset output before reading file (closes lumberjack on Windows)
	log.SetOutput(os.Stdout)

	// Read and verify content
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Test log message")
}

func TestInitLogger_StdoutOutput(t *testing.T) {
	cfg := &config.LoggingConfig{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	}

	err := initLogger(cfg)
	require.NoError(t, err)

	assert.Equal(t, os.Stdout, log.StandardLogger().Out)
	assert.Equal(t, log.ErrorLevel, log.GetLevel())
}

func TestInitLogger_EmptyOutputDefaultsToStdout(t *testing.T) {
	cfg := &config.LoggingConfig{
		Level:  "info",
		Format: "text",
		Output: "",
	}

	err := initLogger(cfg)
	require.NoError(t, err)

	// Empty output should default to stdout
	assert.Equal(t, os.Stdout, log.StandardLogger().Out)
}

func TestInitLogger_AllLogLevels(t *testing.T) {
	tests := []struct {
		level    string
		expected log.Level
	}{
		{"trace", log.TraceLevel},
		{"debug", log.DebugLevel},
		{"info", log.InfoLevel},
		{"warn", log.WarnLevel},
		{"warning", log.WarnLevel},
		{"error", log.ErrorLevel},
		{"fatal", log.FatalLevel},
		{"panic", log.PanicLevel},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			cfg := &config.LoggingConfig{
				Level:  tt.level,
				Format: "json",
				Output: "stdout",
			}

			err := initLogger(cfg)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, log.GetLevel())
		})
	}
}

func TestInitLogger_AllFormats(t *testing.T) {
	tests := []struct {
		format      string
		wantJSON    bool
		wantColored bool
	}{
		{"json", true, false},
		{"text", false, false},
		{"colored", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			cfg := &config.LoggingConfig{
				Level:  "info",
				Format: tt.format,
				Output: "stdout",
			}

			err := initLogger(cfg)
			require.NoError(t, err)

			if tt.wantJSON {
				_, ok := log.StandardLogger().Formatter.(*log.JSONFormatter)
				assert.True(t, ok)
			} else {
				formatter, ok := log.StandardLogger().Formatter.(*log.TextFormatter)
				assert.True(t, ok)
				assert.Equal(t, tt.wantColored, formatter.ForceColors)
			}
		})
	}
}
