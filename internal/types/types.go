// Package types provides shared types for the rentfree cache library.
// This package breaks import cycles between pkg/rentfree and internal/cache.
package types

import "time"

type CacheLevel int

const (
	LevelMemoryOnly CacheLevel = iota + 1
	LevelRedisOnly
	LevelMemoryThenRedis
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

func (l CacheLevel) IncludesMemory() bool {
	return l == LevelMemoryOnly || l == LevelMemoryThenRedis || l == LevelAll
}

func (l CacheLevel) IncludesRedis() bool {
	return l == LevelRedisOnly || l == LevelMemoryThenRedis || l == LevelAll
}

type CachePriority int

const (
	PriorityLow CachePriority = iota + 1
	PriorityNormal
	PriorityHigh
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

type CacheOptions struct {
	TTL            time.Duration
	Level          CacheLevel
	Priority       CachePriority
	FireAndForget  bool
	SkipLocalCache bool
}

func DefaultOptions() *CacheOptions {
	return &CacheOptions{
		TTL: 5 * time.Minute,
	}
}

type CacheEntry struct {
	Key       string
	Value     []byte
	TTL       time.Duration
	Priority  CachePriority
	CreatedAt time.Time
	ExpiresAt time.Time
}

func (e *CacheEntry) IsExpired() bool {
	if e.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(e.ExpiresAt)
}

type MemoryCacheStats struct {
	Hits      int64
	Misses    int64
	Sets      int64
	Deletes   int64
	Evictions int64
}
