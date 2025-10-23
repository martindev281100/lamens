package errors

import (
	"fmt"
	"time"
)

// ErrorType represents different categories of errors
type ErrorType string

const (
	// General error types
	ErrorTypeGeneral    ErrorType = "general"
	ErrorTypeValidation ErrorType = "validation"
	ErrorTypeConfig     ErrorType = "config"
	ErrorTypeNetwork    ErrorType = "network"
	ErrorTypeTimeout    ErrorType = "timeout"
	ErrorTypePermission ErrorType = "permission"
	ErrorTypeNotFound   ErrorType = "not_found"
	ErrorTypeConflict   ErrorType = "conflict"

	// Kubernetes specific error types
	ErrorTypeKubernetes ErrorType = "kubernetes"
	ErrorTypeNamespace  ErrorType = "namespace"
	ErrorTypeResource   ErrorType = "resource"

	// ArgoCD specific error types
	ErrorTypeArgoCD      ErrorType = "argocd"
	ErrorTypeProject     ErrorType = "project"
	ErrorTypeApplication ErrorType = "application"

	// Git specific error types
	ErrorTypeGit        ErrorType = "git"
	ErrorTypeRepository ErrorType = "repository"
	ErrorTypeClone      ErrorType = "clone"
	ErrorTypeAuth       ErrorType = "auth"

	// Helm specific error types
	ErrorTypeHelm   ErrorType = "helm"
	ErrorTypeChart  ErrorType = "chart"
	ErrorTypeValues ErrorType = "values"

	// Environment specific error types
	ErrorTypeEnvironment ErrorType = "environment"
)

// ErrorCode represents a specific error code for programmatic handling
type ErrorCode string

