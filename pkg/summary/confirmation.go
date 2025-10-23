package summary

import (
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
)

// ConfirmationOptions represents options for confirmation prompts
type ConfirmationOptions struct {
	// Message to display to the user
	Message string

	// Default value if user just presses enter
	Default bool

	// Show help text
	ShowHelp bool

	// Custom help text
	HelpText string

	// Allow "modify" option
	AllowModify bool

	// Allow "back" option
	AllowBack bool

	// Non-interactive mode (auto-approve if true)
	AutoApprove bool

	// Dry-run mode (always approve if true)
	DryRun bool
}

// ConfirmationResult represents the result of a confirmation prompt
type ConfirmationResult struct {
	// User's choice
	Confirmed bool

	// User wants to modify configuration
	Modify bool

	// User wants to go back
	Back bool

	// User cancelled the operation
	Cancelled bool

	// Additional user input (if any)
	Input string
}

// ConfirmDeployment prompts the user to confirm a deployment
func ConfirmDeployment(summary *DeploymentSummary, options *ConfirmationOptions) (*ConfirmationResult, error) {
	if options == nil {
		options = &ConfirmationOptions{
			Message:     "Do you want to proceed with this deployment?",
			Default:     false,
			ShowHelp:    true,
			AllowModify: true,
			AllowBack:   true,
		}
	}

	// In non-interactive mode with auto-approve, always confirm
	if options.AutoApprove {
		return &ConfirmationResult{
			Confirmed: true,
		}, nil
	}

	// In dry-run mode, always confirm
	if options.DryRun {
		return &ConfirmationResult{
			Confirmed: true,
		}, nil
	}

	// Display the summary
	fmt.Println(FormatDeploymentSummary(summary))

	// Prepare the prompt
	var prompt survey.Prompt

	if options.AllowModify || options.AllowBack {
		// Use a select prompt with more options
		var choices []string
		if options.AllowModify {
			choices = append(choices, "Proceed with deployment")
			choices = append(choices, "Modify configuration")
		} else {
			choices = append(choices, "Yes")
		}

		if options.AllowBack {
			choices = append(choices, "Go back")
		}

		choices = append(choices, "Cancel")

		prompt = &survey.Select{
			Message: options.Message,
			Options: choices,
			Default: func() string {
				if options.AllowModify {
					return "Proceed with deployment"
				}
				return "Yes"
			}(),
			Help: options.getHelpText(),
		}

		var answer string
		if err := survey.AskOne(prompt, &answer); err != nil {
			if isInterrupted(err) {
				return &ConfirmationResult{
					Cancelled: true,
				}, nil
			}
			return nil, fmt.Errorf("failed to get user input: %w", err)
		}

		// Process the answer
		switch answer {
		case "Proceed with deployment", "Yes":
			return &ConfirmationResult{
				Confirmed: true,
			}, nil
		case "Modify configuration":
			return &ConfirmationResult{
				Modify: true,
			}, nil
		case "Go back":
			return &ConfirmationResult{
				Back: true,
			}, nil
		case "Cancel":
			return &ConfirmationResult{
				Cancelled: true,
			}, nil
		default:
			return &ConfirmationResult{
				Cancelled: true,
			}, nil
		}
	} else {
		// Use a simple confirm prompt
		prompt = &survey.Confirm{
			Message: options.Message,
			Default: options.Default,
			Help:    options.getHelpText(),
		}

		var confirmed bool
		if err := survey.AskOne(prompt, &confirmed); err != nil {
			if isInterrupted(err) {
				return &ConfirmationResult{
					Cancelled: true,
				}, nil
			}
			return nil, fmt.Errorf("failed to get user input: %w", err)
		}

		return &ConfirmationResult{
			Confirmed: confirmed,
		}, nil
	}
}

// ConfirmArgoCDCreate prompts the user to confirm ArgoCD application creation
func ConfirmArgoCDCreate(summary *ArgoCDCreateSummary, options *ConfirmationOptions) (*ConfirmationResult, error) {
	if options == nil {
		options = &ConfirmationOptions{
			Message:     "Do you want to create this ArgoCD Application?",
			Default:     false,
			ShowHelp:    true,
			AllowModify: false,
			AllowBack:   false,
		}
	}

	// In non-interactive mode with auto-approve, always confirm
	if options.AutoApprove {
		return &ConfirmationResult{
			Confirmed: true,
		}, nil
	}

	// In dry-run mode, always confirm
	if options.DryRun {
		return &ConfirmationResult{
			Confirmed: true,
		}, nil
	}

	// Display the summary
	fmt.Println(FormatArgoCDCreateSummary(summary))

	// Use a simple confirm prompt
	prompt := &survey.Confirm{
		Message: options.Message,
		Default: options.Default,
		Help:    options.getHelpText(),
	}

	var confirmed bool
	if err := survey.AskOne(prompt, &confirmed); err != nil {
		if isInterrupted(err) {
			return &ConfirmationResult{
				Cancelled: true,
			}, nil
		}
		return nil, fmt.Errorf("failed to get user input: %w", err)
	}

	return &ConfirmationResult{
		Confirmed: confirmed,
	}, nil
}

