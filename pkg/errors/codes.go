package errors

import (
	"fmt"
)

// ExitCode represents the exit code for the CLI
type ExitCode int

const (
	// Success exit codes
	ExitCodeSuccess ExitCode = 0

	// General error exit codes (1-10)
	ExitCodeGeneralError    ExitCode = 1
	ExitCodeInvalidInput    ExitCode = 2
	ExitCodeConfigError     ExitCode = 3
	ExitCodeNetworkError    ExitCode = 4
	ExitCodeTimeoutError    ExitCode = 5
	ExitCodePermissionError ExitCode = 6
	ExitCodeNotFoundError   ExitCode = 7
	ExitCodeConflictError   ExitCode = 8

	// Kubernetes error exit codes (11-20)
	ExitCodeKubernetesError ExitCode = 11
	ExitCodeNamespaceError  ExitCode = 12
	ExitCodeResourceError   ExitCode = 13

	// ArgoCD error exit codes (21-30)
	ExitCodeArgoCDError      ExitCode = 21
	ExitCodeProjectError     ExitCode = 22
	ExitCodeApplicationError ExitCode = 23

	// Git error exit codes (31-40)
	ExitCodeGitError        ExitCode = 31
	ExitCodeRepositoryError ExitCode = 32
	ExitCodeCloneError      ExitCode = 33
	ExitCodeAuthError       ExitCode = 34

	// Helm error exit codes (41-50)
	ExitCodeHelmError   ExitCode = 41
	ExitCodeChartError  ExitCode = 42
	ExitCodeValuesError ExitCode = 43

	// Environment error exit codes (51-60)
	ExitCodeEnvironmentError ExitCode = 51
)

// ErrorCodeMapping maps error codes to exit codes
var ErrorCodeMapping = map[ErrorCode]ExitCode{
	// General error codes
	ErrCodeUnknown:          ExitCodeGeneralError,
	ErrCodeInvalidInput:     ExitCodeInvalidInput,
	ErrCodeMissingRequired:  ExitCodeInvalidInput,
	ErrCodeInvalidFormat:    ExitCodeInvalidInput,
	ErrCodeConnectionFailed: ExitCodeNetworkError,
	ErrCodeTimeout:          ExitCodeTimeoutError,
	ErrCodePermissionDenied: ExitCodePermissionError,
	ErrCodeNotFound:         ExitCodeNotFoundError,
	ErrCodeAlreadyExists:    ExitCodeConflictError,
	ErrCodeConflict:         ExitCodeConflictError,

	// Kubernetes error codes
	ErrCodeK8sConnection:     ExitCodeKubernetesError,
	ErrCodeK8sConfig:         ExitCodeConfigError,
	ErrCodeNamespaceNotFound: ExitCodeNamespaceError,
	ErrCodeResourceNotFound:  ExitCodeResourceError,
	ErrCodeResourceExists:    ExitCodeResourceError,

	// ArgoCD error codes
	ErrCodeArgoCDConnection: ExitCodeArgoCDError,
	ErrCodeProjectNotFound:  ExitCodeProjectError,
	ErrCodeProjectExists:    ExitCodeProjectError,
	ErrCodeAppNotFound:      ExitCodeApplicationError,
	ErrCodeAppExists:        ExitCodeApplicationError,

	// Git error codes
	ErrCodeGitConnection: ExitCodeGitError,
	ErrCodeRepoNotFound:  ExitCodeRepositoryError,
	ErrCodeAuthFailed:    ExitCodeAuthError,
	ErrCodeCloneFailed:   ExitCodeCloneError,
	ErrCodeInvalidURL:    ExitCodeInvalidInput,

	// Helm error codes
	ErrCodeChartNotFound: ExitCodeChartError,
	ErrCodeInvalidChart:  ExitCodeChartError,
	ErrCodeValuesInvalid: ExitCodeValuesError,

	// Environment error codes
	ErrCodeEnvNotFound: ExitCodeEnvironmentError,
	ErrCodeEnvInvalid:  ExitCodeEnvironmentError,
}

