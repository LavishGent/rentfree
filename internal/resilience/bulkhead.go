package resilience

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/LavishGent/rentfree/internal/config"
)

type Bulkhead struct {
	maxConcurrent  int
	maxQueue       int
	acquireTimeout time.Duration
	semaphore      chan struct{}

	activeCount   atomic.Int32
	queuedCount   atomic.Int32
	rejectedCount atomic.Int64
	totalExecuted atomic.Int64
}

func NewBulkhead(cfg config.BulkheadConfig) *Bulkhead {
	maxConcurrent := cfg.MaxConcurrent
	maxQueue := cfg.MaxQueue
	acquireTimeout := cfg.AcquireTimeout

	if maxConcurrent <= 0 {
		maxConcurrent = 100
	}
	if maxQueue <= 0 {
		maxQueue = 50
	}
	if acquireTimeout <= 0 {
		acquireTimeout = 100 * time.Millisecond
	}

	totalSlots := maxConcurrent + maxQueue

	return &Bulkhead{
		maxConcurrent:  maxConcurrent,
		maxQueue:       maxQueue,
		acquireTimeout: acquireTimeout,
		semaphore:      make(chan struct{}, totalSlots),
	}
}

func (b *Bulkhead) Execute(fn func() error) error {
	return b.ExecuteCtx(context.Background(), func(ctx context.Context) error {
		return fn()
	})
}

// ExecuteCtx executes a function with context through the bulkhead.
func (b *Bulkhead) ExecuteCtx(ctx context.Context, fn func(context.Context) error) error {
	if err := b.acquire(ctx); err != nil {
		return err
	}
	defer b.release()

	b.activeCount.Add(1)
	defer b.activeCount.Add(-1)

	err := fn(ctx)
	b.totalExecuted.Add(1)

	return err
}

// ExecuteWithResult executes a function that returns a result through the bulkhead.
func (b *Bulkhead) ExecuteWithResult(ctx context.Context, fn func(context.Context) (any, error)) (any, error) {
	if err := b.acquire(ctx); err != nil {
		return nil, err
	}
	defer b.release()

	b.activeCount.Add(1)
	defer b.activeCount.Add(-1)

	result, err := fn(ctx)
	b.totalExecuted.Add(1)

	return result, err
}

func (b *Bulkhead) acquire(ctx context.Context) error {
	select {
	case b.semaphore <- struct{}{}:
		return nil
	default:
	}

	currentQueued := b.queuedCount.Load()
	if int(currentQueued) >= b.maxQueue {
		b.rejectedCount.Add(1)
		return ErrBulkheadFull
	}

	b.queuedCount.Add(1)
	defer b.queuedCount.Add(-1)

	timeoutCtx, cancel := context.WithTimeout(ctx, b.acquireTimeout)
	defer cancel()

	select {
	case b.semaphore <- struct{}{}:
		return nil
	case <-timeoutCtx.Done():
		b.rejectedCount.Add(1)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return ErrBulkheadTimeout
	}
}

func (b *Bulkhead) release() {
	<-b.semaphore
}

// ActiveCount returns the number of currently active operations.
func (b *Bulkhead) ActiveCount() int {
	return int(b.activeCount.Load())
}

// QueuedCount returns the number of operations currently queued.
func (b *Bulkhead) QueuedCount() int {
	return int(b.queuedCount.Load())
}

// RejectedCount returns the total number of rejected operations.
func (b *Bulkhead) RejectedCount() int64 {
	return b.rejectedCount.Load()
}

// TotalExecuted returns the total number of executed operations.
func (b *Bulkhead) TotalExecuted() int64 {
	return b.totalExecuted.Load()
}

// AvailableSlots returns the number of available slots.
func (b *Bulkhead) AvailableSlots() int {
	return (b.maxConcurrent + b.maxQueue) - len(b.semaphore)
}

// Stats returns bulkhead statistics.
func (b *Bulkhead) Stats() BulkheadStats {
	return BulkheadStats{
		MaxConcurrent: b.maxConcurrent,
		MaxQueue:      b.maxQueue,
		Active:        int(b.activeCount.Load()),
		Queued:        int(b.queuedCount.Load()),
		Available:     b.AvailableSlots(),
		TotalExecuted: b.totalExecuted.Load(),
		TotalRejected: b.rejectedCount.Load(),
	}
}

// BulkheadStats contains bulkhead statistics.
type BulkheadStats struct {
	MaxConcurrent int
	MaxQueue      int
	Active        int
	Queued        int
	Available     int
	TotalExecuted int64
	TotalRejected int64
}

// DisabledBulkhead is a no-op bulkhead that allows all operations.
type DisabledBulkhead struct{}

// NewDisabledBulkhead creates a disabled bulkhead.
func NewDisabledBulkhead() *DisabledBulkhead {
	return &DisabledBulkhead{}
}

// Execute runs a function without bulkhead protection.
func (b *DisabledBulkhead) Execute(fn func() error) error {
	return fn()
}

// ExecuteCtx runs a function with context without bulkhead protection.
func (b *DisabledBulkhead) ExecuteCtx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

// ExecuteWithResult runs a function that returns a result without bulkhead protection.
func (b *DisabledBulkhead) ExecuteWithResult(ctx context.Context, fn func(context.Context) (any, error)) (any, error) {
	return fn(ctx)
}

// ActiveCount returns 0 as this is a disabled bulkhead.
func (b *DisabledBulkhead) ActiveCount() int { return 0 }

// QueuedCount returns 0 as this is a disabled bulkhead.
func (b *DisabledBulkhead) QueuedCount() int { return 0 }

// RejectedCount returns 0 as this is a disabled bulkhead.
func (b *DisabledBulkhead) RejectedCount() int64 { return 0 }

// TotalExecuted returns 0 as this is a disabled bulkhead.
func (b *DisabledBulkhead) TotalExecuted() int64 { return 0 }

// AvailableSlots returns a large number as this is a disabled bulkhead.
func (b *DisabledBulkhead) AvailableSlots() int { return 1000000 }

// Stats returns empty statistics as this is a disabled bulkhead.
func (b *DisabledBulkhead) Stats() BulkheadStats { return BulkheadStats{} }
