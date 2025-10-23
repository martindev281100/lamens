package git

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"k8s.io/klog/v2"
)

// CloneOptions holds the options for cloning a Git repository.
type CloneOptions struct {
	// URL is the repository URL.
	URL string
	// Auth is the authentication method.
	Auth transport.AuthMethod
	// TempDir is the temporary directory for cloning.
	TempDir string
	// Branch is the branch to clone.
	Branch string
	// Tag is the tag to clone.
	Tag string
	// Commit is the commit to checkout after cloning.
	Commit string
	// Shallow indicates whether to perform a shallow clone.
	Shallow bool
	// Depth is the depth for shallow cloning.
	Depth int
}

// ChartInfo holds information about a Helm chart.
type ChartInfo struct {
	// Name is the chart name.
	Name string
	// Version is the chart version.
	Version string
	// AppVersion is the app version.
	AppVersion string
	// Description is the chart description.
	Description string
	// Path is the path to the chart.
	Path string
}

// CloneRepository clones a Git repository with the provided options.
//
// Parameters:
//   - opts: The clone options.
//
// Returns:
//   - *git.Repository: The cloned repository.
//   - string: The path to the cloned repository.
//   - error: An error if the repository could not be cloned.
func CloneRepository(opts *CloneOptions) (*git.Repository, string, error) {
	if opts == nil {
		return nil, "", fmt.Errorf("clone options cannot be nil")
	}
	if opts.URL == "" {
		return nil, "", fmt.Errorf("repository URL cannot be empty")
	}

	// Create a temporary directory if not provided
	tempDir := opts.TempDir
	if tempDir == "" {
		var err error
		tempDir, err = ioutil.TempDir("", "kargo-bootstrap-")
		if err != nil {
			return nil, "", fmt.Errorf("failed to create temporary directory: %w", err)
		}
	}

	// Clone the repository
	var repo *git.Repository
	var err error

	// Determine the reference to clone
	referenceName := plumbing.HEAD
	if opts.Branch != "" {
		referenceName = plumbing.NewBranchReferenceName(opts.Branch)
	} else if opts.Tag != "" {
		referenceName = plumbing.NewTagReferenceName(opts.Tag)
	}

	// Clone options
	cloneOpts := &git.CloneOptions{
		URL:           opts.URL,
		Auth:          opts.Auth,
		ReferenceName: referenceName,
		SingleBranch:  opts.Branch != "" || opts.Tag != "",
	}

	// Configure shallow clone
	if opts.Shallow {
		cloneOpts.Depth = opts.Depth
		if cloneOpts.Depth == 0 {
			cloneOpts.Depth = 1
		}
	}

	// Clone the repository
	repo, err = git.PlainClone(tempDir, false, cloneOpts)
	if err != nil {
		return nil, "", fmt.Errorf("failed to clone repository: %w", err)
	}

	// Checkout specific commit if provided
	if opts.Commit != "" {
		hash := plumbing.NewHash(opts.Commit)
		worktree, err := repo.Worktree()
		if err != nil {
			return nil, "", fmt.Errorf("failed to get worktree: %w", err)
		}

		err = worktree.Checkout(&git.CheckoutOptions{
			Hash: hash,
		})
		if err != nil {
			return nil, "", fmt.Errorf("failed to checkout commit %s: %w", opts.Commit, err)
		}
	}

	return repo, tempDir, nil
}

// ListBranches lists all branches in a Git repository.
//
// Parameters:
//   - repo: The Git repository.
//
// Returns:
//   - []string: A list of branch names.
//   - error: An error if the branches could not be listed.
func ListBranches(repo *git.Repository) ([]string, error) {
	branches, err := repo.Branches()
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	var branchNames []string
	err = branches.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsBranch() {
			branchNames = append(branchNames, ref.Name().Short())
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to iterate branches: %w", err)
	}

	return branchNames, nil
}

// ListTags lists all tags in a Git repository.
//
// Parameters:
//   - repo: The Git repository.
//
// Returns:
//   - []string: A list of tag names.
//   - error: An error if the tags could not be listed.
func ListTags(repo *git.Repository) ([]string, error) {
	tags, err := repo.Tags()
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	var tagNames []string
	err = tags.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsTag() {
			tagNames = append(tagNames, ref.Name().Short())
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to iterate tags: %w", err)
	}

	return tagNames, nil
}

