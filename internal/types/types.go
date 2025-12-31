// Package types provides shared types for the rentfree cache library.
// This package breaks import cycles between pkg/rentfree and internal/cache.
package types

import "time"

// CacheLevel specifies which cache layers to use for an operation.
type CacheLevel int

const (
	// LevelMemoryOnly uses only the in-memory cache layer.
	LevelMemoryOnly CacheLevel = iota + 1
	// LevelRedisOnly uses only the Redis cache layer.
	LevelRedisOnly
	// LevelMemoryThenRedis checks memory first, then falls back to Redis.
	LevelMemoryThenRedis
	// LevelAll uses all available cache layers.
	LevelAll
)

func (l CacheLevel) String() string {
	switch l {
	case LevelMemoryOnly:
		return "memory-only"
	case LevelRedisOnly:
		return "redis-only"
	case LevelMemoryThenRedis:
		return "memory-then-redis"
	case LevelAll:
		return "all"
	default:
		return "unknown"
	}
}

// IncludesMemory returns true if this cache level includes the memory layer.
func (l CacheLevel) IncludesMemory() bool {
	return l == LevelMemoryOnly || l == LevelMemoryThenRedis || l == LevelAll
}

// IncludesRedis returns true if this cache level includes the Redis layer.
func (l CacheLevel) IncludesRedis() bool {
	return l == LevelRedisOnly || l == LevelMemoryThenRedis || l == LevelAll
}

// CachePriority specifies the eviction priority of a cache entry.
type CachePriority int

const (
	// PriorityLow indicates low priority entries that are evicted first.
	PriorityLow CachePriority = iota + 1
	// PriorityNormal indicates normal priority entries.
	PriorityNormal
	// PriorityHigh indicates high priority entries that are evicted last.
	PriorityHigh
	// PriorityNeverRemove indicates entries that should never be evicted.
	PriorityNeverRemove
)

func (p CachePriority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityNormal:
		return "normal"
	case PriorityHigh:
		return "high"
	case PriorityNeverRemove:
		return "never-remove"
	default:
		return "unknown"
	}
}

// CacheOptions contains options for cache operations.
type CacheOptions struct {
	TTL            time.Duration
	Level          CacheLevel
	Priority       CachePriority
	FireAndForget  bool
	SkipLocalCache bool
}

// DefaultOptions returns a CacheOptions with default values.
func DefaultOptions() *CacheOptions {
	return &CacheOptions{
		TTL: 5 * time.Minute,
	}
}

// CacheEntry represents a cached value with metadata.
//
//nolint:govet // Small struct with mixed types - current ordering is logical
type CacheEntry struct {
	Value     []byte
	CreatedAt time.Time
	ExpiresAt time.Time
	TTL       time.Duration
	Key       string
	Priority  CachePriority
}

// IsExpired returns true if this cache entry has expired.
func (e *CacheEntry) IsExpired() bool {
	if e.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(e.ExpiresAt)
}

// MemoryCacheStats contains statistics about the memory cache.
type MemoryCacheStats struct {
	Hits      int64
	Misses    int64
	Sets      int64
	Deletes   int64
	Evictions int64
}
