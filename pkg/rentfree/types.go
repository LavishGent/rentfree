// Package rentfree provides a unified multi-layer caching solution with
// memory and Redis backends, resilience patterns, and pluggable metrics.
package rentfree

import (
	"github.com/LavishGent/rentfree/internal/types"
)

type (
	// CacheLevel specifies which cache layers to use for an operation.
	CacheLevel = types.CacheLevel
	// CachePriority specifies the eviction priority of a cache entry.
	CachePriority = types.CachePriority
	// CacheEntry represents a cached value with metadata.
	CacheEntry = types.CacheEntry
	// CacheOptions contains options for cache operations.
	CacheOptions = types.CacheOptions
	// MemoryCacheStats contains statistics about the memory cache.
	MemoryCacheStats = types.MemoryCacheStats
	// Serializer provides serialization and deserialization operations.
	Serializer = types.Serializer
	// MetricsRecorder provides operations for recording cache metrics.
	MetricsRecorder = types.MetricsRecorder
	// Logger provides logging operations.
	Logger = types.Logger
)

const (
	// LevelMemoryOnly uses only the in-memory cache layer.
	LevelMemoryOnly = types.LevelMemoryOnly
	// LevelRedisOnly uses only the Redis cache layer.
	LevelRedisOnly = types.LevelRedisOnly
	// LevelMemoryThenRedis checks memory first, then falls back to Redis.
	LevelMemoryThenRedis = types.LevelMemoryThenRedis
	// LevelAll uses all available cache layers.
	LevelAll = types.LevelAll
)

const (
	// PriorityLow indicates low priority entries that are evicted first.
	PriorityLow = types.PriorityLow
	// PriorityNormal indicates normal priority entries.
	PriorityNormal = types.PriorityNormal
	// PriorityHigh indicates high priority entries that are evicted last.
	PriorityHigh = types.PriorityHigh
	// PriorityNeverRemove indicates entries that should never be evicted.
	PriorityNeverRemove = types.PriorityNeverRemove
)

// DefaultOptions returns a default CacheOptions configuration.
func DefaultOptions() *CacheOptions {
	return types.DefaultOptions()
}
