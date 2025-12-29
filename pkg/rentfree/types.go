// Package rentfree provides a unified multi-layer caching solution with
// memory and Redis backends, resilience patterns, and pluggable metrics.
package rentfree

import (
	"github.com/darrell-green/rentfree/internal/types"
)

type (
	CacheLevel       = types.CacheLevel
	CachePriority    = types.CachePriority
	CacheEntry       = types.CacheEntry
	CacheOptions     = types.CacheOptions
	MemoryCacheStats = types.MemoryCacheStats
	Serializer       = types.Serializer
	MetricsRecorder  = types.MetricsRecorder
	Logger           = types.Logger
)

const (
	LevelMemoryOnly      = types.LevelMemoryOnly
	LevelRedisOnly       = types.LevelRedisOnly
	LevelMemoryThenRedis = types.LevelMemoryThenRedis
	LevelAll             = types.LevelAll
)

const (
	PriorityLow         = types.PriorityLow
	PriorityNormal      = types.PriorityNormal
	PriorityHigh        = types.PriorityHigh
	PriorityNeverRemove = types.PriorityNeverRemove
)

func DefaultOptions() *CacheOptions {
	return types.DefaultOptions()
}
