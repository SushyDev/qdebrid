package retry

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"go.uber.org/zap"
)

// TestDefaultConfig verifies default configuration values
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MaxRetries != 10 {
		t.Errorf("expected MaxRetries=10, got %d", cfg.MaxRetries)
	}
	if cfg.InitialBackoff != 2*time.Second {
		t.Errorf("expected InitialBackoff=2s, got %v", cfg.InitialBackoff)
	}
	if cfg.MaxBackoff != 5*time.Minute {
		t.Errorf("expected MaxBackoff=5m, got %v", cfg.MaxBackoff)
	}
	if cfg.Multiplier != 2.0 {
		t.Errorf("expected Multiplier=2.0, got %f", cfg.Multiplier)
	}
	if !cfg.Jitter {
		t.Error("expected Jitter=true")
	}
}

// TestHTTPError verifies HTTPError implementation
func TestHTTPError(t *testing.T) {
	err := NewHTTPError(500, "Internal Server Error")

	if err.StatusCode != 500 {
		t.Errorf("expected StatusCode=500, got %d", err.StatusCode)
	}
	if err.Message != "Internal Server Error" {
		t.Errorf("expected Message='Internal Server Error', got %s", err.Message)
	}

	expectedMsg := "HTTP 500: Internal Server Error"
	if err.Error() != expectedMsg {
		t.Errorf("expected Error()=%q, got %q", expectedMsg, err.Error())
	}
}

