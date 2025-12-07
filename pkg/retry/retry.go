package retry

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

var (
	ErrMaxRetriesExceeded = errors.New("maximum retries exceeded")
	ErrContextCanceled    = errors.New("context canceled")
)

// Config holds retry configuration
type Config struct {
	MaxRetries     int           // Maximum number of retry attempts
	InitialBackoff time.Duration // Initial backoff duration
	MaxBackoff     time.Duration // Maximum backoff duration
	Multiplier     float64       // Backoff multiplier
	Jitter         bool          // Add random jitter to backoff
}

// DefaultConfig returns sensible defaults for Real-Debrid API
func DefaultConfig() Config {
	return Config{
		MaxRetries:     10,
		InitialBackoff: 2 * time.Second,
		MaxBackoff:     5 * time.Minute,
		Multiplier:     2.0,
		Jitter:         true,
	}
}

// RetryableFunc is a function that can be retried
type RetryableFunc func(ctx context.Context) error

// ShouldRetryFunc determines if an error should trigger a retry
type ShouldRetryFunc func(err error) bool

// DefaultShouldRetry determines if an HTTP error should be retried
func DefaultShouldRetry(err error) bool {
	if err == nil {
		return false
	}

	// Always retry on network errors, timeouts, etc.
	var netErr interface{ Temporary() bool }
	if errors.As(err, &netErr) && netErr.Temporary() {
		return true
	}

	// Check for HTTP status codes
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		code := httpErr.StatusCode
		// Retry on rate limit, server errors, and service unavailable
		return code == http.StatusTooManyRequests ||
			code == http.StatusInternalServerError ||
			code == http.StatusBadGateway ||
			code == http.StatusServiceUnavailable ||
			code == http.StatusGatewayTimeout
	}

	return false
}

// HTTPError wraps HTTP response errors
type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

// NewHTTPError creates a new HTTP error
func NewHTTPError(statusCode int, message string) *HTTPError {
	return &HTTPError{
		StatusCode: statusCode,
		Message:    message,
	}
}

// Retryer handles retry logic with exponential backoff
type Retryer struct {
	config      Config
	shouldRetry ShouldRetryFunc
	logger      *zap.Logger
}

// New creates a new Retryer with the given configuration
func New(config Config, logger *zap.Logger) *Retryer {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Retryer{
		config:      config,
		shouldRetry: DefaultShouldRetry,
		logger:      logger,
	}
}

// WithShouldRetry sets a custom retry condition
func (r *Retryer) WithShouldRetry(fn ShouldRetryFunc) *Retryer {
	r.shouldRetry = fn
	return r
}

// Do executes the function with retry logic
func (r *Retryer) Do(ctx context.Context, fn RetryableFunc) error {
	var lastErr error

	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		// Check context before attempting
		select {
		case <-ctx.Done():
			r.logger.Info("retry canceled by context",
				zap.Int("attempt", attempt),
				zap.Error(ctx.Err()))
			return ErrContextCanceled
		default:
		}

		// Execute the function
		lastErr = fn(ctx)
		if lastErr == nil {
			if attempt > 0 {
				r.logger.Info("operation succeeded after retries",
					zap.Int("attempts", attempt))
			}
			return nil
		}

		// Check if we should retry
		if !r.shouldRetry(lastErr) {
			r.logger.Info("error is not retryable",
				zap.Int("attempt", attempt),
				zap.Error(lastErr))
			return lastErr
		}

		// Don't wait after the last attempt
		if attempt == r.config.MaxRetries {
			break
		}

		// Calculate backoff
		backoff := r.calculateBackoff(attempt)
		r.logger.Warn("operation failed, retrying",
			zap.Int("attempt", attempt),
			zap.Duration("backoff", backoff),
			zap.Error(lastErr))

		// Wait with context awareness
		select {
		case <-ctx.Done():
			return ErrContextCanceled
		case <-time.After(backoff):
		}
	}

	r.logger.Error("operation failed after all retries",
		zap.Int("max_retries", r.config.MaxRetries),
		zap.Error(lastErr))
	return fmt.Errorf("%w: %v", ErrMaxRetriesExceeded, lastErr)
}

