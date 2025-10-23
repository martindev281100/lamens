package prompt

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"syscall"

	"github.com/AlecAivazis/survey/v2"
	"github.com/your-org/kargo-bootstrap/pkg/argocd"
	"github.com/your-org/kargo-bootstrap/pkg/git"
	"golang.org/x/term"
)

// Prompter represents an interactive prompt handler
type Prompter struct {
	config *PromptConfig
}

// NewPrompter creates a new prompt handler
func NewPrompter() (*Prompter, error) {
	return &Prompter{
		config: DefaultPromptConfig(),
	}, nil
}

// NewPrompterWithConfig creates a new prompt handler with custom configuration
func NewPrompterWithConfig(config *PromptConfig) (*Prompter, error) {
	if config == nil {
		config = DefaultPromptConfig()
	}
	return &Prompter{
		config: config,
	}, nil
}

// SelectProject prompts the user to select an ArgoCD project from a list
func (p *Prompter) SelectProject(projects []argocd.Project, defaultProject string) (*ProjectSelection, error) {
	if len(projects) == 0 {
		return nil, fmt.Errorf("no projects available for selection")
	}

	// If there's only one project, use it without prompting
	if len(projects) == 1 {
		return &ProjectSelection{
			Project:   &projects[0],
			Confirmed: true,
			Cancelled: false,
		}, nil
	}

	// Prepare project options
	options := make([]string, len(projects))
	defaultIndex := 0

	for i, project := range projects {
		description := project.Description
		if description == "" {
			description = "No description"
		}

		options[i] = fmt.Sprintf("%s - %s", project.Name, description)

		if project.Name == defaultProject {
			defaultIndex = i
		}
	}

	// Create the select prompt
	prompt := &survey.Select{
		Message: "Select an ArgoCD project:",
		Options: options,
		Default: options[defaultIndex],
		Help:    "ArgoCD projects define where applications can be deployed and what repositories can be used.",
	}

	var selectedOption string
	if err := survey.AskOne(prompt, &selectedOption); err != nil {
		if err.Error() == "interrupt" {
			return &ProjectSelection{
				Cancelled: true,
			}, nil
		}
		return nil, NewPromptError(ErrTypeInterrupted, "failed to prompt for project selection", err)
	}

	// Find the selected project
	var selectedProject *argocd.Project
	for i, option := range options {
		if option == selectedOption {
			selectedProject = &projects[i]
			break
		}
	}

	if selectedProject == nil {
		return nil, fmt.Errorf("selected project not found")
	}

	// Show project details
	fmt.Printf("\nSelected project: %s\n", selectedProject.Name)
	if selectedProject.Description != "" {
		fmt.Printf("Description: %s\n", selectedProject.Description)
	}
	fmt.Printf("Source repositories: %d\n", len(selectedProject.SourceRepos))
	fmt.Printf("Destinations: %d\n", len(selectedProject.Destinations))

	// Confirm selection
	confirmed, err := p.ConfirmSelection("Use this project?", true)
	if err != nil {
		return nil, fmt.Errorf("failed to confirm project selection: %w", err)
	}

	return &ProjectSelection{
		Project:   selectedProject,
		Confirmed: confirmed,
		Cancelled: !confirmed,
	}, nil
}

// SelectRepository prompts the user to input a Git repository URL
func (p *Prompter) SelectRepository(defaultURL string) (*RepositoryInput, error) {
	prompt := &survey.Input{
		Message: "Enter Git repository URL:",
		Default: defaultURL,
		Help:    "Enter the URL of the Git repository containing your Helm charts. Supports HTTP(S) and SSH protocols.",
	}

	var repoURL string
	if err := survey.AskOne(prompt, &repoURL); err != nil {
		if err.Error() == "interrupt" {
			return &RepositoryInput{
				Cancelled: true,
			}, nil
		}
		return nil, NewPromptError(ErrTypeInterrupted, "failed to prompt for repository URL", err)
	}

	// Validate the repository URL
	if _, err := git.ValidateRepositoryURL(repoURL); err != nil {
		fmt.Printf("Invalid repository URL: %v\n", err)
		return p.SelectRepository(repoURL) // Retry with current value as default
	}

	return &RepositoryInput{
		URL:       repoURL,
		Validated: true,
		Cancelled: false,
	}, nil
}

