package errors

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// AppValidationResult represents the result of a validation operation
type AppValidationResult struct {
	Valid    bool        `json:"valid"`
	Errors   []*AppError `json:"errors,omitempty"`
	Warnings []string    `json:"warnings,omitempty"`
}

// AddError adds an error to the validation result
func (vr *AppValidationResult) AddError(err *AppError) {
	vr.Valid = false
	vr.Errors = append(vr.Errors, err)
}

// AddWarning adds a warning to the validation result
func (vr *AppValidationResult) AddWarning(warning string) {
	vr.Warnings = append(vr.Warnings, warning)
}

// HasErrors returns true if there are validation errors
func (vr *AppValidationResult) HasErrors() bool {
	return len(vr.Errors) > 0
}

// HasWarnings returns true if there are validation warnings
func (vr *AppValidationResult) HasWarnings() bool {
	return len(vr.Warnings) > 0
}

// Error implements the error interface
func (vr *AppValidationResult) Error() string {
	if !vr.HasErrors() {
		return ""
	}

	var messages []string
	for _, err := range vr.Errors {
		messages = append(messages, err.Error())
	}

	if len(messages) == 1 {
		return messages[0]
	}

	return fmt.Sprintf("validation failed with %d error(s):\n  - %s", len(messages), strings.Join(messages, "\n  - "))
}

// NewAppValidationResult creates a new validation result
func NewAppValidationResult() *AppValidationResult {
	return &AppValidationResult{
		Valid:    true,
		Errors:   make([]*AppError, 0),
		Warnings: make([]string, 0),
	}
}

// ValidateRequired validates that a field is not empty
func ValidateRequired(value, fieldName string) *AppError {
	if strings.TrimSpace(value) == "" {
		return NewAppError(
			ErrorTypeValidation,
			ErrCodeMissingRequired,
			fmt.Sprintf("field '%s' is required", fieldName),
			ErrorSeverityError,
			false,
		).WithField(fieldName, value).WithSuggestions(
			fmt.Sprintf("Provide a value for '%s'", fieldName),
		)
	}
	return nil
}