const (
	// General error codes
	ErrCodeUnknown          ErrorCode = "UNKNOWN"
	ErrCodeInvalidInput     ErrorCode = "INVALID_INPUT"
	ErrCodeMissingRequired  ErrorCode = "MISSING_REQUIRED"
	ErrCodeInvalidFormat    ErrorCode = "INVALID_FORMAT"
	ErrCodeConnectionFailed ErrorCode = "CONNECTION_FAILED"
	ErrCodeTimeout          ErrorCode = "TIMEOUT"
	ErrCodePermissionDenied ErrorCode = "PERMISSION_DENIED"
	ErrCodeNotFound         ErrorCode = "NOT_FOUND"
	ErrCodeAlreadyExists    ErrorCode = "ALREADY_EXISTS"
	ErrCodeConflict         ErrorCode = "CONFLICT"

	// Kubernetes error codes
	ErrCodeK8sConnection     ErrorCode = "K8S_CONNECTION"
	ErrCodeK8sConfig         ErrorCode = "K8S_CONFIG"
	ErrCodeNamespaceNotFound ErrorCode = "NAMESPACE_NOT_FOUND"
	ErrCodeResourceNotFound  ErrorCode = "RESOURCE_NOT_FOUND"
	ErrCodeResourceExists    ErrorCode = "RESOURCE_EXISTS"

	// ArgoCD error codes
	ErrCodeArgoCDConnection ErrorCode = "ARGOCD_CONNECTION"
	ErrCodeProjectNotFound  ErrorCode = "PROJECT_NOT_FOUND"
	ErrCodeProjectExists    ErrorCode = "PROJECT_EXISTS"
	ErrCodeAppNotFound      ErrorCode = "APP_NOT_FOUND"
	ErrCodeAppExists        ErrorCode = "APP_EXISTS"

	// Git error codes
	ErrCodeGitConnection ErrorCode = "GIT_CONNECTION"
	ErrCodeRepoNotFound  ErrorCode = "REPO_NOT_FOUND"
	ErrCodeAuthFailed    ErrorCode = "AUTH_FAILED"
	ErrCodeCloneFailed   ErrorCode = "CLONE_FAILED"
	ErrCodeInvalidURL    ErrorCode = "INVALID_URL"

	// Helm error codes
	ErrCodeChartNotFound ErrorCode = "CHART_NOT_FOUND"
	ErrCodeInvalidChart  ErrorCode = "INVALID_CHART"
	ErrCodeValuesInvalid ErrorCode = "VALUES_INVALID"

	// Environment error codes
	ErrCodeEnvNotFound ErrorCode = "ENV_NOT_FOUND"
	ErrCodeEnvInvalid  ErrorCode = "ENV_INVALID"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity string

const (
	ErrorSeverityInfo    ErrorSeverity = "info"
	ErrorSeverityWarning ErrorSeverity = "warning"
	ErrorSeverityError   ErrorSeverity = "error"
	ErrorSeverityFatal   ErrorSeverity = "fatal"
)

// AppError is the base error type for the application
type AppError struct {
	Type        ErrorType     `json:"type"`
	Code        ErrorCode     `json:"code"`
	Message     string        `json:"message"`
	Severity    ErrorSeverity `json:"severity"`
	Retryable   bool          `json:"retryable"`
	Timestamp   time.Time     `json:"timestamp"`
	Context     ErrorContext  `json:"context,omitempty"`
	Cause       error         `json:"cause,omitempty"`
	Suggestions []string      `json:"suggestions,omitempty"`
}

// ErrorContext provides additional context for an error
type ErrorContext struct {
	Operation string                 `json:"operation,omitempty"`
	Resource  string                 `json:"resource,omitempty"`
	Namespace string                 `json:"namespace,omitempty"`
	Field     string                 `json:"field,omitempty"`
	Value     string                 `json:"value,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	baseMsg := fmt.Sprintf("[%s] %s (code: %s)", e.Type, e.Message, e.Code)

	if e.Context.Operation != "" {
		baseMsg = fmt.Sprintf("%s [operation: %s]", baseMsg, e.Context.Operation)
	}
	if e.Context.Resource != "" {
		baseMsg = fmt.Sprintf("%s [resource: %s]", baseMsg, e.Context.Resource)
	}
	if e.Context.Namespace != "" {
		baseMsg = fmt.Sprintf("%s [namespace: %s]", baseMsg, e.Context.Namespace)
	}

	if e.Cause != nil {
		baseMsg = fmt.Sprintf("%s: %v", baseMsg, e.Cause)
	}

	return baseMsg
}

// Unwrap returns the underlying cause
func (e *AppError) Unwrap() error {
	return e.Cause
}

// IsRetryable returns whether the error is retryable
func (e *AppError) IsRetryable() bool {
	return e.Retryable
}

// GetSeverity returns the severity of the error
func (e *AppError) GetSeverity() ErrorSeverity {
	return e.Severity
}

// GetSuggestions returns suggestions for resolving the error
func (e *AppError) GetSuggestions() []string {
	return e.Suggestions
}

// NewAppError creates a new application error
func NewAppError(errorType ErrorType, code ErrorCode, message string, severity ErrorSeverity, retryable bool) *AppError {
	return &AppError{
		Type:      errorType,
		Code:      code,
		Message:   message,
		Severity:  severity,
		Retryable: retryable,
		Timestamp: time.Now(),
		Context:   ErrorContext{},
	}
}

// WithCause adds a cause to the error
func (e *AppError) WithCause(cause error) *AppError {
	e.Cause = cause
	return e
}

// WithContext adds context to the error
func (e *AppError) WithContext(context ErrorContext) *AppError {
	e.Context = context
	return e
}

// WithOperation adds operation context to the error
func (e *AppError) WithOperation(operation string) *AppError {
	e.Context.Operation = operation
	return e
}

// WithResource adds resource context to the error
func (e *AppError) WithResource(resource string) *AppError {
	e.Context.Resource = resource
	return e
}

// WithNamespace adds namespace context to the error
func (e *AppError) WithNamespace(namespace string) *AppError {
	e.Context.Namespace = namespace
	return e
}

// WithField adds field context to the error
func (e *AppError) WithField(field, value string) *AppError {
	e.Context.Field = field
	e.Context.Value = value
	return e
}

// WithDetails adds additional details to the error
func (e *AppError) WithDetails(details map[string]interface{}) *AppError {
	if e.Context.Details == nil {
		e.Context.Details = make(map[string]interface{})
	}
	for k, v := range details {
		e.Context.Details[k] = v
	}
	return e
}

// WithSuggestions adds suggestions to the error
func (e *AppError) WithSuggestions(suggestions ...string) *AppError {
	e.Suggestions = suggestions
	return e
}

// IsErrorType checks if an error is of a specific type
func IsErrorType(err error, errorType ErrorType) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type == errorType
	}
	return false
}

// IsErrorCode checks if an error has a specific code
func IsErrorCode(err error, code ErrorCode) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == code
	}
	return false
}

// IsRetryableError checks if an error is retryable
func IsRetryableError(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Retryable
	}
	return false
}

// GetErrorSeverity returns the severity of an error
func GetErrorSeverity(err error) ErrorSeverity {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Severity
	}
	return ErrorSeverityError
}

// GetErrorSuggestions returns suggestions for resolving an error
func GetErrorSuggestions(err error) []string {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Suggestions
	}
	return nil
}

// WrapError wraps an error with additional context
func WrapError(err error, errorType ErrorType, code ErrorCode, message string) *AppError {
	if err == nil {
		return nil
	}

	// If it's already an AppError, add context
	if appErr, ok := err.(*AppError); ok {
		return &AppError{
			Type:        errorType,
			Code:        code,
			Message:     fmt.Sprintf("%s: %s", message, appErr.Message),
			Severity:    appErr.Severity,
			Retryable:   appErr.Retryable,
			Timestamp:   time.Now(),
			Context:     appErr.Context,
			Cause:       appErr,
			Suggestions: appErr.Suggestions,
		}
	}

	// Otherwise, create a new AppError with the original error as cause
	return NewAppError(errorType, code, message, ErrorSeverityError, false).WithCause(err)
}
