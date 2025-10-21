package git

import (
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	ssh2 "golang.org/x/crypto/ssh"

	log "github.com/sirupsen/logrus"

	"github.com/rootly/edge-connector/internal/config"
	"github.com/rootly/edge-connector/internal/metrics"
)

// Repository represents a cloned Git repository
type Repository struct {
	Options    *config.GitOptions
	repo       *git.Repository
	mutex      *sync.RWMutex // Read/write lock for safe access
	lastPulled time.Time
	URL        string
	Path       string // Local path where repo is cloned
}

// Manager manages Git repositories
type Manager struct {
	repositories map[string]*Repository
	baseDir      string
	mutex        sync.RWMutex
}

// NewManager creates a new Git repository manager
func NewManager(baseDir string) *Manager {
	return &Manager{
		repositories: make(map[string]*Repository),
		baseDir:      baseDir,
	}
}

// Download clones a Git repository if not already present
func (m *Manager) Download(options *config.GitOptions) (*Repository, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if already downloaded
	if repo, exists := m.repositories[options.URL]; exists {
		log.WithField("repo_url", options.URL).Debug("Repository already cloned")
		return repo, nil
	}

	// Create local path for repository (use hash of URL to avoid path issues)
	repoPath := filepath.Join(m.baseDir, sanitizeRepoName(options.URL))

	// Create base directory if it doesn't exist
	if err := os.MkdirAll(m.baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	// Setup authentication
	auth, err := m.getAuth(options)
	if err != nil {
		return nil, fmt.Errorf("authentication setup failed: %w", err)
	}

	// Clone repository
	branch := options.Branch
	if branch == "" {
		branch = "main"
	}

	log.WithFields(log.Fields{
		"repo_url": options.URL,
		"branch":   branch,
		"path":     repoPath,
	}).Info("Cloning Git repository")

	// Check if directory already exists (partial clone)
	if _, err := os.Stat(repoPath); err == nil {
		log.WithField("path", repoPath).Warn("Repository path already exists, removing and re-cloning")
		if err := os.RemoveAll(repoPath); err != nil {
			return nil, fmt.Errorf("failed to remove existing repository: %w", err)
		}
	}

	repo, err := git.PlainClone(repoPath, false, &git.CloneOptions{
		URL:           options.URL,
		Auth:          auth,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
		SingleBranch:  true,
		Depth:         1, // Shallow clone for faster downloads
		Progress:      nil,
	})
	if err != nil {
		return nil, fmt.Errorf("git clone failed: %w", err)
	}

	// Set permissions
	if err := os.Chmod(repoPath, 0700); err != nil {
		log.WithError(err).Warn("Failed to set repo permissions")
	}

	repository := &Repository{
		URL:        options.URL,
		Path:       repoPath,
		Options:    options,
		repo:       repo,
		lastPulled: time.Now(),
		mutex:      &sync.RWMutex{},
	}

	m.repositories[options.URL] = repository

	log.WithField("repo_url", options.URL).Info("Git repository cloned successfully")

	return repository, nil
}

// PullAll updates all repositories
func (m *Manager) PullAll() {
	m.mutex.RLock()
	repos := make([]*Repository, 0, len(m.repositories))
	for _, repo := range m.repositories {
		repos = append(repos, repo)
	}
	m.mutex.RUnlock()

	for _, repo := range repos {
		if err := m.Pull(repo); err != nil {
			log.WithFields(log.Fields{
				"repo_url": repo.URL,
				"error":    err,
			}).Warn("Failed to pull repository")
		}
	}
}

// Pull updates a specific repository
func (m *Manager) Pull(repo *Repository) error {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	// Check if we should pull based on poll interval
	pollInterval := time.Duration(repo.Options.PollIntervalSec) * time.Second
	if pollInterval == 0 {
		pollInterval = 5 * time.Minute // Default: 5 minutes
	}

	if time.Since(repo.lastPulled) < pollInterval {
		return nil // Too soon, skip
	}

	auth, err := m.getAuth(repo.Options)
	if err != nil {
		return fmt.Errorf("authentication setup failed: %w", err)
	}

	worktree, err := repo.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	log.WithField("repo_url", repo.URL).Debug("Pulling updates for repository")

	pullStart := time.Now()
	err = worktree.Pull(&git.PullOptions{
		Auth:       auth,
		RemoteName: "origin",
		Force:      true, // Hard reset to remote
	})
	pullDuration := time.Since(pullStart)

	if err == git.NoErrAlreadyUpToDate {
		log.WithField("repo_url", repo.URL).Trace("Repository is already up-to-date")
		repo.lastPulled = time.Now()
		metrics.RecordGitPull(repo.URL, "success", pullDuration)
		return nil
	}

	if err != nil {
		metrics.RecordGitPull(repo.URL, "error", pullDuration)
		return fmt.Errorf("git pull failed: %w", err)
	}

	repo.lastPulled = time.Now()
	metrics.RecordGitPull(repo.URL, "success", pullDuration)
	log.WithField("repo_url", repo.URL).Info("Repository updated successfully")

	return nil
}

// GetScriptPath returns the full path to a script in the repository
func (m *Manager) GetScriptPath(repoURL string, scriptPath string) (string, error) {
	m.mutex.RLock()
	repo, exists := m.repositories[repoURL]
	m.mutex.RUnlock()

	if !exists {
		return "", fmt.Errorf("repository not found: %s", repoURL)
	}

	fullPath := filepath.Join(repo.Path, scriptPath)

	// Verify file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("script not found in repository: %s", scriptPath)
	}

	// Security check: ensure path is within repository
	absRepoPath, err := filepath.Abs(repo.Path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute repo path: %w", err)
	}

	absScriptPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute script path: %w", err)
	}

	if !strings.HasPrefix(absScriptPath, absRepoPath) {
		return "", fmt.Errorf("script path escapes repository: %s", scriptPath)
	}

	return fullPath, nil
}