// calculateBackoff computes the backoff duration for a given attempt
func (r *Retryer) calculateBackoff(attempt int) time.Duration {
	backoff := float64(r.config.InitialBackoff) * math.Pow(r.config.Multiplier, float64(attempt))

	// Apply max backoff
	if backoff > float64(r.config.MaxBackoff) {
		backoff = float64(r.config.MaxBackoff)
	}

	// Add jitter if enabled
	if r.config.Jitter {
		jitter := rand.Float64() * backoff * 0.3 // Up to 30% jitter
		backoff = backoff - (backoff * 0.15) + jitter
	}

	return time.Duration(backoff)
}

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	mu          sync.Mutex
	tokens      float64
	maxTokens   float64
	refillRate  float64 // tokens per second
	lastRefill  time.Time
	minInterval time.Duration // minimum time between requests
	lastRequest time.Time
	logger      *zap.Logger
}

// NewRateLimiter creates a rate limiter
// requestsPerMinute: how many requests allowed per minute
// burst: how many requests can be made in a burst
func NewRateLimiter(requestsPerMinute int, burst int, logger *zap.Logger) *RateLimiter {
	if logger == nil {
		logger = zap.NewNop()
	}

	refillRate := float64(requestsPerMinute) / 60.0 // per second
	minInterval := time.Minute / time.Duration(requestsPerMinute)

	return &RateLimiter{
		tokens:      float64(burst),
		maxTokens:   float64(burst),
		refillRate:  refillRate,
		lastRefill:  time.Now(),
		minInterval: minInterval,
		lastRequest: time.Time{},
		logger:      logger,
	}
}

// Wait blocks until a token is available or context is canceled
func (rl *RateLimiter) Wait(ctx context.Context) error {
	for {
		rl.mu.Lock()

		// Refill tokens based on time elapsed
		now := time.Now()
		elapsed := now.Sub(rl.lastRefill).Seconds()
		rl.tokens = math.Min(rl.maxTokens, rl.tokens+elapsed*rl.refillRate)
		rl.lastRefill = now

		// Enforce minimum interval between requests
		if !rl.lastRequest.IsZero() {
			timeSinceLastRequest := now.Sub(rl.lastRequest)
			if timeSinceLastRequest < rl.minInterval {
				waitTime := rl.minInterval - timeSinceLastRequest
				rl.mu.Unlock()

				rl.logger.Debug("enforcing minimum interval",
					zap.Duration("wait_time", waitTime))

				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(waitTime):
					continue
				}
			}
		}

		// Check if we have a token
		if rl.tokens >= 1.0 {
			rl.tokens -= 1.0
			rl.lastRequest = now
			rl.mu.Unlock()
			return nil
		}

		// Calculate how long until next token
		tokensNeeded := 1.0 - rl.tokens
		waitTime := time.Duration(tokensNeeded/rl.refillRate) * time.Second
		rl.mu.Unlock()

		rl.logger.Debug("rate limit reached, waiting",
			zap.Duration("wait_time", waitTime))

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Loop to try again
		}
	}
}

// Queue manages a queue of operations with rate limiting and retry
type Queue struct {
	rateLimiter *RateLimiter
	retryer     *Retryer
	logger      *zap.Logger
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewQueue creates a new operation queue
func NewQueue(requestsPerMinute int, burst int, retryConfig Config, logger *zap.Logger) *Queue {
	if logger == nil {
		logger = zap.NewNop()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Queue{
		rateLimiter: NewRateLimiter(requestsPerMinute, burst, logger),
		retryer:     New(retryConfig, logger),
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Submit submits an operation to the queue
func (q *Queue) Submit(ctx context.Context, name string, fn RetryableFunc) error {
	// Merge contexts - respect both queue and caller context
	mergedCtx, cancel := mergeContexts(q.ctx, ctx)
	defer cancel()

	q.logger.Debug("submitting operation to queue", zap.String("operation", name))

	// Wait for rate limiter
	if err := q.rateLimiter.Wait(mergedCtx); err != nil {
		q.logger.Warn("rate limiter wait canceled",
			zap.String("operation", name),
			zap.Error(err))
		return err
	}

	// Execute with retry
	return q.retryer.Do(mergedCtx, fn)
}

// Shutdown gracefully shuts down the queue
func (q *Queue) Shutdown() {
	q.logger.Info("shutting down queue")
	q.cancel()
	q.wg.Wait()
	q.logger.Info("queue shutdown complete")
}

// mergeContexts creates a context that is canceled when either parent is canceled
func mergeContexts(ctx1, ctx2 context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		select {
		case <-ctx1.Done():
			cancel()
		case <-ctx2.Done():
			cancel()
		case <-ctx.Done():
		}
	}()

	return ctx, cancel
}
