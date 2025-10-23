package errors

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// RetryConfig defines the configuration for retry operations
type RetryConfig struct {
	MaxRetries    int           `json:"max_retries"`
	InitialDelay  time.Duration `json:"initial_delay"`
	MaxDelay      time.Duration `json:"max_delay"`
	BackoffFactor float64       `json:"backoff_factor"`
	Jitter        bool          `json:"jitter"`
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
	}
}

// NetworkRetryConfig returns a retry configuration for network operations
func NetworkRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    5,
		InitialDelay:  2 * time.Second,
		MaxDelay:      60 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
	}
}

// KubernetesRetryConfig returns a retry configuration for Kubernetes operations
func KubernetesRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    4,
		InitialDelay:  500 * time.Millisecond,
		MaxDelay:      10 * time.Second,
		BackoffFactor: 1.5,
		Jitter:        true,
	}
}

// SimpleRetryableOperation represents an operation that can be retried
type SimpleRetryableOperation func() error

// RetryableOperationWithContext represents an operation that can be retried with context
type RetryableOperationWithContext func(ctx context.Context) error

// RetryResult represents the result of a retry operation
type RetryResult struct {
	Success   bool          `json:"success"`
	Attempts  int           `json:"attempts"`
	TotalTime time.Duration `json:"total_time"`
	LastError error         `json:"last_error,omitempty"`
	AllErrors []error       `json:"all_errors,omitempty"`
	Retryable bool          `json:"retryable"`
}

// Retry executes an operation with retry logic
func Retry(operation SimpleRetryableOperation, config RetryConfig) *RetryResult {
	return RetryWithContext(context.Background(), func(ctx context.Context) error {
		return operation()
	}, config)
}