// RLock acquires read lock on a repository (for safe script execution)
func (m *Manager) RLock(repoURL string) error {
	m.mutex.RLock()
	repo, exists := m.repositories[repoURL]
	m.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("repository not found: %s", repoURL)
	}

	repo.mutex.RLock()
	return nil
}

// RUnlock releases read lock on a repository
func (m *Manager) RUnlock(repoURL string) {
	m.mutex.RLock()
	repo, exists := m.repositories[repoURL]
	m.mutex.RUnlock()

	if exists {
		repo.mutex.RUnlock()
	}
}

// getAuth sets up authentication for Git operations
func (m *Manager) getAuth(options *config.GitOptions) (transport.AuthMethod, error) {
	if options.PrivateKeyPath == "" {
		return nil, nil // No authentication
	}

	sshKey, err := os.ReadFile(options.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SSH key: %w", err)
	}

	auth, err := ssh.NewPublicKeys("git", sshKey, options.Passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH auth: %w", err)
	}

	// Disable host key verification (you may want to make this configurable)
	auth.HostKeyCallback = ssh2.InsecureIgnoreHostKey()

	return auth, nil
}

// sanitizeRepoName creates a safe directory name from repo URL
func sanitizeRepoName(repoURL string) string {
	// Use MD5 hash of URL to create a unique, safe directory name
	hash := md5.Sum([]byte(repoURL))
	return fmt.Sprintf("repo_%x", hash)
}

// StartPeriodicPull starts a background goroutine that periodically pulls all repositories
func (m *Manager) StartPeriodicPull(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log.Info("Starting periodic repository pull")

	for {
		select {
		case <-ctx.Done():
			log.Info("Stopping periodic repository pull")
			return
		case <-ticker.C:
			log.Debug("Running periodic pull for all repositories")
			m.PullAll()
		}
	}
}

// GetRepository returns a repository by URL
func (m *Manager) GetRepository(repoURL string) (*Repository, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	repo, exists := m.repositories[repoURL]
	return repo, exists
}
