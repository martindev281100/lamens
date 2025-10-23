package prompt

import (
	"fmt"
	"os"

	"github.com/your-org/kargo-bootstrap/pkg/errors"
)

// PromptError represents different types of prompt errors
type PromptError struct {
	Type    string
	Message string
	Cause   error
}

func (e *PromptError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *PromptError) Unwrap() error {
	return e.Cause
}

// Error types
const (
	ErrTypeCancelled      = "CancelledError"
	ErrTypeInterrupted    = "InterruptedError"
	ErrTypeValidation     = "ValidationError"
	ErrTypeNonInteractive = "NonInteractiveError"
	ErrTypeTimeout        = "TimeoutError"
	ErrTypeEOF            = "EOFError"
)

// NewPromptError creates a new prompt error
func NewPromptError(errorType, message string, cause error) *PromptError {
	return &PromptError{
		Type:    errorType,
		Message: message,
		Cause:   cause,
	}
}

// IsCancelled checks if an error is a cancellation error
func IsCancelled(err error) bool {
	if promptErr, ok := err.(*PromptError); ok {
		return promptErr.Type == ErrTypeCancelled || promptErr.Type == ErrTypeInterrupted
	}
	return false
}

// IsNonInteractiveError checks if an error is a non-interactive mode error
func IsNonInteractiveError(err error) bool {
	if promptErr, ok := err.(*PromptError); ok {
		return promptErr.Type == ErrTypeNonInteractive
	}
	return false
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	if promptErr, ok := err.(*PromptError); ok {
		return promptErr.Type == ErrTypeValidation
	}
	return false
}

// ErrorHandler handles errors in prompts
type ErrorHandler struct {
	nonInteractive bool
	verbose        bool
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(nonInteractive, verbose bool) *ErrorHandler {
	return &ErrorHandler{
		nonInteractive: nonInteractive,
		verbose:        verbose,
	}
}

// Handle handles an error according to the error type and configuration
func (h *ErrorHandler) Handle(err error) error {
	if err == nil {
		return nil
	}

	if IsCancelled(err) {
		if h.verbose {
			fmt.Fprintf(os.Stderr, "\nOperation cancelled by user: %v\n", err)
		} else {
			fmt.Fprintln(os.Stderr, "\nOperation cancelled by user")
		}
		os.Exit(1)
	}

	if IsNonInteractiveError(err) {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		if h.nonInteractive {
			fmt.Fprintln(os.Stderr, "This operation requires interactive mode. Please run without the --non-interactive flag.")
		}
		return err
	}

	if IsValidationError(err) {
		fmt.Fprintf(os.Stderr, "Validation error: %v\n", err)
		return err
	}

	// Default error handling
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	return err
}

// HandleWithRetry handles an error with retry logic
func (h *ErrorHandler) HandleWithRetry(err error, retryFunc func() error, maxRetries int) error {
	if err == nil {
		return nil
	}

	if IsCancelled(err) {
		return h.Handle(err)
	}

	if IsValidationError(err) {
		return h.Handle(err)
	}

	// For other errors, retry if possible
	if maxRetries > 0 {
		if h.verbose {
			fmt.Fprintf(os.Stderr, "Retrying after error: %v\n", err)
		}
		return retryFunc()
	}

	return h.Handle(err)
}

// WithErrorHandling wraps a function with error handling
func WithErrorHandling(fn func() error, handler *ErrorHandler) error {
	err := fn()
	if err != nil {
		return handler.Handle(err)
	}
	return nil
}

// WithErrorHandlingAndRetry wraps a function with error handling and retry logic
func WithErrorHandlingAndRetry(fn func() error, handler *ErrorHandler, maxRetries int) error {
	var err error
	for i := 0; i <= maxRetries; i++ {
		err = fn()
		if err == nil {
			return nil
		}

		if IsCancelled(err) || IsValidationError(err) {
			return handler.Handle(err)
		}

		if i < maxRetries {
			if handler.verbose {
				fmt.Fprintf(os.Stderr, "Attempt %d failed: %v\n", i+1, err)
			}
		}
	}

	return handler.HandleWithRetry(err, fn, 0)
}

// NonInteractiveError creates a non-interactive mode error
func NonInteractiveError(message string) *PromptError {
	return NewPromptError(ErrTypeNonInteractive, message, nil)
}

// NewAppNonInteractiveError creates a non-interactive mode error using the centralized error system
func NewAppNonInteractiveError(message string) error {
	return errors.NewAppError(
		errors.ErrorTypeGeneral,
		errors.ErrCodeInvalidInput,
		message,
		errors.ErrorSeverityError,
		false,
	).WithSuggestions(
		"Run without the --non-interactive flag",
		"Provide all required values via command-line flags",
	)
}

// CancelledError creates a cancellation error
func CancelledError(message string) *PromptError {
	return NewPromptError(ErrTypeCancelled, message, nil)
}

// NewAppCancelledError creates a cancellation error using the centralized error system
func NewAppCancelledError(message string) error {
	return errors.NewAppError(
		errors.ErrorTypeGeneral,
		errors.ErrCodeInvalidInput,
		message,
		errors.ErrorSeverityWarning,
		false,
	)
}

// ValidationError creates a validation error
func ValidationError(message string) *PromptError {
	return NewPromptError(ErrTypeValidation, message, nil)
}

// NewAppValidationError creates a validation error using the centralized error system
func NewAppValidationError(message string) error {
	return errors.NewAppError(
		errors.ErrorTypeValidation,
		errors.ErrCodeInvalidInput,
		message,
		errors.ErrorSeverityError,
		false,
	).WithSuggestions(
		"Check if the input is valid",
		"Verify the input meets the required format",
	)
}

// TimeoutError creates a timeout error
func TimeoutError(message string) *PromptError {
	return NewPromptError(ErrTypeTimeout, message, nil)
}

// NewAppTimeoutError creates a timeout error using the centralized error system
func NewAppTimeoutError(message string) error {
	return errors.NewAppError(
		errors.ErrorTypeTimeout,
		errors.ErrCodeTimeout,
		message,
		errors.ErrorSeverityError,
		false,
	).WithSuggestions(
		"Increase the timeout duration",
		"Check if the operation is taking too long",
	)
}

// EOFError creates an EOF error
func EOFError(message string) *PromptError {
	return NewPromptError(ErrTypeEOF, message, nil)
}

// NewAppEOFError creates an EOF error using the centralized error system
func NewAppEOFError(message string) error {
	return errors.NewAppError(
		errors.ErrorTypeGeneral,
		errors.ErrCodeInvalidInput,
		message,
		errors.ErrorSeverityWarning,
		false,
	).WithSuggestions(
		"Provide valid input",
		"Check if the input stream is properly connected",
	)
}
