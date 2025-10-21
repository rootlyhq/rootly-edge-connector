package executor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Internal tests for unexported functions (can access private methods)
// These tests cover edge cases that are difficult to trigger from external tests

func TestIsAllowedPath_AbsErrorOnScriptPath(t *testing.T) {
	// Test when filepath.Abs fails on scriptPath by using current dir after it's deleted
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "testdir")
	err := os.Mkdir(subDir, 0755)
	if err != nil {
		t.Skip("Cannot create test directory")
	}

	// Save original directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Skip("Cannot get current directory")
	}
	defer os.Chdir(originalDir)

	// Change to subdirectory
	err = os.Chdir(subDir)
	if err != nil {
		t.Skip("Cannot change to test directory")
	}

	// Remove the directory while inside it
	err = os.Remove(subDir)
	if err != nil {
		t.Skip("Cannot remove directory")
	}

	runner := &ScriptRunner{
		allowedPaths: []string{"/some/path"},
	}

	// Try with relative path - should fail because os.Getwd() fails
	result := runner.isAllowedPath("relative/path.sh")

	// Should return false when Abs fails
	assert.False(t, result, "isAllowedPath should return false when filepath.Abs fails")
}

func TestIsAllowedPath_AbsErrorOnAllowedPath(t *testing.T) {
	// Test the continue path when allowedPath causes Abs to fail
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "testdir")
	err := os.Mkdir(subDir, 0755)
	if err != nil {
		t.Skip("Cannot create test directory")
	}

	// Create a valid script in tmpDir
	scriptPath := filepath.Join(tmpDir, "valid.sh")

	// Save and enter subdirectory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(subDir)

	// Remove it
	os.Remove(subDir)

	runner := &ScriptRunner{
		allowedPaths: []string{
			"relative/bad/path", // This will cause Abs to fail (continue)
			tmpDir,              // This is valid absolute path
		},
	}

	// scriptPath is absolute, so its Abs will succeed
	// First allowedPath (relative) will fail Abs and continue
	// Second allowedPath (absolute) will succeed
	result := runner.isAllowedPath(scriptPath)

	// Should return true because second allowedPath matches
	assert.True(t, result, "Should succeed with second allowed path after first fails")
}

func TestDetectInterpreter_PythonFallback(t *testing.T) {
	// Test python fallback by temporarily manipulating PATH
	// Save original PATH
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	// Set PATH to empty or minimal (no python3)
	os.Setenv("PATH", "/nonexistent")

	runner := &ScriptRunner{}

	interpreter, args := runner.detectInterpreter("/tmp/test.py")

	// When python3 is not found, should fall back to "python"
	assert.Equal(t, "python", interpreter, "Should fall back to 'python' when python3 not found")
	assert.Nil(t, args)
}

func TestDetectInterpreter_AllExtensions(t *testing.T) {
	// Comprehensive test for all interpreter types
	runner := &ScriptRunner{}

	tests := []struct {
		path                string
		expectedInterpreter string
		expectedArgs        []string
	}{
		{"/tmp/script.py", "python3", nil}, // python3 found on most systems
		{"/tmp/script.sh", "sh", nil},
		{"/tmp/script.bash", "bash", nil},
		{"/tmp/script.ps1", "powershell", []string{"-File"}},
		{"/tmp/script.rb", "ruby", nil},
		{"/tmp/script.js", "node", nil},
		{"/tmp/script.go", "go", []string{"run"}},
		{"/tmp/script", "", nil},         // No extension, uses shebang
		{"/tmp/script.unknown", "", nil}, // Unknown extension
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			interp, args := runner.detectInterpreter(tt.path)

			// For python, accept either python3 or python (fallback)
			if filepath.Ext(tt.path) == ".py" {
				assert.Contains(t, []string{"python3", "python"}, interp)
			} else {
				assert.Equal(t, tt.expectedInterpreter, interp)
			}

			assert.Equal(t, tt.expectedArgs, args)
		})
	}
}
