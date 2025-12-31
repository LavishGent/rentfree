package resilience

import (
	"context"
	"math"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/LavishGent/rentfree/internal/config"
)

// RetryPolicy implements the retry pattern with exponential backoff.
type RetryPolicy struct {
	maxAttempts    int
	initialBackoff time.Duration
	maxBackoff     time.Duration
	multiplier     float64
	jitter         bool

	totalRetries atomic.Int64
	totalSuccess atomic.Int64
	totalFailure atomic.Int64
}

// NewRetryPolicy creates a new retry policy with the given configuration.
func NewRetryPolicy(cfg config.RetryConfig) *RetryPolicy {
	rp := &RetryPolicy{
		maxAttempts:    cfg.MaxAttempts,
		initialBackoff: cfg.InitialBackoff,
		maxBackoff:     cfg.MaxBackoff,
		multiplier:     cfg.Multiplier,
		jitter:         cfg.Jitter,
	}

	if rp.maxAttempts <= 0 {
		rp.maxAttempts = 3
	}
	if rp.initialBackoff <= 0 {
		rp.initialBackoff = 100 * time.Millisecond
	}
	if rp.maxBackoff <= 0 {
		rp.maxBackoff = 2 * time.Second
	}
	if rp.multiplier <= 0 {
		rp.multiplier = 2.0
	}

	return rp
}

// Execute runs an operation with retry logic.
func (rp *RetryPolicy) Execute(fn func() error) error {
	return rp.ExecuteCtx(context.Background(), func(ctx context.Context) error {
		return fn()
	})
}

// ExecuteCtx runs an operation with retry logic and context.
func (rp *RetryPolicy) ExecuteCtx(ctx context.Context, fn func(context.Context) error) error {
	var lastErr error

	for attempt := 1; attempt <= rp.maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := fn(ctx)
		if err == nil {
			rp.totalSuccess.Add(1)
			return nil
		}

		lastErr = err

		if !IsRetryable(err) {
			rp.totalFailure.Add(1)
			return err
		}

		if attempt == rp.maxAttempts {
			break
		}

		rp.totalRetries.Add(1)

		backoff := rp.calculateBackoff(attempt)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
	}

	rp.totalFailure.Add(1)
	return lastErr
}

// ExecuteWithResult runs an operation that returns a result with retry logic.
func (rp *RetryPolicy) ExecuteWithResult(ctx context.Context, fn func(context.Context) (any, error)) (any, error) {
	var result any
	var lastErr error

	for attempt := 1; attempt <= rp.maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		var err error
		result, err = fn(ctx)
		if err == nil {
			rp.totalSuccess.Add(1)
			return result, nil
		}

		lastErr = err

		if !IsRetryable(err) {
			rp.totalFailure.Add(1)
			return nil, err
		}

		if attempt == rp.maxAttempts {
			break
		}

		rp.totalRetries.Add(1)
		backoff := rp.calculateBackoff(attempt)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
	}

	rp.totalFailure.Add(1)
	return nil, lastErr
}

// calculateBackoff calculates the backoff duration for an attempt.
func (rp *RetryPolicy) calculateBackoff(attempt int) time.Duration {
	// Exponential backoff
	backoff := float64(rp.initialBackoff) * math.Pow(rp.multiplier, float64(attempt-1))

	// Cap at max backoff
	if backoff > float64(rp.maxBackoff) {
		backoff = float64(rp.maxBackoff)
	}

	// Add jitter (±25%)
	if rp.jitter {
		jitterRange := backoff * 0.25
		jitter := (rand.Float64() * 2 * jitterRange) - jitterRange
		backoff += jitter
	}

	return time.Duration(backoff)
}

// Stats returns retry statistics.
func (rp *RetryPolicy) Stats() (retries, success, failure int64) {
	return rp.totalRetries.Load(), rp.totalSuccess.Load(), rp.totalFailure.Load()
}

// Reset resets the statistics.
func (rp *RetryPolicy) Reset() {
	rp.totalRetries.Store(0)
	rp.totalSuccess.Store(0)
	rp.totalFailure.Store(0)
}

// DisabledRetryPolicy is a no-op retry policy that doesn't retry.
type DisabledRetryPolicy struct{}

// NewDisabledRetryPolicy creates a disabled retry policy.
func NewDisabledRetryPolicy() *DisabledRetryPolicy {
	return &DisabledRetryPolicy{}
}

// Execute runs a function without retry logic.
func (rp *DisabledRetryPolicy) Execute(fn func() error) error {
	return fn()
}

// ExecuteCtx runs a function with context without retry logic.
func (rp *DisabledRetryPolicy) ExecuteCtx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

// ExecuteWithResult runs a function that returns a result without retry logic.
func (rp *DisabledRetryPolicy) ExecuteWithResult(ctx context.Context, fn func(context.Context) (any, error)) (any, error) {
	return fn(ctx)
}

// Stats returns zero values as this is a disabled retry policy.
func (rp *DisabledRetryPolicy) Stats() (retries, success, failure int64) {
	return 0, 0, 0
}

// Reset does nothing as this is a disabled retry policy.
func (rp *DisabledRetryPolicy) Reset() {}