// TestDefaultShouldRetry tests the default retry logic
func TestDefaultShouldRetry(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "non-retryable error",
			err:      errors.New("generic error"),
			expected: false,
		},
		{
			name:     "HTTP 429 Too Many Requests",
			err:      NewHTTPError(http.StatusTooManyRequests, "rate limited"),
			expected: true,
		},
		{
			name:     "HTTP 500 Internal Server Error",
			err:      NewHTTPError(http.StatusInternalServerError, "server error"),
			expected: true,
		},
		{
			name:     "HTTP 502 Bad Gateway",
			err:      NewHTTPError(http.StatusBadGateway, "bad gateway"),
			expected: true,
		},
		{
			name:     "HTTP 503 Service Unavailable",
			err:      NewHTTPError(http.StatusServiceUnavailable, "unavailable"),
			expected: true,
		},
		{
			name:     "HTTP 504 Gateway Timeout",
			err:      NewHTTPError(http.StatusGatewayTimeout, "timeout"),
			expected: true,
		},
		{
			name:     "HTTP 400 Bad Request",
			err:      NewHTTPError(http.StatusBadRequest, "bad request"),
			expected: false,
		},
		{
			name:     "HTTP 404 Not Found",
			err:      NewHTTPError(http.StatusNotFound, "not found"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DefaultShouldRetry(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestRetryerSuccessFirstAttempt tests successful execution on first attempt
func TestRetryerSuccessFirstAttempt(t *testing.T) {
	logger := zap.NewNop()
	retryer := New(DefaultConfig(), logger)

	attempts := 0
	fn := func(ctx context.Context) error {
		attempts++
		return nil
	}

	ctx := context.Background()
	err := retryer.Do(ctx, fn)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

// TestRetryerSuccessAfterRetries tests successful execution after retries
func TestRetryerSuccessAfterRetries(t *testing.T) {
	logger := zap.NewNop()
	cfg := Config{
		MaxRetries:     3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		Multiplier:     2.0,
		Jitter:         false, // Disable jitter for predictable testing
	}
	retryer := New(cfg, logger)

	attempts := 0
	fn := func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return NewHTTPError(http.StatusServiceUnavailable, "temporary error")
		}
		return nil
	}

	ctx := context.Background()
	start := time.Now()
	err := retryer.Do(ctx, fn)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}

	// Should have waited at least for 2 backoffs (10ms + 20ms = 30ms)
	minExpected := 30 * time.Millisecond
	if elapsed < minExpected {
		t.Errorf("expected at least %v elapsed, got %v", minExpected, elapsed)
	}
}

// TestRetryerMaxRetriesExceeded tests exhausting all retry attempts
func TestRetryerMaxRetriesExceeded(t *testing.T) {
	logger := zap.NewNop()
	cfg := Config{
		MaxRetries:     2,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		Multiplier:     2.0,
		Jitter:         false,
	}
	retryer := New(cfg, logger)

	attempts := 0
	fn := func(ctx context.Context) error {
		attempts++
		return NewHTTPError(http.StatusServiceUnavailable, "always fails")
	}

	ctx := context.Background()
	err := retryer.Do(ctx, fn)

	if err == nil {
		t.Error("expected error, got nil")
	}
	if !errors.Is(err, ErrMaxRetriesExceeded) {
		t.Errorf("expected ErrMaxRetriesExceeded, got %v", err)
	}
	// MaxRetries=2 means initial attempt + 2 retries = 3 total attempts
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

// TestRetryerNonRetryableError tests that non-retryable errors fail immediately
func TestRetryerNonRetryableError(t *testing.T) {
	logger := zap.NewNop()
	cfg := Config{
		MaxRetries:     5,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		Multiplier:     2.0,
		Jitter:         false,
	}
	retryer := New(cfg, logger)

	attempts := 0
	expectedErr := NewHTTPError(http.StatusBadRequest, "bad request")
	fn := func(ctx context.Context) error {
		attempts++
		return expectedErr
	}

	ctx := context.Background()
	err := retryer.Do(ctx, fn)

	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt (no retries), got %d", attempts)
	}
}

// TestRetryerContextCanceled tests context cancellation
func TestRetryerContextCanceled(t *testing.T) {
	logger := zap.NewNop()
	cfg := Config{
		MaxRetries:     10,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     1 * time.Second,
		Multiplier:     2.0,
		Jitter:         false,
	}
	retryer := New(cfg, logger)

	attempts := 0
	fn := func(ctx context.Context) error {
		attempts++
		return NewHTTPError(http.StatusServiceUnavailable, "always fails")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	err := retryer.Do(ctx, fn)

	if !errors.Is(err, ErrContextCanceled) {
		t.Errorf("expected ErrContextCanceled, got %v", err)
	}
	// Should have made 1-2 attempts before context canceled
	if attempts > 3 {
		t.Errorf("expected <= 3 attempts before cancellation, got %d", attempts)
	}
}

// TestRetryerCustomShouldRetry tests custom retry logic
func TestRetryerCustomShouldRetry(t *testing.T) {
	logger := zap.NewNop()
	cfg := Config{
		MaxRetries:     3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		Multiplier:     2.0,
		Jitter:         false,
	}
	retryer := New(cfg, logger)

	// Custom retry function that only retries on specific error
	customErr := errors.New("custom retryable error")
	retryer.WithShouldRetry(func(err error) bool {
		return errors.Is(err, customErr)
	})

	attempts := 0
	fn := func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return customErr
		}
		return nil
	}

	ctx := context.Background()
	err := retryer.Do(ctx, fn)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

// TestRetryerBackoffCalculation tests backoff calculation
func TestRetryerBackoffCalculation(t *testing.T) {
	cfg := Config{
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     1 * time.Second,
		Multiplier:     2.0,
		Jitter:         false,
	}
	retryer := New(cfg, nil)

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 100 * time.Millisecond},  // 100ms * 2^0 = 100ms
		{1, 200 * time.Millisecond},  // 100ms * 2^1 = 200ms
		{2, 400 * time.Millisecond},  // 100ms * 2^2 = 400ms
		{3, 800 * time.Millisecond},  // 100ms * 2^3 = 800ms
		{4, 1000 * time.Millisecond}, // 100ms * 2^4 = 1600ms, capped at 1000ms
		{5, 1000 * time.Millisecond}, // capped at max
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			backoff := retryer.calculateBackoff(tt.attempt)
			if backoff != tt.expected {
				t.Errorf("attempt %d: expected %v, got %v", tt.attempt, tt.expected, backoff)
			}
		})
	}
}

