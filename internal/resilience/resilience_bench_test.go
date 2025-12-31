package resilience

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/LavishGent/rentfree/internal/config"
)

func BenchmarkCircuitBreaker_Allow(b *testing.B) {
	cfg := config.CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 5,
		SuccessThreshold: 2,
		OpenDuration:     30 * time.Second,
	}
	cb := NewCircuitBreaker(cfg)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = cb.Allow()
	}
}

func BenchmarkCircuitBreaker_RecordSuccess(b *testing.B) {
	cfg := config.CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 5,
		SuccessThreshold: 2,
		OpenDuration:     30 * time.Second,
	}
	cb := NewCircuitBreaker(cfg)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cb.RecordSuccess()
	}
}

func BenchmarkCircuitBreaker_RecordFailure(b *testing.B) {
	cfg := config.CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 1000000, // Prevent opening during benchmark
		SuccessThreshold: 2,
		OpenDuration:     30 * time.Second,
	}
	cb := NewCircuitBreaker(cfg)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cb.RecordFailure()
	}
}

func BenchmarkRetry_Execute_Success(b *testing.B) {
	cfg := config.RetryConfig{
		Enabled:        true,
		MaxAttempts:    3,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     2 * time.Second,
		Multiplier:     2.0,
	}
	rp := NewRetryPolicy(cfg)

	successOp := func() error {
		return nil
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = rp.Execute(successOp)
	}
}

func BenchmarkRetry_Execute_FailThenSuccess(b *testing.B) {
	cfg := config.RetryConfig{
		Enabled:        true,
		MaxAttempts:    3,
		InitialBackoff: time.Microsecond,
		MaxBackoff:     time.Millisecond,
		Multiplier:     2.0,
	}
	rp := NewRetryPolicy(cfg)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		attempt := 0
		failOnceThenSucceed := func() error {
			attempt++
			if attempt == 1 {
				return errors.New("transient error")
			}
			return nil
		}
		_ = rp.Execute(failOnceThenSucceed)
	}
}

func BenchmarkBulkhead_Execute(b *testing.B) {
	cfg := config.BulkheadConfig{
		Enabled:        true,
		MaxConcurrent:  1000,
		MaxQueue:       50,
		AcquireTimeout: 100 * time.Millisecond,
	}
	bh := NewBulkhead(cfg)

	successOp := func() error {
		return nil
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = bh.Execute(successOp)
	}
}

func BenchmarkBulkhead_ExecuteParallel(b *testing.B) {
	cfg := config.BulkheadConfig{
		Enabled:        true,
		MaxConcurrent:  100,
		MaxQueue:       50,
		AcquireTimeout: 100 * time.Millisecond,
	}
	bh := NewBulkhead(cfg)

	successOp := func() error {
		return nil
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = bh.Execute(successOp)
		}
	})
}

func BenchmarkPolicy_Execute_AllEnabled(b *testing.B) {
	cfg := &config.Config{
		CircuitBreaker: config.CircuitBreakerConfig{
			Enabled:          true,
			FailureThreshold: 5,
			SuccessThreshold: 2,
			OpenDuration:     30 * time.Second,
		},
		Retry: config.RetryConfig{
			Enabled:        true,
			MaxAttempts:    3,
			InitialBackoff: 100 * time.Millisecond,
			MaxBackoff:     2 * time.Second,
			Multiplier:     2.0,
		},
		Bulkhead: config.BulkheadConfig{
			Enabled:        true,
			MaxConcurrent:  1000,
			MaxQueue:       50,
			AcquireTimeout: 100 * time.Millisecond,
		},
	}

	policy := NewPolicy(cfg)
	ctx := context.Background()

	successOp := func(ctx context.Context) error {
		return nil
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = policy.Execute(ctx, successOp)
	}
}

func BenchmarkPolicy_Execute_AllDisabled(b *testing.B) {
	cfg := &config.Config{
		CircuitBreaker: config.CircuitBreakerConfig{
			Enabled: false,
		},
		Retry: config.RetryConfig{
			Enabled: false,
		},
		Bulkhead: config.BulkheadConfig{
			Enabled: false,
		},
	}

	policy := NewPolicy(cfg)
	ctx := context.Background()

	successOp := func(ctx context.Context) error {
		return nil
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = policy.Execute(ctx, successOp)
	}
}

func BenchmarkPolicy_ExecuteParallel(b *testing.B) {
	cfg := &config.Config{
		CircuitBreaker: config.CircuitBreakerConfig{
			Enabled:          true,
			FailureThreshold: 5,
			SuccessThreshold: 2,
			OpenDuration:     30 * time.Second,
		},
		Retry: config.RetryConfig{
			Enabled:        true,
			MaxAttempts:    3,
			InitialBackoff: 100 * time.Millisecond,
			MaxBackoff:     2 * time.Second,
			Multiplier:     2.0,
		},
		Bulkhead: config.BulkheadConfig{
			Enabled:        true,
			MaxConcurrent:  100,
			MaxQueue:       50,
			AcquireTimeout: 100 * time.Millisecond,
		},
	}

	policy := NewPolicy(cfg)
	ctx := context.Background()

	successOp := func(ctx context.Context) error {
		return nil
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = policy.Execute(ctx, successOp)
		}
	})
}
