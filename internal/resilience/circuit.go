// Package resilience provides fault tolerance patterns for cache operations.
package resilience

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/LavishGent/rentfree/internal/config"
)

type State int32

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker implements the circuit breaker pattern for fault tolerance.
type CircuitBreaker struct {
	name string

	failureThreshold    int
	successThreshold    int
	openDuration        time.Duration
	halfOpenMaxRequests int

	state atomic.Int32

	mu               sync.Mutex
	consecutiveFails int
	consecutiveSuccs int
	halfOpenRequests int
	openedAt         time.Time

	onStateChange func(from, to State)
}

// stateTransition allows callbacks to be invoked outside the mutex to prevent deadlocks.
type stateTransition struct {
	from     State
	to       State
	callback func(from, to State)
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration.
func NewCircuitBreaker(cfg config.CircuitBreakerConfig) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:                "redis",
		failureThreshold:    cfg.FailureThreshold,
		successThreshold:    cfg.SuccessThreshold,
		openDuration:        cfg.OpenDuration,
		halfOpenMaxRequests: cfg.HalfOpenMaxRequests,
	}

	if cb.failureThreshold <= 0 {
		cb.failureThreshold = 5
	}
	if cb.successThreshold <= 0 {
		cb.successThreshold = 2
	}
	if cb.openDuration <= 0 {
		cb.openDuration = 30 * time.Second
	}
	if cb.halfOpenMaxRequests <= 0 {
		cb.halfOpenMaxRequests = 3
	}

	cb.state.Store(int32(StateClosed))

	return cb
}

// Execute runs a function through the circuit breaker.
func (cb *CircuitBreaker) Execute(fn func() (any, error)) (any, error) {
	if !cb.Allow() {
		return nil, ErrCircuitOpen
	}

	result, err := fn()

	if err != nil {
		cb.RecordFailure()
	} else {
		cb.RecordSuccess()
	}

	return result, err
}

// Allow checks if a request should be allowed through.
func (cb *CircuitBreaker) Allow() bool {
	state := State(cb.state.Load())

	switch state {
	case StateClosed:
		return true

	case StateOpen:
		var transition *stateTransition
		var allowed bool

		cb.mu.Lock()
		if time.Since(cb.openedAt) >= cb.openDuration {
			transition = cb.transitionTo(StateHalfOpen)
			cb.halfOpenRequests = 1
			allowed = true
		}
		cb.mu.Unlock()

		transition.invoke()
		return allowed

	case StateHalfOpen:
		cb.mu.Lock()
		allowed := cb.halfOpenRequests < cb.halfOpenMaxRequests
		if allowed {
			cb.halfOpenRequests++
		}
		cb.mu.Unlock()
		return allowed

	default:
		return true
	}
}

// RecordSuccess records a successful operation.
func (cb *CircuitBreaker) RecordSuccess() {
	var transition *stateTransition

	cb.mu.Lock()
	state := State(cb.state.Load())

	switch state {
	case StateClosed:
		cb.consecutiveFails = 0

	case StateHalfOpen:
		cb.consecutiveSuccs++
		if cb.consecutiveSuccs >= cb.successThreshold {
			transition = cb.transitionTo(StateClosed)
		}
	}
	cb.mu.Unlock()

	// Invoke callback outside mutex to prevent deadlock
	transition.invoke()
}

// RecordFailure records a failed operation.
func (cb *CircuitBreaker) RecordFailure() {
	var transition *stateTransition

	cb.mu.Lock()
	state := State(cb.state.Load())

	switch state {
	case StateClosed:
		cb.consecutiveFails++
		if cb.consecutiveFails >= cb.failureThreshold {
			transition = cb.transitionTo(StateOpen)
		}

	case StateHalfOpen:
		transition = cb.transitionTo(StateOpen)
	}
	cb.mu.Unlock()

	// Invoke callback outside mutex to prevent deadlock
	transition.invoke()
}

