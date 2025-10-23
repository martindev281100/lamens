package prompt

import (
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/your-org/kargo-bootstrap/pkg/git"
)

// InputValidator represents a function that validates input
type InputValidator func(string) error

// CommonValidators provides common validation functions
type CommonValidators struct{}

// NonEmptyValidator validates that input is not empty
func (v *CommonValidators) NonEmptyValidator(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("value cannot be empty")
	}
	return nil
}

// KubernetesNameValidator validates Kubernetes resource names
func (v *CommonValidators) KubernetesNameValidator(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("name cannot be empty")
	}

	// Kubernetes name constraints: lowercase alphanumeric, hyphens, and dots
	// Must start and end with alphanumeric
	validName := regexp.MustCompile(`^[a-z0-9]([a-z0-9\-\.]*[a-z0-9])?$`)
	if !validName.MatchString(value) {
		return fmt.Errorf("name must contain only lowercase alphanumeric characters, hyphens, and dots, and must start and end with alphanumeric characters")
	}

	if len(value) > 63 {
		return fmt.Errorf("name must be 63 characters or less")
	}

	return nil
}

// GitURLValidator validates Git repository URLs
func (v *CommonValidators) GitURLValidator(value string) error {
	_, err := git.ValidateRepositoryURL(value)
	return err
}

// BranchNameValidator validates Git branch names
func (v *CommonValidators) BranchNameValidator(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	// Git branch name constraints
	// Cannot start with a dot, contain double dots, or have problematic characters
	if strings.HasPrefix(value, ".") {
		return fmt.Errorf("branch name cannot start with a dot")
	}

	if strings.Contains(value, "..") {
		return fmt.Errorf("branch name cannot contain consecutive dots")
	}

	// Cannot contain spaces, tabs, or other whitespace
	if strings.ContainsAny(value, " \t\n\r") {
		return fmt.Errorf("branch name cannot contain whitespace")
	}

	// Cannot end with .lock
	if strings.HasSuffix(value, ".lock") {
		return fmt.Errorf("branch name cannot end with .lock")
	}

	// Cannot contain problematic characters
	problematic := []string{"~", "^", ":", "?", "*", "[", "@", "\\"}
	for _, char := range problematic {
		if strings.Contains(value, char) {
			return fmt.Errorf("branch name cannot contain '%s'", char)
		}
	}

	return nil
}

// TagValidator validates Git tag names
func (v *CommonValidators) TagValidator(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("tag name cannot be empty")
	}

	// Git tag name constraints (similar to branch names)
	// Cannot contain spaces or other whitespace
	if strings.ContainsAny(value, " \t\n\r") {
		return fmt.Errorf("tag name cannot contain whitespace")
	}

	// Cannot start with a dot
	if strings.HasPrefix(value, ".") {
		return fmt.Errorf("tag name cannot start with a dot")
	}

	return nil
}

// CommitHashValidator validates Git commit hashes
func (v *CommonValidators) CommitHashValidator(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("commit hash cannot be empty")
	}

	// Git commit hash should be 7-40 hexadecimal characters
	validHash := regexp.MustCompile(`^[a-fA-F0-9]{7,40}$`)
	if !validHash.MatchString(value) {
		return fmt.Errorf("commit hash must be 7-40 hexadecimal characters")
	}

	return nil
}

// InputFormatter provides formatting functions for prompt display
type InputFormatter struct{}

// FormatRepositoryURL formats a repository URL for display
func (f *InputFormatter) FormatRepositoryURL(url string) string {
	// Truncate long URLs for display
	if len(url) > 60 {
		return url[:57] + "..."
	}
	return url
}

// FormatProjectOption formats a project option for display
func (f *InputFormatter) FormatProjectOption(name, description string) string {
	if description == "" {
		description = "No description"
	}
	return fmt.Sprintf("%s - %s", name, description)
}

// FormatChartOption formats a chart option for display
func (f *InputFormatter) FormatChartOption(name, version, description string) string {
	if description == "" {
		description = "No description"
	}
	return fmt.Sprintf("%s (v%s) - %s", name, version, description)
}

// FormatEnvironmentOption formats an environment option for display
func (f *InputFormatter) FormatEnvironmentOption(name, description string, isDefault bool) string {
	formatted := fmt.Sprintf("%s - %s", name, description)
	if isDefault {
		formatted += " [default]"
	}
	return formatted
}

// FormatRevisionOption formats a revision option for display
func (f *InputFormatter) FormatRevisionOption(revisionType, value string, isLatest bool) string {
	formatted := fmt.Sprintf("%s: %s", revisionType, value)
	if isLatest {
		formatted += " [latest]"
	}
	return formatted
}

// InterruptHandler handles interrupt signals gracefully
type InterruptHandler struct {
	interruptChan chan os.Signal
	done          chan bool
	cleanupFunc   func()
}

