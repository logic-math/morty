// Package doing provides job execution functionality with error handling and retry mechanisms.
package doing

import (
	"context"
	"fmt"
	"time"
)

// RetryConfig configures retry behavior.
type RetryConfig struct {
	MaxRetries  int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Multiplier  float64
	RetryableFn func(error) bool
}

// DefaultRetryConfig returns the default retry configuration.
// Task 4: Implement retry logic (max 3 times)
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:  3,
		BaseDelay:   1 * time.Second,
		MaxDelay:    30 * time.Second,
		Multiplier:  2.0,
		RetryableFn: IsRetryableError,
	}
}

// RetryResult represents the result of a retry operation.
type RetryResult struct {
	Attempts   int
	LastError  error
	Success    bool
	Duration   time.Duration
}

// RetryableFunc is a function that can be retried.
type RetryableFunc func(ctx context.Context) error

// Retry executes a function with retry logic.
// Task 4: Implement retry logic (max 3 times)
func Retry(ctx context.Context, config *RetryConfig, operation RetryableFunc) *RetryResult {
	if config == nil {
		config = DefaultRetryConfig()
	}

	result := &RetryResult{
		Attempts: 0,
		Success:  false,
	}

	startTime := time.Now()
	defer func() {
		result.Duration = time.Since(startTime)
	}()

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		result.Attempts = attempt + 1

		// Check context cancellation
		select {
		case <-ctx.Done():
			result.LastError = fmt.Errorf("context cancelled: %w", ctx.Err())
			return result
		default:
		}

		// Execute the operation
		err := operation(ctx)

		// Success!
		if err == nil {
			result.Success = true
			result.LastError = nil
			return result
		}

		result.LastError = err

		// Check if we should retry
		if attempt >= config.MaxRetries {
			// No more retries
			break
		}

		if config.RetryableFn != nil && !config.RetryableFn(err) {
			// Error is not retryable
			break
		}

		// Calculate delay with exponential backoff
		delay := calculateBackoff(attempt, config)

		// Wait before retry
		select {
		case <-ctx.Done():
			result.LastError = fmt.Errorf("context cancelled during retry delay: %w", ctx.Err())
			return result
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return result
}

// calculateBackoff calculates the delay for a retry attempt using exponential backoff.
func calculateBackoff(attempt int, config *RetryConfig) time.Duration {
	delay := config.BaseDelay

	// Exponential backoff: delay * multiplier^attempt
	for i := 0; i < attempt; i++ {
		delay = time.Duration(float64(delay) * config.Multiplier)
		if delay >= config.MaxDelay {
			delay = config.MaxDelay
			break
		}
	}

	return delay
}

// RetryWithErrorHandler executes a function with retry and custom error handling.
func RetryWithErrorHandler(
	ctx context.Context,
	config *RetryConfig,
	onAttempt func(attempt int, err error),
	onRetry func(attempt int, delay time.Duration),
	operation RetryableFunc,
) *RetryResult {
	if config == nil {
		config = DefaultRetryConfig()
	}

	result := &RetryResult{
		Attempts: 0,
		Success:  false,
	}

	startTime := time.Now()
	defer func() {
		result.Duration = time.Since(startTime)
	}()

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		result.Attempts = attempt + 1

		// Check context cancellation
		select {
		case <-ctx.Done():
			result.LastError = fmt.Errorf("context cancelled: %w", ctx.Err())
			if onAttempt != nil {
				onAttempt(attempt, result.LastError)
			}
			return result
		default:
		}

		// Execute the operation
		err := operation(ctx)

		// Call the attempt callback
		if onAttempt != nil {
			onAttempt(attempt, err)
		}

		// Success!
		if err == nil {
			result.Success = true
			result.LastError = nil
			return result
		}

		result.LastError = err

		// Check if we should retry
		if attempt >= config.MaxRetries {
			// No more retries - mark as FAILED
			break
		}

		if config.RetryableFn != nil && !config.RetryableFn(err) {
			// Error is not retryable
			break
		}

		// Calculate delay with exponential backoff
		delay := calculateBackoff(attempt, config)

		// Call the retry callback
		if onRetry != nil {
			onRetry(attempt, delay)
		}

		// Wait before retry
		select {
		case <-ctx.Done():
			result.LastError = fmt.Errorf("context cancelled during retry delay: %w", ctx.Err())
			return result
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return result
}

// IsMaxRetriesExceeded checks if the error is due to max retries being exceeded.
func IsMaxRetriesExceeded(err error) bool {
	if err == nil {
		return false
	}

	var doingErr *DoingError
	if AsDoingError(err, &doingErr) {
		if exceeded, ok := doingErr.Context["max_retries_exceeded"].(bool); ok && exceeded {
			return true
		}
	}

	return false
}

// AsDoingError extracts a DoingError from an error chain.
func AsDoingError(err error, target **DoingError) bool {
	if err == nil {
		return false
	}

	// Check if it's directly a DoingError
	if de, ok := err.(*DoingError); ok {
		*target = de
		return true
	}

	// Check if it wraps a DoingError
	if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
		return AsDoingError(unwrapper.Unwrap(), target)
	}

	return false
}

// ExecuteWithRetry executes an operation with retry and returns a DoingError if it fails.
func ExecuteWithRetry(
	ctx context.Context,
	config *RetryConfig,
	category ErrorCategory,
	operationName string,
	operation RetryableFunc,
) error {
	result := Retry(ctx, config, operation)

	if result.Success {
		return nil
	}

	// Create a comprehensive error
	doingErr := NewDoingError(category, operationName+" failed after retries", result.LastError)
	doingErr.WithContext("attempts", result.Attempts)
	doingErr.WithContext("duration_ms", result.Duration.Milliseconds())

	if !result.Success && result.Attempts > config.MaxRetries {
		doingErr.WithContext("max_retries_exceeded", true)
	}

	return doingErr
}
