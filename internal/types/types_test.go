package types

import (
	"errors"
	"testing"
	"time"
)

func TestCacheLevelString(t *testing.T) {
	tests := []struct {
		level    CacheLevel
		expected string
	}{
		{LevelMemoryOnly, "memory-only"},
		{LevelRedisOnly, "redis-only"},
		{LevelMemoryThenRedis, "memory-then-redis"},
		{LevelAll, "all"},
		{CacheLevel(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("CacheLevel.String() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestCacheLevelIncludesMemory(t *testing.T) {
	tests := []struct {
		level    CacheLevel
		includes bool
	}{
		{LevelMemoryOnly, true},
		{LevelRedisOnly, false},
		{LevelMemoryThenRedis, true},
		{LevelAll, true},
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			if got := tt.level.IncludesMemory(); got != tt.includes {
				t.Errorf("IncludesMemory() = %v, want %v", got, tt.includes)
			}
		})
	}
}

func TestCacheLevelIncludesRedis(t *testing.T) {
	tests := []struct {
		level    CacheLevel
		includes bool
	}{
		{LevelMemoryOnly, false},
		{LevelRedisOnly, true},
		{LevelMemoryThenRedis, true},
		{LevelAll, true},
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			if got := tt.level.IncludesRedis(); got != tt.includes {
				t.Errorf("IncludesRedis() = %v, want %v", got, tt.includes)
			}
		})
	}
}

