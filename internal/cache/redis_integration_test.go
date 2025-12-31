package cache

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LavishGent/rentfree/internal/config"
	"github.com/LavishGent/rentfree/internal/types"
)

// =============================================================================
// Test Option Helpers (to avoid importing pkg/rentfree in internal tests)
// =============================================================================

// withMemoryOnly returns an option for memory-only operations.
func withMemoryOnly() types.Option {
	return func(o *types.CacheOptions) {
		o.Level = types.LevelMemoryOnly
	}
}

// withRedisOnly returns an option for Redis-only operations.
func withRedisOnly() types.Option {
	return func(o *types.CacheOptions) {
		o.Level = types.LevelRedisOnly
	}
}

// withAll returns an option for operations on all cache layers.
func withAll() types.Option {
	return func(o *types.CacheOptions) {
		o.Level = types.LevelAll
	}
}

// redisTestAddress returns the Redis address to use for tests.
// It checks the REDIS_TEST_ADDRESS environment variable first,
// then falls back to localhost:6379.
func redisTestAddress() string {
	if addr := os.Getenv("REDIS_TEST_ADDRESS"); addr != "" {
		return addr
	}
	return "localhost:6379"
}

// skipIfRedisUnavailable skips the test if Redis is not available.
func skipIfRedisUnavailable(t *testing.T) *RedisCache {
	t.Helper()

	cfg := config.RedisConfig{
		Enabled:          true,
		Address:          redisTestAddress(),
		KeyPrefix:        "rentfree:test:",
		DefaultTTL:       5 * time.Minute,
		PoolSize:         5,
		MinIdleConns:     1,
		DialTimeout:      2 * time.Second,
		ReadTimeout:      1 * time.Second,
		WriteTimeout:     1 * time.Second,
		PoolTimeout:      2 * time.Second,
		MaxPendingWrites: 100,
	}

	rc, err := NewRedisCache(cfg, nil)
	if err != nil {
		t.Skipf("Redis unavailable: %v", err)
	}

	if !rc.IsAvailable() {
		rc.Close()
		t.Skip("Redis is not available")
	}

	// Clean up test keys before running tests
	ctx := context.Background()
	_ = rc.Clear(ctx)

	return rc
}

// newTestManagerWithRedis creates a manager with Redis enabled for testing.
func newTestManagerWithRedis(t *testing.T) *Manager {
	t.Helper()

	cfg := config.ForTesting()
	cfg.Redis.Enabled = true
	cfg.Redis.Address = redisTestAddress()
	cfg.Redis.KeyPrefix = "rentfree:test:manager:"
	cfg.Defaults.Level = "all"

	mgr, err := NewManager(cfg, nil)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	if !mgr.IsRedisAvailable() {
		mgr.Close()
		t.Skip("Redis is not available")
	}

	// Clean up test keys
	ctx := context.Background()
	_ = mgr.Clear(ctx, types.LevelAll)

	return mgr
}

// =============================================================================
// Redis Cache Layer Tests
// =============================================================================

func TestRedisCacheGet(t *testing.T) {
	rc := skipIfRedisUnavailable(t)
	defer rc.Close()
	ctx := context.Background()

	t.Run("returns cache miss for non-existent key", func(t *testing.T) {
		_, err := rc.Get(ctx, "non-existent-key")
		assert.ErrorIs(t, err, types.ErrCacheMiss)
	})

	t.Run("retrieves previously set value", func(t *testing.T) {
		key := "test-get-key"
		value := []byte(`{"name":"test"}`)

		err := rc.Set(ctx, key, value, nil)
		require.NoError(t, err)

		got, err := rc.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, got)
	})
}

func TestRedisCacheSet(t *testing.T) {
	rc := skipIfRedisUnavailable(t)
	defer rc.Close()
	ctx := context.Background()

	t.Run("sets value successfully", func(t *testing.T) {
		key := "test-set-key"
		value := []byte("test-value")

		err := rc.Set(ctx, key, value, nil)
		require.NoError(t, err)

		got, err := rc.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, got)
	})

	t.Run("overwrites existing value", func(t *testing.T) {
		key := "test-overwrite-key"
		value1 := []byte("value1")
		value2 := []byte("value2")

		err := rc.Set(ctx, key, value1, nil)
		require.NoError(t, err)

		err = rc.Set(ctx, key, value2, nil)
		require.NoError(t, err)

		got, err := rc.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value2, got)
	})

	t.Run("respects TTL option", func(t *testing.T) {
		key := "test-ttl-key"
		value := []byte("ttl-value")
		opts := &types.CacheOptions{TTL: 100 * time.Millisecond}

		err := rc.Set(ctx, key, value, opts)
		require.NoError(t, err)

		// Verify key exists
		got, err := rc.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, got)

		// Wait for TTL to expire
		time.Sleep(150 * time.Millisecond)

		// Verify key expired
		_, err = rc.Get(ctx, key)
		assert.ErrorIs(t, err, types.ErrCacheMiss)
	})
}

