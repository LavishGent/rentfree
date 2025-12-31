package resilience

import (
	"context"

	"github.com/LavishGent/rentfree/internal/config"
)

// Policy combines multiple resilience patterns (circuit breaker, retry, bulkhead).
type Policy struct {
	circuitBreaker CircuitBreakerExecutor
	retry          RetryExecutor
	bulkhead       BulkheadExecutor
}

// CircuitBreakerExecutor defines the interface for circuit breaker operations.
type CircuitBreakerExecutor interface {
	Execute(fn func() (any, error)) (any, error)
	Allow() bool
	RecordSuccess()
	RecordFailure()
	State() State
	IsOpen() bool
	SetOnStateChange(fn func(from, to State))
}

// RetryExecutor defines the interface for retry operations.
type RetryExecutor interface {
	ExecuteCtx(ctx context.Context, fn func(context.Context) error) error
	ExecuteWithResult(ctx context.Context, fn func(context.Context) (any, error)) (any, error)
}

// BulkheadExecutor defines the interface for bulkhead operations.
type BulkheadExecutor interface {
	ExecuteCtx(ctx context.Context, fn func(context.Context) error) error
	ExecuteWithResult(ctx context.Context, fn func(context.Context) (any, error)) (any, error)
	ActiveCount() int
	QueuedCount() int
	RejectedCount() int64
}

// NewPolicy creates a new resilience policy from the given configuration.
func NewPolicy(cfg *config.Config) *Policy {
	p := &Policy{}

	// Create circuit breaker
	if cfg.CircuitBreaker.Enabled {
		p.circuitBreaker = NewCircuitBreaker(cfg.CircuitBreaker)
	} else {
		p.circuitBreaker = NewDisabledCircuitBreaker()
	}

	if cfg.Retry.Enabled {
		p.retry = NewRetryPolicy(cfg.Retry)
	} else {
		p.retry = NewDisabledRetryPolicy()
	}

	if cfg.Bulkhead.Enabled {
		p.bulkhead = NewBulkhead(cfg.Bulkhead)
	} else {
		p.bulkhead = NewDisabledBulkhead()
	}

	return p
}

// Execute runs an operation through all resilience patterns.
// Execution order: Bulkhead -> Retry -> Circuit Breaker -> Operation
//
// This ordering ensures:
//   - Bulkhead (outermost): Limits concurrent operations to prevent resource exhaustion
//   - Retry: Each retry attempt goes through the circuit breaker independently
//   - Circuit Breaker (innermost): Each attempt (including retries) counts toward
//     circuit state, enabling fast-fail when the downstream service is unhealthy
//
// If circuit breaker wrapped retry instead, a single failing request would exhaust
// all retries before counting as ONE circuit breaker failure, defeating the purpose
// of the circuit breaker pattern.
func (p *Policy) Execute(ctx context.Context, fn func(context.Context) error) error {
	return p.bulkhead.ExecuteCtx(ctx, func(ctx context.Context) error {
		return p.retry.ExecuteCtx(ctx, func(ctx context.Context) error {
			_, err := p.circuitBreaker.Execute(func() (any, error) {
				return nil, fn(ctx)
			})
			return err
		})
	})
}

// ExecuteWithResult runs an operation that returns a result.
// See Execute for details on the ordering rationale.
func (p *Policy) ExecuteWithResult(ctx context.Context, fn func(context.Context) (any, error)) (any, error) {
	return p.bulkhead.ExecuteWithResult(ctx, func(ctx context.Context) (any, error) {
		return p.retry.ExecuteWithResult(ctx, func(ctx context.Context) (any, error) {
			return p.circuitBreaker.Execute(func() (any, error) {
				return fn(ctx)
			})
		})
	})
}

// CircuitBreaker returns the circuit breaker component.
func (p *Policy) CircuitBreaker() CircuitBreakerExecutor {
	return p.circuitBreaker
}

// IsCircuitOpen returns true if the circuit breaker is open.
func (p *Policy) IsCircuitOpen() bool {
	return p.circuitBreaker.IsOpen()
}

// CircuitState returns the current circuit breaker state.
func (p *Policy) CircuitState() State {
	return p.circuitBreaker.State()
}

// SetOnCircuitStateChange sets a callback for circuit state changes.
func (p *Policy) SetOnCircuitStateChange(fn func(from, to State)) {
	p.circuitBreaker.SetOnStateChange(fn)
}

// BulkheadStats returns bulkhead statistics.
func (p *Policy) BulkheadStats() (active, queued int, rejected int64) {
	return p.bulkhead.ActiveCount(), p.bulkhead.QueuedCount(), p.bulkhead.RejectedCount()
}

// DisabledPolicy is a no-op policy that bypasses all resilience patterns.
type DisabledPolicy struct{}

// NewDisabledPolicy creates a disabled policy.
func NewDisabledPolicy() *DisabledPolicy {
	return &DisabledPolicy{}
}

// Execute runs a function without resilience patterns.
func (p *DisabledPolicy) Execute(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

// ExecuteWithResult runs a function that returns a result without resilience patterns.
func (p *DisabledPolicy) ExecuteWithResult(ctx context.Context, fn func(context.Context) (any, error)) (any, error) {
	return fn(ctx)
}

// IsCircuitOpen returns false as this is a disabled policy.
func (p *DisabledPolicy) IsCircuitOpen() bool { return false }

// CircuitState returns StateClosed as this is a disabled policy.
func (p *DisabledPolicy) CircuitState() State { return StateClosed }

// SetOnCircuitStateChange does nothing as this is a disabled policy.
func (p *DisabledPolicy) SetOnCircuitStateChange(fn func(from, to State)) {}

// BulkheadStats returns zero values as this is a disabled policy.
func (p *DisabledPolicy) BulkheadStats() (active, queued int, rejected int64) { return 0, 0, 0 }