// NewInterruptHandler creates a new interrupt handler
func NewInterruptHandler(cleanupFunc func()) *InterruptHandler {
	return &InterruptHandler{
		interruptChan: make(chan os.Signal, 1),
		done:          make(chan bool, 1),
		cleanupFunc:   cleanupFunc,
	}
}

// Start starts listening for interrupt signals
func (h *InterruptHandler) Start() {
	signal.Notify(h.interruptChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-h.interruptChan:
			fmt.Println("\n\nInterrupt received. Cleaning up...")
			if h.cleanupFunc != nil {
				h.cleanupFunc()
			}
			os.Exit(1)
		case <-h.done:
			// Graceful shutdown
			return
		}
	}()
}

// Stop stops the interrupt handler
func (h *InterruptHandler) Stop() {
	signal.Stop(h.interruptChan)
	close(h.done)
}

// PromptHelper provides utility functions for prompting
type PromptHelper struct {
	formatter *InputFormatter
	validator *CommonValidators
}

// NewPromptHelper creates a new prompt helper
func NewPromptHelper() *PromptHelper {
	return &PromptHelper{
		formatter: &InputFormatter{},
		validator: &CommonValidators{},
	}
}

// FormatRepositoryURL formats a repository URL for display
func (h *PromptHelper) FormatRepositoryURL(url string) string {
	return h.formatter.FormatRepositoryURL(url)
}

// FormatProjectOption formats a project option for display
func (h *PromptHelper) FormatProjectOption(name, description string) string {
	return h.formatter.FormatProjectOption(name, description)
}

// FormatChartOption formats a chart option for display
func (h *PromptHelper) FormatChartOption(name, version, description string) string {
	return h.formatter.FormatChartOption(name, version, description)
}

// FormatEnvironmentOption formats an environment option for display
func (h *PromptHelper) FormatEnvironmentOption(name, description string, isDefault bool) string {
	return h.formatter.FormatEnvironmentOption(name, description, isDefault)
}

// ValidateAppName validates an application name
func (h *PromptHelper) ValidateAppName(name string) error {
	return h.validator.KubernetesNameValidator(name)
}

// ValidateRepositoryURL validates a Git repository URL
func (h *PromptHelper) ValidateRepositoryURL(url string) error {
	return h.validator.GitURLValidator(url)
}

// ValidateBranchName validates a Git branch name
func (h *PromptHelper) ValidateBranchName(name string) error {
	return h.validator.BranchNameValidator(name)
}

// ValidateTagName validates a Git tag name
func (h *PromptHelper) ValidateTagName(name string) error {
	return h.validator.TagValidator(name)
}

// ValidateCommitHash validates a Git commit hash
func (h *PromptHelper) ValidateCommitHash(hash string) error {
	return h.validator.CommitHashValidator(hash)
}

// RetryWithBackoff retries a function with exponential backoff
func RetryWithBackoff(fn func() error, maxRetries int, initialDelay time.Duration) error {
	var lastErr error
	delay := initialDelay

	for i := 0; i < maxRetries; i++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
			if i < maxRetries-1 {
				fmt.Printf("Attempt %d failed: %v. Retrying in %v...\n", i+1, err, delay)
				time.Sleep(delay)
				delay *= 2 // Exponential backoff
			}
		}
	}

	return fmt.Errorf("after %d attempts, last error: %w", maxRetries, lastErr)
}

// CreateDefaultEnvironments creates a set of default environments
func CreateDefaultEnvironments() []Environment {
	return []Environment{
		{
			Name:        "development",
			Type:        EnvironmentTypeDevelopment,
			Description: "Development environment for testing changes",
			Default:     true,
		},
		{
			Name:        "staging",
			Type:        EnvironmentTypeStaging,
			Description: "Staging environment for pre-production testing",
			Default:     false,
		},
		{
			Name:        "production",
			Type:        EnvironmentTypeProduction,
			Description: "Production environment for live traffic",
			Default:     false,
		},
	}
}

// FilterOptions filters options based on a search term
func FilterOptions(options []string, searchTerm string) []string {
	if searchTerm == "" {
		return options
	}

	var filtered []string
	searchTerm = strings.ToLower(searchTerm)

	for _, option := range options {
		if strings.Contains(strings.ToLower(option), searchTerm) {
			filtered = append(filtered, option)
		}
	}

	return filtered
}

// PaginateOptions paginates options for display
func PaginateOptions(options []string, pageSize int) [][]string {
	if pageSize <= 0 {
		pageSize = 10
	}

	var pages [][]string
	for i := 0; i < len(options); i += pageSize {
		end := i + pageSize
		if end > len(options) {
			end = len(options)
		}
		pages = append(pages, options[i:end])
	}

	return pages
}
