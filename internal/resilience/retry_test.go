package resilience

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/darrell-green/rentfree/internal/config"
)

func TestNewRetryPolicy(t *testing.T) {
	t.Run("creates with config values", func(t *testing.T) {
		cfg := config.RetryConfig{
			MaxAttempts:    5,
			InitialBackoff: 200 * time.Millisecond,
			MaxBackoff:     5 * time.Second,
			Multiplier:     3.0,
			Jitter:         true,
		}

		rp := NewRetryPolicy(cfg)

		if rp.maxAttempts != 5 {
			t.Errorf("maxAttempts = %v, want 5", rp.maxAttempts)
		}
		if rp.initialBackoff != 200*time.Millisecond {
			t.Errorf("initialBackoff = %v, want 200ms", rp.initialBackoff)
		}
		if rp.maxBackoff != 5*time.Second {
			t.Errorf("maxBackoff = %v, want 5s", rp.maxBackoff)
		}
		if rp.multiplier != 3.0 {
			t.Errorf("multiplier = %v, want 3.0", rp.multiplier)
		}
		if !rp.jitter {
			t.Error("jitter = false, want true")
		}
	})

	t.Run("applies defaults for zero values", func(t *testing.T) {
		cfg := config.RetryConfig{}

		rp := NewRetryPolicy(cfg)

		if rp.maxAttempts != 3 {
			t.Errorf("maxAttempts = %v, want 3", rp.maxAttempts)
		}
		if rp.initialBackoff != 100*time.Millisecond {
			t.Errorf("initialBackoff = %v, want 100ms", rp.initialBackoff)
		}
		if rp.maxBackoff != 2*time.Second {
			t.Errorf("maxBackoff = %v, want 2s", rp.maxBackoff)
		}
		if rp.multiplier != 2.0 {
			t.Errorf("multiplier = %v, want 2.0", rp.multiplier)
		}
	})
}

func TestRetryPolicyExecute(t *testing.T) {
	t.Run("succeeds on first attempt", func(t *testing.T) {
		rp := NewRetryPolicy(config.RetryConfig{MaxAttempts: 3})
		var attempts int

		err := rp.Execute(func() error {
			attempts++
			return nil
		})

		if err != nil {
			t.Errorf("Execute() error = %v, want nil", err)
		}
		if attempts != 1 {
			t.Errorf("attempts = %v, want 1", attempts)
		}
	})

	t.Run("retries on failure", func(t *testing.T) {
		cfg := config.RetryConfig{
			MaxAttempts:    3,
			InitialBackoff: 1 * time.Millisecond,
		}
		rp := NewRetryPolicy(cfg)
		var attempts int

		err := rp.Execute(func() error {
			attempts++
			if attempts < 3 {
				return errors.New("transient error") // Generic errors are retryable by default
			}
			return nil
		})

		if err != nil {
			t.Errorf("Execute() error = %v, want nil", err)
		}
		if attempts != 3 {
			t.Errorf("attempts = %v, want 3", attempts)
		}
	})

	t.Run("returns error after max attempts", func(t *testing.T) {
		cfg := config.RetryConfig{
			MaxAttempts:    3,
			InitialBackoff: 1 * time.Millisecond,
		}
		rp := NewRetryPolicy(cfg)
		var attempts int
		testErr := errors.New("persistent error")

		// Wrap error to make it retryable
		err := rp.Execute(func() error {
			attempts++
			return &retryableError{testErr}
		})

		if err == nil {
			t.Error("Execute() error = nil, want error")
		}
		if attempts != 3 {
			t.Errorf("attempts = %v, want 3", attempts)
		}
	})

	t.Run("does not retry non-retryable errors", func(t *testing.T) {
		cfg := config.RetryConfig{
			MaxAttempts:    5,
			InitialBackoff: 1 * time.Millisecond,
		}
		rp := NewRetryPolicy(cfg)
		var attempts int

		err := rp.Execute(func() error {
			attempts++
			return ErrCircuitOpen // Non-retryable
		})

		if !errors.Is(err, ErrCircuitOpen) {
			t.Errorf("Execute() error = %v, want ErrCircuitOpen", err)
		}
		if attempts != 1 {
			t.Errorf("attempts = %v, want 1", attempts)
		}
	})
}

