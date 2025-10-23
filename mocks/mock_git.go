package mocks

import (
	"fmt"
	"os"
	"sync"

	"github.com/your-org/kargo-bootstrap/pkg/git"
)

// MockGitClient implements a mock Git client for testing
type MockGitClient struct {
	repositories map[string]*MockRepository
	mutex        sync.RWMutex
	tempDir      string
}

// MockRepository represents a mock Git repository
type MockRepository struct {
	Path          string
	URL           string
	DefaultBranch string
	CurrentBranch string
	CurrentCommit string
	Branches      []string
	Tags          []string
	Files         map[string][]byte
}

// NewMockGitClient creates a new mock Git client
func NewMockGitClient() *MockGitClient {
	tempDir, _ := os.MkdirTemp("", "mock-git-*")

	return &MockGitClient{
		repositories: make(map[string]*MockRepository),
		tempDir:      tempDir,
	}
}

// Cleanup cleans up the mock Git client
func (m *MockGitClient) Cleanup() error {
	if m.tempDir != "" {
		return os.RemoveAll(m.tempDir)
	}
	return nil
}

// CloneRepository clones a repository to a temporary directory
func (m *MockGitClient) CloneRepository(opts *git.CloneOptions) (*git.RepositoryInfo, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if opts.URL == "" {
		return nil, fmt.Errorf("repository URL cannot be empty")
	}

	// Create a temporary directory for the repository
	tempDir, err := os.MkdirTemp(m.tempDir, "repo-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}

	// Create a mock repository
	repo := &MockRepository{
		Path:          tempDir,
		URL:           opts.URL,
		DefaultBranch: "main",
		CurrentBranch: "main",
		CurrentCommit: "abc123",
		Branches:      []string{"main", "develop"},
		Tags:          []string{"v1.0.0", "v1.1.0"},
		Files:         make(map[string][]byte),
	}

	// Add some default files
	repo.Files["README.md"] = []byte("# Test Repository")
	repo.Files["Chart.yaml"] = []byte("apiVersion: v2\nname: test-chart\nversion: 0.1.0")

	// Store the repository
	m.repositories[tempDir] = repo

	// Create the repository info
	info := &git.RepositoryInfo{
		Path:          tempDir,
		URL:           opts.URL,
		DefaultBranch: repo.DefaultBranch,
		CurrentBranch: repo.CurrentBranch,
		CurrentCommit: repo.CurrentCommit,
	}

	return info, nil
}

// FetchRepository fetches updates for an existing repository
func (m *MockGitClient) FetchRepository(repoPath string, auth *git.AuthConfig) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	repo, exists := m.repositories[repoPath]
	if !exists {
		return fmt.Errorf("repository not found at path: %s", repoPath)
	}

	// Mock fetch operation - just update the commit
	repo.CurrentCommit = "def456"
	return nil
}

// GetRepository gets or creates a repository handle
func (m *MockGitClient) GetRepository(repoPath string) (*MockRepository, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	repo, exists := m.repositories[repoPath]
	if !exists {
		return nil, fmt.Errorf("repository not found at path: %s", repoPath)
	}

	return repo, nil
}

// CleanupTempDir cleans up temporary directories
func (m *MockGitClient) CleanupTempDir(tempDir string) error {
	return os.RemoveAll(tempDir)
}

// ValidateRepositoryURL validates and normalizes repository URLs
func (m *MockGitClient) ValidateRepositoryURL(repoURL string) (string, error) {
	if repoURL == "" {
		return "", fmt.Errorf("repository URL cannot be empty")
	}

	// Simple validation for mock
	if !contains(repoURL, "://") && !contains(repoURL, "@") {
		return "https://" + repoURL, nil
	}

	return repoURL, nil
}

// GetDefaultBranch detects the default branch of a repository
func (m *MockGitClient) GetDefaultBranch(repoPath string) (string, error) {
	repo, err := m.GetRepository(repoPath)
	if err != nil {
		return "", err
	}

	return repo.DefaultBranch, nil
}