// ValidateKubernetesName validates a Kubernetes resource name
func ValidateKubernetesName(name, fieldName string) *AppError {
	if strings.TrimSpace(name) == "" {
		return NewAppError(
			ErrorTypeValidation,
			ErrCodeMissingRequired,
			fmt.Sprintf("field '%s' cannot be empty", fieldName),
			ErrorSeverityError,
			false,
		).WithField(fieldName, name).WithSuggestions(
			fmt.Sprintf("Provide a valid Kubernetes name for '%s'", fieldName),
		)
	}

	// Kubernetes name validation: max 63 characters, lowercase letters, numbers, hyphens, and dots
	if len(name) > 63 {
		return NewAppError(
			ErrorTypeValidation,
			ErrCodeInvalidFormat,
			fmt.Sprintf("field '%s' cannot be longer than 63 characters", fieldName),
			ErrorSeverityError,
			false,
		).WithField(fieldName, name).WithSuggestions(
			fmt.Sprintf("Shorten '%s' to 63 characters or less", fieldName),
		)
	}

	// Check for valid characters
	validNameRegex := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`)
	if !validNameRegex.MatchString(name) {
		return NewAppError(
			ErrorTypeValidation,
			ErrCodeInvalidFormat,
			fmt.Sprintf("field '%s' contains invalid characters. Only lowercase letters, numbers, hyphens, and dots are allowed, and must start and end with a letter or number", fieldName),
			ErrorSeverityError,
			false,
		).WithField(fieldName, name).WithSuggestions(
			fmt.Sprintf("Ensure '%s' follows Kubernetes naming conventions", fieldName),
			"Use only lowercase letters, numbers, hyphens, and dots",
			"Start and end with a letter or number",
		)
	}

	return nil
}

// ValidateNamespace validates a Kubernetes namespace name
func ValidateNamespace(namespace string) *AppError {
	return ValidateKubernetesName(namespace, "namespace")
}

// ValidateApplicationName validates an application name
func ValidateApplicationName(name string) *AppError {
	return ValidateKubernetesName(name, "application")
}

// ValidateURL validates a URL format
func ValidateURL(rawURL, fieldName string) *AppError {
	if strings.TrimSpace(rawURL) == "" {
		return NewAppError(
			ErrorTypeValidation,
			ErrCodeMissingRequired,
			fmt.Sprintf("field '%s' cannot be empty", fieldName),
			ErrorSeverityError,
			false,
		).WithField(fieldName, rawURL).WithSuggestions(
			fmt.Sprintf("Provide a valid URL for '%s'", fieldName),
		)
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return NewAppError(
			ErrorTypeValidation,
			ErrCodeInvalidFormat,
			fmt.Sprintf("field '%s' is not a valid URL: %v", fieldName, err),
			ErrorSeverityError,
			false,
		).WithField(fieldName, rawURL).WithCause(err).WithSuggestions(
			fmt.Sprintf("Ensure '%s' is a valid URL format", fieldName),
			"Include protocol (http:// or https://)",
		)
	}

	if parsedURL.Scheme == "" {
		return NewAppError(
			ErrorTypeValidation,
			ErrCodeInvalidFormat,
			fmt.Sprintf("field '%s' must include a protocol (http:// or https://)", fieldName),
			ErrorSeverityError,
			false,
		).WithField(fieldName, rawURL).WithSuggestions(
			fmt.Sprintf("Add protocol to '%s' (e.g., https://example.com)", fieldName),
		)
	}

	if parsedURL.Host == "" {
		return NewAppError(
			ErrorTypeValidation,
			ErrCodeInvalidFormat,
			fmt.Sprintf("field '%s' must include a host", fieldName),
			ErrorSeverityError,
			false,
		).WithField(fieldName, rawURL).WithSuggestions(
			fmt.Sprintf("Add host to '%s' (e.g., https://example.com/repo.git)", fieldName),
		)
	}

	return nil
}

// ValidateGitURL validates a Git repository URL
func ValidateGitURL(repoURL string) *AppError {
	err := ValidateURL(repoURL, "repository URL")
	if err != nil {
		return NewAppError(
			ErrorTypeGit,
			ErrCodeInvalidURL,
			err.Message,
			err.Severity,
			err.Retryable,
		).WithContext(err.Context).WithCause(err.Cause)
	}

	// Additional Git-specific validation
	parsedURL, _ := url.Parse(repoURL)

	// Check if it's a Git repository (ends with .git or looks like a Git URL)
	if !strings.HasSuffix(parsedURL.Path, ".git") &&
		!strings.Contains(parsedURL.Host, "github.com") &&
		!strings.Contains(parsedURL.Host, "gitlab.com") &&
		!strings.Contains(parsedURL.Host, "bitbucket.org") {
		return NewAppError(
			ErrorTypeGit,
			ErrCodeInvalidURL,
			"URL does not appear to be a Git repository",
			ErrorSeverityWarning,
			false,
		).WithField("repository URL", repoURL).WithSuggestions(
			"Ensure the URL points to a Git repository",
			"Git URLs typically end with .git",
			"Common Git hosts: github.com, gitlab.com, bitbucket.org",
		)
	}

	return nil
}

// ValidateArgoCDProjectName validates an Argo CD project name
func ValidateArgoCDProjectName(name string) *AppError {
	if strings.TrimSpace(name) == "" {
		return NewAppError(
			ErrorTypeValidation,
			ErrCodeMissingRequired,
			"ArgoCD project name cannot be empty",
			ErrorSeverityError,
			false,
		).WithField("project", name).WithSuggestions(
			"Provide a valid ArgoCD project name",
		)
	}

	// ArgoCD project name validation: max 63 characters, lowercase letters, numbers, hyphens
	if len(name) > 63 {
		return NewAppError(
			ErrorTypeValidation,
			ErrCodeInvalidFormat,
			"ArgoCD project name cannot be longer than 63 characters",
			ErrorSeverityError,
			false,
		).WithField("project", name).WithSuggestions(
			"Shorten the project name to 63 characters or less",
		)
	}

	// Check for valid characters
	validProjectRegex := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	if !validProjectRegex.MatchString(name) {
		return NewAppError(
			ErrorTypeValidation,
			ErrCodeInvalidFormat,
			"ArgoCD project name contains invalid characters. Only lowercase letters, numbers, and hyphens are allowed, and must start and end with a letter or number",
			ErrorSeverityError,
			false,
		).WithField("project", name).WithSuggestions(
			"Use only lowercase letters, numbers, and hyphens",
			"Start and end with a letter or number",
		)
	}

	return nil
}

// ValidateHelmChartName validates a Helm chart name
func ValidateHelmChartName(name string) *AppError {
	if strings.TrimSpace(name) == "" {
		return NewAppError(
			ErrorTypeValidation,
			ErrCodeMissingRequired,
			"Helm chart name cannot be empty",
			ErrorSeverityError,
			false,
		).WithField("chart", name).WithSuggestions(
			"Provide a valid Helm chart name",
		)
	}

	// Helm chart name validation: max 63 characters, lowercase letters, numbers, hyphens
	if len(name) > 63 {
		return NewAppError(
			ErrorTypeValidation,
			ErrCodeInvalidFormat,
			"Helm chart name cannot be longer than 63 characters",
			ErrorSeverityError,
			false,
		).WithField("chart", name).WithSuggestions(
			"Shorten the chart name to 63 characters or less",
		)
	}

	// Check for valid characters
	validChartRegex := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	if !validChartRegex.MatchString(name) {
		return NewAppError(
			ErrorTypeValidation,
			ErrCodeInvalidFormat,
			"Helm chart name contains invalid characters. Only lowercase letters, numbers, and hyphens are allowed, and must start and end with a letter or number",
			ErrorSeverityError,
			false,
		).WithField("chart", name).WithSuggestions(
			"Use only lowercase letters, numbers, and hyphens",
			"Start and end with a letter or number",
		)
	}

	return nil
}

// ValidateHelmChartVersion validates a Helm chart version
func ValidateHelmChartVersion(version string) *AppError {
	if strings.TrimSpace(version) == "" {
		return NewAppError(
			ErrorTypeValidation,
			ErrCodeMissingRequired,
			"Helm chart version cannot be empty",
			ErrorSeverityError,
			false,
		).WithField("version", version).WithSuggestions(
			"Provide a valid Helm chart version",
		)
	}

	// Basic semantic version validation
	versionRegex := regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+([a-zA-Z0-9\-\.\+]*)?$`)
	if !versionRegex.MatchString(version) {
		return NewAppError(
			ErrorTypeValidation,
			ErrCodeInvalidFormat,
			"Helm chart version must follow semantic versioning (e.g., 1.0.0, v1.0.0, 1.0.0-alpha)",
			ErrorSeverityWarning,
			false,
		).WithField("version", version).WithSuggestions(
			"Use semantic versioning (e.g., 1.0.0, v1.0.0)",
			"Include pre-release tags if needed (e.g., 1.0.0-alpha)",
		)
	}

	return nil
}

