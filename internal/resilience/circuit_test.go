package resilience

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LavishGent/rentfree/internal/config"
)

func TestCircuitBreakerStateString(t *testing.T) {
	//nolint:govet // Test table - alignment not critical
	tests := []struct {
		state    State
		expected string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{State(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("State.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewCircuitBreaker(t *testing.T) {
	t.Run("creates with config values", func(t *testing.T) {
		cfg := config.CircuitBreakerConfig{
			FailureThreshold:    10,
			SuccessThreshold:    5,
			OpenDuration:        1 * time.Minute,
			HalfOpenMaxRequests: 7,
		}

		cb := NewCircuitBreaker(cfg)

		if cb.failureThreshold != 10 {
			t.Errorf("failureThreshold = %v, want 10", cb.failureThreshold)
		}
		if cb.successThreshold != 5 {
			t.Errorf("successThreshold = %v, want 5", cb.successThreshold)
		}
		if cb.openDuration != 1*time.Minute {
			t.Errorf("openDuration = %v, want 1m", cb.openDuration)
		}
		if cb.halfOpenMaxRequests != 7 {
			t.Errorf("halfOpenMaxRequests = %v, want 7", cb.halfOpenMaxRequests)
		}
		if cb.State() != StateClosed {
			t.Errorf("initial state = %v, want closed", cb.State())
		}
	})

	t.Run("applies defaults for zero values", func(t *testing.T) {
		cfg := config.CircuitBreakerConfig{}

		cb := NewCircuitBreaker(cfg)

		if cb.failureThreshold != 5 {
			t.Errorf("failureThreshold = %v, want 5", cb.failureThreshold)
		}
		if cb.successThreshold != 2 {
			t.Errorf("successThreshold = %v, want 2", cb.successThreshold)
		}
		if cb.openDuration != 30*time.Second {
			t.Errorf("openDuration = %v, want 30s", cb.openDuration)
		}
		if cb.halfOpenMaxRequests != 3 {
			t.Errorf("halfOpenMaxRequests = %v, want 3", cb.halfOpenMaxRequests)
		}
	})
}

func TestCircuitBreakerStateTransitions(t *testing.T) {
	t.Run("closed to open after failure threshold", func(t *testing.T) {
		cfg := config.CircuitBreakerConfig{
			FailureThreshold: 3,
			OpenDuration:     1 * time.Second,
		}
		cb := NewCircuitBreaker(cfg)

		// Record failures below threshold
		cb.RecordFailure()
		cb.RecordFailure()
		if cb.State() != StateClosed {
			t.Errorf("state after 2 failures = %v, want closed", cb.State())
		}

		// Third failure should open
		cb.RecordFailure()
		if cb.State() != StateOpen {
			t.Errorf("state after 3 failures = %v, want open", cb.State())
		}
	})

	t.Run("open to half-open after duration", func(t *testing.T) {
		cfg := config.CircuitBreakerConfig{
			FailureThreshold: 1,
			OpenDuration:     50 * time.Millisecond,
		}
		cb := NewCircuitBreaker(cfg)

		// Open the circuit
		cb.RecordFailure()
		if cb.State() != StateOpen {
			t.Fatalf("state = %v, want open", cb.State())
		}

		// Should not allow immediately
		if cb.Allow() {
			t.Error("Allow() = true, want false while open")
		}

		// Wait for open duration
		time.Sleep(60 * time.Millisecond)

		// Should transition to half-open on next Allow()
		if !cb.Allow() {
			t.Error("Allow() = false, want true after open duration")
		}
		if cb.State() != StateHalfOpen {
			t.Errorf("state = %v, want half-open", cb.State())
		}
	})

	t.Run("half-open to closed after success threshold", func(t *testing.T) {
		cfg := config.CircuitBreakerConfig{
			FailureThreshold:    1,
			SuccessThreshold:    2,
			OpenDuration:        10 * time.Millisecond,
			HalfOpenMaxRequests: 5,
		}
		cb := NewCircuitBreaker(cfg)

		// Open and wait for half-open
		cb.RecordFailure()
		time.Sleep(20 * time.Millisecond)
		cb.Allow() // Transition to half-open

		// Record successes
		cb.RecordSuccess()
		if cb.State() != StateHalfOpen {
			t.Errorf("state after 1 success = %v, want half-open", cb.State())
		}

		cb.RecordSuccess()
		if cb.State() != StateClosed {
			t.Errorf("state after 2 successes = %v, want closed", cb.State())
		}
	})

	t.Run("half-open to open on failure", func(t *testing.T) {
		cfg := config.CircuitBreakerConfig{
			FailureThreshold:    1,
			SuccessThreshold:    2,
			OpenDuration:        10 * time.Millisecond,
			HalfOpenMaxRequests: 5,
		}
		cb := NewCircuitBreaker(cfg)

		// Open and wait for half-open
		cb.RecordFailure()
		time.Sleep(20 * time.Millisecond)
		cb.Allow()

		if cb.State() != StateHalfOpen {
			t.Fatalf("state = %v, want half-open", cb.State())
		}

		// Any failure in half-open should reopen
		cb.RecordFailure()
		if cb.State() != StateOpen {
			t.Errorf("state after failure in half-open = %v, want open", cb.State())
		}
	})
}

func TestCircuitBreakerAllow(t *testing.T) {
	t.Run("always allows in closed state", func(t *testing.T) {
		cb := NewCircuitBreaker(config.CircuitBreakerConfig{})

		for i := 0; i < 100; i++ {
			if !cb.Allow() {
				t.Errorf("Allow() = false in closed state")
			}
		}
	})

	t.Run("blocks in open state", func(t *testing.T) {
		cfg := config.CircuitBreakerConfig{
			FailureThreshold: 1,
			OpenDuration:     1 * time.Hour,
		}
		cb := NewCircuitBreaker(cfg)
		cb.RecordFailure() // Open the circuit

		if cb.Allow() {
			t.Error("Allow() = true, want false in open state")
		}
	})

	t.Run("limits requests in half-open state", func(t *testing.T) {
		cfg := config.CircuitBreakerConfig{
			FailureThreshold:    1,
			OpenDuration:        10 * time.Millisecond,
			HalfOpenMaxRequests: 3,
		}
		cb := NewCircuitBreaker(cfg)

		// Open and wait
		cb.RecordFailure()
		time.Sleep(20 * time.Millisecond)

		// Should allow up to HalfOpenMaxRequests
		for i := 0; i < 3; i++ {
			if !cb.Allow() {
				t.Errorf("Allow() = false, want true for request %d in half-open", i+1)
			}
		}

		// Should block additional requests
		if cb.Allow() {
			t.Error("Allow() = true, want false after max half-open requests")
		}
	})
}

func TestCircuitBreakerExecute(t *testing.T) {
	t.Run("executes function and returns result", func(t *testing.T) {
		cb := NewCircuitBreaker(config.CircuitBreakerConfig{})

		result, err := cb.Execute(func() (any, error) {
			return "success", nil
		})

		if err != nil {
			t.Errorf("Execute() error = %v, want nil", err)
		}
		if result != "success" {
			t.Errorf("Execute() result = %v, want success", result)
		}
	})

	t.Run("returns ErrCircuitOpen when open", func(t *testing.T) {
		cfg := config.CircuitBreakerConfig{
			FailureThreshold: 1,
			OpenDuration:     1 * time.Hour,
		}
		cb := NewCircuitBreaker(cfg)
		cb.RecordFailure()

		_, err := cb.Execute(func() (any, error) {
			return "should not run", nil
		})

		if !errors.Is(err, ErrCircuitOpen) {
			t.Errorf("Execute() error = %v, want ErrCircuitOpen", err)
		}
	})

	t.Run("records success on nil error", func(t *testing.T) {
		cb := NewCircuitBreaker(config.CircuitBreakerConfig{})

		// Record some failures first
		cb.RecordFailure()
		cb.RecordFailure()

		// Execute success should reset
		_, _ = cb.Execute(func() (any, error) {
			return nil, nil
		})

		stats := cb.Stats()
		if stats.ConsecutiveFails != 0 {
			t.Errorf("ConsecutiveFails = %v, want 0", stats.ConsecutiveFails)
		}
	})

	t.Run("records failure on error", func(t *testing.T) {
		cfg := config.CircuitBreakerConfig{
			FailureThreshold: 5,
		}
		cb := NewCircuitBreaker(cfg)

		_, _ = cb.Execute(func() (any, error) {
			return nil, errors.New("test error")
		})

		stats := cb.Stats()
		if stats.ConsecutiveFails != 1 {
			t.Errorf("ConsecutiveFails = %v, want 1", stats.ConsecutiveFails)
		}
	})
}

func TestCircuitBreakerOnStateChange(t *testing.T) {
	cfg := config.CircuitBreakerConfig{
		FailureThreshold: 1,
		OpenDuration:     10 * time.Millisecond,
	}
	cb := NewCircuitBreaker(cfg)

	var changes []struct{ from, to State }
	var mu sync.Mutex

	cb.SetOnStateChange(func(from, to State) {
		mu.Lock()
		changes = append(changes, struct{ from, to State }{from, to})
		mu.Unlock()
	})

	// Trigger state changes
	cb.RecordFailure() // closed -> open
	time.Sleep(20 * time.Millisecond)
	cb.Allow()         // open -> half-open
	cb.RecordSuccess() // still half-open
	cb.RecordSuccess() // half-open -> closed (default threshold is 2)

	// Wait for async callbacks
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(changes) < 2 {
		t.Errorf("expected at least 2 state changes, got %d", len(changes))
	}
}

// TestCircuitBreakerCallbackCanReadState verifies that callbacks can safely
// read circuit breaker state without deadlocking. This was a bug where
// callbacks were invoked while holding the mutex, causing deadlock if the
// callback called State(), Stats(), or other methods that acquire the mutex.
func TestCircuitBreakerCallbackCanReadState(t *testing.T) {
	cfg := config.CircuitBreakerConfig{
		FailureThreshold: 1,
		SuccessThreshold: 1,
		OpenDuration:     10 * time.Millisecond,
	}
	cb := NewCircuitBreaker(cfg)

	// Use a channel with timeout to detect deadlock
	done := make(chan struct{})
	var capturedState State
	var capturedStats CircuitBreakerStats

	cb.SetOnStateChange(func(from, to State) {
		// These calls would have deadlocked before the fix because
		// they acquire the mutex, and we were inside the mutex.
		capturedState = cb.State()
		capturedStats = cb.Stats()
	})

	// Run in goroutine with timeout to detect deadlock
	go func() {
		cb.RecordFailure() // closed -> open (triggers callback)
		close(done)
	}()

	select {
	case <-done:
		// Success - no deadlock
	case <-time.After(1 * time.Second):
		t.Fatal("deadlock detected: callback could not read circuit breaker state")
	}

	// Verify the callback was able to read state correctly
	if capturedState != StateOpen {
		t.Errorf("callback captured state = %v, want open", capturedState)
	}
	if capturedStats.State != StateOpen {
		t.Errorf("callback captured stats.State = %v, want open", capturedStats.State)
	}
}

func TestCircuitBreakerReset(t *testing.T) {
	cfg := config.CircuitBreakerConfig{
		FailureThreshold: 1,
		OpenDuration:     1 * time.Hour,
	}
	cb := NewCircuitBreaker(cfg)

	// Open the circuit
	cb.RecordFailure()
	if cb.State() != StateOpen {
		t.Fatalf("state = %v, want open", cb.State())
	}

	// Reset
	cb.Reset()

	if cb.State() != StateClosed {
		t.Errorf("state after reset = %v, want closed", cb.State())
	}

	stats := cb.Stats()
	if stats.ConsecutiveFails != 0 || stats.ConsecutiveSuccs != 0 {
		t.Errorf("counters not reset: fails=%d, succs=%d", stats.ConsecutiveFails, stats.ConsecutiveSuccs)
	}
}

func TestCircuitBreakerConcurrency(t *testing.T) {
	cfg := config.CircuitBreakerConfig{
		FailureThreshold: 100,
		OpenDuration:     1 * time.Second,
	}
	cb := NewCircuitBreaker(cfg)

	var wg sync.WaitGroup
	var successCount, failCount atomic.Int64

	// Run concurrent operations
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				if cb.Allow() {
					if j%2 == 0 {
						cb.RecordSuccess()
						successCount.Add(1)
					} else {
						cb.RecordFailure()
						failCount.Add(1)
					}
				}
			}
		}(i)
	}

	wg.Wait()

	// Should have processed many operations
	total := successCount.Load() + failCount.Load()
	if total < 1000 {
		t.Errorf("total operations = %d, want >= 1000", total)
	}
}

func TestDisabledCircuitBreaker(t *testing.T) {
	cb := NewDisabledCircuitBreaker()

	t.Run("always allows", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			if !cb.Allow() {
				t.Error("Allow() = false, want true")
			}
		}
	})

	t.Run("executes function", func(t *testing.T) {
		result, err := cb.Execute(func() (any, error) {
			return "test", nil
		})
		if err != nil || result != "test" {
			t.Errorf("Execute() = (%v, %v), want (test, nil)", result, err)
		}
	})

	t.Run("always reports closed", func(t *testing.T) {
		if cb.State() != StateClosed {
			t.Errorf("State() = %v, want closed", cb.State())
		}
		if cb.IsOpen() {
			t.Error("IsOpen() = true, want false")
		}
		if !cb.IsClosed() {
			t.Error("IsClosed() = false, want true")
		}
	})
}
