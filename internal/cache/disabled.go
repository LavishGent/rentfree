package cache

import (
	"context"

	"gitlab.com/appliedsystems/experimental/users/ddavis/stuff/rentfree/internal/types"
)

type DisabledMemoryCache struct{}

func NewDisabledMemoryCache() *DisabledMemoryCache {
	return &DisabledMemoryCache{}
}

func (c *DisabledMemoryCache) Name() string                                             { return "memory-disabled" }
func (c *DisabledMemoryCache) IsAvailable() bool                                        { return false }
func (c *DisabledMemoryCache) Close() error                                             { return nil }
func (c *DisabledMemoryCache) EntryCount() int                                          { return 0 }
func (c *DisabledMemoryCache) Size() int64                                              { return 0 }
func (c *DisabledMemoryCache) MaxSize() int64                                           { return 0 }
func (c *DisabledMemoryCache) UsagePercentage() float64                                 { return 0 }
func (c *DisabledMemoryCache) HitRatio() float64                                        { return 0 }
func (c *DisabledMemoryCache) Stats() types.MemoryCacheStats                            { return types.MemoryCacheStats{} }
func (c *DisabledMemoryCache) Clear(ctx context.Context) error                          { return nil }
func (c *DisabledMemoryCache) ClearByPattern(ctx context.Context, pattern string) error { return nil }

func (c *DisabledMemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	return nil, types.ErrCacheMiss
}

func (c *DisabledMemoryCache) Set(ctx context.Context, key string, value []byte, opts *types.CacheOptions) error {
	return nil
}

func (c *DisabledMemoryCache) Delete(ctx context.Context, key string) error {
	return nil
}

func (c *DisabledMemoryCache) Contains(ctx context.Context, key string) (bool, error) {
	return false, nil
}

type DisabledRedisCache struct{}

func NewDisabledRedisCache() *DisabledRedisCache {
	return &DisabledRedisCache{}
}

func (c *DisabledRedisCache) Name() string                                             { return "redis-disabled" }
func (c *DisabledRedisCache) IsAvailable() bool                                        { return false }
func (c *DisabledRedisCache) Close() error                                             { return nil }
func (c *DisabledRedisCache) PendingWrites() int                                       { return 0 }
func (c *DisabledRedisCache) DroppedWrites() int64                                     { return 0 }
func (c *DisabledRedisCache) Clear(ctx context.Context) error                          { return nil }
func (c *DisabledRedisCache) ClearByPattern(ctx context.Context, pattern string) error { return nil }

func (c *DisabledRedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	return nil, types.ErrRedisUnavailable
}

func (c *DisabledRedisCache) Set(ctx context.Context, key string, value []byte, opts *types.CacheOptions) error {
	return nil
}

func (c *DisabledRedisCache) Delete(ctx context.Context, key string) error {
	return nil
}

func (c *DisabledRedisCache) Contains(ctx context.Context, key string) (bool, error) {
	return false, nil
}

func (c *DisabledRedisCache) GetMany(ctx context.Context, keys []string) (map[string][]byte, error) {
	return make(map[string][]byte), nil
}

func (c *DisabledRedisCache) SetMany(ctx context.Context, items map[string][]byte, opts *types.CacheOptions) error {
	return nil
}

var _ types.MemoryCacheLayer = (*DisabledMemoryCache)(nil)
var _ types.RedisCacheLayer = (*DisabledRedisCache)(nil)
