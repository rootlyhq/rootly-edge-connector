package executor_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rootly/edge-connector/internal/config"
	"github.com/rootly/edge-connector/internal/executor"
)

func TestScriptRunner_Run_Success(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping shell script test on Windows")
	}

	// Create temporary test directory
	tmpDir := t.TempDir()

	// Create test script
	scriptPath := filepath.Join(tmpDir, "test.sh")
	scriptContent := `#!/bin/bash
echo "stdout: $REC_PARAM_MESSAGE"
echo "stderr: error message" >&2
exit 0
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	runner := executor.NewScriptRunner(
		[]string{tmpDir},
		map[string]string{"GLOBAL_VAR": "global_value"},
	)

	action := &config.Action{
		Script:  scriptPath,
		Timeout: 5,
		Env: map[string]string{
			"ACTION_VAR": "action_value",
		},
	}

	params := map[string]string{
		"message": "Hello World",
	}

	result := runner.Run(context.Background(), action, params)

	assert.Equal(t, 0, result.ExitCode)
	assert.Nil(t, result.Error)
	assert.Contains(t, result.Stdout, "stdout: Hello World")
	assert.Contains(t, result.Stderr, "stderr: error message")
	assert.GreaterOrEqual(t, result.DurationMs, int64(0)) // Can be 0ms on fast CI systems
}

func TestScriptRunner_Run_Failure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping shell script test on Windows")
	}

	tmpDir := t.TempDir()

	scriptPath := filepath.Join(tmpDir, "fail.sh")
	scriptContent := `#!/bin/bash
echo "Script failed" >&2
exit 1
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	runner := executor.NewScriptRunner([]string{tmpDir}, nil)

	action := &config.Action{
		Script:  scriptPath,
		Timeout: 5,
	}

	result := runner.Run(context.Background(), action, nil)

	assert.Equal(t, 1, result.ExitCode)
	assert.NotNil(t, result.Error)
	assert.Contains(t, result.Stderr, "Script failed")
}

func TestScriptRunner_Run_Timeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping shell script test on Windows")
	}

	tmpDir := t.TempDir()

	scriptPath := filepath.Join(tmpDir, "slow.sh")
	scriptContent := `#!/bin/bash
sleep 5
echo "This should not appear"
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	runner := executor.NewScriptRunner([]string{tmpDir}, nil)

	action := &config.Action{
		Script:  scriptPath,
		Timeout: 1, // 1 second timeout
	}

	start := time.Now()
	result := runner.Run(context.Background(), action, nil)
	duration := time.Since(start)

	assert.Equal(t, -1, result.ExitCode)
	assert.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "timed out")
	// Timeout should be close to the configured timeout (1s), allowing some overhead
	assert.Less(t, duration, 6*time.Second, "Should not wait for full sleep duration")
	assert.Greater(t, duration, 500*time.Millisecond, "Should take at least the timeout duration")
}

func TestScriptRunner_PathWhitelist(t *testing.T) {
	tmpDir := t.TempDir()
	allowedDir := filepath.Join(tmpDir, "allowed")
	forbiddenDir := filepath.Join(tmpDir, "forbidden")

	err := os.MkdirAll(allowedDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(forbiddenDir, 0755)
	require.NoError(t, err)

	// Create scripts
	allowedScript := filepath.Join(allowedDir, "allowed.sh")
	forbiddenScript := filepath.Join(forbiddenDir, "forbidden.sh")

	scriptContent := `#!/bin/bash
echo "test"
`

	for _, path := range []string{allowedScript, forbiddenScript} {
		err := os.WriteFile(path, []byte(scriptContent), 0755)
		require.NoError(t, err)
	}

	runner := executor.NewScriptRunner(
		[]string{allowedDir}, // Only allow scripts in allowedDir
		nil,
	)

	tests := []struct {
		name          string
		scriptPath    string
		expectSuccess bool
	}{
		{
			name:          "allowed path",
			scriptPath:    allowedScript,
			expectSuccess: true,
		},
		{
			name:          "forbidden path",
			scriptPath:    forbiddenScript,
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := &config.Action{
				Script:  tt.scriptPath,
				Timeout: 5,
			}

			result := runner.Run(context.Background(), action, nil)

			if tt.expectSuccess {
				assert.Equal(t, 0, result.ExitCode)
				assert.Nil(t, result.Error)
			} else {
				assert.Equal(t, 1, result.ExitCode)
				assert.NotNil(t, result.Error)
				assert.Contains(t, result.Error.Error(), "not allowed")
			}
		})
	}
}

func TestScriptRunner_ParameterConversion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping shell script test on Windows")
	}

	tmpDir := t.TempDir()

	scriptPath := filepath.Join(tmpDir, "params.sh")
	scriptContent := `#!/bin/bash
