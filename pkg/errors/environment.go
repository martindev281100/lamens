package errors

import (
	"fmt"
	"strings"
	"time"

	"github.com/your-org/kargo-bootstrap/pkg/config"
)

// EnvironmentError represents an error that occurred in a specific environment
type EnvironmentError struct {
	Environment string
	Timestamp   time.Time
	Message     string
	Retryable   bool
	Details     map[string]interface{}
}

// Error implements the error interface
func (e *EnvironmentError) Error() string {
	if e.Retryable {
		return fmt.Sprintf("[RETRYABLE] Environment '%s' error at %s: %s",
			e.Environment, e.Timestamp.Format(time.RFC3339), e.Message)
	}
	return fmt.Sprintf("[FATAL] Environment '%s' error at %s: %s",
		e.Environment, e.Timestamp.Format(time.RFC3339), e.Message)
}

// EnvironmentErrorCollector collects errors across multiple environments
type EnvironmentErrorCollector struct {
	environmentErrors map[string][]*EnvironmentError
	globalError       *EnvironmentError
}

// NewEnvironmentErrorCollector creates a new environment error collector
func NewEnvironmentErrorCollector() *EnvironmentErrorCollector {
	return &EnvironmentErrorCollector{
		environmentErrors: make(map[string][]*EnvironmentError),
	}
}

// AddError adds an error for a specific environment
func (c *EnvironmentErrorCollector) AddError(environment, message string, retryable bool, details map[string]interface{}) {
	error := &EnvironmentError{
		Environment: environment,
		Timestamp:   time.Now(),
		Message:     message,
		Retryable:   retryable,
		Details:     details,
	}

	if c.environmentErrors[environment] == nil {
		c.environmentErrors[environment] = make([]*EnvironmentError, 0)
	}

	c.environmentErrors[environment] = append(c.environmentErrors[environment], error)
}

// AddGlobalError adds a global error that affects all environments
func (c *EnvironmentErrorCollector) AddGlobalError(message string, retryable bool, details map[string]interface{}) {
	c.globalError = &EnvironmentError{
		Environment: "GLOBAL",
		Timestamp:   time.Now(),
		Message:     message,
		Retryable:   retryable,
		Details:     details,
	}
}

// HasErrors returns true if there are any errors
func (c *EnvironmentErrorCollector) HasErrors() bool {
	if c.globalError != nil {
		return true
	}

	for _, errors := range c.environmentErrors {
		if len(errors) > 0 {
			return true
		}
	}

	return false
}

// HasRetryableErrors returns true if there are any retryable errors
func (c *EnvironmentErrorCollector) HasRetryableErrors() bool {
	if c.globalError != nil && c.globalError.Retryable {
		return true
	}

	for _, errors := range c.environmentErrors {
		for _, error := range errors {
			if error.Retryable {
				return true
			}
		}
	}

	return false
}

// GetErrorsForEnvironment returns all errors for a specific environment
func (c *EnvironmentErrorCollector) GetErrorsForEnvironment(environment string) []*EnvironmentError {
	return c.environmentErrors[environment]
}

// GetAllErrors returns all errors across all environments
func (c *EnvironmentErrorCollector) GetAllErrors() []*EnvironmentError {
	var allErrors []*EnvironmentError

	if c.globalError != nil {
		allErrors = append(allErrors, c.globalError)
	}

	for _, errors := range c.environmentErrors {
		allErrors = append(allErrors, errors...)
	}

	return allErrors
}

// GetEnvironmentsWithErrors returns a list of environments that have errors
func (c *EnvironmentErrorCollector) GetEnvironmentsWithErrors() []string {
	var environments []string

	for environment, errors := range c.environmentErrors {
		if len(errors) > 0 {
			environments = append(environments, environment)
		}
	}

	return environments
}

// GetRetryableEnvironments returns a list of environments with retryable errors
func (c *EnvironmentErrorCollector) GetRetryableEnvironments() []string {
	var environments []string

	for environment, errors := range c.environmentErrors {
		for _, error := range errors {
			if error.Retryable {
				environments = append(environments, environment)
				break
			}
		}
	}

	return environments
}

// GetFatalEnvironments returns a list of environments with fatal (non-retryable) errors
func (c *EnvironmentErrorCollector) GetFatalEnvironments() []string {
	var environments []string

	for environment, errors := range c.environmentErrors {
		hasFatal := false
		for _, error := range errors {
			if !error.Retryable {
				hasFatal = true
				break
			}
		}
		if hasFatal {
			environments = append(environments, environment)
		}
	}

	return environments
}