// ValidateEnvironmentName validates an environment name
func ValidateEnvironmentName(name string) *AppError {
	if strings.TrimSpace(name) == "" {
		return NewAppError(
			ErrorTypeValidation,
			ErrCodeMissingRequired,
			"Environment name cannot be empty",
			ErrorSeverityError,
			false,
		).WithField("environment", name).WithSuggestions(
			"Provide a valid environment name",
		)
	}

	// Environment name validation: max 20 characters, lowercase letters, numbers, hyphens
	if len(name) > 20 {
		return NewAppError(
			ErrorTypeValidation,
			ErrCodeInvalidFormat,
			"Environment name cannot be longer than 20 characters",
			ErrorSeverityError,
			false,
		).WithField("environment", name).WithSuggestions(
			"Shorten the environment name to 20 characters or less",
		)
	}

	// Check for valid characters
	validEnvRegex := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	if !validEnvRegex.MatchString(name) {
		return NewAppError(
			ErrorTypeValidation,
			ErrCodeInvalidFormat,
			"Environment name contains invalid characters. Only lowercase letters, numbers, and hyphens are allowed, and must start and end with a letter or number",
			ErrorSeverityError,
			false,
		).WithField("environment", name).WithSuggestions(
			"Use only lowercase letters, numbers, and hyphens",
			"Start and end with a letter or number",
		)
	}

	return nil
}

// ValidateKubernetesConfiguration validates Kubernetes configuration
func ValidateKubernetesConfiguration(kubeconfigPath, context string) *AppValidationResult {
	result := NewAppValidationResult()

	// Validate kubeconfig path if provided
	if kubeconfigPath != "" {
		// This is a basic validation - in a real implementation, you would check if the file exists and is valid
		if !strings.Contains(kubeconfigPath, ".yaml") && !strings.Contains(kubeconfigPath, ".yml") {
			result.AddError(NewAppError(
				ErrorTypeKubernetes,
				ErrCodeK8sConfig,
				"kubeconfig file must be a YAML file",
				ErrorSeverityError,
				false,
			).WithField("kubeconfig", kubeconfigPath).WithSuggestions(
				"Ensure kubeconfig file has .yaml or .yml extension",
				"Check that the kubeconfig file exists and is valid",
			))
		}
	}

	// Validate context name if provided
	if context != "" {
		err := ValidateKubernetesName(context, "context")
		if err != nil {
			result.AddError(NewAppError(
				ErrorTypeKubernetes,
				ErrCodeK8sConfig,
				err.Message,
				err.Severity,
				err.Retryable,
			).WithContext(err.Context).WithCause(err.Cause))
		}
	}

	return result
}