func TestRetryPolicyExecuteCtx(t *testing.T) {
	t.Run("respects context cancellation", func(t *testing.T) {
		cfg := config.RetryConfig{
			MaxAttempts:    10,
			InitialBackoff: 100 * time.Millisecond,
		}
		rp := NewRetryPolicy(cfg)
		var attempts int

		ctx, cancel := context.WithCancel(context.Background())

		// Cancel after a short delay
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		err := rp.ExecuteCtx(ctx, func(ctx context.Context) error {
			attempts++
			return &retryableError{errors.New("keep failing")}
		})

		if !errors.Is(err, context.Canceled) {
			t.Errorf("ExecuteCtx() error = %v, want context.Canceled", err)
		}
		// Should have made some attempts before cancellation
		if attempts < 1 {
			t.Errorf("attempts = %v, want >= 1", attempts)
		}
	})

	t.Run("checks context before each attempt", func(t *testing.T) {
		cfg := config.RetryConfig{
			MaxAttempts:    5,
			InitialBackoff: 1 * time.Millisecond,
		}
		rp := NewRetryPolicy(cfg)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := rp.ExecuteCtx(ctx, func(ctx context.Context) error {
			t.Error("function should not be called when context is cancelled")
			return nil
		})

		if !errors.Is(err, context.Canceled) {
			t.Errorf("ExecuteCtx() error = %v, want context.Canceled", err)
		}
	})
}

func TestRetryPolicyExecuteWithResult(t *testing.T) {
	t.Run("returns result on success", func(t *testing.T) {
		rp := NewRetryPolicy(config.RetryConfig{MaxAttempts: 3})

		result, err := rp.ExecuteWithResult(context.Background(), func(ctx context.Context) (any, error) {
			return "success", nil
		})

		if err != nil {
			t.Errorf("ExecuteWithResult() error = %v, want nil", err)
		}
		if result != "success" {
			t.Errorf("ExecuteWithResult() result = %v, want success", result)
		}
	})

	t.Run("retries and returns final result", func(t *testing.T) {
		cfg := config.RetryConfig{
			MaxAttempts:    3,
			InitialBackoff: 1 * time.Millisecond,
		}
		rp := NewRetryPolicy(cfg)
		var attempts int

		result, err := rp.ExecuteWithResult(context.Background(), func(ctx context.Context) (any, error) {
			attempts++
			if attempts < 3 {
				return nil, &retryableError{errors.New("not yet")}
			}
			return attempts, nil
		})

		if err != nil {
			t.Errorf("ExecuteWithResult() error = %v, want nil", err)
		}
		if result != 3 {
			t.Errorf("ExecuteWithResult() result = %v, want 3", result)
		}
	})
}