func TestCachePriorityString(t *testing.T) {
	tests := []struct {
		priority CachePriority
		expected string
	}{
		{PriorityLow, "low"},
		{PriorityNormal, "normal"},
		{PriorityHigh, "high"},
		{PriorityNeverRemove, "never-remove"},
		{CachePriority(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.priority.String(); got != tt.expected {
				t.Errorf("CachePriority.String() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts == nil {
		t.Fatal("DefaultOptions() returned nil")
	}

	if opts.TTL != 5*time.Minute {
		t.Errorf("TTL = %v, want 5m", opts.TTL)
	}

	// Level and Priority should be unset (0)
	if opts.Level != 0 {
		t.Errorf("Level = %v, want 0 (unset)", opts.Level)
	}
	if opts.Priority != 0 {
		t.Errorf("Priority = %v, want 0 (unset)", opts.Priority)
	}
}

func TestApplyOptions(t *testing.T) {
	t.Run("applies no options", func(t *testing.T) {
		opts := ApplyOptions()
		if opts.TTL != 5*time.Minute {
			t.Errorf("TTL = %v, want 5m (default)", opts.TTL)
		}
	})

	t.Run("applies custom options", func(t *testing.T) {
		opts := ApplyOptions(
			func(o *CacheOptions) { o.TTL = 1 * time.Hour },
			func(o *CacheOptions) { o.Level = LevelRedisOnly },
			func(o *CacheOptions) { o.FireAndForget = true },
		)

		if opts.TTL != 1*time.Hour {
			t.Errorf("TTL = %v, want 1h", opts.TTL)
		}
		if opts.Level != LevelRedisOnly {
			t.Errorf("Level = %v, want RedisOnly", opts.Level)
		}
		if !opts.FireAndForget {
			t.Error("FireAndForget = false, want true")
		}
	})
}

func TestCacheEntryIsExpired(t *testing.T) {
	t.Run("not expired when ExpiresAt is zero", func(t *testing.T) {
		entry := &CacheEntry{
			Key:       "key",
			Value:     []byte("value"),
			ExpiresAt: time.Time{},
		}

		if entry.IsExpired() {
			t.Error("IsExpired() = true, want false (no expiry)")
		}
	})

	t.Run("not expired when ExpiresAt is in future", func(t *testing.T) {
		entry := &CacheEntry{
			Key:       "key",
			Value:     []byte("value"),
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}

		if entry.IsExpired() {
			t.Error("IsExpired() = true, want false")
		}
	})

	t.Run("expired when ExpiresAt is in past", func(t *testing.T) {
		entry := &CacheEntry{
			Key:       "key",
			Value:     []byte("value"),
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}

		if !entry.IsExpired() {
			t.Error("IsExpired() = false, want true")
		}
	})
}

func TestCacheErrorError(t *testing.T) {
	t.Run("with key", func(t *testing.T) {
		err := &CacheError{
			Op:    "Get",
			Key:   "user:123",
			Layer: "redis",
			Err:   errors.New("connection refused"),
		}

		expected := "cache Get on redis [user:123]: connection refused"
		if got := err.Error(); got != expected {
			t.Errorf("Error() = %s, want %s", got, expected)
		}
	})

	t.Run("without key", func(t *testing.T) {
		err := &CacheError{
			Op:    "Clear",
			Layer: "memory",
			Err:   errors.New("operation failed"),
		}

		expected := "cache Clear on memory: operation failed"
		if got := err.Error(); got != expected {
			t.Errorf("Error() = %s, want %s", got, expected)
		}
	})
}

func TestCacheErrorUnwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := &CacheError{
		Op:    "Set",
		Key:   "key",
		Layer: "redis",
		Err:   underlying,
	}

	if err.Unwrap() != underlying {
		t.Error("Unwrap() did not return underlying error")
	}

	// Test errors.Is works
	if !errors.Is(err, underlying) {
		t.Error("errors.Is should work with wrapped error")
	}
}

func TestNewCacheError(t *testing.T) {
	underlying := errors.New("test error")
	err := NewCacheError("Get", "key:123", "redis", underlying)

	if err.Op != "Get" {
		t.Errorf("Op = %s, want Get", err.Op)
	}
	if err.Key != "key:123" {
		t.Errorf("Key = %s, want key:123", err.Key)
	}
	if err.Layer != "redis" {
		t.Errorf("Layer = %s, want redis", err.Layer)
	}
	if err.Err != underlying {
		t.Error("Err is not the underlying error")
	}
}

func TestIsCacheMiss(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect bool
	}{
		{"direct ErrCacheMiss", ErrCacheMiss, true},
		{"wrapped ErrCacheMiss", NewCacheError("Get", "key", "memory", ErrCacheMiss), true},
		{"other error", errors.New("other"), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsCacheMiss(tt.err); got != tt.expect {
				t.Errorf("IsCacheMiss() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestIsRedisUnavailable(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect bool
	}{
		{"direct ErrRedisUnavailable", ErrRedisUnavailable, true},
		{"wrapped ErrRedisUnavailable", NewCacheError("Get", "key", "redis", ErrRedisUnavailable), true},
		{"other error", errors.New("other"), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRedisUnavailable(tt.err); got != tt.expect {
				t.Errorf("IsRedisUnavailable() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestIsCircuitOpen(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect bool
	}{
		{"direct ErrCircuitOpen", ErrCircuitOpen, true},
		{"wrapped ErrCircuitOpen", NewCacheError("Get", "key", "redis", ErrCircuitOpen), true},
		{"other error", errors.New("other"), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsCircuitOpen(tt.err); got != tt.expect {
				t.Errorf("IsCircuitOpen() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect bool
	}{
		{"nil error", nil, false},
		{"cache miss", ErrCacheMiss, false},
		{"circuit open", ErrCircuitOpen, false},
		{"closed", ErrClosed, false},
		{"invalid key", ErrInvalidKey, false},
		{"redis unavailable", ErrRedisUnavailable, true},
		{"write queue full", ErrWriteQueueFull, true},
		{"bulkhead full", ErrBulkheadFull, true},
		{"bulkhead timeout", ErrBulkheadTimeout, true},
		{"serialization failed", ErrSerializationFailed, true},
		{"generic error", errors.New("network error"), true},
		{"wrapped retryable", NewCacheError("Get", "key", "redis", errors.New("timeout")), true},
		{"wrapped non-retryable", NewCacheError("Get", "key", "redis", ErrCacheMiss), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryable(tt.err); got != tt.expect {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestMemoryCacheStats(t *testing.T) {
	stats := MemoryCacheStats{
		Hits:      100,
		Misses:    20,
		Sets:      50,
		Deletes:   10,
		Evictions: 5,
	}

	if stats.Hits != 100 {
		t.Errorf("Hits = %d, want 100", stats.Hits)
	}
	if stats.Misses != 20 {
		t.Errorf("Misses = %d, want 20", stats.Misses)
	}
	if stats.Sets != 50 {
		t.Errorf("Sets = %d, want 50", stats.Sets)
	}
	if stats.Deletes != 10 {
		t.Errorf("Deletes = %d, want 10", stats.Deletes)
	}
	if stats.Evictions != 5 {
		t.Errorf("Evictions = %d, want 5", stats.Evictions)
	}
}
