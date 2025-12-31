package cache

import (
	"context"

	"github.com/LavishGent/rentfree/internal/types"
)

// DisabledMemoryCache is a no-op memory cache implementation.
type DisabledMemoryCache struct{}

// NewDisabledMemoryCache creates a new disabled memory cache.
func NewDisabledMemoryCache() *DisabledMemoryCache {
	return &DisabledMemoryCache{}
}

// Name returns the cache layer name.
func (c *DisabledMemoryCache) Name() string { return "memory-disabled" }

// IsAvailable returns false as this cache is disabled.
func (c *DisabledMemoryCache) IsAvailable() bool { return false }

// Close does nothing as this cache is disabled.
func (c *DisabledMemoryCache) Close() error { return nil }

// EntryCount returns 0 as this cache is disabled.
func (c *DisabledMemoryCache) EntryCount() int { return 0 }

// Size returns 0 as this cache is disabled.
func (c *DisabledMemoryCache) Size() int64 { return 0 }

// MaxSize returns 0 as this cache is disabled.
func (c *DisabledMemoryCache) MaxSize() int64 { return 0 }

// UsagePercentage returns 0 as this cache is disabled.
func (c *DisabledMemoryCache) UsagePercentage() float64 { return 0 }

// HitRatio returns 0 as this cache is disabled.
func (c *DisabledMemoryCache) HitRatio() float64 { return 0 }

// Stats returns empty statistics as this cache is disabled.
func (c *DisabledMemoryCache) Stats() types.MemoryCacheStats { return types.MemoryCacheStats{} }

// Clear does nothing as this cache is disabled.
func (c *DisabledMemoryCache) Clear(ctx context.Context) error { return nil }

// ClearByPattern does nothing as this cache is disabled.
func (c *DisabledMemoryCache) ClearByPattern(ctx context.Context, pattern string) error { return nil }

// Get returns ErrCacheMiss as this cache is disabled.
func (c *DisabledMemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	return nil, types.ErrCacheMiss
}

// Set does nothing as this cache is disabled.
func (c *DisabledMemoryCache) Set(ctx context.Context, key string, value []byte, opts *types.CacheOptions) error {
	return nil
}

// Delete does nothing as this cache is disabled.
func (c *DisabledMemoryCache) Delete(ctx context.Context, key string) error {
	return nil
}

// Contains returns false as this cache is disabled.
func (c *DisabledMemoryCache) Contains(ctx context.Context, key string) (bool, error) {
	return false, nil
}

// DisabledRedisCache is a no-op Redis cache implementation.
type DisabledRedisCache struct{}

// NewDisabledRedisCache creates a new disabled Redis cache.
func NewDisabledRedisCache() *DisabledRedisCache {
	return &DisabledRedisCache{}
}

// Name returns the cache layer name.
func (c *DisabledRedisCache) Name() string { return "redis-disabled" }

// IsAvailable returns false as this cache is disabled.
func (c *DisabledRedisCache) IsAvailable() bool { return false }

// Close does nothing as this cache is disabled.
func (c *DisabledRedisCache) Close() error { return nil }

// PendingWrites returns 0 as this cache is disabled.
func (c *DisabledRedisCache) PendingWrites() int { return 0 }

// DroppedWrites returns 0 as this cache is disabled.
func (c *DisabledRedisCache) DroppedWrites() int64 { return 0 }

// Clear does nothing as this cache is disabled.
func (c *DisabledRedisCache) Clear(ctx context.Context) error { return nil }

// ClearByPattern does nothing as this cache is disabled.
func (c *DisabledRedisCache) ClearByPattern(ctx context.Context, pattern string) error { return nil }

// Get returns ErrRedisUnavailable as this cache is disabled.
func (c *DisabledRedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	return nil, types.ErrRedisUnavailable
}

// Set does nothing as this cache is disabled.
func (c *DisabledRedisCache) Set(ctx context.Context, key string, value []byte, opts *types.CacheOptions) error {
	return nil
}

// Delete does nothing as this cache is disabled.
func (c *DisabledRedisCache) Delete(ctx context.Context, key string) error {
	return nil
}

// Contains returns false as this cache is disabled.
func (c *DisabledRedisCache) Contains(ctx context.Context, key string) (bool, error) {
	return false, nil
}

// GetMany returns an empty map as this cache is disabled.
func (c *DisabledRedisCache) GetMany(ctx context.Context, keys []string) (map[string][]byte, error) {
	return make(map[string][]byte), nil
}

// SetMany does nothing as this cache is disabled.
func (c *DisabledRedisCache) SetMany(ctx context.Context, items map[string][]byte, opts *types.CacheOptions) error {
	return nil
}

var _ types.MemoryCacheLayer = (*DisabledMemoryCache)(nil)
var _ types.RedisCacheLayer = (*DisabledRedisCache)(nil)
