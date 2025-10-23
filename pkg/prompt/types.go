package prompt

import (
	"github.com/your-org/kargo-bootstrap/pkg/argocd"
	"github.com/your-org/kargo-bootstrap/pkg/git"
)

// PromptResult represents the result of a prompt operation
type PromptResult struct {
	Success bool
	Value   interface{}
	Error   error
}

// ProjectSelection represents the result of a project selection prompt
type ProjectSelection struct {
	Project   *argocd.Project
	Confirmed bool
	Cancelled bool
}

// RepositoryInput represents the result of a repository input prompt
type RepositoryInput struct {
	URL       string
	Validated bool
	Cancelled bool
}

// RevisionSelection represents the result of a revision selection prompt
type RevisionSelection struct {
	Type      string // "branch", "tag", or "commit"
	Value     string
	Confirmed bool
	Cancelled bool
}

// ChartSelection represents the result of a chart selection prompt
type ChartSelection struct {
	Chart     *git.ChartInfo
	Confirmed bool
	Cancelled bool
}

// EnvironmentSelection represents the result of an environment selection prompt
type EnvironmentSelection struct {
	Environments []string
	Confirmed    bool
	Cancelled    bool
}

// AppNameInput represents the result of an application name input prompt
type AppNameInput struct {
	Name      string
	Validated bool
	Cancelled bool
}

// PromptConfig contains configuration for prompt behavior
type PromptConfig struct {
	ShowHelp         bool
	ShowDefaults     bool
	ConfirmDangerous bool
	TimeoutSeconds   int
	PageSize         int
	ColorOutput      bool
}

// DefaultPromptConfig returns the default configuration for prompts
func DefaultPromptConfig() *PromptConfig {
	return &PromptConfig{
		ShowHelp:         true,
		ShowDefaults:     true,
		ConfirmDangerous: true,
		TimeoutSeconds:   0, // No timeout by default
		PageSize:         10,
		ColorOutput:      true,
	}
}

// PromptType represents different types of prompts
type PromptType string

const (
	PromptTypeSelect      PromptType = "select"
	PromptTypeInput       PromptType = "input"
	PromptTypeMultiSelect PromptType = "multiselect"
	PromptTypeConfirm     PromptType = "confirm"
	PromptTypePassword    PromptType = "password"
)

// ValidationRule represents a validation rule for input
type ValidationRule struct {
	Name        string
	Description string
	Validator   func(string) error
}

// EnvironmentType represents different types of deployment environments
type EnvironmentType string

const (
	EnvironmentTypeDevelopment EnvironmentType = "development"
	EnvironmentTypeStaging     EnvironmentType = "staging"
	EnvironmentTypeProduction  EnvironmentType = "production"
	EnvironmentTypeTesting     EnvironmentType = "testing"
	EnvironmentTypeCustom      EnvironmentType = "custom"
)

// Environment represents a deployment environment
type Environment struct {
	Name        string
	Type        EnvironmentType
	Description string
	Default     bool
}

// RevisionType represents different types of Git revisions
type RevisionType string

const (
	RevisionTypeBranch RevisionType = "branch"
	RevisionTypeTag    RevisionType = "tag"
	RevisionTypeCommit RevisionType = "commit"
)

// Revision represents a Git revision
type Revision struct {
	Type        RevisionType
	Name        string
	Hash        string
	Latest      bool
	Description string
}

// PromptOptions contains options for creating a prompt
type PromptOptions struct {
	Message      string
	Default      interface{}
	Help         string
	Required     bool
	Validate     func(interface{}) error
	PageSize     int
	FilterFunc   func(string) []string
	ShowHelp     bool
	ShowDefaults bool
	Timeout      int
}

// SelectOptions contains options for select prompts
type SelectOptions struct {
	PromptOptions
	Items        []string
	DefaultIndex int
}

// MultiSelectOptions contains options for multi-select prompts
type MultiSelectOptions struct {
	PromptOptions
	Items         []string
	DefaultValues []string
	Min           int
	Max           int
}

// InputOptions contains options for input prompts
type InputOptions struct {
	PromptOptions
	Mask     bool
	Validate func(string) error
}

// ConfirmOptions contains options for confirmation prompts
type ConfirmOptions struct {
	PromptOptions
	Default bool
}
