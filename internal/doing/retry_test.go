package doing

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", config.MaxRetries)
	}
	if config.BaseDelay != 1*time.Second {
		t.Errorf("BaseDelay = %v, want 1s", config.BaseDelay)
	}
	if config.MaxDelay != 30*time.Second {
		t.Errorf("MaxDelay = %v, want 30s", config.MaxDelay)
	}
	if config.Multiplier != 2.0 {
		t.Errorf("Multiplier = %f, want 2.0", config.Multiplier)
	}
	if config.RetryableFn == nil {
		t.Error("RetryableFn should not be nil")
	}
}

func TestRetry_Success(t *testing.T) {
	callCount := 0
	operation := func(ctx context.Context) error {
		callCount++
		return nil
	}

	config := &RetryConfig{
		MaxRetries:  3,
		BaseDelay:   10 * time.Millisecond,
		RetryableFn: func(err error) bool { return true },
	}

	result := Retry(context.Background(), config, operation)

	if !result.Success {
		t.Error("Expected success")
	}
	if result.Attempts != 1 {
		t.Errorf("Attempts = %d, want 1", result.Attempts)
	}
	if callCount != 1 {
		t.Errorf("CallCount = %d, want 1", callCount)
	}
}

func TestRetry_EventualSuccess(t *testing.T) {
	callCount := 0
	operation := func(ctx context.Context) error {
		callCount++
		if callCount < 3 {
			return errors.New("temporary error")
		}
		return nil
	}

	config := &RetryConfig{
		MaxRetries:  5,
		BaseDelay:   10 * time.Millisecond,
		RetryableFn: func(err error) bool { return true },
	}

	result := Retry(context.Background(), config, operation)

	if !result.Success {
		t.Error("Expected eventual success")
	}
	if result.Attempts != 3 {
		t.Errorf("Attempts = %d, want 3", result.Attempts)
	}
}

func TestRetry_MaxRetriesExceeded(t *testing.T) {
	callCount := 0
	operation := func(ctx context.Context) error {
		callCount++
		return errors.New("persistent error")
	}

	config := &RetryConfig{
		MaxRetries:  2,
		BaseDelay:   10 * time.Millisecond,
		RetryableFn: func(err error) bool { return true },
	}

	result := Retry(context.Background(), config, operation)

	if result.Success {
		t.Error("Expected failure")
	}
	if result.Attempts != 3 { // initial + 2 retries
		t.Errorf("Attempts = %d, want 3", result.Attempts)
	}
	if callCount != 3 {
		t.Errorf("CallCount = %d, want 3", callCount)
	}
	if result.LastError == nil {
		t.Error("Expected LastError to be set")
	}
}

func TestRetry_NonRetryableError(t *testing.T) {
	callCount := 0
	operation := func(ctx context.Context) error {
		callCount++
		return errors.New("fatal error")
	}

	config := &RetryConfig{
		MaxRetries:  3,
		BaseDelay:   10 * time.Millisecond,
		RetryableFn: func(err error) bool { return false }, // Non-retryable
	}

	result := Retry(context.Background(), config, operation)

	if result.Success {
		t.Error("Expected failure")
	}
	if result.Attempts != 1 { // Should not retry
		t.Errorf("Attempts = %d, want 1", result.Attempts)
	}
}

func TestRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	operation := func(ctx context.Context) error {
		return errors.New("error")
	}

	config := &RetryConfig{
		MaxRetries:  5,
		BaseDelay:   1 * time.Second, // Long delay
		RetryableFn: func(err error) bool { return true },
	}

	// Cancel context after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	result := Retry(ctx, config, operation)

	if result.Success {
		t.Error("Expected failure due to context cancellation")
	}
	if result.LastError == nil || !errors.Is(result.LastError, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got %v", result.LastError)
	}
}

func TestCalculateBackoff(t *testing.T) {
	config := &RetryConfig{
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   1 * time.Second,
		Multiplier: 2.0,
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 400 * time.Millisecond},
		{3, 800 * time.Millisecond},
		{4, 1 * time.Second}, // capped at MaxDelay
		{5, 1 * time.Second}, // capped at MaxDelay
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			delay := calculateBackoff(tt.attempt, config)
			if delay != tt.expected {
				t.Errorf("calculateBackoff(%d) = %v, want %v", tt.attempt, delay, tt.expected)
			}
		})
	}
}