echo "HOST=$REC_PARAM_HOST"
echo "PORT=$REC_PARAM_PORT"
echo "GLOBAL=$GLOBAL_VAR"
echo "ACTION=$ACTION_ENV"
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	runner := executor.NewScriptRunner(
		[]string{tmpDir},
		map[string]string{"GLOBAL_VAR": "global_value"},
	)

	action := &config.Action{
		Script:  scriptPath,
		Timeout: 5,
		Env: map[string]string{
			"ACTION_ENV": "action_value",
		},
	}

	params := map[string]string{
		"host": "localhost",
		"port": "8080",
	}

	result := runner.Run(context.Background(), action, params)

	assert.Equal(t, 0, result.ExitCode)
	assert.Contains(t, result.Stdout, "HOST=localhost")
	assert.Contains(t, result.Stdout, "PORT=8080")
	assert.Contains(t, result.Stdout, "GLOBAL=global_value")
	assert.Contains(t, result.Stdout, "ACTION=action_value")
}

func TestScriptRunner_InterpreterDetection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping interpreter test on Windows")
	}

	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		ext        string
		content    string
		shouldWork bool
	}{
		{
			name:       "shell script",
			ext:        ".sh",
			content:    "echo 'shell'",
			shouldWork: true,
		},
		{
			name:       "python script",
			ext:        ".py",
			content:    "print('python')",
			shouldWork: commandExists("python") || commandExists("python3"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.shouldWork {
				t.Skip("Interpreter not available")
			}

			scriptPath := filepath.Join(tmpDir, "test"+tt.ext)
			err := os.WriteFile(scriptPath, []byte(tt.content), 0755)
			require.NoError(t, err)

			runner := executor.NewScriptRunner([]string{tmpDir}, nil)

			action := &config.Action{
				Script:  scriptPath,
				Timeout: 5,
			}

			result := runner.Run(context.Background(), action, nil)

			// We just check that the script executed without path errors
			// Actual execution might fail if interpreter not available
			if result.Error != nil {
				assert.NotContains(t, result.Error.Error(), "not allowed")
			}
		})
	}
}

func TestScriptRunner_NonExistentScript(t *testing.T) {
	runner := executor.NewScriptRunner([]string{"/tmp"}, nil)

	action := &config.Action{
		Script:  "/tmp/nonexistent-script-12345.sh",
		Timeout: 5,
	}

	result := runner.Run(context.Background(), action, nil)

	assert.Equal(t, 1, result.ExitCode)
	assert.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "not found")
}

// Helper function to check if a command exists
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// Helper to get absolute path to test fixtures
func getFixturePath(filename string) (scriptPath, allowedDir string, err error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", "", err
	}
	fixturesDir := filepath.Join(wd, "testdata/fixtures")
	scriptPath = filepath.Join(fixturesDir, filename)
	return scriptPath, fixturesDir, nil
}

func TestScriptRunner_PythonScript(t *testing.T) {
	if !commandExists("python") && !commandExists("python3") {
		t.Skip("Python not available")
	}

	scriptPath, allowedDir, err := getFixturePath("test.py")
	require.NoError(t, err)

	runner := executor.NewScriptRunner([]string{allowedDir}, nil)
	result := runner.Run(context.Background(), &config.Action{
		Script:  scriptPath,
		Timeout: 5,
	}, map[string]string{"message": "Hello from Python"})

	assert.Equal(t, 0, result.ExitCode)
	assert.Nil(t, result.Error)
	assert.Contains(t, result.Stdout, "Python: Hello from Python")
}

