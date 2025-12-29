package resilience

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"math"
	"sync/atomic"
	"time"

	"gitlab.com/appliedsystems/experimental/users/ddavis/stuff/rentfree/internal/config"
)

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

func (rp *RetryPolicy) Execute(fn func() error) error {
	return rp.ExecuteCtx(context.Background(), func(ctx context.Context) error {
		return fn()
	})
}

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
		jitter := (secureRandomFloat64() * 2 * jitterRange) - jitterRange
		backoff += jitter
	}

	return time.Duration(backoff)
}

// secureRandomFloat64 generates a cryptographically secure random float64 in [0, 1).
func secureRandomFloat64() float64 {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Fallback to 0.5 if random generation fails (should be extremely rare)
		return 0.5
	}
	// Convert to uint64 and scale to [0, 1)
	return float64(binary.BigEndian.Uint64(b[:])) / float64(1<<64)
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

func (rp *DisabledRetryPolicy) Execute(fn func() error) error {
	return fn()
}

func (rp *DisabledRetryPolicy) ExecuteCtx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func (rp *DisabledRetryPolicy) ExecuteWithResult(ctx context.Context, fn func(context.Context) (any, error)) (any, error) {
	return fn(ctx)
}

func (rp *DisabledRetryPolicy) Stats() (retries, success, failure int64) {
	return 0, 0, 0
}

func (rp *DisabledRetryPolicy) Reset() {}
