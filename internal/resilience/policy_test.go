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

func testConfig() config.Config {
	return config.Config{
		CircuitBreaker: config.CircuitBreakerConfig{
			Enabled:             true,
			FailureThreshold:    3,
			SuccessThreshold:    2,
			OpenDuration:        50 * time.Millisecond,
			HalfOpenMaxRequests: 3,
		},
		Retry: config.RetryConfig{
			Enabled:        true,
			MaxAttempts:    3,
			InitialBackoff: 1 * time.Millisecond,
			MaxBackoff:     10 * time.Millisecond,
			Multiplier:     2.0,
			Jitter:         false,
		},
		Bulkhead: config.BulkheadConfig{
			Enabled:        true,
			MaxConcurrent:  10,
			MaxQueue:       5,
			AcquireTimeout: 50 * time.Millisecond,
		},
	}
}

func TestNewPolicy(t *testing.T) {
	t.Run("creates enabled policy", func(t *testing.T) {
		cfg := testConfig()
		p := NewPolicy(cfg)

		if p.circuitBreaker == nil {
			t.Error("circuitBreaker is nil")
		}
		if p.retry == nil {
			t.Error("retry is nil")
		}
		if p.bulkhead == nil {
			t.Error("bulkhead is nil")
		}
	})

	t.Run("creates disabled components when not enabled", func(t *testing.T) {
		cfg := config.Config{
			CircuitBreaker: config.CircuitBreakerConfig{Enabled: false},
			Retry:          config.RetryConfig{Enabled: false},
			Bulkhead:       config.BulkheadConfig{Enabled: false},
		}
		p := NewPolicy(cfg)

		// Should use disabled implementations
		if _, ok := p.circuitBreaker.(*DisabledCircuitBreaker); !ok {
			t.Error("expected DisabledCircuitBreaker")
		}
		if _, ok := p.retry.(*DisabledRetryPolicy); !ok {
			t.Error("expected DisabledRetryPolicy")
		}
		if _, ok := p.bulkhead.(*DisabledBulkhead); !ok {
			t.Error("expected DisabledBulkhead")
		}
	})
}

func TestPolicyExecute(t *testing.T) {
	t.Run("executes function successfully", func(t *testing.T) {
		p := NewPolicy(testConfig())
		var executed bool

		err := p.Execute(context.Background(), func(ctx context.Context) error {
			executed = true
			return nil
		})

		if err != nil {
			t.Errorf("Execute() error = %v, want nil", err)
		}
		if !executed {
			t.Error("function was not executed")
		}
	})

	t.Run("propagates error", func(t *testing.T) {
		p := NewPolicy(testConfig())
		testErr := errors.New("test error")

		err := p.Execute(context.Background(), func(ctx context.Context) error {
			return testErr
		})

		if !errors.Is(err, testErr) {
			t.Errorf("Execute() error = %v, want %v", err, testErr)
		}
	})
}

func TestPolicyExecuteWithResult(t *testing.T) {
	t.Run("returns result", func(t *testing.T) {
		p := NewPolicy(testConfig())

		result, err := p.ExecuteWithResult(context.Background(), func(ctx context.Context) (any, error) {
			return "success", nil
		})

		if err != nil {
			t.Errorf("ExecuteWithResult() error = %v, want nil", err)
		}
		if result != "success" {
			t.Errorf("ExecuteWithResult() result = %v, want success", result)
		}
	})
}