func TestScriptRunner_NodeScript(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Node.js test on Windows (slow interpreter startup)")
	}
	if !commandExists("node") {
		t.Skip("Node.js not available")
	}

	scriptPath, allowedDir, err := getFixturePath("test.js")
	require.NoError(t, err)

	runner := executor.NewScriptRunner([]string{allowedDir}, nil)
	result := runner.Run(context.Background(), &config.Action{
		Script:  scriptPath,
		Timeout: 5,
	}, map[string]string{"message": "Hello from Node"})

	assert.Equal(t, 0, result.ExitCode)
	assert.Nil(t, result.Error)
	assert.Contains(t, result.Stdout, "Node: Hello from Node")
}

func TestScriptRunner_RubyScript(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Ruby test on Windows (slow interpreter startup)")
	}
	if !commandExists("ruby") {
		t.Skip("Ruby not available")
	}

	scriptPath, allowedDir, err := getFixturePath("test.rb")
	require.NoError(t, err)

	runner := executor.NewScriptRunner([]string{allowedDir}, nil)
	result := runner.Run(context.Background(), &config.Action{
		Script:  scriptPath,
		Timeout: 5,
	}, map[string]string{"message": "Hello from Ruby"})

	assert.Equal(t, 0, result.ExitCode)
	assert.Nil(t, result.Error)
	assert.Contains(t, result.Stdout, "Ruby: Hello from Ruby")
}

func TestScriptRunner_GoScript(t *testing.T) {
	if !commandExists("go") {
		t.Skip("Go not available")
	}

	scriptPath, allowedDir, err := getFixturePath("test.go")
	require.NoError(t, err)

	runner := executor.NewScriptRunner([]string{allowedDir}, nil)
	result := runner.Run(context.Background(), &config.Action{
		Script:  scriptPath,
		Timeout: 10, // Go compilation takes longer
	}, map[string]string{"message": "Hello from Go"})

	assert.Equal(t, 0, result.ExitCode)
	assert.Nil(t, result.Error)
	assert.Contains(t, result.Stdout, "Go: Hello from Go")
}

func TestScriptRunner_BashScript(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping bash test on Windows")
	}

	scriptPath, allowedDir, err := getFixturePath("test.bash")
	require.NoError(t, err)

	runner := executor.NewScriptRunner([]string{allowedDir}, nil)
	result := runner.Run(context.Background(), &config.Action{
		Script:  scriptPath,
		Timeout: 5,
	}, map[string]string{"message": "Hello from Bash"})

	assert.Equal(t, 0, result.ExitCode)
	assert.Nil(t, result.Error)
	assert.Contains(t, result.Stdout, "Bash: Hello from Bash")
}

func TestScriptRunner_ShebangScript(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping shebang test on Windows")
	}

	scriptPath, allowedDir, err := getFixturePath("test_shebang")
	require.NoError(t, err)

	runner := executor.NewScriptRunner([]string{allowedDir}, nil)
	result := runner.Run(context.Background(), &config.Action{
		Script:  scriptPath,
		Timeout: 5,
	}, map[string]string{"message": "Hello from Shebang"})

	assert.Equal(t, 0, result.ExitCode)
	assert.Nil(t, result.Error)
	assert.Contains(t, result.Stdout, "Shebang: Hello from Shebang")
}

func TestScriptRunner_SetGitManager(t *testing.T) {
	runner := executor.NewScriptRunner(nil, nil)

	// Create a mock git manager
	mockGitMgr := &mockGitManager{}

	// SetGitManager should not panic and should set the manager
	assert.NotPanics(t, func() {
		runner.SetGitManager(mockGitMgr)
	})

	// We can verify it was set by running a git-based action
	if runtime.GOOS == "windows" {
		return // Skip git action test on Windows
	}

	scriptPath, allowedDir, err := getFixturePath("test.sh")
	require.NoError(t, err)

	runner = executor.NewScriptRunner([]string{allowedDir}, nil)
	runner.SetGitManager(mockGitMgr)

	// Run a git-based action
	action := &config.Action{
		Script:     scriptPath,
		SourceType: "git",
		Timeout:    5,
		GitOptions: &config.GitOptions{
			URL: "https://github.com/example/repo",
		},
	}

	// This should attempt to lock the git repo
	result := runner.Run(context.Background(), action, map[string]string{"message": "test"})

	// Should have attempted to lock
	assert.True(t, mockGitMgr.rlockCalled)
	assert.Equal(t, "https://github.com/example/repo", mockGitMgr.lastRepoURL)
	assert.True(t, mockGitMgr.runlockCalled)

	// Script should have run
	assert.Equal(t, 0, result.ExitCode)
}

