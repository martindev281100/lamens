package git

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/skeema/knownhosts"
	"github.com/your-org/kargo-bootstrap/pkg/errors"
	"golang.org/x/crypto/ssh"
	"k8s.io/klog/v2"
)

// AuthMethod represents the type of authentication to use
type AuthMethod int

const (
	AuthNone AuthMethod = iota
	AuthToken
	AuthUserPass
	AuthSSHKey
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Method     AuthMethod
	Token      string
	Username   string
	Password   string
	SSHKey     string
	KeyPath    string
	KnownHosts string
}

// CloneOptions contains options for cloning a repository
type CloneOptions struct {
	URL     string
	Branch  string
	Tag     string
	Commit  string
	Shallow bool
	Auth    *AuthConfig
	TempDir string
}

// RepositoryInfo contains information about a repository
type RepositoryInfo struct {
	Path          string
	URL           string
	DefaultBranch string
	CurrentBranch string
	CurrentCommit string
}

// CloneRepository clones a repository to a temporary directory
func CloneRepository(opts *CloneOptions) (*RepositoryInfo, error) {
	// Validate and normalize the repository URL
	normalizedURL, err := ValidateRepositoryURL(opts.URL)
	if err != nil {
		return nil, errors.NewAppError(
			errors.ErrorTypeGit,
			errors.ErrCodeInvalidURL,
			"invalid repository URL",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithField("repository URL", opts.URL).WithSuggestions(
			"Check if the repository URL is correct",
			"Ensure the URL includes the protocol (http://, https://, ssh://)",
			"Verify the repository exists",
		)
	}

	// Create a temporary directory if not specified
	tempDir := opts.TempDir
	if tempDir == "" {
		tempDir, err = ioutil.TempDir("", "kargo-bootstrap-git-")
		if err != nil {
			return nil, errors.NewAppError(
				errors.ErrorTypeGeneral,
				errors.ErrCodeInvalidInput,
				"failed to create temporary directory",
				errors.ErrorSeverityError,
				false,
			).WithCause(err).WithSuggestions(
				"Check if the temporary directory is accessible",
				"Verify disk space is available",
				"Check file system permissions",
			)
		}
	}

	// Prepare clone options
	cloneOpts := &git.CloneOptions{
		URL:           normalizedURL,
		Depth:         1,
		Tags:          git.AllTags,
		SingleBranch:  false,
		ReferenceName: plumbing.ReferenceName(""),
	}

	// Configure shallow clone if requested
	if opts.Shallow {
		cloneOpts.Depth = 1
	} else {
		cloneOpts.Depth = 0
	}

	// Set reference if branch or tag is specified
	if opts.Branch != "" {
		cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(opts.Branch)
		cloneOpts.SingleBranch = true
	} else if opts.Tag != "" {
		cloneOpts.ReferenceName = plumbing.NewTagReferenceName(opts.Tag)
		cloneOpts.SingleBranch = true
	}

	// Configure authentication
	auth, err := getAuthMethod(normalizedURL, opts.Auth)
	if err != nil {
		CleanupTempDir(tempDir)
		return nil, errors.NewAppError(
			errors.ErrorTypeAuth,
			errors.ErrCodeAuthFailed,
			"failed to configure authentication",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithSuggestions(
			"Check if authentication credentials are correct",
			"Verify the authentication method is supported",
			"Ensure the repository URL matches the authentication method",
		)
	}
	if auth != nil {
		cloneOpts.Auth = auth
	}

	// Clone the repository
	klog.V(2).Infof("Cloning repository %s to %s", normalizedURL, tempDir)
	repo, err := git.PlainClone(tempDir, false, cloneOpts)
	if err != nil {
		CleanupTempDir(tempDir)
		return nil, errors.NewAppError(
			errors.ErrorTypeClone,
			errors.ErrCodeCloneFailed,
			"failed to clone repository",
			errors.ErrorSeverityError,
			true, // Network issues might be retryable
		).WithCause(err).WithField("repository URL", normalizedURL).WithSuggestions(
			"Check if the repository URL is correct",
			"Verify network connectivity to the repository",
			"Check authentication credentials",
			"Ensure the repository is accessible",
		)
	}

	// Get repository information
	info, err := getRepositoryInfo(repo, tempDir, normalizedURL)
	if err != nil {
		CleanupTempDir(tempDir)
		return nil, errors.NewAppError(
			errors.ErrorTypeRepository,
			errors.ErrCodeRepoNotFound,
			"failed to get repository information",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithSuggestions(
			"Check if the repository is valid",
			"Verify the repository was cloned successfully",
		)
	}

	// Checkout specific commit if specified
	if opts.Commit != "" {
		if err := CheckoutRevision(repo, opts.Commit); err != nil {
			CleanupTempDir(tempDir)
			return nil, errors.NewAppError(
				errors.ErrorTypeGit,
				errors.ErrCodeInvalidFormat,
				fmt.Sprintf("failed to checkout commit %s", opts.Commit),
				errors.ErrorSeverityError,
				false,
			).WithCause(err).WithField("commit", opts.Commit).WithSuggestions(
				"Check if the commit hash is correct",
				"Verify the commit exists in the repository",
				"Ensure the commit is reachable from the current branch",
			)
		}
		// Update the current commit in the info
		hash, err := repo.Head()
		if err == nil {
			info.CurrentCommit = hash.Hash().String()
		}
	}

	return info, nil
}

// FetchRepository fetches updates for an existing repository
func FetchRepository(repoPath string, auth *AuthConfig) error {
	// Open the existing repository
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return errors.NewAppError(
			errors.ErrorTypeRepository,
			errors.ErrCodeRepoNotFound,
			"failed to open repository",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithField("repository path", repoPath).WithSuggestions(
			"Check if the repository path is correct",
			"Verify the repository exists",
			"Ensure the repository is a valid Git repository",
		)
	}

	// Get the remote URL
	remote, err := repo.Remote("origin")
	if err != nil {
		return errors.NewAppError(
			errors.ErrorTypeRepository,
			errors.ErrCodeRepoNotFound,
			"failed to get remote",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithSuggestions(
			"Check if the repository has a remote named 'origin'",
			"Verify the repository is a valid Git repository",
		)
	}

	if len(remote.Config().URLs) == 0 {
		return errors.NewAppError(
			errors.ErrorTypeRepository,
			errors.ErrCodeRepoNotFound,
			"no remote URL found",
			errors.ErrorSeverityError,
			false,
		).WithSuggestions(
			"Add a remote URL to the repository",
			"Check if the repository has been properly initialized",
		)
	}

	remoteURL := remote.Config().URLs[0]

	// Configure authentication
	authMethod, err := getAuthMethod(remoteURL, auth)
	if err != nil {
		return errors.NewAppError(
			errors.ErrorTypeAuth,
			errors.ErrCodeAuthFailed,
			"failed to configure authentication",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithSuggestions(
			"Check if authentication credentials are correct",
			"Verify the authentication method is supported",
			"Ensure the repository URL matches the authentication method",
		)
	}

	// Prepare fetch options
	fetchOpts := &git.FetchOptions{
		RemoteName: "origin",
		Tags:       git.AllTags,
	}

	if authMethod != nil {
		fetchOpts.Auth = authMethod
	}

	// Fetch updates
	klog.V(2).Infof("Fetching updates for repository at %s", repoPath)
	if err := repo.Fetch(fetchOpts); err != nil && err != git.NoErrAlreadyUpToDate {
		return errors.NewAppError(
			errors.ErrorTypeGit,
			errors.ErrCodeGitConnection,
			"failed to fetch updates",
			errors.ErrorSeverityError,
			true, // Network issues might be retryable
		).WithCause(err).WithSuggestions(
			"Check network connectivity to the repository",
			"Verify authentication credentials",
			"Ensure the repository is accessible",
		)
	}

	return nil
}

// GetRepository gets or creates a repository handle
func GetRepository(repoPath string) (*git.Repository, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, errors.NewAppError(
			errors.ErrorTypeRepository,
			errors.ErrCodeRepoNotFound,
			"failed to open repository",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithField("repository path", repoPath).WithSuggestions(
			"Check if the repository path is correct",
			"Verify the repository exists",
			"Ensure the repository is a valid Git repository",
		)
	}
	return repo, nil
}

// CleanupTempDir cleans up temporary directories
func CleanupTempDir(tempDir string) error {
	if tempDir == "" {
		return nil
	}

	klog.V(2).Infof("Cleaning up temporary directory: %s", tempDir)
	return os.RemoveAll(tempDir)
}

// ValidateRepositoryURL validates and normalizes repository URLs
func ValidateRepositoryURL(repoURL string) (string, error) {
	if repoURL == "" {
		return "", errors.NewAppError(
			errors.ErrorTypeValidation,
			errors.ErrCodeMissingRequired,
			"repository URL cannot be empty",
			errors.ErrorSeverityError,
			false,
		).WithField("repository URL", repoURL).WithSuggestions(
			"Provide a valid repository URL",
		)
	}

	// Parse the URL
	u, err := url.Parse(repoURL)
	if err != nil {
		// If it's not a valid URL, it might be an SSH URL like git@github.com:user/repo.git
		if strings.Contains(repoURL, "@") && strings.Contains(repoURL, ":") {
			// This looks like an SSH URL, return as-is
			return repoURL, nil
		}
		return "", errors.NewAppError(
			errors.ErrorTypeGit,
			errors.ErrCodeInvalidURL,
			"invalid repository URL",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithField("repository URL", repoURL).WithSuggestions(
			"Check if the repository URL is correct",
			"Ensure the URL includes the protocol (http://, https://, ssh://)",
			"Verify the repository exists",
		)
	}

	// Ensure the URL has a scheme
	if u.Scheme == "" {
		// Default to HTTPS for HTTP-like URLs
		if strings.Contains(u.String(), "://") == false {
			return "https://" + repoURL, nil
		}
		return "", errors.NewAppError(
			errors.ErrorTypeGit,
			errors.ErrCodeInvalidURL,
			"repository URL must have a scheme (http://, https://, ssh://, etc.)",
			errors.ErrorSeverityError,
			false,
		).WithField("repository URL", repoURL).WithSuggestions(
			"Add a protocol to the URL (e.g., https://github.com/user/repo.git)",
			"Use ssh:// for SSH URLs",
		)
	}

	// Validate the scheme
	switch u.Scheme {
	case "http", "https", "ssh", "git":
		// Valid schemes
	default:
		return "", errors.NewAppError(
			errors.ErrorTypeGit,
			errors.ErrCodeInvalidURL,
			fmt.Sprintf("unsupported URL scheme: %s", u.Scheme),
			errors.ErrorSeverityError,
			false,
		).WithField("scheme", u.Scheme).WithField("repository URL", repoURL).WithSuggestions(
			"Use a supported URL scheme (http, https, ssh, git)",
			"Check if the repository URL is correct",
		)
	}

	return u.String(), nil
}

// GetDefaultBranch detects the default branch of a repository
func GetDefaultBranch(repoPath string) (string, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", errors.NewAppError(
			errors.ErrorTypeRepository,
			errors.ErrCodeRepoNotFound,
			"failed to open repository",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithField("repository path", repoPath).WithSuggestions(
			"Check if the repository path is correct",
			"Verify the repository exists",
			"Ensure the repository is a valid Git repository",
		)
	}

	// Try to get the default branch from the remote
	remote, err := repo.Remote("origin")
	if err != nil {
		return "", errors.NewAppError(
			errors.ErrorTypeRepository,
			errors.ErrCodeRepoNotFound,
			"failed to get remote",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithSuggestions(
			"Check if the repository has a remote named 'origin'",
			"Verify the repository is a valid Git repository",
		)
	}

	if len(remote.Config().URLs) == 0 {
		return "", errors.NewAppError(
			errors.ErrorTypeRepository,
			errors.ErrCodeRepoNotFound,
			"no remote URL found",
			errors.ErrorSeverityError,
			false,
		).WithSuggestions(
			"Add a remote URL to the repository",
			"Check if the repository has been properly initialized",
		)
	}

	// Get the remote HEAD reference
	refs, err := remote.List(&git.ListOptions{})
	if err != nil {
		return "", errors.NewAppError(
			errors.ErrorTypeGit,
			errors.ErrCodeGitConnection,
			"failed to list remote references",
			errors.ErrorSeverityError,
			true, // Network issues might be retryable
		).WithCause(err).WithSuggestions(
			"Check network connectivity to the repository",
			"Verify authentication credentials",
			"Ensure the repository is accessible",
		)
	}

	for _, ref := range refs {
		if ref.Name() == "HEAD" {
			// For HEAD reference, we need to check the symbolic reference target
			// This is a simplified approach - in a real implementation, you might need
			// to resolve the symbolic reference properly
			return "main", nil // Default to main as a fallback
		}
	}

	// Fallback to common default branch names
	commonBranches := []string{"main", "master", "develop"}
	for _, branch := range commonBranches {
		_, err := repo.Reference(plumbing.NewBranchReferenceName(branch), true)
		if err == nil {
			return branch, nil
		}
	}

	return "", errors.NewAppError(
		errors.ErrorTypeGit,
		errors.ErrCodeInvalidFormat,
		"could not determine default branch",
		errors.ErrorSeverityWarning,
		false,
	).WithSuggestions(
		"Check if the repository has any branches",
		"Ensure the repository is properly initialized",
		"Verify the repository is accessible",
	)
}

// ListBranches lists available branches in a repository
func ListBranches(repoPath string) ([]string, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, errors.NewAppError(
			errors.ErrorTypeRepository,
			errors.ErrCodeRepoNotFound,
			"failed to open repository",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithField("repository path", repoPath).WithSuggestions(
			"Check if the repository path is correct",
			"Verify the repository exists",
			"Ensure the repository is a valid Git repository",
		)
	}

	branches, err := repo.Branches()
	if err != nil {
		return nil, errors.NewAppError(
			errors.ErrorTypeGit,
			errors.ErrCodeGitConnection,
			"failed to list branches",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithSuggestions(
			"Check if the repository has any branches",
			"Verify the repository is properly initialized",
		)
	}

	var branchNames []string
	err = branches.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsBranch() {
			branchNames = append(branchNames, ref.Name().Short())
		}
		return nil
	})

	if err != nil {
		return nil, errors.NewAppError(
			errors.ErrorTypeGit,
			errors.ErrCodeGitConnection,
			"failed to iterate branches",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithSuggestions(
			"Check if the repository has any branches",
			"Verify the repository is properly initialized",
		)
	}

	return branchNames, nil
}

// ListTags lists available tags in a repository
func ListTags(repoPath string) ([]string, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, errors.NewAppError(
			errors.ErrorTypeRepository,
			errors.ErrCodeRepoNotFound,
			"failed to open repository",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithField("repository path", repoPath).WithSuggestions(
			"Check if the repository path is correct",
			"Verify the repository exists",
			"Ensure the repository is a valid Git repository",
		)
	}

	tags, err := repo.Tags()
	if err != nil {
		return nil, errors.NewAppError(
			errors.ErrorTypeGit,
			errors.ErrCodeGitConnection,
			"failed to list tags",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithSuggestions(
			"Check if the repository has any tags",
			"Verify the repository is properly initialized",
		)
	}

	var tagNames []string
	err = tags.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsTag() {
			tagNames = append(tagNames, ref.Name().Short())
		}
		return nil
	})

	if err != nil {
		return nil, errors.NewAppError(
			errors.ErrorTypeGit,
			errors.ErrCodeGitConnection,
			"failed to iterate tags",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithSuggestions(
			"Check if the repository has any tags",
			"Verify the repository is properly initialized",
		)
	}

	return tagNames, nil
}

// CheckoutRevision checks out a specific branch, tag, or commit
func CheckoutRevision(repo *git.Repository, revision string) error {
	// Try to resolve the revision
	hash, err := repo.ResolveRevision(plumbing.Revision(revision))
	if err != nil {
		return errors.NewAppError(
			errors.ErrorTypeGit,
			errors.ErrCodeInvalidFormat,
			fmt.Sprintf("failed to resolve revision %s", revision),
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithField("revision", revision).WithSuggestions(
			"Check if the revision is correct",
			"Verify the revision exists in the repository",
			"Ensure the revision is reachable from the current branch",
		)
	}

	// Get the commit object
	commit, err := repo.CommitObject(*hash)
	if err != nil {
		return errors.NewAppError(
			errors.ErrorTypeGit,
			errors.ErrCodeInvalidFormat,
			fmt.Sprintf("failed to get commit %s", hash),
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithField("commit", hash.String()).WithSuggestions(
			"Check if the commit hash is correct",
			"Verify the commit exists in the repository",
		)
	}

	// Create a worktree checkout
	worktree, err := repo.Worktree()
	if err != nil {
		return errors.NewAppError(
			errors.ErrorTypeGit,
			errors.ErrCodeInvalidFormat,
			"failed to get worktree",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithSuggestions(
			"Check if the repository is a valid Git repository",
			"Verify the repository is properly initialized",
		)
	}

	// Checkout the commit
	err = worktree.Checkout(&git.CheckoutOptions{
		Hash:  commit.Hash,
		Force: true,
	})
	if err != nil {
		return errors.NewAppError(
			errors.ErrorTypeGit,
			errors.ErrCodeInvalidFormat,
			fmt.Sprintf("failed to checkout revision %s", revision),
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithField("revision", revision).WithSuggestions(
			"Check if the revision is correct",
			"Verify the revision exists in the repository",
			"Ensure the revision is reachable from the current branch",
		)
	}

	return nil
}

// getRepositoryInfo extracts information about a repository
func getRepositoryInfo(repo *git.Repository, repoPath, repoURL string) (*RepositoryInfo, error) {
	info := &RepositoryInfo{
		Path: repoPath,
		URL:  repoURL,
	}

	// Get HEAD reference
	head, err := repo.Head()
	if err != nil {
		return nil, errors.NewAppError(
			errors.ErrorTypeGit,
			errors.ErrCodeInvalidFormat,
			"failed to get HEAD",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithSuggestions(
			"Check if the repository has any commits",
			"Verify the repository is properly initialized",
		)
	}

	info.CurrentBranch = head.Name().Short()
	info.CurrentCommit = head.Hash().String()

	// Get default branch
	defaultBranch, err := GetDefaultBranch(repoPath)
	if err != nil {
		klog.V(2).Infof("Could not determine default branch: %v", err)
		defaultBranch = "main" // Default fallback
	}
	info.DefaultBranch = defaultBranch

	return info, nil
}

// getAuthMethod creates an appropriate authentication method based on the URL and config
func getAuthMethod(repoURL string, auth *AuthConfig) (transport.AuthMethod, error) {
	if auth == nil {
		return nil, nil
	}

	// Parse the URL to determine the scheme
	u, err := url.Parse(repoURL)
	if err != nil {
		// If it's an SSH URL, handle it separately
		if strings.Contains(repoURL, "@") && strings.Contains(repoURL, ":") {
			return getSSHAuth(auth)
		}
		return nil, errors.NewAppError(
			errors.ErrorTypeGit,
			errors.ErrCodeInvalidURL,
			"failed to parse repository URL",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithField("repository URL", repoURL).WithSuggestions(
			"Check if the repository URL is correct",
			"Ensure the URL includes the protocol (http://, https://, ssh://)",
		)
	}

	switch u.Scheme {
	case "http", "https":
		return getHTTPAuth(auth)
	case "ssh", "git":
		return getSSHAuth(auth)
	default:
		return nil, nil
	}
}

// getHTTPAuth creates HTTP authentication method
func getHTTPAuth(auth *AuthConfig) (transport.AuthMethod, error) {
	switch auth.Method {
	case AuthToken:
		return &http.BasicAuth{
			Username: "token", // Use "token" as username for token-based auth
			Password: auth.Token,
		}, nil
	case AuthUserPass:
		return &http.BasicAuth{
			Username: auth.Username,
			Password: auth.Password,
		}, nil
	default:
		return nil, nil
	}
}

// getSSHAuth creates SSH authentication method
func getSSHAuth(auth *AuthConfig) (transport.AuthMethod, error) {
	if auth.Method != AuthSSHKey {
		return nil, nil
	}

	var sshAuth *gitssh.PublicKeys
	var err error

	if auth.KeyPath != "" {
		// Use key file
		sshAuth, err = gitssh.NewPublicKeysFromFile("git", auth.KeyPath, "")
		if err != nil {
			return nil, errors.NewAppError(
				errors.ErrorTypeAuth,
				errors.ErrCodeAuthFailed,
				"failed to create SSH auth from key file",
				errors.ErrorSeverityError,
				false,
			).WithCause(err).WithField("key path", auth.KeyPath).WithSuggestions(
				"Check if the SSH key file exists",
				"Verify the SSH key file is readable",
				"Ensure the SSH key file is in the correct format",
			)
		}
	} else if auth.SSHKey != "" {
		// Use provided key content
		sshAuth, err = gitssh.NewPublicKeys("git", []byte(auth.SSHKey), "")
		if err != nil {
			return nil, errors.NewAppError(
				errors.ErrorTypeAuth,
				errors.ErrCodeAuthFailed,
				"failed to create SSH auth from key content",
				errors.ErrorSeverityError,
				false,
			).WithCause(err).WithSuggestions(
				"Check if the SSH key content is valid",
				"Ensure the SSH key is in the correct format",
				"Verify the SSH key is not encrypted",
			)
		}
	} else {
		return nil, errors.NewAppError(
			errors.ErrorTypeAuth,
			errors.ErrCodeAuthFailed,
			"SSH key not provided",
			errors.ErrorSeverityError,
			false,
		).WithSuggestions(
			"Provide an SSH key file path or key content",
			"Ensure the SSH key is properly configured",
		)
	}

	// Configure known hosts if provided
	if auth.KnownHosts != "" {
		// Use the provided known hosts file for host key verification
		hostKeyCallback, err := knownhosts.New(auth.KnownHosts)
		if err != nil {
			return nil, errors.NewAppError(
				errors.ErrorTypeAuth,
				errors.ErrCodeAuthFailed,
				"failed to create known hosts callback",
				errors.ErrorSeverityError,
				false,
			).WithCause(err).WithField("known hosts", auth.KnownHosts).WithSuggestions(
				"Check if the known hosts file exists",
				"Verify the known hosts file is readable",
				"Ensure the known hosts file is in the correct format",
			)
		}
		// Convert knownhosts.HostKeyCallback to ssh.HostKeyCallback
		sshAuth.HostKeyCallback = ssh.HostKeyCallback(hostKeyCallback)
	} else {
		// Default to accepting all host keys (not recommended for production)
		sshAuth.HostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	return sshAuth, nil
}
