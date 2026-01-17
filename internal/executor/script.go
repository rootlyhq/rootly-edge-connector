package executor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/rootly/edge-connector/internal/config"
	"github.com/rootly/edge-connector/internal/reporter"
)

// GitManager interface for repository locking
type GitManager interface {
	RLock(repoURL string) error
	RUnlock(repoURL string)
}

// ScriptRunner handles script execution
type ScriptRunner struct {
	gitManager   GitManager
	globalEnv    map[string]string
	allowedPaths []string
}

// NewScriptRunner creates a new script runner
func NewScriptRunner(allowedPaths []string, globalEnv map[string]string) *ScriptRunner {
	return &ScriptRunner{
		allowedPaths: allowedPaths,
		globalEnv:    globalEnv,
	}
}

// SetGitManager sets the git manager for repository locking
func (r *ScriptRunner) SetGitManager(gitManager GitManager) {
	r.gitManager = gitManager
}

// Run executes a script with the given action configuration and parameters
func (r *ScriptRunner) Run(ctx context.Context, action *config.Action, params map[string]string) reporter.ScriptResult {
	start := time.Now()

	// Acquire read lock on git repository if this is a git-based action
	if r.gitManager != nil && action.SourceType == "git" && action.GitOptions != nil {
		if err := r.gitManager.RLock(action.GitOptions.URL); err != nil {
			log.WithError(err).WithField("repo_url", action.GitOptions.URL).Error("Failed to acquire git repository lock")
			return reporter.ScriptResult{
				ExitCode:   1,
				DurationMs: 0,
				Error:      fmt.Errorf("failed to acquire git repository lock: %w", err),
			}
		}
		defer r.gitManager.RUnlock(action.GitOptions.URL)
		log.WithField("repo_url", action.GitOptions.URL).Debug("Acquired read lock on git repository")
	}

	// Validate script path
	if !r.isAllowedPath(action.Script) {
		var allowedPathsMsg string
		if len(r.allowedPaths) == 0 {
			allowedPathsMsg = "all paths (no restrictions)"
		} else {
			allowedPathsMsg = fmt.Sprintf("%v", r.allowedPaths)
		}

		return reporter.ScriptResult{
			ExitCode:   1,
			DurationMs: 0,
			Error: fmt.Errorf(
				"script path '%s' is not within allowed paths. "+
					"Allowed paths: %s. "+
					"To fix: add your script directory to 'security.allowed_script_paths' in config.yml, "+
					"or set 'allowed_script_paths: []' to allow all paths",
				action.Script,
				allowedPathsMsg,
			),
		}
	}

	// Check if script exists
	if _, err := os.Stat(action.Script); os.IsNotExist(err) {
		return reporter.ScriptResult{
			ExitCode:   1,
			DurationMs: 0,
			Error:      fmt.Errorf("script not found: %s", action.Script),
		}
	}

	// Create context with timeout
	timeout := time.Duration(action.Timeout) * time.Second
	if timeout == 0 {
		timeout = 300 * time.Second // Default 5 minutes
	}
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Detect script interpreter based on file extension
	interpreter, interpreterArgs := r.detectInterpreter(action.Script)

	// Build command arguments: flags first, then positional args
	cmdArgs := []string{}

	// Add flags (e.g., --verbose, --config=value)
	for key, value := range action.Flags {
		if value == "" || value == "true" {
			// Boolean flag: --verbose
			cmdArgs = append(cmdArgs, fmt.Sprintf("--%s", key))
		} else {
			// Value flag: --config=value
			cmdArgs = append(cmdArgs, fmt.Sprintf("--%s=%s", key, value))
		}
	}

	// Add positional arguments
	cmdArgs = append(cmdArgs, action.Args...)

	// Build command
	var cmd *exec.Cmd
	if interpreter != "" {
		args := append(interpreterArgs, action.Script)
		args = append(args, cmdArgs...)
		cmd = exec.CommandContext(ctxWithTimeout, interpreter, args...)
	} else {
		// Execute directly (assumes script has shebang)
		allArgs := append([]string{action.Script}, cmdArgs...)
		cmd = exec.CommandContext(ctxWithTimeout, allArgs[0], allArgs[1:]...)
	}

	// Set working directory to script directory
	cmd.Dir = filepath.Dir(action.Script)

	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range r.globalEnv {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}
	for key, value := range action.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}
	for key, value := range params {
		cmd.Env = append(cmd.Env, fmt.Sprintf("REC_PARAM_%s=%s", strings.ToUpper(key), value))
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	log.WithFields(log.Fields{
		"script":  action.Script,
		"timeout": timeout,
	}).Info("Executing script")

	// Execute command
	err := cmd.Run()

	duration := time.Since(start)
	result := reporter.ScriptResult{
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		DurationMs: duration.Milliseconds(),
	}

	// Check for timeout
	if ctxWithTimeout.Err() == context.DeadlineExceeded {
		result.ExitCode = -1
		result.Error = fmt.Errorf("script timed out after %v", timeout)
		log.WithFields(log.Fields{
			"script":  action.Script,
			"timeout": timeout,
		}).Error("Script execution timed out")
		return result
	}

	// Check for execution error
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
		result.Error = err
		log.WithFields(log.Fields{
			"script":    action.Script,
			"exit_code": result.ExitCode,
			"error":     err,
		}).Error("Script execution failed")

		// Log stdout at DEBUG level if present
		if result.Stdout != "" {
			log.WithFields(log.Fields{
				"script": action.Script,
				"output": "stdout",
			}).Debugf("Script output (stdout):\n%s", result.Stdout)
		}

		// Log stderr at DEBUG level
		if result.Stderr != "" {
			log.WithFields(log.Fields{
				"script": action.Script,
				"output": "stderr",
			}).Debugf("Script output (stderr):\n%s", result.Stderr)
		}

		return result
	}

	result.ExitCode = 0
	log.WithFields(log.Fields{
		"script":      action.Script,
		"duration_ms": result.DurationMs,
	}).Info("Script executed successfully")

	// Log stdout at DEBUG level if present
	if result.Stdout != "" {
		log.WithFields(log.Fields{
			"script": action.Script,
			"output": "stdout",
		}).Debugf("Script output (stdout):\n%s", result.Stdout)
	}

	// Log stderr at DEBUG level if present (even on success, stderr may have warnings)
	if result.Stderr != "" {
		log.WithFields(log.Fields{
			"script": action.Script,
			"output": "stderr",
		}).Debugf("Script output (stderr):\n%s", result.Stderr)
	}

	// Write output to files if configured
	if action.Stdout != "" {
		if err := os.WriteFile(action.Stdout, stdout.Bytes(), 0644); err != nil {
			log.WithError(err).Warn("Failed to write stdout to file")
		}
	}
	if action.Stderr != "" {
		if err := os.WriteFile(action.Stderr, stderr.Bytes(), 0644); err != nil {
			log.WithError(err).Warn("Failed to write stderr to file")
		}
	}

	return result
}