func TestRetryWithErrorHandler(t *testing.T) {
	attempts := []int{}
	retries := []int{}

	operation := func(ctx context.Context) error {
		return errors.New("error")
	}

	onAttempt := func(attempt int, err error) {
		attempts = append(attempts, attempt)
	}

	onRetry := func(attempt int, delay time.Duration) {
		retries = append(retries, attempt)
	}

	config := &RetryConfig{
		MaxRetries:  2,
		BaseDelay:   10 * time.Millisecond,
		RetryableFn: func(err error) bool { return true },
	}

	result := RetryWithErrorHandler(context.Background(), config, onAttempt, onRetry, operation)

	if result.Success {
		t.Error("Expected failure")
	}
	if len(attempts) != 3 { // initial + 2 retries
		t.Errorf("onAttempt called %d times, want 3", len(attempts))
	}
	if len(retries) != 2 { // 2 retry delays
		t.Errorf("onRetry called %d times, want 2", len(retries))
	}
}

func TestExecuteWithRetry_Success(t *testing.T) {
	operation := func(ctx context.Context) error {
		return nil
	}

	config := &RetryConfig{
		MaxRetries:  3,
		BaseDelay:   10 * time.Millisecond,
		RetryableFn: func(err error) bool { return true },
	}

	err := ExecuteWithRetry(context.Background(), config, ErrorCategoryExecution, "test", operation)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestExecuteWithRetry_Failure(t *testing.T) {
	operation := func(ctx context.Context) error {
		return errors.New("persistent failure")
	}

	config := &RetryConfig{
		MaxRetries:  1,
		BaseDelay:   10 * time.Millisecond,
		RetryableFn: func(err error) bool { return true },
	}

	err := ExecuteWithRetry(context.Background(), config, ErrorCategoryExecution, "test operation", operation)

	if err == nil {
		t.Error("Expected error")
	}

	doingErr, ok := err.(*DoingError)
	if !ok {
		t.Error("Expected DoingError")
	}
	if doingErr.Category != ErrorCategoryExecution {
		t.Errorf("Category = %v, want Execution", doingErr.Category)
	}
}

func TestRetry_NilConfig(t *testing.T) {
	callCount := 0
	operation := func(ctx context.Context) error {
		callCount++
		return nil
	}

	result := Retry(context.Background(), nil, operation)

	if !result.Success {
		t.Error("Expected success with default config")
	}
	if callCount != 1 {
		t.Errorf("CallCount = %d, want 1", callCount)
	}
}

func TestRetry_DurationTracked(t *testing.T) {
	operation := func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	}

	config := &RetryConfig{
		MaxRetries:  0,
		BaseDelay:   0,
		RetryableFn: func(err error) bool { return true },
	}

	result := Retry(context.Background(), config, operation)

	if result.Duration < 10*time.Millisecond {
		t.Errorf("Duration = %v, expected at least 10ms", result.Duration)
	}
}

func TestIsMaxRetriesExceeded(t *testing.T) {
	t.Run("with max_retries_exceeded flag", func(t *testing.T) {
		err := NewDoingError(ErrorCategoryExecution, "failed", nil).
			WithContext("max_retries_exceeded", true)
		if !IsMaxRetriesExceeded(err) {
			t.Error("Expected IsMaxRetriesExceeded to return true")
		}
	})

	t.Run("without flag", func(t *testing.T) {
		err := NewDoingError(ErrorCategoryExecution, "failed", nil)
		if IsMaxRetriesExceeded(err) {
			t.Error("Expected IsMaxRetriesExceeded to return false")
		}
	})

	t.Run("nil error", func(t *testing.T) {
		if IsMaxRetriesExceeded(nil) {
			t.Error("Expected IsMaxRetriesExceeded to return false for nil")
		}
	})

	t.Run("regular error", func(t *testing.T) {
		if IsMaxRetriesExceeded(errors.New("regular error")) {
			t.Error("Expected IsMaxRetriesExceeded to return false for regular error")
		}
	})
}

func TestRetry_ContextTimeoutDuringDelay(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	callCount := 0
	operation := func(ctx context.Context) error {
		callCount++
		return errors.New("error")
	}

	config := &RetryConfig{
		MaxRetries:  5,
		BaseDelay:   100 * time.Millisecond, // Longer than context timeout
		RetryableFn: func(err error) bool { return true },
	}

	result := Retry(ctx, config, operation)

	if result.Success {
		t.Error("Expected failure")
	}
	if !errors.Is(result.LastError, context.DeadlineExceeded) {
		t.Errorf("Expected DeadlineExceeded, got %v", result.LastError)
	}
}
