package cache

import (
	"context"
	"log/slog"
	"strings"
	"sync/atomic"

	"github.com/allegro/bigcache/v3"

	"github.com/darrell-green/rentfree/internal/config"
	"github.com/darrell-green/rentfree/internal/types"
)

type MemoryCache struct {
	cache  *bigcache.BigCache
	config config.MemoryConfig
	logger *slog.Logger

	hits      atomic.Int64
	misses    atomic.Int64
	sets      atomic.Int64
	deletes   atomic.Int64
	evictions atomic.Int64

	closed atomic.Bool
}

func NewMemoryCache(cfg config.MemoryConfig, logger *slog.Logger) (*MemoryCache, error) {
	if logger == nil {
		logger = slog.Default()
	}

	mc := &MemoryCache{
		config: cfg,
		logger: logger.With("component", "memory-cache"),
	}

	bcConfig := bigcache.Config{
		Shards:             cfg.Shards,
		LifeWindow:         cfg.DefaultTTL,
		CleanWindow:        cfg.CleanupInterval,
		MaxEntriesInWindow: 1000 * 10 * 60, // Estimated entries in LifeWindow
		MaxEntrySize:       cfg.MaxEntrySize,
		HardMaxCacheSize:   cfg.MaxSizeMB,
		Verbose:            false,
		Logger:             &bigcacheLogger{logger: logger},
		OnRemoveWithReason: func(key string, entry []byte, reason bigcache.RemoveReason) {
			if reason == bigcache.NoSpace || reason == bigcache.Expired {
				mc.evictions.Add(1)
			}
		},
	}

	bc, err := bigcache.New(context.Background(), bcConfig)
	if err != nil {
		return nil, err
	}

	mc.cache = bc
	return mc, nil
}

func (c *MemoryCache) Name() string {
	return "memory"
}

func (c *MemoryCache) IsAvailable() bool {
	return !c.closed.Load()
}

func (c *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	if c.closed.Load() {
		return nil, types.ErrClosed
	}

	data, err := c.cache.Get(key)
	if err != nil {
		if err == bigcache.ErrEntryNotFound {
			c.misses.Add(1)
			return nil, types.ErrCacheMiss
		}
		return nil, types.NewCacheError("Get", key, "memory", err)
	}

	c.hits.Add(1)
	return data, nil
}

func (c *MemoryCache) Set(ctx context.Context, key string, value []byte, opts *types.CacheOptions) error {
	if c.closed.Load() {
		return types.ErrClosed
	}

	if err := c.cache.Set(key, value); err != nil {
		return types.NewCacheError("Set", key, "memory", err)
	}

	c.sets.Add(1)
	return nil
}

func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	if c.closed.Load() {
		return types.ErrClosed
	}

	if err := c.cache.Delete(key); err != nil {
		if err != bigcache.ErrEntryNotFound {
			return types.NewCacheError("Delete", key, "memory", err)
		}
	}

	c.deletes.Add(1)
	return nil
}

func (c *MemoryCache) Contains(ctx context.Context, key string) (bool, error) {
	if c.closed.Load() {
		return false, types.ErrClosed
	}

	_, err := c.cache.Get(key)
	if err != nil {
		if err == bigcache.ErrEntryNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (c *MemoryCache) Clear(ctx context.Context) error {
	if c.closed.Load() {
		return types.ErrClosed
	}

	return c.cache.Reset()
}

func (c *MemoryCache) ClearByPattern(ctx context.Context, pattern string) error {
	if c.closed.Load() {
		return types.ErrClosed
	}

	var keysToDelete []string

	iter := c.cache.Iterator()
	for iter.SetNext() {
		entry, err := iter.Value()
		if err != nil {
			continue
		}

		if matchPattern(entry.Key(), pattern) {
			keysToDelete = append(keysToDelete, entry.Key())
		}
	}

	for _, key := range keysToDelete {
		_ = c.cache.Delete(key)
	}

	c.logger.Debug("Cleared entries by pattern",
		"pattern", pattern,
		"deleted", len(keysToDelete),
	)

	return nil
}

func (c *MemoryCache) Close() error {
	if c.closed.Swap(true) {
		return nil
	}
	return c.cache.Close()
}

func (c *MemoryCache) Stats() types.MemoryCacheStats {
	return types.MemoryCacheStats{
		Hits:      c.hits.Load(),
		Misses:    c.misses.Load(),
		Sets:      c.sets.Load(),
		Deletes:   c.deletes.Load(),
		Evictions: c.evictions.Load(),
	}
}

func (c *MemoryCache) EntryCount() int {
	return c.cache.Len()
}

func (c *MemoryCache) Size() int64 {
	return int64(c.cache.Capacity())
}

func (c *MemoryCache) MaxSize() int64 {
	return int64(c.config.MaxSizeMB) * 1024 * 1024
}

func (c *MemoryCache) UsagePercentage() float64 {
	maxBytes := c.MaxSize()
	if maxBytes == 0 {
		return 0
	}
	return float64(c.Size()) / float64(maxBytes) * 100
}

func (c *MemoryCache) HitRatio() float64 {
	hits := c.hits.Load()
	misses := c.misses.Load()
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total)
}

func matchPattern(key, pattern string) bool {
	if pattern == "*" {
		return true
	}

	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(key, prefix)
	}

	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(key, suffix)
	}

	if strings.Contains(pattern, "*") {
		parts := strings.Split(pattern, "*")
		if len(parts) == 2 {
			return strings.HasPrefix(key, parts[0]) && strings.HasSuffix(key, parts[1])
		}
	}

	return key == pattern
}

type bigcacheLogger struct {
	logger *slog.Logger
}

func (l *bigcacheLogger) Printf(format string, args ...any) {
	l.logger.Debug("bigcache: "+format, args...)
}

var _ types.MemoryCacheLayer = (*MemoryCache)(nil)