// GetExitCode returns the appropriate exit code for an error
func GetExitCode(err error) ExitCode {
	if err == nil {
		return ExitCodeSuccess
	}

	if appErr, ok := err.(*AppError); ok {
		if exitCode, exists := ErrorCodeMapping[appErr.Code]; exists {
			return exitCode
		}

		// Fallback to error type mapping
		switch appErr.Type {
		case ErrorTypeGeneral:
			return ExitCodeGeneralError
		case ErrorTypeValidation:
			return ExitCodeInvalidInput
		case ErrorTypeConfig:
			return ExitCodeConfigError
		case ErrorTypeNetwork:
			return ExitCodeNetworkError
		case ErrorTypeTimeout:
			return ExitCodeTimeoutError
		case ErrorTypePermission:
			return ExitCodePermissionError
		case ErrorTypeNotFound:
			return ExitCodeNotFoundError
		case ErrorTypeConflict:
			return ExitCodeConflictError
		case ErrorTypeKubernetes:
			return ExitCodeKubernetesError
		case ErrorTypeNamespace:
			return ExitCodeNamespaceError
		case ErrorTypeResource:
			return ExitCodeResourceError
		case ErrorTypeArgoCD:
			return ExitCodeArgoCDError
		case ErrorTypeProject:
			return ExitCodeProjectError
		case ErrorTypeApplication:
			return ExitCodeApplicationError
		case ErrorTypeGit:
			return ExitCodeGitError
		case ErrorTypeRepository:
			return ExitCodeRepositoryError
		case ErrorTypeClone:
			return ExitCodeCloneError
		case ErrorTypeAuth:
			return ExitCodeAuthError
		case ErrorTypeHelm:
			return ExitCodeHelmError
		case ErrorTypeChart:
			return ExitCodeChartError
		case ErrorTypeValues:
			return ExitCodeValuesError
		case ErrorTypeEnvironment:
			return ExitCodeEnvironmentError
		default:
			return ExitCodeGeneralError
		}
	}

	return ExitCodeGeneralError
}

// GetExitCodeMessage returns a human-readable message for an exit code
func GetExitCodeMessage(code ExitCode) string {
	switch code {
	case ExitCodeSuccess:
		return "Success"
	case ExitCodeGeneralError:
		return "General error occurred"
	case ExitCodeInvalidInput:
		return "Invalid input provided"
	case ExitCodeConfigError:
		return "Configuration error occurred"
	case ExitCodeNetworkError:
		return "Network error occurred"
	case ExitCodeTimeoutError:
		return "Operation timed out"
	case ExitCodePermissionError:
		return "Permission denied"
	case ExitCodeNotFoundError:
		return "Resource not found"
	case ExitCodeConflictError:
		return "Conflict with existing resource"
	case ExitCodeKubernetesError:
		return "Kubernetes error occurred"
	case ExitCodeNamespaceError:
		return "Namespace error occurred"
	case ExitCodeResourceError:
		return "Resource error occurred"
	case ExitCodeArgoCDError:
		return "ArgoCD error occurred"
	case ExitCodeProjectError:
		return "Project error occurred"
	case ExitCodeApplicationError:
		return "Application error occurred"
	case ExitCodeGitError:
		return "Git error occurred"
	case ExitCodeRepositoryError:
		return "Repository error occurred"
	case ExitCodeCloneError:
		return "Clone error occurred"
	case ExitCodeAuthError:
		return "Authentication error occurred"
	case ExitCodeHelmError:
		return "Helm error occurred"
	case ExitCodeChartError:
		return "Chart error occurred"
	case ExitCodeValuesError:
		return "Values error occurred"
	case ExitCodeEnvironmentError:
		return "Environment error occurred"
	default:
		return fmt.Sprintf("Unknown exit code: %d", code)
	}
}

// IsSuccess returns true if the exit code indicates success
func (code ExitCode) IsSuccess() bool {
	return code == ExitCodeSuccess
}

// IsError returns true if the exit code indicates an error
func (code ExitCode) IsError() bool {
	return code != ExitCodeSuccess
}

// String returns the string representation of the exit code
func (code ExitCode) String() string {
	return GetExitCodeMessage(code)
}

// Int returns the integer value of the exit code
func (code ExitCode) Int() int {
	return int(code)
}