// ConfirmEnvOperation prompts the user to confirm an environment operation
func ConfirmEnvOperation(summary *EnvOperationSummary, options *ConfirmationOptions) (*ConfirmationResult, error) {
	if options == nil {
		options = &ConfirmationOptions{
			Message:     "Do you want to proceed with this operation?",
			Default:     false,
			ShowHelp:    true,
			AllowModify: false,
			AllowBack:   false,
		}
	}

	// In non-interactive mode with auto-approve, always confirm
	if options.AutoApprove {
		return &ConfirmationResult{
			Confirmed: true,
		}, nil
	}

	// Display the summary
	fmt.Println(FormatEnvOperationSummary(summary))

	// Use a simple confirm prompt
	prompt := &survey.Confirm{
		Message: options.Message,
		Default: options.Default,
		Help:    options.getHelpText(),
	}

	var confirmed bool
	if err := survey.AskOne(prompt, &confirmed); err != nil {
		if isInterrupted(err) {
			return &ConfirmationResult{
				Cancelled: true,
			}, nil
		}
		return nil, fmt.Errorf("failed to get user input: %w", err)
	}

	return &ConfirmationResult{
		Confirmed: confirmed,
	}, nil
}

// ConfirmRiskyOperation prompts the user for confirmation of a risky operation
func ConfirmRiskyOperation(message string, warnings []string, options *ConfirmationOptions) (*ConfirmationResult, error) {
	if options == nil {
		options = &ConfirmationOptions{
			Message:     message,
			Default:     false,
			ShowHelp:    true,
			AllowModify: false,
			AllowBack:   true,
		}
	}

	// In non-interactive mode with auto-approve, always confirm
	if options.AutoApprove {
		return &ConfirmationResult{
			Confirmed: true,
		}, nil
	}

	// Display warnings
	if len(warnings) > 0 {
		fmt.Println(warningColor("⚠️  Warnings:"))
		for i, warning := range warnings {
			fmt.Printf("  %d. %s\n", i+1, warning)
		}
		fmt.Println()
	}

	// Use a more explicit prompt
	var choices []string
	choices = append(choices, "Yes, I understand the risks")
	choices = append(choices, "No, cancel the operation")

	if options.AllowBack {
		choices = append(choices, "Go back")
	}

	prompt := &survey.Select{
		Message: options.Message,
		Options: choices,
		Default: "No, cancel the operation",
		Help:    options.getHelpText(),
	}

	var answer string
	if err := survey.AskOne(prompt, &answer); err != nil {
		if isInterrupted(err) {
			return &ConfirmationResult{
				Cancelled: true,
			}, nil
		}
		return nil, fmt.Errorf("failed to get user input: %w", err)
	}

	// Process the answer
	switch answer {
	case "Yes, I understand the risks":
		return &ConfirmationResult{
			Confirmed: true,
		}, nil
	case "No, cancel the operation":
		return &ConfirmationResult{
			Cancelled: true,
		}, nil
	case "Go back":
		return &ConfirmationResult{
			Back: true,
		}, nil
	default:
		return &ConfirmationResult{
			Cancelled: true,
		}, nil
	}
}

// PromptForInput prompts the user for text input
func PromptForInput(message, defaultValue string, required bool) (string, error) {
	prompt := &survey.Input{
		Message: message,
		Default: defaultValue,
		Help:    "Press Enter to use the default value",
	}

	var answer string
	if err := survey.AskOne(prompt, &answer); err != nil {
		if isInterrupted(err) {
			return "", fmt.Errorf("user cancelled input")
		}
		return "", fmt.Errorf("failed to get user input: %w", err)
	}

	if required && strings.TrimSpace(answer) == "" {
		return "", fmt.Errorf("input is required")
	}

	return answer, nil
}

// PromptForPassword prompts the user for a password
func PromptForPassword(message string) (string, error) {
	prompt := &survey.Password{
		Message: message,
		Help:    "Password will not be displayed as you type",
	}

	var answer string
	if err := survey.AskOne(prompt, &answer); err != nil {
		if isInterrupted(err) {
			return "", fmt.Errorf("user cancelled input")
		}
		return "", fmt.Errorf("failed to get user input: %w", err)
	}

	return answer, nil
}

// PromptForMultiSelect prompts the user to select multiple options
func PromptForMultiSelect(message string, options []string, defaults []string) ([]string, error) {
	prompt := &survey.MultiSelect{
		Message: message,
		Options: options,
		Default: defaults,
		Help:    "Use arrow keys to navigate, space to select, enter to confirm",
	}

	var answers []string
	if err := survey.AskOne(prompt, &answers); err != nil {
		if isInterrupted(err) {
			return nil, fmt.Errorf("user cancelled input")
		}
		return nil, fmt.Errorf("failed to get user input: %w", err)
	}

	return answers, nil
}

// Helper functions

// getHelpText returns the help text for the confirmation prompt
func (o *ConfirmationOptions) getHelpText() string {
	if o.HelpText != "" {
		return o.HelpText
	}

	if o.ShowHelp {
		return "Use arrow keys to navigate, enter to select"
	}

	return ""
}

// warningColor returns a colored warning string
func warningColor(s string) string {
	return colors.WarningColor(s)
}

// IsTerminal checks if the current output is a terminal
func IsTerminal() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// CanPrompt checks if interactive prompts are available
func CanPrompt() bool {
	return IsTerminal() && os.Getenv("CI") == "" && os.Getenv("NON_INTERACTIVE") == ""
}

// PromptOrAutoConfirm either prompts the user or auto-confirms based on environment
func PromptOrAutoConfirm(summary *DeploymentSummary, options *ConfirmationOptions) (*ConfirmationResult, error) {
	if !CanPrompt() {
		// Auto-confirm in non-interactive environments
		return &ConfirmationResult{
			Confirmed: true,
		}, nil
	}

	return ConfirmDeployment(summary, options)
}

// isInterrupted checks if the error is an interruption error
func isInterrupted(err error) bool {
	if err == nil {
		return false
	}

	// Check for keyboard interrupt
	if err.Error() == "interrupt" {
		return true
	}

	// Check for EOF (Ctrl+D)
	if err.Error() == "EOF" {
		return true
	}

	return false
}
