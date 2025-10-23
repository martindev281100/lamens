package yaml

import (
	"fmt"
	"strings"

	"github.com/your-org/kargo-bootstrap/pkg/errors"
)

// ErrorType represents different types of errors that can occur during YAML processing
type ErrorType string

const (
	ErrorTypeValidation    ErrorType = "validation"
	ErrorTypeRendering     ErrorType = "rendering"
	ErrorTypeTemplate      ErrorType = "template"
	ErrorTypeMarshaling    ErrorType = "marshaling"
	ErrorTypeConfiguration ErrorType = "configuration"
)

// YamlError represents a structured error for YAML processing
type YamlError struct {
	Type    ErrorType `json:"type"`
	Message string    `json:"message"`
	Field   string    `json:"field,omitempty"`
	Value   string    `json:"value,omitempty"`
	Cause   error     `json:"cause,omitempty"`
}

// Error implements the error interface
func (e *YamlError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s error in field '%s': %s", e.Type, e.Field, e.Message)
	}
	return fmt.Sprintf("%s error: %s", e.Type, e.Message)
}

// Unwrap returns the underlying cause
func (e *YamlError) Unwrap() error {
	return e.Cause
}

// NewValidationError creates a new validation error
func NewValidationError(message, field, value string) *YamlError {
	return &YamlError{
		Type:    ErrorTypeValidation,
		Message: message,
		Field:   field,
		Value:   value,
	}
}

// NewAppValidationError creates a new validation error using the centralized error system
func NewAppValidationError(message, field, value string) error {
	return errors.NewAppError(
		errors.ErrorTypeValidation,
		errors.ErrCodeInvalidFormat,
		message,
		errors.ErrorSeverityError,
		false,
	).WithField(field, value).WithResource(field)
}

// NewRenderingError creates a new rendering error
func NewRenderingError(message string, cause error) *YamlError {
	return &YamlError{
		Type:    ErrorTypeRendering,
		Message: message,
		Cause:   cause,
	}
}

// NewAppRenderingError creates a new rendering error using the centralized error system
func NewAppRenderingError(message string, cause error) error {
	return errors.NewAppError(
		errors.ErrorTypeValues,
		errors.ErrCodeValuesInvalid,
		message,
		errors.ErrorSeverityError,
		false,
	).WithCause(cause)
}

// NewTemplateError creates a new template error
func NewTemplateError(message string, cause error) *YamlError {
	return &YamlError{
		Type:    ErrorTypeTemplate,
		Message: message,
		Cause:   cause,
	}
}

// NewAppTemplateError creates a new template error using the centralized error system
func NewAppTemplateError(message string, cause error) error {
	return errors.NewAppError(
		errors.ErrorTypeValues,
		errors.ErrCodeValuesInvalid,
		message,
		errors.ErrorSeverityError,
		false,
	).WithCause(cause)
}

// NewMarshalingError creates a new marshaling error
func NewMarshalingError(message string, cause error) *YamlError {
	return &YamlError{
		Type:    ErrorTypeMarshaling,
		Message: message,
		Cause:   cause,
	}
}

// NewAppMarshalingError creates a new marshaling error using the centralized error system
func NewAppMarshalingError(message string, cause error) error {
	return errors.NewAppError(
		errors.ErrorTypeValues,
		errors.ErrCodeValuesInvalid,
		message,
		errors.ErrorSeverityError,
		false,
	).WithCause(cause)
}

// NewConfigurationError creates a new configuration error
func NewConfigurationError(message, field string, cause error) *YamlError {
	return &YamlError{
		Type:    ErrorTypeConfiguration,
		Message: message,
		Field:   field,
		Cause:   cause,
	}
}

// NewAppConfigurationError creates a new configuration error using the centralized error system
func NewAppConfigurationError(message, field string, cause error) error {
	return errors.NewAppError(
		errors.ErrorTypeConfig,
		errors.ErrCodeInvalidInput,
		message,
		errors.ErrorSeverityError,
		false,
	).WithCause(cause).WithField(field, "")
}

// ErrorCollector collects multiple errors during processing
type ErrorCollector struct {
	errors []*YamlError
}

// NewErrorCollector creates a new error collector
func NewErrorCollector() *ErrorCollector {
	return &ErrorCollector{
		errors: make([]*YamlError, 0),
	}
}

// Add adds an error to the collector
func (ec *ErrorCollector) Add(err *YamlError) {
	ec.errors = append(ec.errors, err)
}

// AddError adds a generic error to the collector
func (ec *ErrorCollector) AddError(err error) {
	if yamlErr, ok := err.(*YamlError); ok {
		ec.errors = append(ec.errors, yamlErr)
	} else {
		ec.errors = append(ec.errors, &YamlError{
			Type:    ErrorTypeConfiguration,
			Message: err.Error(),
			Cause:   err,
		})
	}
}

// HasErrors returns true if there are any errors
func (ec *ErrorCollector) HasErrors() bool {
	return len(ec.errors) > 0
}

// GetErrors returns all collected errors
func (ec *ErrorCollector) GetErrors() []*YamlError {
	return ec.errors
}

// Error implements the error interface
func (ec *ErrorCollector) Error() string {
	if !ec.HasErrors() {
		return ""
	}

	var messages []string
	for _, err := range ec.errors {
		messages = append(messages, err.Error())
	}

	if len(messages) == 1 {
		return messages[0]
	}

	return fmt.Sprintf("multiple errors occurred:\n  - %s", strings.Join(messages, "\n  - "))
}