// SelectRevision prompts the user to select a Git revision (branch/tag/commit)
func (p *Prompter) SelectRevision(repoURL string, defaultBranch string) (*RevisionSelection, error) {
	// First, select the revision type
	revisionTypePrompt := &survey.Select{
		Message: "Select revision type:",
		Options: []string{
			string(RevisionTypeBranch),
			string(RevisionTypeTag),
			string(RevisionTypeCommit),
		},
		Default: string(RevisionTypeBranch),
		Help:    "Select whether to use a branch, tag, or specific commit hash.",
	}

	var revisionType string
	if err := survey.AskOne(revisionTypePrompt, &revisionType); err != nil {
		if err.Error() == "interrupt" {
			return &RevisionSelection{
				Cancelled: true,
			}, nil
		}
		return nil, NewPromptError(ErrTypeInterrupted, "failed to prompt for revision type", err)
	}

	// Then prompt for the specific revision value
	var revisionPrompt survey.Prompt
	var helpText string

	switch RevisionType(revisionType) {
	case RevisionTypeBranch:
		helpText = "Enter the branch name to use for deployment."
		revisionPrompt = &survey.Input{
			Message: "Enter branch name:",
			Default: defaultBranch,
			Help:    helpText,
		}
	case RevisionTypeTag:
		helpText = "Enter the tag name to use for deployment."
		revisionPrompt = &survey.Input{
			Message: "Enter tag name:",
			Help:    helpText,
		}
	case RevisionTypeCommit:
		helpText = "Enter the commit hash to use for deployment."
		revisionPrompt = &survey.Input{
			Message: "Enter commit hash:",
			Help:    helpText,
		}
	default:
		return nil, fmt.Errorf("unsupported revision type: %s", revisionType)
	}

	var revisionValue string
	if err := survey.AskOne(revisionPrompt, &revisionValue); err != nil {
		if err.Error() == "interrupt" {
			return &RevisionSelection{
				Cancelled: true,
			}, nil
		}
		return nil, NewPromptError(ErrTypeInterrupted, "failed to prompt for revision value", err)
	}

	// Validate the revision value
	if strings.TrimSpace(revisionValue) == "" {
		fmt.Println("Revision value cannot be empty")
		return p.SelectRevision(repoURL, defaultBranch) // Retry
	}

	// Confirm selection
	confirmed, err := p.ConfirmSelection(
		fmt.Sprintf("Use %s '%s' from repository %s?", revisionType, revisionValue, repoURL),
		true,
	)
	if err != nil {
		return nil, NewPromptError(ErrTypeInterrupted, "failed to confirm revision selection", err)
	}

	return &RevisionSelection{
		Type:      revisionType,
		Value:     revisionValue,
		Confirmed: confirmed,
		Cancelled: !confirmed,
	}, nil
}

// SelectChartPath prompts the user to select from discovered chart paths
func (p *Prompter) SelectChartPath(charts []*git.ChartInfo) (*ChartSelection, error) {
	if len(charts) == 0 {
		return nil, fmt.Errorf("no charts available for selection")
	}

	// If there's only one chart, use it without prompting
	if len(charts) == 1 {
		return &ChartSelection{
			Chart:     charts[0],
			Confirmed: true,
			Cancelled: false,
		}, nil
	}

	// Prepare chart options
	options := make([]string, len(charts))
	for i, chart := range charts {
		description := chart.Description
		if description == "" {
			description = "No description"
		}
		options[i] = fmt.Sprintf("%s (v%s) - %s", chart.Name, chart.Version, description)
	}

	// Create the select prompt
	prompt := &survey.Select{
		Message: "Select a Helm chart:",
		Options: options,
		Help:    "Select the Helm chart to deploy from the repository.",
	}

	var selectedOption string
	if err := survey.AskOne(prompt, &selectedOption); err != nil {
		if err.Error() == "interrupt" {
			return &ChartSelection{
				Cancelled: true,
			}, nil
		}
		return nil, NewPromptError(ErrTypeInterrupted, "failed to prompt for chart selection", err)
	}

	// Find the selected chart
	var selectedChart *git.ChartInfo
	for i, option := range options {
		if option == selectedOption {
			selectedChart = charts[i]
			break
		}
	}

	if selectedChart == nil {
		return nil, fmt.Errorf("selected chart not found")
	}

	// Show chart details
	fmt.Printf("\nSelected chart: %s\n", selectedChart.Name)
	fmt.Printf("Version: %s\n", selectedChart.Version)
	if selectedChart.AppVersion != "" {
		fmt.Printf("App Version: %s\n", selectedChart.AppVersion)
	}
	if selectedChart.Description != "" {
		fmt.Printf("Description: %s\n", selectedChart.Description)
	}
	fmt.Printf("Path: %s\n", selectedChart.Path)

	// Confirm selection
	confirmed, err := p.ConfirmSelection("Use this chart?", true)
	if err != nil {
		return nil, NewPromptError(ErrTypeInterrupted, "failed to confirm chart selection", err)
	}

	return &ChartSelection{
		Chart:     selectedChart,
		Confirmed: confirmed,
		Cancelled: !confirmed,
	}, nil
}

