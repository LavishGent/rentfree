package cache

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/darrell-green/rentfree/internal/config"
	"github.com/darrell-green/rentfree/internal/types"
)

func testMemoryConfig() config.MemoryConfig {
	return config.MemoryConfig{
		Enabled:         true,
		MaxSizeMB:       16,
		DefaultTTL:      1 * time.Minute,
		CleanupInterval: 1 * time.Second,
		Shards:          64,
		MaxEntrySize:    1024 * 1024, // 1MB
	}
}

func TestNewMemoryCache(t *testing.T) {
	t.Run("creates with nil logger", func(t *testing.T) {
		cache, err := NewMemoryCache(testMemoryConfig(), nil)
		if err != nil {
			t.Fatalf("NewMemoryCache() error = %v", err)
		}
		defer cache.Close()

		if cache == nil {
			t.Fatal("NewMemoryCache() returned nil")
		}
	})

	t.Run("creates with custom logger", func(t *testing.T) {
		logger := slog.Default()
		cache, err := NewMemoryCache(testMemoryConfig(), logger)
		if err != nil {
			t.Fatalf("NewMemoryCache() error = %v", err)
		}
		defer cache.Close()

		if cache == nil {
			t.Fatal("NewMemoryCache() returned nil")
		}
	})
}

func TestMemoryCacheName(t *testing.T) {
	cache, err := NewMemoryCache(testMemoryConfig(), nil)
	if err != nil {
		t.Fatalf("NewMemoryCache() error = %v", err)
	}
	defer cache.Close()

	if name := cache.Name(); name != "memory" {
		t.Errorf("Name() = %s, want memory", name)
	}
}

func TestMemoryCacheIsAvailable(t *testing.T) {
	cache, err := NewMemoryCache(testMemoryConfig(), nil)
	if err != nil {
		t.Fatalf("NewMemoryCache() error = %v", err)
	}

	t.Run("available when open", func(t *testing.T) {
		if !cache.IsAvailable() {
			t.Error("IsAvailable() = false, want true")
		}
	})

	t.Run("unavailable when closed", func(t *testing.T) {
		cache.Close()
		if cache.IsAvailable() {
			t.Error("IsAvailable() = true, want false after close")
		}
	})
}

func TestMemoryCacheGet(t *testing.T) {
	ctx := context.Background()

	t.Run("returns miss for non-existent key", func(t *testing.T) {
		cache, _ := NewMemoryCache(testMemoryConfig(), nil)
		defer cache.Close()

		_, err := cache.Get(ctx, "non-existent")
		if !errors.Is(err, types.ErrCacheMiss) {
			t.Errorf("Get() error = %v, want ErrCacheMiss", err)
		}
	})

	t.Run("returns value for existing key", func(t *testing.T) {
		cache, _ := NewMemoryCache(testMemoryConfig(), nil)
		defer cache.Close()

		// Set a value first
		value := []byte("test value")
		_ = cache.Set(ctx, "key1", value, nil)

		got, err := cache.Get(ctx, "key1")
		if err != nil {
			t.Errorf("Get() error = %v, want nil", err)
		}
		if string(got) != string(value) {
			t.Errorf("Get() = %s, want %s", string(got), string(value))
		}
	})

	t.Run("returns error when closed", func(t *testing.T) {
		cache, _ := NewMemoryCache(testMemoryConfig(), nil)
		cache.Close()

		_, err := cache.Get(ctx, "key")
		if !errors.Is(err, types.ErrClosed) {
			t.Errorf("Get() error = %v, want ErrClosed", err)
		}
	})

	t.Run("tracks hits and misses", func(t *testing.T) {
		cache, _ := NewMemoryCache(testMemoryConfig(), nil)
		defer cache.Close()

		_ = cache.Set(ctx, "key1", []byte("value"), nil)

		_, _ = cache.Get(ctx, "key1")       // hit
		_, _ = cache.Get(ctx, "key1")       // hit
		_, _ = cache.Get(ctx, "non-exist")  // miss
		_, _ = cache.Get(ctx, "non-exist2") // miss

		stats := cache.Stats()
		if stats.Hits != 2 {
			t.Errorf("Hits = %d, want 2", stats.Hits)
		}
		if stats.Misses != 2 {
			t.Errorf("Misses = %d, want 2", stats.Misses)
		}
	})
}