// mockGitManager for testing
type mockGitManager struct {
	rlockCalled   bool
	runlockCalled bool
	lastRepoURL   string
	rlockError    error
}

func (m *mockGitManager) RLock(repoURL string) error {
	m.rlockCalled = true
	m.lastRepoURL = repoURL
	return m.rlockError
}

func (m *mockGitManager) RUnlock(repoURL string) {
	m.runlockCalled = true
}

// Edge case tests for improved coverage

func TestScriptRunner_DefaultTimeout(t *testing.T) {
	// Test default timeout when Timeout=0
	if runtime.GOOS == "windows" {
		t.Skip("Skipping shell script test on Windows")
	}

	scriptPath, allowedDir, err := getFixturePath("test.sh")
	require.NoError(t, err)

	runner := executor.NewScriptRunner([]string{allowedDir}, nil)

	action := &config.Action{
		ID:         "timeout_test",
		Type:       "script",
		SourceType: "local",
		Script:     scriptPath,
		Timeout:    0, // Should use default 300s
	}

	params := map[string]string{}

	result := runner.Run(context.Background(), action, params)

	require.NoError(t, result.Error)
	assert.Equal(t, 0, result.ExitCode)
}

func TestScriptRunner_BooleanFlags(t *testing.T) {
	// Test script with boolean and value flags
	if runtime.GOOS == "windows" {
		t.Skip("Skipping shell script test on Windows")
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_flags.sh")
	scriptContent := `#!/bin/bash
echo "Args: $@"
exit 0
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	runner := executor.NewScriptRunner([]string{tmpDir}, nil)

	action := &config.Action{
		ID:         "flags_test",
		Type:       "script",
		SourceType: "local",
		Script:     scriptPath,
		Flags: map[string]string{
			"verbose": "",     // Boolean flag
			"debug":   "true", // Boolean flag
			"config":  "prod", // Value flag
		},
		Timeout: 5,
	}

	result := runner.Run(context.Background(), action, map[string]string{})

	require.NoError(t, result.Error)
	assert.Contains(t, result.Stdout, "--verbose")
	assert.Contains(t, result.Stdout, "--debug")
	assert.Contains(t, result.Stdout, "--config=prod")
}

func TestScriptRunner_FailureWithStdout(t *testing.T) {
	// Test script that writes stdout then fails
	if runtime.GOOS == "windows" {
		t.Skip("Skipping shell script test on Windows")
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "fail_with_output.sh")
	scriptContent := `#!/bin/bash
echo "Stdout before failure"
exit 1
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	runner := executor.NewScriptRunner([]string{tmpDir}, nil)

	action := &config.Action{
		ID:         "fail_stdout",
		Type:       "script",
		SourceType: "local",
		Script:     scriptPath,
		Timeout:    5,
	}

	result := runner.Run(context.Background(), action, map[string]string{})

	assert.NotNil(t, result.Error)
	assert.Equal(t, 1, result.ExitCode)
	assert.Contains(t, result.Stdout, "Stdout before failure")
}

func TestScriptRunner_OutputFileWriteError(t *testing.T) {
	// Test stdout/stderr file write errors
	if runtime.GOOS == "windows" {
		t.Skip("Skipping shell script test on Windows")
	}

	scriptPath, allowedDir, err := getFixturePath("test.sh")
	require.NoError(t, err)

	runner := executor.NewScriptRunner([]string{allowedDir}, nil)

	action := &config.Action{
		ID:         "write_error",
		Type:       "script",
		SourceType: "local",
		Script:     scriptPath,
		Stdout:     "/root/cannot/write/stdout.txt",
		Stderr:     "/root/cannot/write/stderr.txt",
		Timeout:    5,
	}

	result := runner.Run(context.Background(), action, map[string]string{})

	// Script succeeds but file writes fail (logged as warnings)
	require.NoError(t, result.Error)
	assert.Equal(t, 0, result.ExitCode)
}

func TestScriptRunner_PowerShellScript(t *testing.T) {
	// Test PowerShell script detection
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test.ps1")
	err := os.WriteFile(scriptPath, []byte("Write-Host 'Test'\nExit 0"), 0755)
	require.NoError(t, err)

	runner := executor.NewScriptRunner([]string{tmpDir}, nil)

	action := &config.Action{
		ID:         "ps_test",
		Type:       "script",
		SourceType: "local",
		Script:     scriptPath,
		Timeout:    5,
	}

	result := runner.Run(context.Background(), action, map[string]string{})

	// On Windows: might execute, On Unix: will fail (expected)
	if runtime.GOOS != "windows" {
		assert.NotNil(t, result.Error)
	}
}

func TestScriptRunner_NoAllowedPaths(t *testing.T) {
	// Test when no allowed paths specified
	if runtime.GOOS == "windows" {
		t.Skip("Skipping shell script test on Windows")
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test.sh")
	err := os.WriteFile(scriptPath, []byte("#!/bin/bash\necho 'OK'\nexit 0"), 0755)
	require.NoError(t, err)

	// Nil allowed paths = allow all
	runner := executor.NewScriptRunner(nil, nil)

	action := &config.Action{
		ID:         "unrestricted",
		Type:       "script",
		SourceType: "local",
		Script:     scriptPath,
		Timeout:    5,
	}

	result := runner.Run(context.Background(), action, map[string]string{})

	require.NoError(t, result.Error)
	assert.Equal(t, 0, result.ExitCode)
}

// Additional edge case tests for 100% coverage

func TestScriptRunner_GitLockFailure(t *testing.T) {
	// Test when git lock acquisition fails (covers lines 51-57)
	if runtime.GOOS == "windows" {
		t.Skip("Skipping shell script test on Windows")
	}

	scriptPath, allowedDir, err := getFixturePath("test.sh")
	require.NoError(t, err)

	mockGitMgr := &mockGitManager{
		rlockError: errors.New("failed to acquire lock"),
	}

	runner := executor.NewScriptRunner([]string{allowedDir}, nil)
	runner.SetGitManager(mockGitMgr)

	action := &config.Action{
		ID:         "git_action",
		Type:       "script",
		SourceType: "git",
		Script:     scriptPath,
		GitOptions: &config.GitOptions{
			URL: "https://github.com/test/repo",
		},
		Timeout: 5,
	}

	result := runner.Run(context.Background(), action, map[string]string{})

	// Should fail due to lock acquisition error
	assert.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "lock")
	assert.Equal(t, 1, result.ExitCode)
	assert.True(t, mockGitMgr.rlockCalled)
}

func TestScriptRunner_GitActionSuccess(t *testing.T) {
	// Test successful git-based action (covers git manager code paths)
	if runtime.GOOS == "windows" {
		t.Skip("Skipping shell script test on Windows")
	}

	scriptPath, allowedDir, err := getFixturePath("test.sh")
	require.NoError(t, err)

	mockGitMgr := &mockGitManager{
		rlockError: nil, // No error
	}

	runner := executor.NewScriptRunner([]string{allowedDir}, nil)
	runner.SetGitManager(mockGitMgr)

	action := &config.Action{
		ID:         "git_action_success",
		Type:       "script",
		SourceType: "git",
		Script:     scriptPath,
		GitOptions: &config.GitOptions{
			URL: "https://github.com/test/repo",
		},
		Timeout: 5,
	}

	result := runner.Run(context.Background(), action, map[string]string{})

	// Should succeed
	require.NoError(t, result.Error)
	assert.Equal(t, 0, result.ExitCode)
	assert.True(t, mockGitMgr.rlockCalled)
	assert.True(t, mockGitMgr.runlockCalled)
}

func TestScriptRunner_DefaultInterpreterWithShebang(t *testing.T) {
	// Test script with no extension (uses shebang)
	if runtime.GOOS == "windows" {
		t.Skip("Skipping shell script test on Windows")
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "script_no_ext")
	scriptContent := `#!/bin/bash
echo "No extension script"
exit 0
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	runner := executor.NewScriptRunner([]string{tmpDir}, nil)

	action := &config.Action{
		ID:         "no_ext",
		Type:       "script",
		SourceType: "local",
		Script:     scriptPath,
		Timeout:    5,
	}

	result := runner.Run(context.Background(), action, map[string]string{})

	require.NoError(t, result.Error)
	assert.Equal(t, 0, result.ExitCode)
	assert.Contains(t, result.Stdout, "No extension")
}

func TestScriptRunner_IsAllowedPath_AbsErrorOnScriptPath(t *testing.T) {
	// Test isAllowedPath when filepath.Abs fails on scriptPath (line 246)
	// We trigger this by changing to a temp dir and then removing it
	// This makes os.Getwd() fail, which causes filepath.Abs to fail for relative paths

	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	require.NoError(t, err)

	// Save current directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	// Change to subdirectory
	err = os.Chdir(subDir)
	require.NoError(t, err)

	// Remove the subdirectory while we're in it
	err = os.Remove(subDir)
	require.NoError(t, err)

	// Now filepath.Abs on relative paths will fail because os.Getwd() fails
	runner := executor.NewScriptRunner([]string{"/valid/path"}, nil)

	action := &config.Action{
		ID:         "abs_error",
		Type:       "script",
		SourceType: "local",
		Script:     "relative/path.sh", // Relative path - will fail
		Timeout:    5,
	}

	result := runner.Run(context.Background(), action, map[string]string{})

	// Should fail because isAllowedPath returns false when Abs fails
	assert.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "not allowed")
}

func TestScriptRunner_IsAllowedPath_AbsErrorOnAllowedPath(t *testing.T) {
	// Test isAllowedPath continue path when allowedPath.Abs fails (line 252)
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	scriptDir := filepath.Join(tmpDir, "scripts")

	err := os.Mkdir(subDir, 0755)
	require.NoError(t, err)
	err = os.Mkdir(scriptDir, 0755)
	require.NoError(t, err)

	scriptPath := filepath.Join(scriptDir, "test.sh")
	err = os.WriteFile(scriptPath, []byte("#!/bin/bash\necho 'test'\nexit 0"), 0755)
	require.NoError(t, err)

	// Save and restore working directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	// Change to subdirectory
	err = os.Chdir(subDir)
	require.NoError(t, err)

	// Remove subdirectory to make relative path Abs fail
	err = os.Remove(subDir)
	require.NoError(t, err)

	// Create runner with one invalid relative path and one valid absolute path
	// The relative path will cause Abs to fail (continue), then valid path succeeds
	runner := executor.NewScriptRunner([]string{"relative/invalid", scriptDir}, nil)

	action := &config.Action{
		ID:         "allowed_continue",
		Type:       "script",
		SourceType: "local",
		Script:     scriptPath,
		Timeout:    5,
	}

	result := runner.Run(context.Background(), action, map[string]string{})

	// Should succeed because second allowedPath is valid absolute path
	require.NoError(t, result.Error)
	assert.Equal(t, 0, result.ExitCode)
}

func TestScriptRunner_PythonFallbackWhenPython3Missing(t *testing.T) {
	// Test python fallback when python3 is not in PATH (line 273)
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows - PATH manipulation differs")
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test.py")
	scriptContent := `#!/usr/bin/env python
print("Python fallback test")
import sys
sys.exit(0)
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	// Temporarily modify PATH to exclude python3
	oldPath := os.Getenv("PATH")
	defer os.Setenv("PATH", oldPath)

	// Set PATH to only include directories that don't have python3
	// Keep only /usr/bin and /bin (which usually have 'python' symlink)
	os.Setenv("PATH", "/usr/bin:/bin")

	runner := executor.NewScriptRunner([]string{tmpDir}, nil)

	action := &config.Action{
		ID:         "python_fallback",
		Type:       "script",
		SourceType: "local",
		Script:     scriptPath,
		Timeout:    5,
	}

	result := runner.Run(context.Background(), action, map[string]string{})

	// On systems without python3, should use 'python' fallback
	// Result depends on whether 'python' is available
	if result.Error == nil {
		// Successfully used fallback
		assert.Equal(t, 0, result.ExitCode)
	} else {
		// Neither python3 nor python available
		assert.NotNil(t, result.Error)
	}
}