// ListBranches lists available branches in a repository
func (m *MockGitClient) ListBranches(repoPath string) ([]string, error) {
	repo, err := m.GetRepository(repoPath)
	if err != nil {
		return nil, err
	}

	return repo.Branches, nil
}

// ListTags lists available tags in a repository
func (m *MockGitClient) ListTags(repoPath string) ([]string, error) {
	repo, err := m.GetRepository(repoPath)
	if err != nil {
		return nil, err
	}

	return repo.Tags, nil
}

// CheckoutRevision checks out a specific branch, tag, or commit
func (m *MockGitClient) CheckoutRevision(repoPath string, revision string) error {
	repo, err := m.GetRepository(repoPath)
	if err != nil {
		return err
	}

	// Check if revision is a branch
	for _, branch := range repo.Branches {
		if branch == revision {
			repo.CurrentBranch = revision
			repo.CurrentCommit = "branch-" + revision
			return nil
		}
	}

	// Check if revision is a tag
	for _, tag := range repo.Tags {
		if tag == revision {
			repo.CurrentBranch = "main" // Tags are usually on main
			repo.CurrentCommit = "tag-" + tag
			return nil
		}
	}

	// Assume it's a commit hash
	repo.CurrentCommit = revision
	return nil
}

// AddTestRepository adds a test repository to the mock
func (m *MockGitClient) AddTestRepository(url string, repo *MockRepository) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.repositories[repo.Path] = repo
}

// RemoveTestRepository removes a test repository from the mock
func (m *MockGitClient) RemoveTestRepository(path string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.repositories, path)
}

// AddFile adds a file to a mock repository
func (m *MockGitClient) AddFile(repoPath, filename string, content []byte) error {
	repo, err := m.GetRepository(repoPath)
	if err != nil {
		return err
	}

	repo.Files[filename] = content
	return nil
}

// GetFile gets a file from a mock repository
func (m *MockGitClient) GetFile(repoPath, filename string) ([]byte, error) {
	repo, err := m.GetRepository(repoPath)
	if err != nil {
		return nil, err
	}

	content, exists := repo.Files[filename]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", filename)
	}

	return content, nil
}

// ListFiles lists all files in a mock repository
func (m *MockGitClient) ListFiles(repoPath string) ([]string, error) {
	repo, err := m.GetRepository(repoPath)
	if err != nil {
		return nil, err
	}

	var files []string
	for filename := range repo.Files {
		files = append(files, filename)
	}

	return files, nil
}

// CreateTestRepo creates a test repository with basic structure
func (m *MockGitClient) CreateTestRepo(url string) (*MockRepository, error) {
	tempDir, err := os.MkdirTemp(m.tempDir, "test-repo-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}

	repo := &MockRepository{
		Path:          tempDir,
		URL:           url,
		DefaultBranch: "main",
		CurrentBranch: "main",
		CurrentCommit: "abc123",
		Branches:      []string{"main", "develop"},
		Tags:          []string{"v1.0.0", "v1.1.0"},
		Files:         make(map[string][]byte),
	}

	// Add basic Helm chart structure
	repo.Files["Chart.yaml"] = []byte(`apiVersion: v2
name: test-chart
description: A Helm chart for testing
type: application
version: 0.1.0
appVersion: "1.0.0"`)

	repo.Files["values.yaml"] = []byte(`replicaCount: 1
image:
  repository: nginx
  pullPolicy: IfNotPresent
  tag: "latest"
service:
  type: ClusterIP
  port: 80`)

	repo.Files["templates/deployment.yaml"] = []byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "test-chart.fullname" . }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      app: {{ include "test-chart.name" . }}
  template:
    metadata:
      labels:
        app: {{ include "test-chart.name" . }}
    spec:
      containers:
        - name: {{ .Chart.Name }}
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
          ports:
            - name: http
              containerPort: 80`)

	m.repositories[tempDir] = repo
	return repo, nil
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())))
}