// transitionTo changes the circuit breaker state.
// Must be called while holding the mutex.
// Returns a stateTransition if a callback should be invoked, nil otherwise.
// The caller MUST invoke the callback (if non-nil) AFTER releasing the mutex
// to prevent deadlocks.
func (cb *CircuitBreaker) transitionTo(newState State) *stateTransition {
	oldState := State(cb.state.Load())
	if oldState == newState {
		return nil
	}

	// Reset counters based on new state
	switch newState {
	case StateClosed:
		cb.consecutiveFails = 0
		cb.consecutiveSuccs = 0
		cb.halfOpenRequests = 0

	case StateOpen:
		cb.openedAt = time.Now()
		cb.consecutiveSuccs = 0

	case StateHalfOpen:
		cb.consecutiveSuccs = 0
		cb.halfOpenRequests = 0
	}

	cb.state.Store(int32(newState))

	// Return transition info for callback invocation outside the mutex.
	// This prevents deadlocks if the callback reads circuit breaker state.
	if cb.onStateChange != nil {
		return &stateTransition{
			from:     oldState,
			to:       newState,
			callback: cb.onStateChange,
		}
	}
	return nil
}

// invokeCallback safely invokes a state transition callback.
// Must be called AFTER releasing the mutex.
func (t *stateTransition) invoke() {
	if t != nil && t.callback != nil {
		t.callback(t.from, t.to)
	}
}

// State returns the current circuit breaker state.
func (cb *CircuitBreaker) State() State {
	return State(cb.state.Load())
}

// IsOpen returns true if the circuit is open.
func (cb *CircuitBreaker) IsOpen() bool {
	return cb.State() == StateOpen
}

// IsClosed returns true if the circuit is closed.
func (cb *CircuitBreaker) IsClosed() bool {
	return cb.State() == StateClosed
}

// IsHalfOpen returns true if the circuit is half-open.
func (cb *CircuitBreaker) IsHalfOpen() bool {
	return cb.State() == StateHalfOpen
}

// SetOnStateChange sets a callback for state changes.
// The callback is invoked synchronously after state transitions complete.
// The callback may safely read circuit breaker state (State(), Stats(), etc.)
// without risk of deadlock. The callback should be reasonably fast
// (e.g., logging, metrics recording) to avoid blocking operations.
func (cb *CircuitBreaker) SetOnStateChange(fn func(from, to State)) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.onStateChange = fn
}

// Reset resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.consecutiveFails = 0
	cb.consecutiveSuccs = 0
	cb.halfOpenRequests = 0
	cb.state.Store(int32(StateClosed))
}

// Stats returns circuit breaker statistics.
func (cb *CircuitBreaker) Stats() CircuitBreakerStats {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	return CircuitBreakerStats{
		State:            cb.State(),
		ConsecutiveFails: cb.consecutiveFails,
		ConsecutiveSuccs: cb.consecutiveSuccs,
		HalfOpenRequests: cb.halfOpenRequests,
	}
}

// CircuitBreakerStats contains circuit breaker statistics.
type CircuitBreakerStats struct {
	State            State
	ConsecutiveFails int
	ConsecutiveSuccs int
	HalfOpenRequests int
}

// DisabledCircuitBreaker is a no-op circuit breaker that allows all requests.
type DisabledCircuitBreaker struct{}

// NewDisabledCircuitBreaker creates a disabled circuit breaker.
func NewDisabledCircuitBreaker() *DisabledCircuitBreaker {
	return &DisabledCircuitBreaker{}
}

// Execute runs a function without circuit breaker protection.
func (cb *DisabledCircuitBreaker) Execute(fn func() (any, error)) (any, error) {
	return fn()
}

// Allow returns true as this is a disabled circuit breaker.
func (cb *DisabledCircuitBreaker) Allow() bool { return true }

// RecordSuccess does nothing as this is a disabled circuit breaker.
func (cb *DisabledCircuitBreaker) RecordSuccess() {}

// RecordFailure does nothing as this is a disabled circuit breaker.
func (cb *DisabledCircuitBreaker) RecordFailure() {}

// State returns StateClosed as this is a disabled circuit breaker.
func (cb *DisabledCircuitBreaker) State() State { return StateClosed }

// IsOpen returns false as this is a disabled circuit breaker.
func (cb *DisabledCircuitBreaker) IsOpen() bool { return false }

// IsClosed returns true as this is a disabled circuit breaker.
func (cb *DisabledCircuitBreaker) IsClosed() bool { return true }

// IsHalfOpen returns false as this is a disabled circuit breaker.
func (cb *DisabledCircuitBreaker) IsHalfOpen() bool { return false }

// Reset does nothing as this is a disabled circuit breaker.
func (cb *DisabledCircuitBreaker) Reset() {}

// SetOnStateChange does nothing as this is a disabled circuit breaker.
func (cb *DisabledCircuitBreaker) SetOnStateChange(fn func(from, to State)) {}