func TestMemoryCacheSet(t *testing.T) {
	ctx := context.Background()

	t.Run("stores value", func(t *testing.T) {
		cache, _ := NewMemoryCache(testMemoryConfig(), nil)
		defer cache.Close()

		err := cache.Set(ctx, "key1", []byte("value1"), nil)
		if err != nil {
			t.Errorf("Set() error = %v, want nil", err)
		}

		got, err := cache.Get(ctx, "key1")
		if err != nil {
			t.Errorf("Get() error = %v", err)
		}
		if string(got) != "value1" {
			t.Errorf("Get() = %s, want value1", string(got))
		}
	})

	t.Run("overwrites existing value", func(t *testing.T) {
		cache, _ := NewMemoryCache(testMemoryConfig(), nil)
		defer cache.Close()

		_ = cache.Set(ctx, "key1", []byte("value1"), nil)
		_ = cache.Set(ctx, "key1", []byte("value2"), nil)

		got, _ := cache.Get(ctx, "key1")
		if string(got) != "value2" {
			t.Errorf("Get() = %s, want value2", string(got))
		}
	})

	t.Run("returns error when closed", func(t *testing.T) {
		cache, _ := NewMemoryCache(testMemoryConfig(), nil)
		cache.Close()

		err := cache.Set(ctx, "key", []byte("value"), nil)
		if !errors.Is(err, types.ErrClosed) {
			t.Errorf("Set() error = %v, want ErrClosed", err)
		}
	})

	t.Run("tracks set count", func(t *testing.T) {
		cache, _ := NewMemoryCache(testMemoryConfig(), nil)
		defer cache.Close()

		_ = cache.Set(ctx, "key1", []byte("value1"), nil)
		_ = cache.Set(ctx, "key2", []byte("value2"), nil)
		_ = cache.Set(ctx, "key3", []byte("value3"), nil)

		stats := cache.Stats()
		if stats.Sets != 3 {
			t.Errorf("Sets = %d, want 3", stats.Sets)
		}
	})
}

func TestMemoryCacheDelete(t *testing.T) {
	ctx := context.Background()

	t.Run("deletes existing key", func(t *testing.T) {
		cache, _ := NewMemoryCache(testMemoryConfig(), nil)
		defer cache.Close()

		cache.Set(ctx, "key1", []byte("value1"), nil)

		err := cache.Delete(ctx, "key1")
		if err != nil {
			t.Errorf("Delete() error = %v, want nil", err)
		}

		_, err = cache.Get(ctx, "key1")
		if !errors.Is(err, types.ErrCacheMiss) {
			t.Errorf("Get() after delete error = %v, want ErrCacheMiss", err)
		}
	})

	t.Run("no error for non-existent key", func(t *testing.T) {
		cache, _ := NewMemoryCache(testMemoryConfig(), nil)
		defer cache.Close()

		err := cache.Delete(ctx, "non-existent")
		if err != nil {
			t.Errorf("Delete() error = %v, want nil", err)
		}
	})

	t.Run("returns error when closed", func(t *testing.T) {
		cache, _ := NewMemoryCache(testMemoryConfig(), nil)
		cache.Close()

		err := cache.Delete(ctx, "key")
		if !errors.Is(err, types.ErrClosed) {
			t.Errorf("Delete() error = %v, want ErrClosed", err)
		}
	})

	t.Run("tracks delete count", func(t *testing.T) {
		cache, _ := NewMemoryCache(testMemoryConfig(), nil)
		defer cache.Close()

		_ = cache.Set(ctx, "key1", []byte("value1"), nil)
		_ = cache.Delete(ctx, "key1")
		_ = cache.Delete(ctx, "key2") // non-existent

		stats := cache.Stats()
		if stats.Deletes != 2 {
			t.Errorf("Deletes = %d, want 2", stats.Deletes)
		}
	})
}