// ValidateArgoCDConfiguration validates Argo CD configuration
func ValidateArgoCDConfiguration(namespace, serverURL string) *AppValidationResult {
	result := NewAppValidationResult()

	// Validate namespace
	if namespace == "" {
		result.AddError(NewAppError(
			ErrorTypeArgoCD,
			ErrCodeMissingRequired,
			"ArgoCD namespace is required",
			ErrorSeverityError,
			false,
		).WithField("namespace", namespace).WithSuggestions(
			"Provide the namespace where ArgoCD is installed",
			"Default is usually 'argocd'",
		))
	} else {
		err := ValidateNamespace(namespace)
		if err != nil {
			result.AddError(NewAppError(
				ErrorTypeArgoCD,
				err.Code,
				err.Message,
				err.Severity,
				err.Retryable,
			).WithContext(err.Context).WithCause(err.Cause))
		}
	}

	// Validate server URL if provided
	if serverURL != "" {
		err := ValidateURL(serverURL, "ArgoCD server URL")
		if err != nil {
			result.AddError(NewAppError(
				ErrorTypeArgoCD,
				ErrCodeArgoCDConnection,
				err.Message,
				err.Severity,
				err.Retryable,
			).WithContext(err.Context).WithCause(err.Cause))
		}
	}

	return result
}

// ValidateGitRepository validates Git repository configuration
func ValidateGitRepository(repoURL, branch, path string) *AppValidationResult {
	result := NewAppValidationResult()

	// Validate repository URL
	if repoURL == "" {
		result.AddError(NewAppError(
			ErrorTypeGit,
			ErrCodeMissingRequired,
			"Git repository URL is required",
			ErrorSeverityError,
			false,
		).WithField("repository URL", repoURL).WithSuggestions(
			"Provide a valid Git repository URL",
		))
	} else {
		err := ValidateGitURL(repoURL)
		if err != nil {
			result.AddError(err)
		}
	}

	// Validate branch if provided
	if branch != "" {
		// Basic branch name validation
		if len(branch) > 255 {
			result.AddError(NewAppError(
				ErrorTypeGit,
				ErrCodeInvalidFormat,
				"Git branch name cannot be longer than 255 characters",
				ErrorSeverityError,
				false,
			).WithField("branch", branch).WithSuggestions(
				"Shorten the branch name to 255 characters or less",
			))
		}

		// Check for invalid characters
		invalidBranchChars := regexp.MustCompile(`[^\w\-.\/]`)
		if invalidBranchChars.MatchString(branch) {
			result.AddError(NewAppError(
				ErrorTypeGit,
				ErrCodeInvalidFormat,
				"Git branch name contains invalid characters",
				ErrorSeverityWarning,
				false,
			).WithField("branch", branch).WithSuggestions(
				"Use only letters, numbers, hyphens, underscores, dots, and forward slashes",
			))
		}
	}

	// Validate path if provided
	if path != "" {
		// Basic path validation
		if strings.Contains(path, "..") {
			result.AddError(NewAppError(
				ErrorTypeGit,
				ErrCodeInvalidFormat,
				"Git path cannot contain '..' for security reasons",
				ErrorSeverityError,
				false,
			).WithField("path", path).WithSuggestions(
				"Use a valid path within the repository",
			))
		}
	}

	return result
}

// ValidateHelmChart validates Helm chart configuration
func ValidateHelmChart(chartName, chartVersion, repoURL string) *AppValidationResult {
	result := NewAppValidationResult()

	// Validate chart name
	if chartName == "" {
		result.AddError(NewAppError(
			ErrorTypeHelm,
			ErrCodeMissingRequired,
			"Helm chart name is required",
			ErrorSeverityError,
			false,
		).WithField("chart", chartName).WithSuggestions(
			"Provide a valid Helm chart name",
		))
	} else {
		err := ValidateHelmChartName(chartName)
		if err != nil {
			result.AddError(err)
		}
	}

	// Validate chart version if provided
	if chartVersion != "" {
		err := ValidateHelmChartVersion(chartVersion)
		if err != nil {
			result.AddError(err)
		}
	}

	// Validate repository URL if provided
	if repoURL != "" {
		err := ValidateURL(repoURL, "Helm repository URL")
		if err != nil {
			result.AddError(NewAppError(
				ErrorTypeHelm,
				err.Code,
				err.Message,
				err.Severity,
				err.Retryable,
			).WithContext(err.Context).WithCause(err.Cause))
		}
	}

	return result
}