// RetryWithContext executes an operation with retry logic and context
func RetryWithContext(ctx context.Context, operation RetryableOperationWithContext, config RetryConfig) *RetryResult {
	result := &RetryResult{
		Success:   false,
		Attempts:  0,
		AllErrors: make([]error, 0),
		Retryable: true,
	}

	startTime := time.Now()
	var lastErr error

	for result.Attempts < config.MaxRetries {
		result.Attempts++

		// Check if context is cancelled
		if ctx.Err() != nil {
			lastErr = ctx.Err()
			result.LastError = lastErr
			result.AllErrors = append(result.AllErrors, lastErr)
			result.Retryable = false
			result.TotalTime = time.Since(startTime)
			return result
		}

		// Execute the operation
		err := operation(ctx)
		if err == nil {
			// Operation succeeded
			result.Success = true
			result.TotalTime = time.Since(startTime)
			return result
		}

		// Store the error
		lastErr = err
		result.LastError = lastErr
		result.AllErrors = append(result.AllErrors, lastErr)

		// Check if the error is retryable
		if !IsRetryableError(err) {
			result.Retryable = false
			result.TotalTime = time.Since(startTime)
			return result
		}

		// If this was the last attempt, don't wait
		if result.Attempts >= config.MaxRetries {
			break
		}

		// Calculate delay for next attempt
		delay := calculateDelay(result.Attempts, config)

		// Wait with context cancellation
		select {
		case <-ctx.Done():
			lastErr = ctx.Err()
			result.LastError = lastErr
			result.AllErrors = append(result.AllErrors, lastErr)
			result.Retryable = false
			result.TotalTime = time.Since(startTime)
			return result
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	result.TotalTime = time.Since(startTime)
	return result
}

// calculateDelay calculates the delay for the next retry attempt
func calculateDelay(attempt int, config RetryConfig) time.Duration {
	// Calculate exponential backoff
	delay := float64(config.InitialDelay) * math.Pow(config.BackoffFactor, float64(attempt-1))

	// Apply maximum delay limit
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}

	// Add jitter if enabled
	if config.Jitter {
		// Add up to 25% random jitter
		jitter := delay * 0.25 * (rand.Float64()*2 - 1)
		delay += jitter
	}

	return time.Duration(delay)
}

// RetryForErrorTypes retries an operation only for specific error types
func RetryForErrorTypes(operation SimpleRetryableOperation, config RetryConfig, retryableTypes ...ErrorType) *RetryResult {
	return RetryWithContext(context.Background(), func(ctx context.Context) error {
		err := operation()
		if err != nil {
			// Check if the error type is in the allowed retryable types
			for _, retryableType := range retryableTypes {
				if IsErrorType(err, retryableType) {
					return err
				}
			}
			// If error type is not in the allowed list, make it non-retryable
			if appErr, ok := err.(*AppError); ok {
				return NewAppError(
					appErr.Type,
					appErr.Code,
					appErr.Message,
					appErr.Severity,
					false, // Make it non-retryable
				).WithContext(appErr.Context).WithCause(appErr.Cause)
			}
		}
		return err
	}, config)
}

// RetryForErrorCodes retries an operation only for specific error codes
func RetryForErrorCodes(operation SimpleRetryableOperation, config RetryConfig, retryableCodes ...ErrorCode) *RetryResult {
	return RetryWithContext(context.Background(), func(ctx context.Context) error {
		err := operation()
		if err != nil {
			// Check if the error code is in the allowed retryable codes
			for _, retryableCode := range retryableCodes {
				if IsErrorCode(err, retryableCode) {
					return err
				}
			}
			// If error code is not in the allowed list, make it non-retryable
			if appErr, ok := err.(*AppError); ok {
				return NewAppError(
					appErr.Type,
					appErr.Code,
					appErr.Message,
					appErr.Severity,
					false, // Make it non-retryable
				).WithContext(appErr.Context).WithCause(appErr.Cause)
			}
		}
		return err
	}, config)
}

// RetryWithProgress executes an operation with retry logic and progress reporting
func RetryWithProgress(operation SimpleRetryableOperation, config RetryConfig, progressFunc func(attempt int, maxAttempts int, err error)) *RetryResult {
	return RetryWithContext(context.Background(), func(ctx context.Context) error {
		err := operation()
		if progressFunc != nil {
			progressFunc(1, config.MaxRetries, err)
		}
		return err
	}, config)
}

// NetworkRetry retries a network operation with standard network retry configuration
func NetworkRetry(operation SimpleRetryableOperation) *RetryResult {
	return Retry(operation, NetworkRetryConfig())
}

// KubernetesRetry retries a Kubernetes operation with standard Kubernetes retry configuration
func KubernetesRetry(operation SimpleRetryableOperation) *RetryResult {
	return Retry(operation, KubernetesRetryConfig())
}

// DoWithRetry is a convenience function that executes an operation with default retry configuration
func DoWithRetry(operation SimpleRetryableOperation) error {
	result := Retry(operation, DefaultRetryConfig())
	if result.Success {
		return nil
	}
	return result.LastError
}

// DoWithRetryConfig is a convenience function that executes an operation with custom retry configuration
func DoWithRetryConfig(operation SimpleRetryableOperation, config RetryConfig) error {
	result := Retry(operation, config)
	if result.Success {
		return nil
	}
	return result.LastError
}

// DoWithRetryAndProgress is a convenience function that executes an operation with retry and progress reporting
func DoWithRetryAndProgress(operation SimpleRetryableOperation, config RetryConfig, progressFunc func(attempt int, maxAttempts int, err error)) error {
	result := RetryWithProgress(operation, config, progressFunc)
	if result.Success {
		return nil
	}
	return result.LastError
}

// RetryError represents an error that occurred during retry operations
type RetryError struct {
	Result *RetryResult
}

// Error implements the error interface
func (re *RetryError) Error() string {
	if re.Result.LastError == nil {
		return "operation failed after retries"
	}
	return fmt.Sprintf("operation failed after %d attempts: %v", re.Result.Attempts, re.Result.LastError)
}

// Unwrap returns the last error
func (re *RetryError) Unwrap() error {
	return re.Result.LastError
}

// IsRetryable returns whether the operation was retryable
func (re *RetryError) IsRetryable() bool {
	return re.Result.Retryable
}

// GetAttempts returns the number of attempts made
func (re *RetryError) GetAttempts() int {
	return re.Result.Attempts
}

// GetAllErrors returns all errors that occurred during retry
func (re *RetryError) GetAllErrors() []error {
	return re.Result.AllErrors
}

// NewRetryError creates a new retry error
func NewRetryError(result *RetryResult) *RetryError {
	return &RetryError{Result: result}
}

// IsRetryError checks if an error is a retry error
func IsRetryError(err error) bool {
	_, ok := err.(*RetryError)
	return ok
}

// GetRetryResult returns the retry result from an error
func GetRetryResult(err error) *RetryResult {
	if retryErr, ok := err.(*RetryError); ok {
		return retryErr.Result
	}
	return nil
}