// ToError returns nil if no errors, otherwise returns the collector as an error
func (ec *ErrorCollector) ToError() error {
	if !ec.HasErrors() {
		return nil
	}
	return ec
}

// FilterErrorsByType returns errors of a specific type
func (ec *ErrorCollector) FilterErrorsByType(errorType ErrorType) []*YamlError {
	var filtered []*YamlError
	for _, err := range ec.errors {
		if err.Type == errorType {
			filtered = append(filtered, err)
		}
	}
	return filtered
}

// GetValidationErrors returns all validation errors
func (ec *ErrorCollector) GetValidationErrors() []*YamlError {
	return ec.FilterErrorsByType(ErrorTypeValidation)
}

// GetRenderingErrors returns all rendering errors
func (ec *ErrorCollector) GetRenderingErrors() []*YamlError {
	return ec.FilterErrorsByType(ErrorTypeRendering)
}

// GetTemplateErrors returns all template errors
func (ec *ErrorCollector) GetTemplateErrors() []*YamlError {
	return ec.FilterErrorsByType(ErrorTypeTemplate)
}

// ErrorSummary provides a human-readable summary of all errors
func (ec *ErrorCollector) ErrorSummary() string {
	if !ec.HasErrors() {
		return "No errors"
	}

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Found %d error(s):\n", len(ec.errors)))

	// Group errors by type
	errorGroups := make(map[ErrorType][]*YamlError)
	for _, err := range ec.errors {
		errorGroups[err.Type] = append(errorGroups[err.Type], err)
	}

	// Print errors by type
	for _, errorType := range []ErrorType{
		ErrorTypeValidation,
		ErrorTypeRendering,
		ErrorTypeTemplate,
		ErrorTypeMarshaling,
		ErrorTypeConfiguration,
	} {
		if errors, exists := errorGroups[errorType]; exists {
			summary.WriteString(fmt.Sprintf("\n%s (%d):\n", string(errorType), len(errors)))
			for _, err := range errors {
				summary.WriteString(fmt.Sprintf("  - %s\n", err.Message))
				if err.Field != "" {
					summary.WriteString(fmt.Sprintf("    Field: %s", err.Field))
					if err.Value != "" {
						summary.WriteString(fmt.Sprintf(" (value: %s)", err.Value))
					}
					summary.WriteString("\n")
				}
			}
		}
	}

	return summary.String()
}

// ValidateRequiredField validates that a required field is not empty
func ValidateRequiredField(value, fieldName string) error {
	if strings.TrimSpace(value) == "" {
		return NewAppValidationError(
			fmt.Sprintf("field '%s' is required", fieldName),
			fieldName,
			value,
		)
	}
	return nil
}

// ValidateURL validates a URL format (basic validation)
func ValidateURL(url, fieldName string) error {
	if strings.TrimSpace(url) == "" {
		return NewAppValidationError(
			fmt.Sprintf("field '%s' cannot be empty", fieldName),
			fieldName,
			url,
		)
	}

	// Basic URL validation - should contain a protocol
	if !strings.Contains(url, "://") {
		return NewAppValidationError(
			fmt.Sprintf("field '%s' must be a valid URL with protocol (e.g., https://)", fieldName),
			fieldName,
			url,
		)
	}

	return nil
}

// ValidateKubernetesName validates a Kubernetes resource name
func ValidateKubernetesName(name, fieldName string) error {
	if strings.TrimSpace(name) == "" {
		return NewAppValidationError(
			fmt.Sprintf("field '%s' cannot be empty", fieldName),
			fieldName,
			name,
		)
	}

	// Basic Kubernetes name validation
	if len(name) > 63 {
		return NewAppValidationError(
			fmt.Sprintf("field '%s' cannot be longer than 63 characters", fieldName),
			fieldName,
			name,
		)
	}

	// Check for invalid characters
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '.') {
			return NewAppValidationError(
				fmt.Sprintf("field '%s' contains invalid character '%c'. Only lowercase letters, numbers, hyphens, and dots are allowed", fieldName, char),
				fieldName,
				name,
			)
		}
	}

	return nil
}

// WrapError wraps an error with additional context
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}

	if yamlErr, ok := err.(*YamlError); ok {
		// Add context to the existing error
		return &YamlError{
			Type:    yamlErr.Type,
			Message: fmt.Sprintf("%s: %s", message, yamlErr.Message),
			Field:   yamlErr.Field,
			Value:   yamlErr.Value,
			Cause:   yamlErr.Cause,
		}
	}

	return &YamlError{
		Type:    ErrorTypeConfiguration,
		Message: fmt.Sprintf("%s: %s", message, err.Error()),
		Cause:   err,
	}
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	if yamlErr, ok := err.(*YamlError); ok {
		return yamlErr.Type == ErrorTypeValidation
	}
	return false
}

// IsRenderingError checks if an error is a rendering error
func IsRenderingError(err error) bool {
	if yamlErr, ok := err.(*YamlError); ok {
		return yamlErr.Type == ErrorTypeRendering
	}
	return false
}

// IsTemplateError checks if an error is a template error
func IsTemplateError(err error) bool {
	if yamlErr, ok := err.(*YamlError); ok {
		return yamlErr.Type == ErrorTypeTemplate
	}
	return false
}