// TestRetryerBackoffWithJitter tests that jitter is applied
func TestRetryerBackoffWithJitter(t *testing.T) {
	cfg := Config{
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     1 * time.Second,
		Multiplier:     2.0,
		Jitter:         true,
	}
	retryer := New(cfg, nil)

	// Run multiple times to ensure jitter varies
	backoffs := make(map[time.Duration]bool)
	for i := 0; i < 10; i++ {
		backoff := retryer.calculateBackoff(1)
		backoffs[backoff] = true

		// Should be roughly 200ms ± 30%
		if backoff < 140*time.Millisecond || backoff > 260*time.Millisecond {
			t.Errorf("backoff %v out of expected range [140ms, 260ms]", backoff)
		}
	}

	// With jitter, we should see at least a few different values
	if len(backoffs) < 3 {
		t.Errorf("expected varied backoffs with jitter, got %d unique values", len(backoffs))
	}
}

// TestRateLimiterBasic tests basic rate limiting
func TestRateLimiterBasic(t *testing.T) {
	logger := zap.NewNop()
	// 60 requests per minute = 1 per second, burst 2
	rl := NewRateLimiter(60, 2, logger)

	ctx := context.Background()

	// First request uses a token
	start := time.Now()
	if err := rl.Wait(ctx); err != nil {
		t.Errorf("unexpected error on request 0: %v", err)
	}

	// Second request uses another token but must respect minInterval (1 second)
	if err := rl.Wait(ctx); err != nil {
		t.Errorf("unexpected error on request 1: %v", err)
	}
	elapsed := time.Since(start)

	// Second request should wait for minInterval (~1 second)
	if elapsed < 900*time.Millisecond {
		t.Errorf("expected ~1s for second request, got %v", elapsed)
	}

	// Third request should also wait for minInterval (another ~1 second)
	start = time.Now()
	if err := rl.Wait(ctx); err != nil {
		t.Errorf("unexpected error on request 2: %v", err)
	}
	elapsed = time.Since(start)

	// Should have waited at least 900ms (allowing some margin)
	if elapsed < 900*time.Millisecond {
		t.Errorf("expected rate limit wait, got %v", elapsed)
	}
}