// SelectEnvironments prompts the user to select deployment environments
func (p *Prompter) SelectEnvironments(availableEnvs []Environment, defaults []string) (*EnvironmentSelection, error) {
	if len(availableEnvs) == 0 {
		return nil, fmt.Errorf("no environments available for selection")
	}

	// Prepare environment options
	options := make([]string, len(availableEnvs))
	defaultOptions := make([]string, 0, len(defaults))

	for i, env := range availableEnvs {
		options[i] = fmt.Sprintf("%s - %s", env.Name, env.Description)
		if env.Default || contains(defaults, env.Name) {
			defaultOptions = append(defaultOptions, options[i])
		}
	}

	// Create the multi-select prompt
	prompt := &survey.MultiSelect{
		Message: "Select deployment environments:",
		Options: options,
		Default: defaultOptions,
		Help:    "Select the environments where you want to deploy this application.",
	}

	var selectedOptions []string
	if err := survey.AskOne(prompt, &selectedOptions); err != nil {
		if err.Error() == "interrupt" {
			return &EnvironmentSelection{
				Cancelled: true,
			}, nil
		}
		return nil, NewPromptError(ErrTypeInterrupted, "failed to prompt for environment selection", err)
	}

	// Extract environment names from selected options
	selectedEnvs := make([]string, 0, len(selectedOptions))
	for _, selectedOption := range selectedOptions {
		for _, env := range availableEnvs {
			if strings.HasPrefix(selectedOption, env.Name+" -") {
				selectedEnvs = append(selectedEnvs, env.Name)
				break
			}
		}
	}

	if len(selectedEnvs) == 0 {
		fmt.Println("At least one environment must be selected")
		return p.SelectEnvironments(availableEnvs, defaults) // Retry
	}

	// Confirm selection
	confirmed, err := p.ConfirmSelection(
		fmt.Sprintf("Deploy to environments: %s?", strings.Join(selectedEnvs, ", ")),
		true,
	)
	if err != nil {
		return nil, NewPromptError(ErrTypeInterrupted, "failed to confirm environment selection", err)
	}

	return &EnvironmentSelection{
		Environments: selectedEnvs,
		Confirmed:    confirmed,
		Cancelled:    !confirmed,
	}, nil
}

// SelectEnvironmentsFromConfig prompts the user to select deployment environments using config
func (p *Prompter) SelectEnvironmentsFromConfig(envManager interface{}, defaults []string) (*EnvironmentSelection, error) {
	// We need to use interface{} to avoid import cycle
	// The actual implementation will be in the cmd package where it has access to both packages
	return nil, fmt.Errorf("SelectEnvironmentsFromConfig should be implemented in the cmd package to avoid import cycles")
}

// InputAppName prompts the user to input an application name
func (p *Prompter) InputAppName(defaultName string) (*AppNameInput, error) {
	prompt := &survey.Input{
		Message: "Enter application name:",
		Default: defaultName,
		Help:    "Enter a unique name for this application. This will be used to identify the application in ArgoCD and Kargo.",
	}

	var appName string
	if err := survey.AskOne(prompt, &appName); err != nil {
		if err.Error() == "interrupt" {
			return &AppNameInput{
				Cancelled: true,
			}, nil
		}
		return nil, NewPromptError(ErrTypeInterrupted, "failed to prompt for application name", err)
	}

	// Validate the application name
	if err := ValidateAppName(appName); err != nil {
		fmt.Printf("Invalid application name: %v\n", err)
		return p.InputAppName(appName) // Retry with current value as default
	}

	return &AppNameInput{
		Name:      appName,
		Validated: true,
		Cancelled: false,
	}, nil
}

// ConfirmSelection prompts the user to confirm a selection
func (p *Prompter) ConfirmSelection(message string, defaultChoice bool) (bool, error) {
	if !p.config.ConfirmDangerous {
		return true, nil
	}

	prompt := &survey.Confirm{
		Message: message,
		Default: defaultChoice,
	}

	var confirmed bool
	if err := survey.AskOne(prompt, &confirmed); err != nil {
		if err.Error() == "interrupt" {
			return false, nil
		}
		return false, NewPromptError(ErrTypeInterrupted, "failed to confirm selection", err)
	}

	return confirmed, nil
}

// ValidateInput provides common validation functions for user input
func ValidateInput(input string, rules []ValidationRule) error {
	for _, rule := range rules {
		if err := rule.Validator(input); err != nil {
			return fmt.Errorf("%s: %w", rule.Name, err)
		}
	}
	return nil
}

// ValidateAppName validates an application name
func ValidateAppName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("application name cannot be empty")
	}

	// Kubernetes name constraints: lowercase alphanumeric, hyphens, and dots
	// Must start and end with alphanumeric
	validName := regexp.MustCompile(`^[a-z0-9]([a-z0-9\-\.]*[a-z0-9])?$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("application name must contain only lowercase alphanumeric characters, hyphens, and dots, and must start and end with alphanumeric characters")
	}

	if len(name) > 63 {
		return fmt.Errorf("application name must be 63 characters or less")
	}

	return nil
}

// ValidateRepositoryURL validates a Git repository URL
func ValidateRepositoryURL(url string) error {
	_, err := git.ValidateRepositoryURL(url)
	if err != nil {
		return NewPromptError(ErrTypeValidation, "invalid repository URL", err)
	}
	return nil
}

// FormatOptions formats options for display in prompts
func FormatOptions(options []string, pageSize int) [][]string {
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

// HandleInterrupt handles Ctrl+C and other interruptions gracefully
func HandleInterrupt() {
	// Check if we're in a terminal
	if !term.IsTerminal(int(syscall.Stdin)) {
		return
	}

	// Set up signal handling for graceful interruption
	// This is a placeholder for more sophisticated interrupt handling
	fmt.Println("\nOperation cancelled by user.")
	os.Exit(0)
}

// contains checks if a string slice contains a specific string
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