// isAllowedPath checks if the script path is within allowed paths
func (r *ScriptRunner) isAllowedPath(scriptPath string) bool {
	// If no allowed paths specified, allow all
	if len(r.allowedPaths) == 0 {
		return true
	}

	absPath, err := filepath.Abs(scriptPath)
	if err != nil {
		return false
	}

	for _, allowedPath := range r.allowedPaths {
		absAllowed, err := filepath.Abs(allowedPath)
		if err != nil {
			continue
		}
		// Check if script is within allowed path
		if strings.HasPrefix(absPath, absAllowed) {
			return true
		}
	}

	return false
}

// detectInterpreter detects the appropriate interpreter based on file extension
func (r *ScriptRunner) detectInterpreter(scriptPath string) (string, []string) {
	ext := strings.ToLower(filepath.Ext(scriptPath))

	switch ext {
	case ".py":
		// Try python3 first (modern systems), fall back to python
		if _, err := exec.LookPath("python3"); err == nil {
			return "python3", nil
		}
		return "python", nil
	case ".sh":
		return "sh", nil
	case ".bash":
		return "bash", nil
	case ".ps1":
		return "powershell", []string{"-File"}
	case ".rb":
		return "ruby", nil
	case ".js":
		return "node", nil
	case ".go":
		return "go", []string{"run"}
	default:
		// No interpreter, assume executable with shebang
		return "", nil
	}
}
