package errors

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// ErrorHandler handles errors in a consistent way
type ErrorHandler struct {
	verbose      bool
	outputWriter io.Writer
	errorWriter  io.Writer
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(verbose bool) *ErrorHandler {
	return &ErrorHandler{
		verbose:      verbose,
		outputWriter: os.Stdout,
		errorWriter:  os.Stderr,
	}
}

// NewErrorHandlerWithWriters creates a new error handler with custom writers
func NewErrorHandlerWithWriters(verbose bool, outputWriter, errorWriter io.Writer) *ErrorHandler {
	return &ErrorHandler{
		verbose:      verbose,
		outputWriter: outputWriter,
		errorWriter:  errorWriter,
	}
}

// Handle handles an error according to its type and severity
func (h *ErrorHandler) Handle(err error) error {
	if err == nil {
		return nil
	}

	// Check if it's an AppError
	if appErr, ok := err.(*AppError); ok {
		return h.handleAppError(appErr)
	}

	// Check if it's a RetryError
	if retryErr, ok := err.(*RetryError); ok {
		return h.handleRetryError(retryErr)
	}

	// Handle generic errors
	return h.handleGenericError(err)
}

// handleAppError handles an AppError
func (h *ErrorHandler) handleAppError(err *AppError) error {
	// Format the error message
	message := h.formatAppError(err)

	// Write to error writer
	fmt.Fprintln(h.errorWriter, message)

	// Print suggestions if available and verbose mode is enabled
	if h.verbose && len(err.Suggestions) > 0 {
		fmt.Fprintln(h.errorWriter, "Suggestions:")
		for _, suggestion := range err.Suggestions {
			fmt.Fprintf(h.errorWriter, "  - %s\n", suggestion)
		}
	}

	// Print context details if verbose mode is enabled
	if h.verbose {
		h.printErrorContext(err.Context)
	}

	return err
}

// handleRetryError handles a RetryError
func (h *ErrorHandler) handleRetryError(err *RetryError) error {
	message := fmt.Sprintf("Operation failed after %d attempts", err.Result.Attempts)

	if err.Result.LastError != nil {
		message = fmt.Sprintf("%s: %v", message, err.Result.LastError)
	}

	fmt.Fprintln(h.errorWriter, message)

	// Print all errors if verbose mode is enabled
	if h.verbose && len(err.Result.AllErrors) > 1 {
		fmt.Fprintln(h.errorWriter, "All errors:")
		for i, e := range err.Result.AllErrors {
			fmt.Fprintf(h.errorWriter, "  Attempt %d: %v\n", i+1, e)
		}
	}

	return err
}

// handleGenericError handles a generic error
func (h *ErrorHandler) handleGenericError(err error) error {
	fmt.Fprintf(h.errorWriter, "Error: %v\n", err)
	return err
}

// formatAppError formats an AppError for display
func (h *ErrorHandler) formatAppError(err *AppError) string {
	var builder strings.Builder

	// Add severity prefix
	switch err.Severity {
	case ErrorSeverityInfo:
		builder.WriteString("INFO: ")
	case ErrorSeverityWarning:
		builder.WriteString("WARNING: ")
	case ErrorSeverityError:
		builder.WriteString("ERROR: ")
	case ErrorSeverityFatal:
		builder.WriteString("FATAL: ")
	default:
		builder.WriteString("ERROR: ")
	}

	// Add error type and code
	builder.WriteString(fmt.Sprintf("[%s:%s] ", err.Type, err.Code))

	// Add message
	builder.WriteString(err.Message)

	// Add retryable indicator
	if err.Retryable {
		builder.WriteString(" (retryable)")
	}

	return builder.String()
}

// printErrorContext prints error context details
func (h *ErrorHandler) printErrorContext(context ErrorContext) {
	if context.Operation != "" {
		fmt.Fprintf(h.errorWriter, "Operation: %s\n", context.Operation)
	}
	if context.Resource != "" {
		fmt.Fprintf(h.errorWriter, "Resource: %s\n", context.Resource)
	}
	if context.Namespace != "" {
		fmt.Fprintf(h.errorWriter, "Namespace: %s\n", context.Namespace)
	}
	if context.Field != "" {
		fmt.Fprintf(h.errorWriter, "Field: %s", context.Field)
		if context.Value != "" {
			fmt.Fprintf(h.errorWriter, " (value: %s)", context.Value)
		}
		fmt.Fprintln(h.errorWriter)
	}
	if len(context.Details) > 0 {
		fmt.Fprintln(h.errorWriter, "Details:")
		for key, value := range context.Details {
			fmt.Fprintf(h.errorWriter, "  %s: %v\n", key, value)
		}
	}
}

// HandleWithExit handles an error and exits with the appropriate exit code
func (h *ErrorHandler) HandleWithExit(err error) {
	if err == nil {
		return
	}

	h.Handle(err)
	exitCode := GetExitCode(err)
	os.Exit(exitCode.Int())
}

// HandleValidationResult handles a validation result
func (h *ErrorHandler) HandleValidationResult(result *AppValidationResult) error {
	if result.Valid {
		return nil
	}

	if result.HasErrors() {
		fmt.Fprintln(h.errorWriter, "Validation failed:")
		for _, err := range result.Errors {
			h.Handle(err)
		}
	}

	if result.HasWarnings() && h.verbose {
		fmt.Fprintln(h.errorWriter, "Warnings:")
		for _, warning := range result.Warnings {
			fmt.Fprintf(h.errorWriter, "  - %s\n", warning)
		}
	}

	if result.HasErrors() {
		return result
	}

	return nil
}

// FormatErrorForJSON formats an error for JSON output
func FormatErrorForJSON(err error) map[string]interface{} {
	if err == nil {
		return nil
	}

	if appErr, ok := err.(*AppError); ok {
		return map[string]interface{}{
			"type":        string(appErr.Type),
			"code":        string(appErr.Code),
			"message":     appErr.Message,
			"severity":    string(appErr.Severity),
			"retryable":   appErr.Retryable,
			"timestamp":   appErr.Timestamp,
			"context":     appErr.Context,
			"suggestions": appErr.Suggestions,
		}
	}

	if retryErr, ok := err.(*RetryError); ok {
		return map[string]interface{}{
			"type":     "retry_error",
			"attempts": retryErr.Result.Attempts,
			"success":  retryErr.Result.Success,
			"last_error": func() interface{} {
				if retryErr.Result.LastError != nil {
					return FormatErrorForJSON(retryErr.Result.LastError)
				}
				return nil
			}(),
		}
	}

	return map[string]interface{}{
		"type":    "generic_error",
		"message": err.Error(),
	}
}

// FormatErrorForCLI formats an error for CLI output
func FormatErrorForCLI(err error, verbose bool) string {
	if err == nil {
		return ""
	}

	handler := NewErrorHandler(verbose)

	if appErr, ok := err.(*AppError); ok {
		return handler.formatAppError(appErr)
	}

	if retryErr, ok := err.(*RetryError); ok {
		return fmt.Sprintf("Operation failed after %d attempts: %v", retryErr.Result.Attempts, retryErr.Result.LastError)
	}

	return fmt.Sprintf("Error: %v", err)
}

// ErrorReporter reports errors in different formats
type ErrorReporter struct {
	handler    *ErrorHandler
	format     string
	outputFile string
}

// NewErrorReporter creates a new error reporter
func NewErrorReporter(format string, verbose bool) *ErrorReporter {
	return &ErrorReporter{
		handler: NewErrorHandler(verbose),
		format:  format,
	}
}

// SetOutputFile sets the output file for error reporting
func (r *ErrorReporter) SetOutputFile(filename string) {
	r.outputFile = filename
}

// Report reports an error
func (r *ErrorReporter) Report(err error) error {
	if err == nil {
		return nil
	}

	switch r.format {
	case "json":
		return r.reportJSON(err)
	case "cli":
		return r.reportCLI(err)
	default:
		return r.handler.Handle(err)
	}
}

// reportJSON reports an error in JSON format
func (r *ErrorReporter) reportJSON(err error) error {
	errorData := FormatErrorForJSON(err)

	var output io.Writer = os.Stderr
	if r.outputFile != "" {
		file, err := os.Create(r.outputFile)
		if err != nil {
			return fmt.Errorf("failed to create error report file: %w", err)
		}
		defer file.Close()
		output = file
	}

	// In a real implementation, you would use a JSON encoder here
	fmt.Fprintf(output, "%+v\n", errorData)

	return err
}

// reportCLI reports an error in CLI format
func (r *ErrorReporter) reportCLI(err error) error {
	return r.handler.Handle(err)
}

// MultiErrorHandler handles multiple errors
type MultiErrorHandler struct {
	handlers []*ErrorHandler
}

// NewMultiErrorHandler creates a new multi-error handler
func NewMultiErrorHandler(handlers ...*ErrorHandler) *MultiErrorHandler {
	return &MultiErrorHandler{handlers: handlers}
}

// Handle handles an error with multiple handlers
func (m *MultiErrorHandler) Handle(err error) error {
	if err == nil {
		return nil
	}

	var lastErr error
	for _, handler := range m.handlers {
		lastErr = handler.Handle(err)
	}

	return lastErr
}

// ErrorCollector collects multiple errors
type ErrorCollector struct {
	errors []error
}

// NewErrorCollector creates a new error collector
func NewErrorCollector() *ErrorCollector {
	return &ErrorCollector{
		errors: make([]error, 0),
	}
}

// Add adds an error to the collector
func (ec *ErrorCollector) Add(err error) {
	if err != nil {
		ec.errors = append(ec.errors, err)
	}
}

// HasErrors returns true if there are errors
func (ec *ErrorCollector) HasErrors() bool {
	return len(ec.errors) > 0
}

// GetErrors returns all collected errors
func (ec *ErrorCollector) GetErrors() []error {
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

// HandleWith handles the collected errors with a handler
func (ec *ErrorCollector) HandleWith(handler *ErrorHandler) error {
	if !ec.HasErrors() {
		return nil
	}

	for _, err := range ec.errors {
		handler.Handle(err)
	}

	return ec
}