func TestMemoryCacheContains(t *testing.T) {
	ctx := context.Background()

	t.Run("returns true for existing key", func(t *testing.T) {
		cache, _ := NewMemoryCache(testMemoryConfig(), nil)
		defer cache.Close()

		cache.Set(ctx, "key1", []byte("value1"), nil)

		exists, err := cache.Contains(ctx, "key1")
		if err != nil {
			t.Errorf("Contains() error = %v, want nil", err)
		}
		if !exists {
			t.Error("Contains() = false, want true")
		}
	})

	t.Run("returns false for non-existent key", func(t *testing.T) {
		cache, _ := NewMemoryCache(testMemoryConfig(), nil)
		defer cache.Close()

		exists, err := cache.Contains(ctx, "non-existent")
		if err != nil {
			t.Errorf("Contains() error = %v, want nil", err)
		}
		if exists {
			t.Error("Contains() = true, want false")
		}
	})

	t.Run("returns error when closed", func(t *testing.T) {
		cache, _ := NewMemoryCache(testMemoryConfig(), nil)
		cache.Close()

		_, err := cache.Contains(ctx, "key")
		if !errors.Is(err, types.ErrClosed) {
			t.Errorf("Contains() error = %v, want ErrClosed", err)
		}
	})
}

func TestMemoryCacheClear(t *testing.T) {
	ctx := context.Background()

	t.Run("removes all entries", func(t *testing.T) {
		cache, _ := NewMemoryCache(testMemoryConfig(), nil)
		defer cache.Close()

		_ = cache.Set(ctx, "key1", []byte("value1"), nil)
		_ = cache.Set(ctx, "key2", []byte("value2"), nil)
		_ = cache.Set(ctx, "key3", []byte("value3"), nil)

		err := cache.Clear(ctx)
		if err != nil {
			t.Errorf("Clear() error = %v, want nil", err)
		}

		if count := cache.EntryCount(); count != 0 {
			t.Errorf("EntryCount() after Clear = %d, want 0", count)
		}
	})

	t.Run("returns error when closed", func(t *testing.T) {
		cache, _ := NewMemoryCache(testMemoryConfig(), nil)
		cache.Close()

		err := cache.Clear(ctx)
		if !errors.Is(err, types.ErrClosed) {
			t.Errorf("Clear() error = %v, want ErrClosed", err)
		}
	})
}

func TestMemoryCacheClearByPattern(t *testing.T) {
	ctx := context.Background()

	t.Run("clears matching keys", func(t *testing.T) {
		cache, _ := NewMemoryCache(testMemoryConfig(), nil)
		defer cache.Close()

		cache.Set(ctx, "user:1", []byte("data1"), nil)
		cache.Set(ctx, "user:2", []byte("data2"), nil)
		cache.Set(ctx, "session:1", []byte("sess1"), nil)

		err := cache.ClearByPattern(ctx, "user:*")
		if err != nil {
			t.Errorf("ClearByPattern() error = %v, want nil", err)
		}

		// user keys should be gone
		_, err = cache.Get(ctx, "user:1")
		if !errors.Is(err, types.ErrCacheMiss) {
			t.Error("user:1 should be deleted")
		}
		_, err = cache.Get(ctx, "user:2")
		if !errors.Is(err, types.ErrCacheMiss) {
			t.Error("user:2 should be deleted")
		}

		// session key should remain
		_, err = cache.Get(ctx, "session:1")
		if err != nil {
			t.Error("session:1 should still exist")
		}
	})

	t.Run("returns error when closed", func(t *testing.T) {
		cache, _ := NewMemoryCache(testMemoryConfig(), nil)
		cache.Close()

		err := cache.ClearByPattern(ctx, "*")
		if !errors.Is(err, types.ErrClosed) {
			t.Errorf("ClearByPattern() error = %v, want ErrClosed", err)
		}
	})
}

func TestMemoryCacheClose(t *testing.T) {
	t.Run("closes successfully", func(t *testing.T) {
		cache, _ := NewMemoryCache(testMemoryConfig(), nil)

		err := cache.Close()
		if err != nil {
			t.Errorf("Close() error = %v, want nil", err)
		}
	})

	t.Run("double close is safe", func(t *testing.T) {
		cache, _ := NewMemoryCache(testMemoryConfig(), nil)

		cache.Close()
		err := cache.Close()
		if err != nil {
			t.Errorf("second Close() error = %v, want nil", err)
		}
	})
}

