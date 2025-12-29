package resilience

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gitlab.com/appliedsystems/experimental/users/ddavis/stuff/rentfree/internal/config"
)

func TestNewBulkhead(t *testing.T) {
	t.Run("creates with config values", func(t *testing.T) {
		cfg := config.BulkheadConfig{
			MaxConcurrent:  20,
			MaxQueue:       10,
			AcquireTimeout: 500 * time.Millisecond,
		}

		b := NewBulkhead(cfg)

		if b.maxConcurrent != 20 {
			t.Errorf("maxConcurrent = %v, want 20", b.maxConcurrent)
		}
		if b.maxQueue != 10 {
			t.Errorf("maxQueue = %v, want 10", b.maxQueue)
		}
		if b.acquireTimeout != 500*time.Millisecond {
			t.Errorf("acquireTimeout = %v, want 500ms", b.acquireTimeout)
		}
	})

	t.Run("applies defaults for zero values", func(t *testing.T) {
		cfg := config.BulkheadConfig{}

		b := NewBulkhead(cfg)

		if b.maxConcurrent != 100 {
			t.Errorf("maxConcurrent = %v, want 100", b.maxConcurrent)
		}
		if b.maxQueue != 50 {
			t.Errorf("maxQueue = %v, want 50", b.maxQueue)
		}
		if b.acquireTimeout != 100*time.Millisecond {
			t.Errorf("acquireTimeout = %v, want 100ms", b.acquireTimeout)
		}
	})
}