func TestPolicyRetryIntegration(t *testing.T) {
	t.Run("retries on retryable failure", func(t *testing.T) {
		cfg := testConfig()
		cfg.CircuitBreaker.Enabled = false // Disable circuit breaker for this test
		p := NewPolicy(cfg)

		var attempts int

		err := p.Execute(context.Background(), func(ctx context.Context) error {
			attempts++
			if attempts < 3 {
				return &retryableError{errors.New("transient error")}
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
}

func TestPolicyCircuitBreakerIntegration(t *testing.T) {
	t.Run("opens circuit after failures", func(t *testing.T) {
		cfg := testConfig()
		cfg.CircuitBreaker.FailureThreshold = 3
		cfg.Retry.Enabled = false // Disable retry for this test
		p := NewPolicy(cfg)

		// Cause failures to open circuit
		for i := 0; i < 3; i++ {
			_ = p.Execute(context.Background(), func(ctx context.Context) error {
				return errors.New("failure")
			})
		}

		if !p.IsCircuitOpen() {
			t.Error("IsCircuitOpen() = false, want true")
		}

		// Next call should fail with circuit open
		err := p.Execute(context.Background(), func(ctx context.Context) error {
			return nil
		})

		if !errors.Is(err, ErrCircuitOpen) {
			t.Errorf("Execute() error = %v, want ErrCircuitOpen", err)
		}
	})

	t.Run("recovers after circuit opens", func(t *testing.T) {
		cfg := testConfig()
		cfg.CircuitBreaker.FailureThreshold = 2
		cfg.CircuitBreaker.SuccessThreshold = 1
		cfg.CircuitBreaker.OpenDuration = 20 * time.Millisecond
		cfg.Retry.Enabled = false
		p := NewPolicy(cfg)

		// Open the circuit
		for i := 0; i < 2; i++ {
			_ = p.Execute(context.Background(), func(ctx context.Context) error {
				return errors.New("failure")
			})
		}

		if !p.IsCircuitOpen() {
			t.Fatal("circuit should be open")
		}

		// Wait for half-open transition
		time.Sleep(30 * time.Millisecond)

		// Execute success to close circuit
		err := p.Execute(context.Background(), func(ctx context.Context) error {
			return nil
		})

		if err != nil {
			t.Errorf("Execute() error = %v, want nil", err)
		}

		// Circuit should be closed now
		if p.IsCircuitOpen() {
			t.Error("circuit should be closed after recovery")
		}
	})
}

func TestPolicyBulkheadIntegration(t *testing.T) {
	t.Run("limits concurrency", func(t *testing.T) {
		cfg := testConfig()
		cfg.Bulkhead.MaxConcurrent = 2
		cfg.Bulkhead.MaxQueue = 1 // Use 1 to avoid default of 50, total slots = 3
		cfg.Bulkhead.AcquireTimeout = 10 * time.Millisecond
		cfg.CircuitBreaker.Enabled = false
		cfg.Retry.Enabled = false
		p := NewPolicy(cfg)

		// Fill all bulkhead slots (concurrent + queue = 3)
		started := make(chan struct{}, 3)
		blocking := make(chan struct{})

		for i := 0; i < 3; i++ {
			go func() {
				_ = p.Execute(context.Background(), func(ctx context.Context) error {
					started <- struct{}{}
					<-blocking
					return nil
				})
			}()
		}

		// Wait for all operations to start
		<-started
		<-started
		<-started

		// This should be rejected (all slots full)
		err := p.Execute(context.Background(), func(ctx context.Context) error {
			return nil
		})

		close(blocking)

		if !errors.Is(err, ErrBulkheadFull) && !errors.Is(err, ErrBulkheadTimeout) {
			t.Errorf("Execute() error = %v, want ErrBulkheadFull or ErrBulkheadTimeout", err)
		}
	})
}

func TestPolicyCircuitState(t *testing.T) {
	p := NewPolicy(testConfig())

	if state := p.CircuitState(); state != StateClosed {
		t.Errorf("CircuitState() = %v, want closed", state)
	}

	if p.IsCircuitOpen() {
		t.Error("IsCircuitOpen() = true, want false")
	}
}

func TestPolicySetOnCircuitStateChange(t *testing.T) {
	cfg := testConfig()
	cfg.CircuitBreaker.FailureThreshold = 1
	p := NewPolicy(cfg)

	var stateChanges int
	var mu sync.Mutex

	p.SetOnCircuitStateChange(func(from, to State) {
		mu.Lock()
		stateChanges++
		mu.Unlock()
	})

	// Trigger state change
	p.Execute(context.Background(), func(ctx context.Context) error {
		return errors.New("failure")
	})

	// Wait for async callback
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if stateChanges < 1 {
		t.Errorf("stateChanges = %v, want >= 1", stateChanges)
	}
	mu.Unlock()
}

func TestPolicyBulkheadStats(t *testing.T) {
	cfg := testConfig()
	cfg.Bulkhead.MaxConcurrent = 5
	p := NewPolicy(cfg)

	// Execute some operations
	for i := 0; i < 5; i++ {
		p.Execute(context.Background(), func(ctx context.Context) error {
			return nil
		})
	}

	active, queued, rejected := p.BulkheadStats()

	// After completion, active should be 0
	if active != 0 {
		t.Errorf("active = %v, want 0", active)
	}
	if queued != 0 {
		t.Errorf("queued = %v, want 0", queued)
	}
	// No rejections expected
	if rejected != 0 {
		t.Errorf("rejected = %v, want 0", rejected)
	}
}

func TestPolicyCircuitBreaker(t *testing.T) {
	p := NewPolicy(testConfig())

	cb := p.CircuitBreaker()
	if cb == nil {
		t.Error("CircuitBreaker() returned nil")
	}
}

func TestDisabledPolicy(t *testing.T) {
	p := NewDisabledPolicy()

	t.Run("executes function", func(t *testing.T) {
		var executed bool
		err := p.Execute(context.Background(), func(ctx context.Context) error {
			executed = true
			return nil
		})

		if err != nil {
			t.Errorf("Execute() error = %v, want nil", err)
		}
		if !executed {
			t.Error("function not executed")
		}
	})

	t.Run("returns result", func(t *testing.T) {
		result, err := p.ExecuteWithResult(context.Background(), func(ctx context.Context) (any, error) {
			return "test", nil
		})

		if err != nil {
			t.Errorf("ExecuteWithResult() error = %v, want nil", err)
		}
		if result != "test" {
			t.Errorf("result = %v, want test", result)
		}
	})

	t.Run("circuit is never open", func(t *testing.T) {
		if p.IsCircuitOpen() {
			t.Error("IsCircuitOpen() = true, want false")
		}
		if p.CircuitState() != StateClosed {
			t.Errorf("CircuitState() = %v, want closed", p.CircuitState())
		}
	})

	t.Run("bulkhead stats are zero", func(t *testing.T) {
		active, queued, rejected := p.BulkheadStats()
		if active != 0 || queued != 0 || rejected != 0 {
			t.Errorf("BulkheadStats() = (%d, %d, %d), want (0, 0, 0)", active, queued, rejected)
		}
	})
}

func TestPolicyConcurrency(t *testing.T) {
	// Use a larger bulkhead and longer timeout for concurrency test
	cfg := testConfig()
	cfg.Bulkhead.MaxConcurrent = 50
	cfg.Bulkhead.MaxQueue = 50
	cfg.Bulkhead.AcquireTimeout = 500 * time.Millisecond
	p := NewPolicy(cfg)

	var wg sync.WaitGroup
	var successCount atomic.Int64

	// Run many concurrent operations
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := p.Execute(context.Background(), func(ctx context.Context) error {
				time.Sleep(1 * time.Millisecond)
				return nil
			})
			if err == nil {
				successCount.Add(1)
			}
		}()
	}

	wg.Wait()

	// Most should succeed
	if s := successCount.Load(); s < 50 {
		t.Errorf("successCount = %v, want >= 50", s)
	}
}