func TestMemoryCacheStats(t *testing.T) {
	ctx := context.Background()
	cache, _ := NewMemoryCache(testMemoryConfig(), nil)
	defer cache.Close()

	// Perform operations
	_ = cache.Set(ctx, "key1", []byte("value1"), nil)
	_ = cache.Set(ctx, "key2", []byte("value2"), nil)
	_, _ = cache.Get(ctx, "key1")         // hit
	_, _ = cache.Get(ctx, "key1")         // hit
	_, _ = cache.Get(ctx, "non-existent") // miss
	_ = cache.Delete(ctx, "key1")

	stats := cache.Stats()
	if stats.Sets != 2 {
		t.Errorf("Sets = %d, want 2", stats.Sets)
	}
	if stats.Hits != 2 {
		t.Errorf("Hits = %d, want 2", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Misses = %d, want 1", stats.Misses)
	}
	if stats.Deletes != 1 {
		t.Errorf("Deletes = %d, want 1", stats.Deletes)
	}
}

func TestMemoryCacheEntryCount(t *testing.T) {
	ctx := context.Background()
	cache, _ := NewMemoryCache(testMemoryConfig(), nil)
	defer cache.Close()

	if count := cache.EntryCount(); count != 0 {
		t.Errorf("initial EntryCount() = %d, want 0", count)
	}

	cache.Set(ctx, "key1", []byte("value1"), nil)
	cache.Set(ctx, "key2", []byte("value2"), nil)

	if count := cache.EntryCount(); count != 2 {
		t.Errorf("EntryCount() = %d, want 2", count)
	}
}

func TestMemoryCacheMaxSize(t *testing.T) {
	cfg := testMemoryConfig()
	cfg.MaxSizeMB = 32
	cache, _ := NewMemoryCache(cfg, nil)
	defer cache.Close()

	expected := int64(32 * 1024 * 1024) // 32MB in bytes
	if maxSize := cache.MaxSize(); maxSize != expected {
		t.Errorf("MaxSize() = %d, want %d", maxSize, expected)
	}
}

func TestMemoryCacheHitRatio(t *testing.T) {
	ctx := context.Background()
	cache, _ := NewMemoryCache(testMemoryConfig(), nil)
	defer cache.Close()

	t.Run("returns 0 when no operations", func(t *testing.T) {
		if ratio := cache.HitRatio(); ratio != 0 {
			t.Errorf("initial HitRatio() = %f, want 0", ratio)
		}
	})

	t.Run("calculates correctly", func(t *testing.T) {
		_ = cache.Set(ctx, "key1", []byte("value1"), nil)

		_, _ = cache.Get(ctx, "key1")         // hit
		_, _ = cache.Get(ctx, "key1")         // hit
		_, _ = cache.Get(ctx, "key1")         // hit
		_, _ = cache.Get(ctx, "non-existent") // miss

		// 3 hits out of 4 = 0.75
		ratio := cache.HitRatio()
		if ratio < 0.74 || ratio > 0.76 {
			t.Errorf("HitRatio() = %f, want ~0.75", ratio)
		}
	})
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		key      string
		pattern  string
		expected bool
	}{
		// Wildcard all
		{"anything", "*", true},
		{"", "*", true},

		// Prefix patterns
		{"user:123", "user:*", true},
		{"user:", "user:*", true},
		{"session:123", "user:*", false},

		// Suffix patterns
		{"key:session", "*:session", true},
		{":session", "*:session", true},
		{"key:data", "*:session", false},

		// Middle wildcard patterns
		{"user:123:session", "user:*:session", true},
		{"user::session", "user:*:session", true},
		{"session:123:data", "user:*:session", false},

		// Exact match
		{"user:123", "user:123", true},
		{"user:123", "user:124", false},
	}

	for _, tt := range tests {
		t.Run(tt.key+"_"+tt.pattern, func(t *testing.T) {
			result := matchPattern(tt.key, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.key, tt.pattern, result, tt.expected)
			}
		})
	}
}