func TestBulkheadExecute(t *testing.T) {
	t.Run("executes function successfully", func(t *testing.T) {
		b := NewBulkhead(config.BulkheadConfig{MaxConcurrent: 10})

		var executed bool
		err := b.Execute(func() error {
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

	t.Run("propagates function error", func(t *testing.T) {
		b := NewBulkhead(config.BulkheadConfig{MaxConcurrent: 10})
		testErr := errors.New("test error")

		err := b.Execute(func() error {
			return testErr
		})

		if !errors.Is(err, testErr) {
			t.Errorf("Execute() error = %v, want %v", err, testErr)
		}
	})
}

func TestBulkheadConcurrencyLimit(t *testing.T) {
	t.Run("allows up to max concurrent plus queue", func(t *testing.T) {
		cfg := config.BulkheadConfig{
			MaxConcurrent:  3,
			MaxQueue:       2, // Total 5 slots
			AcquireTimeout: 100 * time.Millisecond,
		}
		b := NewBulkhead(cfg)

		var activeCount atomic.Int32
		var maxActive atomic.Int32
		blocking := make(chan struct{})
		started := make(chan struct{}, 5)

		// Launch exactly max concurrent operations
		for i := 0; i < 3; i++ {
			go func() {
				_ = b.ExecuteCtx(context.Background(), func(ctx context.Context) error {
					current := activeCount.Add(1)
					// Track max concurrent
					for {
						old := maxActive.Load()
						if current <= old || maxActive.CompareAndSwap(old, current) {
							break
						}
					}
					started <- struct{}{}
					<-blocking
					activeCount.Add(-1)
					return nil
				})
			}()
		}

		// Wait for all to start
		for i := 0; i < 3; i++ {
			<-started
		}

		// Check active count equals max
		if active := b.ActiveCount(); active != 3 {
			t.Errorf("ActiveCount() = %v, want 3", active)
		}

		close(blocking)
		time.Sleep(10 * time.Millisecond)

		if max := maxActive.Load(); max != 3 {
			t.Errorf("max concurrent = %v, want 3", max)
		}
	})

	t.Run("rejects when semaphore full", func(t *testing.T) {
		cfg := config.BulkheadConfig{
			MaxConcurrent:  2,
			MaxQueue:       1, // Total 3 slots
			AcquireTimeout: 10 * time.Millisecond,
		}
		b := NewBulkhead(cfg)

		// Fill all slots (concurrent + queue)
		blocking := make(chan struct{})
		started := make(chan struct{}, 3)

		for i := 0; i < 3; i++ {
			go func() {
				_ = b.ExecuteCtx(context.Background(), func(ctx context.Context) error {
					started <- struct{}{}
					<-blocking
					return nil
				})
			}()
		}

		// Wait for all slots to be taken
		for i := 0; i < 3; i++ {
			<-started
		}

		// This should be rejected (all slots full)
		err := b.Execute(func() error {
			return nil
		})

		close(blocking)

		if !errors.Is(err, ErrBulkheadFull) && !errors.Is(err, ErrBulkheadTimeout) {
			t.Errorf("Execute() error = %v, want ErrBulkheadFull or ErrBulkheadTimeout", err)
		}
	})
}

func TestBulkheadQueue(t *testing.T) {
	t.Run("queues requests when full", func(t *testing.T) {
		cfg := config.BulkheadConfig{
			MaxConcurrent:  1,
			MaxQueue:       5,
			AcquireTimeout: 500 * time.Millisecond,
		}
		b := NewBulkhead(cfg)

		// Start a long-running operation
		started := make(chan struct{})
		release := make(chan struct{})

		go func() {
			_ = b.ExecuteCtx(context.Background(), func(ctx context.Context) error {
				started <- struct{}{}
				<-release
				return nil
			})
		}()
		<-started

		// Start queued operations
		var completed atomic.Int32
		var wg sync.WaitGroup

		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := b.Execute(func() error {
					completed.Add(1)
					return nil
				})
				if err != nil {
					t.Errorf("queued operation error = %v", err)
				}
			}()
		}

		// Give time for operations to queue
		time.Sleep(10 * time.Millisecond)

		// Release the blocking operation
		close(release)

		wg.Wait()

		if c := completed.Load(); c != 3 {
			t.Errorf("completed = %v, want 3", c)
		}
	})

	t.Run("rejects when queue is full", func(t *testing.T) {
		cfg := config.BulkheadConfig{
			MaxConcurrent:  1,
			MaxQueue:       1,
			AcquireTimeout: 100 * time.Millisecond,
		}
		b := NewBulkhead(cfg)

		// Fill the concurrent slot and queue
		blocking := make(chan struct{})
		started := make(chan struct{}, 2)

		// Start blocking operation
		go func() {
			_ = b.ExecuteCtx(context.Background(), func(ctx context.Context) error {
				started <- struct{}{}
				<-blocking
				return nil
			})
		}()
		<-started

		// Fill the queue
		go func() {
			_ = b.ExecuteCtx(context.Background(), func(ctx context.Context) error {
				started <- struct{}{}
				<-blocking
				return nil
			})
		}()

		// Wait a bit for the queue to fill
		time.Sleep(20 * time.Millisecond)

		// This should be rejected
		err := b.Execute(func() error {
			return nil
		})

		close(blocking)

		if !errors.Is(err, ErrBulkheadFull) && !errors.Is(err, ErrBulkheadTimeout) {
			t.Errorf("Execute() error = %v, want ErrBulkheadFull or ErrBulkheadTimeout", err)
		}
	})
}

func TestBulkheadTimeout(t *testing.T) {
	t.Run("times out waiting for slot", func(t *testing.T) {
		cfg := config.BulkheadConfig{
			MaxConcurrent:  1,
			MaxQueue:       1, // Use 1 to avoid default of 50, total slots = 2
			AcquireTimeout: 50 * time.Millisecond,
		}
		b := NewBulkhead(cfg)

		// Fill all slots (concurrent + queue = 2)
		blocking := make(chan struct{})
		started := make(chan struct{}, 2)

		for i := 0; i < 2; i++ {
			go func() {
				_ = b.ExecuteCtx(context.Background(), func(ctx context.Context) error {
					started <- struct{}{}
					<-blocking
					return nil
				})
			}()
		}
		<-started
		<-started

		// This should timeout (all slots full, including queue)
		start := time.Now()
		err := b.Execute(func() error {
			return nil
		})
		elapsed := time.Since(start)

		close(blocking)

		if !errors.Is(err, ErrBulkheadTimeout) {
			t.Errorf("Execute() error = %v, want ErrBulkheadTimeout", err)
		}

		// Should have waited approximately the timeout duration
		if elapsed < 40*time.Millisecond || elapsed > 200*time.Millisecond {
			t.Errorf("elapsed = %v, expected ~50ms", elapsed)
		}
	})
}

func TestBulkheadContextCancellation(t *testing.T) {
	t.Run("respects context cancellation", func(t *testing.T) {
		cfg := config.BulkheadConfig{
			MaxConcurrent:  1,
			MaxQueue:       1, // Use 1 to avoid default of 50, total slots = 2
			AcquireTimeout: 1 * time.Second,
		}
		b := NewBulkhead(cfg)

		// Fill all slots (concurrent + queue = 2)
		blocking := make(chan struct{})
		started := make(chan struct{}, 2)

		for i := 0; i < 2; i++ {
			go func() {
				_ = b.ExecuteCtx(context.Background(), func(ctx context.Context) error {
					started <- struct{}{}
					<-blocking
					return nil
				})
			}()
		}
		<-started
		<-started

		// Try with cancellable context - should block waiting for slot then cancel
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(30 * time.Millisecond)
			cancel()
		}()

		err := b.ExecuteCtx(ctx, func(ctx context.Context) error {
			return nil
		})

		close(blocking)

		if !errors.Is(err, context.Canceled) {
			t.Errorf("ExecuteCtx() error = %v, want context.Canceled", err)
		}
	})
}

func TestBulkheadExecuteWithResult(t *testing.T) {
	t.Run("returns result", func(t *testing.T) {
		b := NewBulkhead(config.BulkheadConfig{MaxConcurrent: 10})

		result, err := b.ExecuteWithResult(context.Background(), func(ctx context.Context) (any, error) {
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

func TestBulkheadStats(t *testing.T) {
	cfg := config.BulkheadConfig{
		MaxConcurrent:  5,
		MaxQueue:       3,
		AcquireTimeout: 10 * time.Millisecond,
	}
	b := NewBulkhead(cfg)

	// Execute some operations
	for i := 0; i < 10; i++ {
		_ = b.Execute(func() error { return nil })
	}

	stats := b.Stats()

	if stats.MaxConcurrent != 5 {
		t.Errorf("MaxConcurrent = %v, want 5", stats.MaxConcurrent)
	}
	if stats.MaxQueue != 3 {
		t.Errorf("MaxQueue = %v, want 3", stats.MaxQueue)
	}
	if stats.TotalExecuted != 10 {
		t.Errorf("TotalExecuted = %v, want 10", stats.TotalExecuted)
	}
	if stats.Active != 0 {
		t.Errorf("Active = %v, want 0", stats.Active)
	}
}

func TestBulkheadRejectedCount(t *testing.T) {
	// Note: MaxQueue defaults to 50 if <= 0, so we use MaxQueue=1
	// to have total slots = MaxConcurrent + MaxQueue = 1 + 1 = 2
	cfg := config.BulkheadConfig{
		MaxConcurrent:  1,
		MaxQueue:       1,
		AcquireTimeout: 1 * time.Millisecond,
	}
	b := NewBulkhead(cfg)

	// Fill all slots (concurrent + queue = 2)
	blocking := make(chan struct{})
	started := make(chan struct{}, 2)

	for i := 0; i < 2; i++ {
		go func() {
			_ = b.ExecuteCtx(context.Background(), func(ctx context.Context) error {
				started <- struct{}{}
				<-blocking
				return nil
			})
		}()
	}
	<-started
	<-started

	// Try to execute more (should be rejected since all slots are full)
	for i := 0; i < 5; i++ {
		_ = b.Execute(func() error { return nil })
	}

	close(blocking)

	if rejected := b.RejectedCount(); rejected < 1 {
		t.Errorf("RejectedCount() = %v, want >= 1", rejected)
	}
}

func TestDisabledBulkhead(t *testing.T) {
	b := NewDisabledBulkhead()

	t.Run("executes all operations", func(t *testing.T) {
		var wg sync.WaitGroup
		var count atomic.Int32

		for i := 0; i < 1000; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := b.Execute(func() error {
					count.Add(1)
					return nil
				})
				if err != nil {
					t.Errorf("Execute() error = %v, want nil", err)
				}
			}()
		}

		wg.Wait()

		if c := count.Load(); c != 1000 {
			t.Errorf("count = %v, want 1000", c)
		}
	})

	t.Run("stats are zero", func(t *testing.T) {
		stats := b.Stats()
		if stats.Active != 0 || stats.Queued != 0 || stats.TotalRejected != 0 {
			t.Errorf("Stats() = %+v, want all zeros", stats)
		}
	})

	t.Run("available slots is large", func(t *testing.T) {
		if slots := b.AvailableSlots(); slots < 1000 {
			t.Errorf("AvailableSlots() = %v, want large number", slots)
		}
	})
}

func TestBulkheadConcurrency(t *testing.T) {
	cfg := config.BulkheadConfig{
		MaxConcurrent:  10,
		MaxQueue:       20,
		AcquireTimeout: 100 * time.Millisecond,
	}
	b := NewBulkhead(cfg)

	var wg sync.WaitGroup
	var successCount atomic.Int64

	// Run many concurrent operations
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := b.Execute(func() error {
				time.Sleep(5 * time.Millisecond)
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