func TestRetryPolicyBackoff(t *testing.T) {
	t.Run("backoff increases exponentially", func(t *testing.T) {
		cfg := config.RetryConfig{
			InitialBackoff: 100 * time.Millisecond,
			MaxBackoff:     10 * time.Second,
			Multiplier:     2.0,
			Jitter:         false,
		}
		rp := NewRetryPolicy(cfg)

		b1 := rp.calculateBackoff(1)
		b2 := rp.calculateBackoff(2)
		b3 := rp.calculateBackoff(3)

		if b1 != 100*time.Millisecond {
			t.Errorf("backoff(1) = %v, want 100ms", b1)
		}
		if b2 != 200*time.Millisecond {
			t.Errorf("backoff(2) = %v, want 200ms", b2)
		}
		if b3 != 400*time.Millisecond {
			t.Errorf("backoff(3) = %v, want 400ms", b3)
		}
	})

	t.Run("backoff capped at max", func(t *testing.T) {
		cfg := config.RetryConfig{
			InitialBackoff: 100 * time.Millisecond,
			MaxBackoff:     500 * time.Millisecond,
			Multiplier:     10.0,
			Jitter:         false,
		}
		rp := NewRetryPolicy(cfg)

		b := rp.calculateBackoff(5)
		if b > 500*time.Millisecond {
			t.Errorf("backoff(5) = %v, want <= 500ms", b)
		}
	})

	t.Run("jitter adds variation", func(t *testing.T) {
		cfg := config.RetryConfig{
			InitialBackoff: 100 * time.Millisecond,
			MaxBackoff:     10 * time.Second,
			Multiplier:     2.0,
			Jitter:         true,
		}
		rp := NewRetryPolicy(cfg)

		// Calculate backoff multiple times and check for variation
		values := make(map[time.Duration]bool)
		for i := 0; i < 20; i++ {
			b := rp.calculateBackoff(2)
			values[b] = true
		}

		// With jitter, we should see different values
		if len(values) < 2 {
			t.Error("jitter not producing variation")
		}
	})
}

func TestRetryPolicyStats(t *testing.T) {
	cfg := config.RetryConfig{
		MaxAttempts:    3,
		InitialBackoff: 1 * time.Millisecond,
	}
	rp := NewRetryPolicy(cfg)

	// Success on first try
	_ = rp.Execute(func() error { return nil })

	// Success after retry
	attempts := 0
	_ = rp.Execute(func() error {
		attempts++
		if attempts < 2 {
			return &retryableError{errors.New("fail")}
		}
		return nil
	})

	// Failure after all retries
	_ = rp.Execute(func() error {
		return &retryableError{errors.New("always fail")}
	})

	retries, success, failure := rp.Stats()

	if success != 2 {
		t.Errorf("success = %v, want 2", success)
	}
	if failure != 1 {
		t.Errorf("failure = %v, want 1", failure)
	}
	if retries < 1 {
		t.Errorf("retries = %v, want >= 1", retries)
	}
}

func TestRetryPolicyReset(t *testing.T) {
	rp := NewRetryPolicy(config.RetryConfig{MaxAttempts: 3})

	// Accumulate some stats
	rp.Execute(func() error { return nil })
	rp.Execute(func() error { return nil })

	// Reset
	rp.Reset()

	retries, success, failure := rp.Stats()
	if retries != 0 || success != 0 || failure != 0 {
		t.Errorf("Stats after reset = (%d, %d, %d), want (0, 0, 0)", retries, success, failure)
	}
}

func TestDisabledRetryPolicy(t *testing.T) {
	rp := NewDisabledRetryPolicy()

	t.Run("executes without retry", func(t *testing.T) {
		var attempts int

		err := rp.Execute(func() error {
			attempts++
			return errors.New("always fail")
		})

		if err == nil {
			t.Error("Execute() error = nil, want error")
		}
		if attempts != 1 {
			t.Errorf("attempts = %v, want 1", attempts)
		}
	})

	t.Run("stats are zero", func(t *testing.T) {
		retries, success, failure := rp.Stats()
		if retries != 0 || success != 0 || failure != 0 {
			t.Errorf("Stats() = (%d, %d, %d), want (0, 0, 0)", retries, success, failure)
		}
	})
}

func TestRetryPolicyConcurrency(t *testing.T) {
	cfg := config.RetryConfig{
		MaxAttempts:    2,
		InitialBackoff: 1 * time.Millisecond,
	}
	rp := NewRetryPolicy(cfg)

	var successCount atomic.Int64
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := rp.Execute(func() error {
				return nil
			})
			if err == nil {
				successCount.Add(1)
			}
		}()
	}

	wg.Wait()

	if successCount.Load() != 50 {
		t.Errorf("successCount = %v, want 50", successCount.Load())
	}
}

// retryableError wraps an error to make it retryable
type retryableError struct {
	err error
}

func (e *retryableError) Error() string { return e.err.Error() }
func (e *retryableError) Unwrap() error { return e.err }