// TestRateLimiterRefill tests token refill over time
func TestRateLimiterRefill(t *testing.T) {
	logger := zap.NewNop()
	// 60 requests per minute = 1 per second, burst 1
	rl := NewRateLimiter(60, 1, logger)

	ctx := context.Background()

	// Use the initial token
	if err := rl.Wait(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Wait for a token to refill
	time.Sleep(1100 * time.Millisecond)

	// Should be able to make another request immediately
	start := time.Now()
	if err := rl.Wait(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	elapsed := time.Since(start)

	// Should complete quickly since token was refilled
	if elapsed > 100*time.Millisecond {
		t.Errorf("expected immediate request after refill, took %v", elapsed)
	}
}

// TestRateLimiterContextCanceled tests context cancellation
func TestRateLimiterContextCanceled(t *testing.T) {
	logger := zap.NewNop()
	// Very slow rate: 1 request per minute
	rl := NewRateLimiter(1, 1, logger)

	// Use the initial token
	ctx := context.Background()
	if err := rl.Wait(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Try another request with canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := rl.Wait(ctx)
	if err == nil {
		t.Error("expected error from canceled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// TestRateLimiterHighThroughput tests many requests
func TestRateLimiterHighThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping high throughput test in short mode")
	}

	logger := zap.NewNop()
	// 60 requests per minute = 1 per second, burst 5
	rl := NewRateLimiter(60, 5, logger)

	ctx := context.Background()
	requestCount := 10

	start := time.Now()
	for i := 0; i < requestCount; i++ {
		if err := rl.Wait(ctx); err != nil {
			t.Errorf("unexpected error on request %d: %v", i, err)
		}
	}
	elapsed := time.Since(start)

	// With minInterval enforcement (1 second between requests):
	// Request 0: instant
	// Requests 1-9: each waits ~1 second = ~9 seconds total
	// Should take at least 8.5 seconds
	minExpected := 8500 * time.Millisecond
	if elapsed < minExpected {
		t.Errorf("requests completed too quickly: %v (expected >= %v)", elapsed, minExpected)
	}

	// Should not take longer than 10 seconds
	maxExpected := 10000 * time.Millisecond
	if elapsed > maxExpected {
		t.Errorf("requests took too long: %v (expected <= %v)", elapsed, maxExpected)
	}
}

// TestQueueSubmit tests queue submission with rate limiting and retry
func TestQueueSubmit(t *testing.T) {
	logger := zap.NewNop()
	cfg := Config{
		MaxRetries:     2,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		Multiplier:     2.0,
		Jitter:         false,
	}

	// 60 requests per minute, burst 2
	queue := NewQueue(60, 2, cfg, logger)
	defer queue.Shutdown()

	ctx := context.Background()

	// Test successful operation
	executed := false
	err := queue.Submit(ctx, "test-op", func(ctx context.Context) error {
		executed = true
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !executed {
		t.Error("operation was not executed")
	}
}

// TestQueueSubmitWithRetries tests queue with failing operations
func TestQueueSubmitWithRetries(t *testing.T) {
	logger := zap.NewNop()
	cfg := Config{
		MaxRetries:     3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		Multiplier:     2.0,
		Jitter:         false,
	}

	queue := NewQueue(60, 5, cfg, logger)
	defer queue.Shutdown()

	ctx := context.Background()

	attempts := 0
	err := queue.Submit(ctx, "retry-op", func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return NewHTTPError(http.StatusServiceUnavailable, "temporary")
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

// TestQueueSubmitMultiple tests multiple operations
func TestQueueSubmitMultiple(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multiple submit test in short mode")
	}

	logger := zap.NewNop()
	cfg := DefaultConfig()

	// 120 requests per minute, burst 3
	queue := NewQueue(120, 3, cfg, logger)
	defer queue.Shutdown()

	ctx := context.Background()
	count := 5

	start := time.Now()
	for i := 0; i < count; i++ {
		err := queue.Submit(ctx, "op", func(ctx context.Context) error {
			return nil
		})
		if err != nil {
			t.Errorf("operation %d failed: %v", i, err)
		}
	}
	elapsed := time.Since(start)

	// First 3 are burst (instant), then 2 more at 2/second = ~1 second total
	// Should take at least 900ms
	minExpected := 900 * time.Millisecond
	if elapsed < minExpected {
		t.Errorf("operations completed too quickly: %v", elapsed)
	}
}

// TestQueueShutdown tests graceful shutdown
func TestQueueShutdown(t *testing.T) {
	logger := zap.NewNop()
	queue := NewQueue(60, 5, DefaultConfig(), logger)

	// Shutdown should not hang
	done := make(chan bool)
	go func() {
		queue.Shutdown()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Error("shutdown did not complete in time")
	}
}

// TestQueueContextMerging tests that both queue and caller contexts are respected
func TestQueueContextMerging(t *testing.T) {
	logger := zap.NewNop()
	cfg := Config{
		MaxRetries:     10,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     1 * time.Second,
		Multiplier:     2.0,
		Jitter:         false,
	}

	queue := NewQueue(60, 5, cfg, logger)
	defer queue.Shutdown()

	// Test caller context cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	err := queue.Submit(ctx, "timeout-op", func(ctx context.Context) error {
		return NewHTTPError(http.StatusServiceUnavailable, "always fails")
	})

	if err == nil {
		t.Error("expected error from timeout")
	}
}