func TestRedisCacheDelete(t *testing.T) {
	rc := skipIfRedisUnavailable(t)
	defer rc.Close()
	ctx := context.Background()

	t.Run("deletes existing key", func(t *testing.T) {
		key := "test-delete-key"
		value := []byte("to-be-deleted")

		err := rc.Set(ctx, key, value, nil)
		require.NoError(t, err)

		err = rc.Delete(ctx, key)
		require.NoError(t, err)

		_, err = rc.Get(ctx, key)
		assert.ErrorIs(t, err, types.ErrCacheMiss)
	})

	t.Run("deleting non-existent key succeeds", func(t *testing.T) {
		err := rc.Delete(ctx, "non-existent-delete-key")
		assert.NoError(t, err)
	})
}

func TestRedisCacheContains(t *testing.T) {
	rc := skipIfRedisUnavailable(t)
	defer rc.Close()
	ctx := context.Background()

	t.Run("returns true for existing key", func(t *testing.T) {
		key := "test-contains-key"
		value := []byte("exists")

		err := rc.Set(ctx, key, value, nil)
		require.NoError(t, err)

		exists, err := rc.Contains(ctx, key)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("returns false for non-existent key", func(t *testing.T) {
		exists, err := rc.Contains(ctx, "non-existent-contains-key")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestRedisCacheGetMany(t *testing.T) {
	rc := skipIfRedisUnavailable(t)
	defer rc.Close()
	ctx := context.Background()

	t.Run("returns all found keys", func(t *testing.T) {
		// Set up test data
		items := map[string][]byte{
			"mget-key1": []byte("value1"),
			"mget-key2": []byte("value2"),
			"mget-key3": []byte("value3"),
		}

		for k, v := range items {
			err := rc.Set(ctx, k, v, nil)
			require.NoError(t, err)
		}

		// Get multiple keys (including one that doesn't exist)
		keys := []string{"mget-key1", "mget-key2", "mget-key3", "mget-key-missing"}
		results, err := rc.GetMany(ctx, keys)
		require.NoError(t, err)

		assert.Len(t, results, 3)
		assert.Equal(t, []byte("value1"), results["mget-key1"])
		assert.Equal(t, []byte("value2"), results["mget-key2"])
		assert.Equal(t, []byte("value3"), results["mget-key3"])
		_, exists := results["mget-key-missing"]
		assert.False(t, exists)
	})

	t.Run("returns empty map for empty keys", func(t *testing.T) {
		results, err := rc.GetMany(ctx, []string{})
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestRedisCacheSetMany(t *testing.T) {
	rc := skipIfRedisUnavailable(t)
	defer rc.Close()
	ctx := context.Background()

	t.Run("sets multiple keys", func(t *testing.T) {
		items := map[string][]byte{
			"mset-key1": []byte("value1"),
			"mset-key2": []byte("value2"),
			"mset-key3": []byte("value3"),
		}

		err := rc.SetMany(ctx, items, nil)
		require.NoError(t, err)

		// Verify all keys were set
		for k, v := range items {
			got, err := rc.Get(ctx, k)
			require.NoError(t, err)
			assert.Equal(t, v, got)
		}
	})

	t.Run("empty map succeeds", func(t *testing.T) {
		err := rc.SetMany(ctx, map[string][]byte{}, nil)
		assert.NoError(t, err)
	})
}

func TestRedisCacheClear(t *testing.T) {
	rc := skipIfRedisUnavailable(t)
	defer rc.Close()
	ctx := context.Background()

	t.Run("clears all entries with prefix", func(t *testing.T) {
		// Set up test data
		for i := 0; i < 10; i++ {
			key := "clear-key-" + string(rune('a'+i))
			err := rc.Set(ctx, key, []byte("value"), nil)
			require.NoError(t, err)
		}

		// Clear all
		err := rc.Clear(ctx)
		require.NoError(t, err)

		// Verify all cleared
		for i := 0; i < 10; i++ {
			key := "clear-key-" + string(rune('a'+i))
			_, err := rc.Get(ctx, key)
			assert.ErrorIs(t, err, types.ErrCacheMiss)
		}
	})
}

func TestRedisCacheClearByPattern(t *testing.T) {
	rc := skipIfRedisUnavailable(t)
	defer rc.Close()
	ctx := context.Background()

	t.Run("clears keys matching pattern", func(t *testing.T) {
		// Set up test data with different prefixes
		testKeys := []string{
			"user:1:profile",
			"user:2:profile",
			"user:3:profile",
			"session:abc",
			"session:def",
		}

		for _, key := range testKeys {
			err := rc.Set(ctx, key, []byte("value"), nil)
			require.NoError(t, err)
		}

		// Clear only user keys
		err := rc.ClearByPattern(ctx, "user:*")
		require.NoError(t, err)

		// Verify user keys are cleared
		for _, key := range testKeys[:3] {
			_, err := rc.Get(ctx, key)
			assert.ErrorIs(t, err, types.ErrCacheMiss, "key %s should be cleared", key)
		}

		// Verify session keys still exist
		for _, key := range testKeys[3:] {
			_, err := rc.Get(ctx, key)
			assert.NoError(t, err, "key %s should still exist", key)
		}
	})
}

func TestRedisCacheFireAndForget(t *testing.T) {
	rc := skipIfRedisUnavailable(t)
	defer rc.Close()
	ctx := context.Background()

	t.Run("async write is eventually persisted", func(t *testing.T) {
		key := "async-key"
		value := []byte("async-value")
		opts := &types.CacheOptions{
			TTL:           5 * time.Minute,
			FireAndForget: true,
		}

		err := rc.Set(ctx, key, value, opts)
		require.NoError(t, err)

		// Wait for async write to complete
		time.Sleep(100 * time.Millisecond)

		// Verify value was written
		got, err := rc.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, got)
	})

	t.Run("multiple async writes are processed", func(t *testing.T) {
		opts := &types.CacheOptions{
			TTL:           5 * time.Minute,
			FireAndForget: true,
		}

		// Queue multiple writes
		for i := 0; i < 10; i++ {
			key := "async-batch-" + string(rune('a'+i))
			value := []byte("value-" + string(rune('a'+i)))
			err := rc.Set(ctx, key, value, opts)
			require.NoError(t, err)
		}

		// Wait for async writes to complete
		time.Sleep(200 * time.Millisecond)

		// Verify all values were written
		for i := 0; i < 10; i++ {
			key := "async-batch-" + string(rune('a'+i))
			expected := []byte("value-" + string(rune('a'+i)))
			got, err := rc.Get(ctx, key)
			require.NoError(t, err)
			assert.Equal(t, expected, got)
		}
	})
}

func TestRedisCachePingAndReconnect(t *testing.T) {
	rc := skipIfRedisUnavailable(t)
	defer rc.Close()
	ctx := context.Background()

	t.Run("ping succeeds when connected", func(t *testing.T) {
		err := rc.Ping(ctx)
		assert.NoError(t, err)
	})

	t.Run("reconnect succeeds when server is available", func(t *testing.T) {
		err := rc.Reconnect(ctx)
		assert.NoError(t, err)
		assert.True(t, rc.IsAvailable())
	})
}

func TestRedisCacheConcurrency(t *testing.T) {
	rc := skipIfRedisUnavailable(t)
	defer rc.Close()
	ctx := context.Background()

	t.Run("handles concurrent reads and writes", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 50
		numOps := 20

		// Initialize test key
		err := rc.Set(ctx, "concurrent-key", []byte("initial"), nil)
		require.NoError(t, err)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOps; j++ {
					key := "concurrent-key"

					// Mix of reads and writes
					if j%2 == 0 {
						_, _ = rc.Get(ctx, key)
					} else {
						_ = rc.Set(ctx, key, []byte("updated"), nil)
					}
				}
			}(i)
		}

		wg.Wait()

		// Verify key is still accessible
		_, err = rc.Get(ctx, "concurrent-key")
		assert.NoError(t, err)
	})
}

// =============================================================================
// Manager with Redis Integration Tests
// =============================================================================

func TestManagerWithRedisGet(t *testing.T) {
	mgr := newTestManagerWithRedis(t)
	defer mgr.Close()
	ctx := context.Background()

	t.Run("retrieves value from Redis when memory misses", func(t *testing.T) {
		key := "redis-only-key"
		value := "redis-value"

		// Set with Redis-only level
		err := mgr.Set(ctx, key, value, withRedisOnly())
		require.NoError(t, err)

		// Get should fetch from Redis
		var result string
		err = mgr.Get(ctx, key, &result)
		require.NoError(t, err)
		assert.Equal(t, value, result)
	})

	t.Run("populates memory cache from Redis on get", func(t *testing.T) {
		key := "populate-memory-key"
		value := "populate-value"

		// Set only in Redis
		err := mgr.Set(ctx, key, value, withRedisOnly())
		require.NoError(t, err)

		// First get - fetches from Redis, populates memory (async)
		var result1 string
		err = mgr.Get(ctx, key, &result1, withAll())
		require.NoError(t, err)
		assert.Equal(t, value, result1)

		// Wait for async memory population to complete
		time.Sleep(50 * time.Millisecond)

		// Second get with memory-only should hit memory cache
		var result2 string
		err = mgr.Get(ctx, key, &result2, withMemoryOnly())
		require.NoError(t, err)
		assert.Equal(t, value, result2)
	})
}

func TestManagerWithRedisSet(t *testing.T) {
	mgr := newTestManagerWithRedis(t)
	defer mgr.Close()
	ctx := context.Background()

	t.Run("sets value in both layers with LevelAll", func(t *testing.T) {
		key := "both-layers-key"
		value := "both-layers-value"

		err := mgr.Set(ctx, key, value, withAll())
		require.NoError(t, err)

		// Verify in memory-only
		var memResult string
		err = mgr.Get(ctx, key, &memResult, withMemoryOnly())
		require.NoError(t, err)
		assert.Equal(t, value, memResult)

		// Verify in Redis-only
		var redisResult string
		err = mgr.Get(ctx, key, &redisResult, withRedisOnly())
		require.NoError(t, err)
		assert.Equal(t, value, redisResult)
	})

	t.Run("sets value only in Redis with LevelRedisOnly", func(t *testing.T) {
		key := "redis-only-set-key"
		value := "redis-only-set-value"

		err := mgr.Set(ctx, key, value, withRedisOnly())
		require.NoError(t, err)

		// Should miss in memory
		var memResult string
		err = mgr.Get(ctx, key, &memResult, withMemoryOnly())
		assert.ErrorIs(t, err, types.ErrCacheMiss)

		// Should hit in Redis
		var redisResult string
		err = mgr.Get(ctx, key, &redisResult, withRedisOnly())
		require.NoError(t, err)
		assert.Equal(t, value, redisResult)
	})
}

func TestManagerWithRedisDelete(t *testing.T) {
	mgr := newTestManagerWithRedis(t)
	defer mgr.Close()
	ctx := context.Background()

	t.Run("deletes from both layers", func(t *testing.T) {
		key := "delete-both-key"
		value := "delete-both-value"

		// Set in both layers
		err := mgr.Set(ctx, key, value, withAll())
		require.NoError(t, err)

		// Delete
		err = mgr.Delete(ctx, key)
		require.NoError(t, err)

		// Verify deleted from memory
		var memResult string
		err = mgr.Get(ctx, key, &memResult, withMemoryOnly())
		assert.ErrorIs(t, err, types.ErrCacheMiss)

		// Verify deleted from Redis
		var redisResult string
		err = mgr.Get(ctx, key, &redisResult, withRedisOnly())
		assert.ErrorIs(t, err, types.ErrCacheMiss)
	})
}

func TestManagerWithRedisGetOrCreate(t *testing.T) {
	mgr := newTestManagerWithRedis(t)
	defer mgr.Close()
	ctx := context.Background()

	t.Run("fetches from Redis before calling factory", func(t *testing.T) {
		key := "getorcreate-redis-key"
		value := "cached-in-redis"
		factoryCalled := false

		// Pre-populate Redis only
		err := mgr.Set(ctx, key, value, withRedisOnly())
		require.NoError(t, err)

		// GetOrCreate should find in Redis, not call factory
		var result string
		err = mgr.GetOrCreate(ctx, key, &result, func() (any, error) {
			factoryCalled = true
			return "factory-value", nil
		}, withAll())
		require.NoError(t, err)

		assert.Equal(t, value, result)
		assert.False(t, factoryCalled, "factory should not be called when value exists in Redis")
	})

	t.Run("stores in both layers when factory is called", func(t *testing.T) {
		key := "getorcreate-both-key"
		expectedValue := "created-value"

		var result string
		err := mgr.GetOrCreate(ctx, key, &result, func() (any, error) {
			return expectedValue, nil
		}, withAll())
		require.NoError(t, err)
		assert.Equal(t, expectedValue, result)

		// Verify in memory
		var memResult string
		err = mgr.Get(ctx, key, &memResult, withMemoryOnly())
		require.NoError(t, err)
		assert.Equal(t, expectedValue, memResult)

		// Verify in Redis
		var redisResult string
		err = mgr.Get(ctx, key, &redisResult, withRedisOnly())
		require.NoError(t, err)
		assert.Equal(t, expectedValue, redisResult)
	})
}

func TestManagerWithRedisGetMany(t *testing.T) {
	mgr := newTestManagerWithRedis(t)
	defer mgr.Close()
	ctx := context.Background()

	t.Run("retrieves from both layers", func(t *testing.T) {
		// Set some keys in memory only
		err := mgr.Set(ctx, "many-mem-1", "mem-value-1", withMemoryOnly())
		require.NoError(t, err)

		// Set some keys in Redis only
		err = mgr.Set(ctx, "many-redis-1", "redis-value-1", withRedisOnly())
		require.NoError(t, err)

		// Set some keys in both
		err = mgr.Set(ctx, "many-both-1", "both-value-1", withAll())
		require.NoError(t, err)

		// Get all keys - GetMany returns raw bytes
		keys := []string{"many-mem-1", "many-redis-1", "many-both-1", "many-missing"}
		results, err := mgr.GetMany(ctx, keys)
		require.NoError(t, err)

		// Note: GetMany returns serialized JSON bytes
		assert.Contains(t, string(results["many-mem-1"]), "mem-value-1")
		assert.Contains(t, string(results["many-redis-1"]), "redis-value-1")
		assert.Contains(t, string(results["many-both-1"]), "both-value-1")
		_, exists := results["many-missing"]
		assert.False(t, exists)
	})
}

func TestManagerWithRedisSetMany(t *testing.T) {
	mgr := newTestManagerWithRedis(t)
	defer mgr.Close()
	ctx := context.Background()

	t.Run("sets multiple keys in both layers", func(t *testing.T) {
		items := map[string]any{
			"setmany-key1": "value1",
			"setmany-key2": "value2",
			"setmany-key3": "value3",
		}

		err := mgr.SetMany(ctx, items, withAll())
		require.NoError(t, err)

		// Verify in memory
		for k, v := range items {
			var result string
			err := mgr.Get(ctx, k, &result, withMemoryOnly())
			require.NoError(t, err)
			assert.Equal(t, v, result)
		}

		// Verify in Redis
		for k, v := range items {
			var result string
			err := mgr.Get(ctx, k, &result, withRedisOnly())
			require.NoError(t, err)
			assert.Equal(t, v, result)
		}
	})
}

func TestManagerWithRedisClear(t *testing.T) {
	mgr := newTestManagerWithRedis(t)
	defer mgr.Close()
	ctx := context.Background()

	t.Run("clears both layers", func(t *testing.T) {
		// Set keys in both layers
		for i := 0; i < 5; i++ {
			key := "clear-test-" + string(rune('a'+i))
			err := mgr.Set(ctx, key, "value", withAll())
			require.NoError(t, err)
		}

		// Clear
		err := mgr.Clear(ctx, types.LevelAll)
		require.NoError(t, err)

		// Verify cleared from memory
		for i := 0; i < 5; i++ {
			key := "clear-test-" + string(rune('a'+i))
			var result string
			err := mgr.Get(ctx, key, &result, withMemoryOnly())
			assert.ErrorIs(t, err, types.ErrCacheMiss)
		}

		// Verify cleared from Redis
		for i := 0; i < 5; i++ {
			key := "clear-test-" + string(rune('a'+i))
			var result string
			err := mgr.Get(ctx, key, &result, withRedisOnly())
			assert.ErrorIs(t, err, types.ErrCacheMiss)
		}
	})
}

func TestManagerWithRedisHealth(t *testing.T) {
	mgr := newTestManagerWithRedis(t)
	defer mgr.Close()
	ctx := context.Background()

	t.Run("reports healthy when Redis is available", func(t *testing.T) {
		health, err := mgr.Health(ctx)
		require.NoError(t, err)

		assert.Equal(t, types.HealthStatusHealthy, health.Status)
		assert.True(t, health.Redis.Available)
		assert.True(t, health.Memory.Available)
	})

	t.Run("IsRedisAvailable returns true", func(t *testing.T) {
		assert.True(t, mgr.IsRedisAvailable())
	})
}

// =============================================================================
// Graceful Degradation Tests
// =============================================================================

func TestGracefulDegradationToMemory(t *testing.T) {
	// Create config with invalid Redis address to simulate unavailability
	cfg := config.ForTesting()
	cfg.Redis.Enabled = true
	cfg.Redis.Address = "localhost:59999" // Invalid port
	cfg.Redis.DialTimeout = 100 * time.Millisecond
	cfg.Defaults.Level = "all"

	mgr, err := NewManager(cfg, nil)
	require.NoError(t, err)
	defer mgr.Close()

	ctx := context.Background()

	t.Run("continues to work with memory when Redis unavailable", func(t *testing.T) {
		// Redis should be unavailable
		assert.False(t, mgr.IsRedisAvailable())

		// But memory operations should still work
		key := "degraded-key"
		value := "degraded-value"

		// Set should succeed (writes to memory)
		err := mgr.Set(ctx, key, value)
		require.NoError(t, err)

		// Get should succeed (reads from memory)
		var result string
		err = mgr.Get(ctx, key, &result)
		require.NoError(t, err)
		assert.Equal(t, value, result)
	})

	t.Run("health reports degraded status", func(t *testing.T) {
		health, err := mgr.Health(ctx)
		require.NoError(t, err)

		assert.Equal(t, types.HealthStatusDegraded, health.Status)
		assert.False(t, health.Redis.Available)
		assert.True(t, health.Memory.Available)
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkRedisCacheGet(b *testing.B) {
	cfg := config.RedisConfig{
		Enabled:          true,
		Address:          redisTestAddress(),
		KeyPrefix:        "rentfree:bench:",
		DefaultTTL:       5 * time.Minute,
		PoolSize:         10,
		MinIdleConns:     2,
		DialTimeout:      2 * time.Second,
		ReadTimeout:      1 * time.Second,
		WriteTimeout:     1 * time.Second,
		PoolTimeout:      2 * time.Second,
		MaxPendingWrites: 1000,
	}

	rc, err := NewRedisCache(cfg, nil)
	if err != nil || !rc.IsAvailable() {
		b.Skip("Redis unavailable")
	}
	defer rc.Close()

	ctx := context.Background()
	key := "bench-key"
	value := []byte(`{"id":123,"name":"benchmark test","data":"some test data for benchmarking"}`)

	_ = rc.Set(ctx, key, value, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = rc.Get(ctx, key)
	}
}

func BenchmarkRedisCacheSet(b *testing.B) {
	cfg := config.RedisConfig{
		Enabled:          true,
		Address:          redisTestAddress(),
		KeyPrefix:        "rentfree:bench:",
		DefaultTTL:       5 * time.Minute,
		PoolSize:         10,
		MinIdleConns:     2,
		DialTimeout:      2 * time.Second,
		ReadTimeout:      1 * time.Second,
		WriteTimeout:     1 * time.Second,
		PoolTimeout:      2 * time.Second,
		MaxPendingWrites: 1000,
	}

	rc, err := NewRedisCache(cfg, nil)
	if err != nil || !rc.IsAvailable() {
		b.Skip("Redis unavailable")
	}
	defer rc.Close()

	ctx := context.Background()
	value := []byte(`{"id":123,"name":"benchmark test","data":"some test data for benchmarking"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "bench-set-" + string(rune(i%26+'a'))
		_ = rc.Set(ctx, key, value, nil)
	}
}

func BenchmarkRedisCacheSetAsync(b *testing.B) {
	cfg := config.RedisConfig{
		Enabled:          true,
		Address:          redisTestAddress(),
		KeyPrefix:        "rentfree:bench:",
		DefaultTTL:       5 * time.Minute,
		PoolSize:         10,
		MinIdleConns:     2,
		DialTimeout:      2 * time.Second,
		ReadTimeout:      1 * time.Second,
		WriteTimeout:     1 * time.Second,
		PoolTimeout:      2 * time.Second,
		MaxPendingWrites: 10000,
	}

	rc, err := NewRedisCache(cfg, nil)
	if err != nil || !rc.IsAvailable() {
		b.Skip("Redis unavailable")
	}
	defer rc.Close()

	ctx := context.Background()
	value := []byte(`{"id":123,"name":"benchmark test","data":"some test data for benchmarking"}`)
	opts := &types.CacheOptions{TTL: 5 * time.Minute, FireAndForget: true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "bench-async-" + string(rune(i%26+'a'))
		_ = rc.Set(ctx, key, value, opts)
	}
}

func BenchmarkManagerWithRedisConcurrent(b *testing.B) {
	cfg := config.ForTesting()
	cfg.Redis.Enabled = true
	cfg.Redis.Address = redisTestAddress()
	cfg.Redis.KeyPrefix = "rentfree:bench:manager:"
	cfg.Defaults.Level = "all"

	mgr, err := NewManager(cfg, nil)
	if err != nil {
		b.Fatalf("failed to create manager: %v", err)
	}
	if !mgr.IsRedisAvailable() {
		b.Skip("Redis unavailable")
	}
	defer mgr.Close()

	ctx := context.Background()

	// Pre-populate some keys
	for i := 0; i < 100; i++ {
		key := "bench-concurrent-" + string(rune(i%26+'a'))
		_ = mgr.Set(ctx, key, "benchmark-value")
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "bench-concurrent-" + string(rune(i%26+'a'))
			if i%2 == 0 {
				var result string
				_ = mgr.Get(ctx, key, &result)
			} else {
				_ = mgr.Set(ctx, key, "updated-value")
			}
			i++
		}
	})
}

// =============================================================================
// Health Check Tests
// =============================================================================

func TestRedisHealthCheck(t *testing.T) {
	t.Run("health check worker starts and stops cleanly", func(t *testing.T) {
		cfg := config.RedisConfig{
			Enabled:             true,
			Address:             redisTestAddress(),
			KeyPrefix:           "rentfree:test:healthcheck:",
			DefaultTTL:          5 * time.Minute,
			PoolSize:            5,
			MinIdleConns:        1,
			DialTimeout:         2 * time.Second,
			ReadTimeout:         1 * time.Second,
			WriteTimeout:        1 * time.Second,
			PoolTimeout:         2 * time.Second,
			MaxPendingWrites:    100,
			HealthCheckInterval: 100 * time.Millisecond, // Fast interval for testing
		}

		rc, err := NewRedisCache(cfg, nil)
		require.NoError(t, err)

		// Let health check run a few times
		time.Sleep(350 * time.Millisecond)

		// Close should not hang or panic
		err = rc.Close()
		assert.NoError(t, err)
	})

	t.Run("health check restores connection after recovery", func(t *testing.T) {
		rc := skipIfRedisUnavailable(t)
		defer rc.Close()

		// Simulate disconnection by setting connected to false
		rc.connected.Store(false)
		assert.False(t, rc.IsAvailable())

		// Perform health check manually
		rc.performHealthCheck()

		// Should be connected again since Redis is available
		assert.True(t, rc.IsAvailable())
	})

	t.Run("health check with disabled interval does not start worker", func(t *testing.T) {
		cfg := config.RedisConfig{
			Enabled:             true,
			Address:             redisTestAddress(),
			KeyPrefix:           "rentfree:test:healthcheck:",
			DefaultTTL:          5 * time.Minute,
			PoolSize:            5,
			MinIdleConns:        1,
			DialTimeout:         2 * time.Second,
			ReadTimeout:         1 * time.Second,
			WriteTimeout:        1 * time.Second,
			PoolTimeout:         2 * time.Second,
			MaxPendingWrites:    100,
			HealthCheckInterval: 0, // Disabled
		}

		rc, err := NewRedisCache(cfg, nil)
		require.NoError(t, err)

		// Close should work without issues even though health check was disabled
		err = rc.Close()
		assert.NoError(t, err)
	})
}