// Error implements the error interface
func (c *EnvironmentErrorCollector) Error() string {
	if !c.HasErrors() {
		return "No errors"
	}

	var builder strings.Builder

	if c.globalError != nil {
		builder.WriteString(fmt.Sprintf("Global error: %s\n", c.globalError.Error()))
	}

	if len(c.environmentErrors) > 0 {
		builder.WriteString("Environment errors:\n")
		for environment, errors := range c.environmentErrors {
			if len(errors) > 0 {
				builder.WriteString(fmt.Sprintf("  %s (%d error(s)):\n", environment, len(errors)))
				for _, error := range errors {
					builder.WriteString(fmt.Sprintf("    - %s\n", error.Message))
				}
			}
		}
	}

	return builder.String()
}

// Summary returns a human-readable summary of all errors
func (c *EnvironmentErrorCollector) Summary() string {
	if !c.HasErrors() {
		return "✅ No errors"
	}

	var builder strings.Builder
	totalErrors := len(c.GetAllErrors())

	builder.WriteString(fmt.Sprintf("❌ Found %d error(s):\n", totalErrors))

	if c.globalError != nil {
		builder.WriteString(fmt.Sprintf("  Global: %s\n", c.globalError.Message))
	}

	for environment, errors := range c.environmentErrors {
		if len(errors) > 0 {
			retryableCount := 0
			fatalCount := 0

			for _, error := range errors {
				if error.Retryable {
					retryableCount++
				} else {
					fatalCount++
				}
			}

			builder.WriteString(fmt.Sprintf("  %s: %d error(s) (%d retryable, %d fatal)\n",
				environment, len(errors), retryableCount, fatalCount))
		}
	}

	return builder.String()
}

// ValidationResult represents the result of a validation
type ValidationResult struct {
	Valid    bool
	Errors   []*EnvironmentError
	Warnings []string
}

// ValidateEnvironmentSetup validates a multi-environment setup
func ValidateEnvironmentSetup(envManager *config.EnvironmentManager, environments []string) ValidationResult {
	result := ValidationResult{
		Valid:  true,
		Errors: make([]*EnvironmentError, 0),
	}

	// Validate that all environments exist
	validationResult := envManager.ValidateEnvironmentSetup(environments)
	if !validationResult.Valid {
		result.Valid = false
		for _, errMsg := range validationResult.Errors {
			result.Errors = append(result.Errors, &EnvironmentError{
				Environment: "GLOBAL",
				Timestamp:   time.Now(),
				Message:     errMsg,
				Retryable:   false,
			})
		}
	}

	// Check for warnings
	result.Warnings = validationResult.Warnings

	return result
}

// RetryStrategy provides configuration for retrying failed environments
type RetryStrategy struct {
	MaxRetries    int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
}

// DefaultRetryStrategy returns a default retry strategy
func DefaultRetryStrategy() RetryStrategy {
	return RetryStrategy{
		MaxRetries:    3,
		InitialDelay:  5 * time.Second,
		MaxDelay:      60 * time.Second,
		BackoffFactor: 2.0,
	}
}

// RetryableOperation represents an operation that can be retried
type RetryableOperation func(environment string) error

// RetryEnvironments retries operations for environments with retryable errors
func RetryEnvironments(
	collector *EnvironmentErrorCollector,
	operation RetryableOperation,
	strategy RetryStrategy,
	progress func(environment string, attempt int, maxAttempts int),
) *EnvironmentErrorCollector {
	retryCollector := NewEnvironmentErrorCollector()
	environments := collector.GetRetryableEnvironments()

	for _, environment := range environments {
		errors := collector.GetErrorsForEnvironment(environment)
		if len(errors) == 0 {
			continue
		}

		// Retry the operation
		var lastError error
		for attempt := 1; attempt <= strategy.MaxRetries; attempt++ {
			if progress != nil {
				progress(environment, attempt, strategy.MaxRetries)
			}

			if err := operation(environment); err != nil {
				lastError = err
				if attempt < strategy.MaxRetries {
					// Calculate delay for next attempt
					delay := strategy.InitialDelay
					for i := 1; i < attempt; i++ {
						delay = time.Duration(float64(delay) * strategy.BackoffFactor)
						if delay > strategy.MaxDelay {
							delay = strategy.MaxDelay
						}
					}
					time.Sleep(delay)
				}
			} else {
				// Success, no more retries needed
				lastError = nil
				break
			}
		}

		// If still failing after retries, add to retry collector
		if lastError != nil {
			retryCollector.AddError(environment,
				fmt.Sprintf("Failed after %d retries: %v", strategy.MaxRetries, lastError),
				false,
				map[string]interface{}{
					"max_retries": strategy.MaxRetries,
					"last_error":  lastError.Error(),
				})
		}
	}

	return retryCollector
}

// MergeCollectors merges multiple error collectors into one
func MergeCollectors(collectors ...*EnvironmentErrorCollector) *EnvironmentErrorCollector {
	merged := NewEnvironmentErrorCollector()

	for _, collector := range collectors {
		if collector.globalError != nil {
			merged.globalError = collector.globalError
		}

		for environment, errors := range collector.environmentErrors {
			for _, error := range errors {
				merged.AddError(environment, error.Message, error.Retryable, error.Details)
			}
		}
	}

	return merged
}