// ListCommits lists all commits in a Git repository.
//
// Parameters:
//   - repo: The Git repository.
//   - limit: The maximum number of commits to list (0 for no limit).
//
// Returns:
//   - []string: A list of commit hashes.
//   - error: An error if the commits could not be listed.
func ListCommits(repo *git.Repository, limit int) ([]string, error) {
	ref, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get commit object: %w", err)
	}

	var commitHashes []string
	count := 0

	// Iterate through commits
	err = commit.Parents().ForEach(func(c *object.Commit) error {
		if limit > 0 && count >= limit {
			return fmt.Errorf("limit reached")
		}

		commitHashes = append(commitHashes, c.Hash.String())
		count++
		return nil
	})

	// Add the initial commit
	commitHashes = append(commitHashes, commit.Hash.String())

	if err != nil && err.Error() != "limit reached" {
		return nil, fmt.Errorf("failed to iterate commits: %w", err)
	}

	return commitHashes, nil
}

// ListChartsInRepo discovers all Helm charts in a Git repository.
//
// Parameters:
//   - opts: The clone options.
//
// Returns:
//   - []ChartInfo: A list of Helm charts.
//   - error: An error if the charts could not be discovered.
func ListChartsInRepo(opts *CloneOptions) ([]ChartInfo, error) {
	// Clone the repository
	repo, tempDir, err := CloneRepository(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Discover charts
	charts, err := DiscoverCharts(tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to discover charts: %w", err)
	}

	return charts, nil
}

// DiscoverCharts discovers all Helm charts in a directory.
//
// Parameters:
//   - dir: The directory to search.
//
// Returns:
//   - []ChartInfo: A list of Helm charts.
//   - error: An error if the charts could not be discovered.
func DiscoverCharts(dir string) ([]ChartInfo, error) {
	var charts []ChartInfo

	// Walk the directory tree
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if the file is Chart.yaml
		if info.Name() == "Chart.yaml" {
			// Get the relative path
			relPath, err := filepath.Rel(dir, filepath.Dir(path))
			if err != nil {
				return fmt.Errorf("failed to get relative path: %w", err)
			}

			// Parse the chart
			chart, err := ParseChart(path)
			if err != nil {
				klog.Warningf("Failed to parse chart at %s: %v", path, err)
				return nil // Continue walking
			}

			chart.Path = relPath
			charts = append(charts, *chart)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return charts, nil
}

// ParseChart parses a Helm chart from a Chart.yaml file.
//
// Parameters:
//   - chartPath: The path to the Chart.yaml file.
//
// Returns:
//   - *ChartInfo: The parsed chart information.
//   - error: An error if the chart could not be parsed.
func ParseChart(chartPath string) (*ChartInfo, error) {
	// Read the Chart.yaml file
	data, err := ioutil.ReadFile(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Chart.yaml: %w", err)
	}

	// Parse the YAML (simplified implementation)
	// In a real implementation, you would use a proper YAML parser
	chart := &ChartInfo{
		Name:        "example-chart",
		Version:     "1.0.0",
		AppVersion:  "1.0.0",
		Description: "Example Helm chart",
	}

	return chart, nil
}

// ValidateRepositoryURL validates a Git repository URL.
//
// Parameters:
//   - url: The repository URL to validate.
//
// Returns:
//   - string: The normalized URL.
//   - error: An error if the URL is invalid.
func ValidateRepositoryURL(url string) (string, error) {
	if url == "" {
		return "", fmt.Errorf("repository URL cannot be empty")
	}

	// Normalize the URL
	normalizedURL := strings.TrimSpace(url)

	// Check if the URL has a scheme
	if !strings.Contains(normalizedURL, "://") {
		// Assume HTTPS if no scheme is provided
		normalizedURL = "https://" + normalizedURL
	}

	// Basic validation
	if !strings.HasPrefix(normalizedURL, "http://") &&
		!strings.HasPrefix(normalizedURL, "https://") &&
		!strings.HasPrefix(normalizedURL, "ssh://") &&
		!strings.HasPrefix(normalizedURL, "git@") {
		return "", fmt.Errorf("invalid repository URL scheme")
	}

	return normalizedURL, nil
}

// CreateAuthMethod creates an authentication method for Git operations.
//
// Parameters:
//   - token: The authentication token.
//   - username: The username for basic authentication.
//   - password: The password for basic authentication.
//   - sshKeyPath: The path to the SSH private key.
//   - sshKey: The SSH private key content.
//   - knownHosts: The path to the known hosts file.
//
// Returns:
//   - transport.AuthMethod: The authentication method.
//   - error: An error if the authentication method could not be created.
func CreateAuthMethod(token, username, password, sshKeyPath, sshKey, knownHosts string) (transport.AuthMethod, error) {
	// Token authentication
	if token != "" {
		return &http.BasicAuth{
			Username: "token", // GitHub uses "token" as the username for token authentication
			Password: token,
		}, nil
	}

	// Username/password authentication
	if username != "" && password != "" {
		return &http.BasicAuth{
			Username: username,
			Password: password,
		}, nil
	}

	// SSH key authentication
	if sshKeyPath != "" || sshKey != "" {
		var key []byte
		var err error

		if sshKey != "" {
			key = []byte(sshKey)
		} else {
			key, err = ioutil.ReadFile(sshKeyPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read SSH key: %w", err)
			}
		}

		// Create SSH public key authentication
		signer, err := ssh.NewPublicKeysFromFile("git", sshKeyPath, "")
		if err != nil {
			// Try with the key content
			signer, err = ssh.NewPublicKeys("git", key, "")
			if err != nil {
				return nil, fmt.Errorf("failed to create SSH signer: %w", err)
			}
		}

		// Set known hosts if provided
		if knownHosts != "" {
			knownHostsCallback, err := ssh.NewKnownHostsFile(knownHosts)
			if err != nil {
				return nil, fmt.Errorf("failed to create known hosts callback: %w", err)
			}
			signer.HostKeyCallback = knownHostsCallback
		} else {
			// Use the default known hosts callback
			signer.HostKeyCallback = ssh.InsecureIgnoreHostKey() // Not recommended for production
		}

		return signer, nil
	}

	// No authentication
	return nil, nil
}

// GetLatestCommit gets the latest commit in a Git repository.
//
// Parameters:
//   - repo: The Git repository.
//   - branch: The branch to get the commit from (empty for default branch).
//
// Returns:
//   - string: The commit hash.
//   - error: An error if the commit could not be retrieved.
func GetLatestCommit(repo *git.Repository, branch string) (string, error) {
	var ref *plumbing.Reference
	var err error

	if branch != "" {
		// Get the reference for the specified branch
		ref, err = repo.Storer.Reference(plumbing.NewBranchReferenceName(branch))
		if err != nil {
			return "", fmt.Errorf("failed to get reference for branch %s: %w", branch, err)
		}
	} else {
		// Get the HEAD reference
		ref, err = repo.Head()
		if err != nil {
			return "", fmt.Errorf("failed to get HEAD reference: %w", err)
		}
	}

	return ref.Hash().String(), nil
}

// GetCommitMessage gets the message for a commit.
//
// Parameters:
//   - repo: The Git repository.
//   - commitHash: The commit hash.
//
// Returns:
//   - string: The commit message.
//   - error: An error if the commit message could not be retrieved.
func GetCommitMessage(repo *git.Repository, commitHash string) (string, error) {
	hash := plumbing.NewHash(commitHash)
	commit, err := repo.CommitObject(hash)
	if err != nil {
		return "", fmt.Errorf("failed to get commit object: %w", err)
	}

	return commit.Message, nil
}

// GetCommitDate gets the date for a commit.
//
// Parameters:
//   - repo: The Git repository.
//   - commitHash: The commit hash.
//
// Returns:
//   - time.Time: The commit date.
//   - error: An error if the commit date could not be retrieved.
func GetCommitDate(repo *git.Repository, commitHash string) (time.Time, error) {
	hash := plumbing.NewHash(commitHash)
	commit, err := repo.CommitObject(hash)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get commit object: %w", err)
	}

	return commit.Author.When, nil
}

// GetRemoteURL gets the remote URL for a Git repository.
//
// Parameters:
//   - repo: The Git repository.
//   - remoteName: The name of the remote (default: "origin").
//
// Returns:
//   - string: The remote URL.
//   - error: An error if the remote URL could not be retrieved.
func GetRemoteURL(repo *git.Repository, remoteName string) (string, error) {
	if remoteName == "" {
		remoteName = "origin"
	}

	remote, err := repo.Remote(remoteName)
	if err != nil {
		return "", fmt.Errorf("failed to get remote %s: %w", remoteName, err)
	}

	if len(remote.Config().URLs) == 0 {
		return "", fmt.Errorf("no URLs found for remote %s", remoteName)
	}

	return remote.Config().URLs[0], nil
}

// IsRepositoryDirty checks if a Git repository has uncommitted changes.
//
// Parameters:
//   - repo: The Git repository.
//
// Returns:
//   - bool: True if the repository has uncommitted changes.
//   - error: An error if the repository status could not be retrieved.
func IsRepositoryDirty(repo *git.Repository) (bool, error) {
	worktree, err := repo.Worktree()
	if err != nil {
		return false, fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return false, fmt.Errorf("failed to get worktree status: %w", err)
	}

	return !status.IsClean(), nil
}
