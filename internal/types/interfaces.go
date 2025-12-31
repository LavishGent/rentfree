package types

import (
	"context"
	"time"
)

// CacheInfo provides basic information about a cache layer.
type CacheInfo interface {
	Name() string
	IsAvailable() bool
}

// CacheReader provides read operations for a cache layer.
type CacheReader interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Contains(ctx context.Context, key string) (bool, error)
}

// CacheWriter provides write operations for a cache layer.
type CacheWriter interface {
	Set(ctx context.Context, key string, value []byte, opts *CacheOptions) error
	Delete(ctx context.Context, key string) error
}

// CacheClearer provides clear operations for a cache layer.
type CacheClearer interface {
	Clear(ctx context.Context) error
	ClearByPattern(ctx context.Context, pattern string) error
}

// CacheCloser provides close operations for a cache layer.
type CacheCloser interface {
	Close() error
}

// BatchReader provides batch read operations for a cache layer.
type BatchReader interface {
	GetMany(ctx context.Context, keys []string) (map[string][]byte, error)
}

// BatchWriter provides batch write operations for a cache layer.
type BatchWriter interface {
	SetMany(ctx context.Context, items map[string][]byte, opts *CacheOptions) error
}

// MemoryStatsProvider provides statistics about memory cache usage.
type MemoryStatsProvider interface {
	Stats() MemoryCacheStats
	EntryCount() int
	Size() int64
	MaxSize() int64
	UsagePercentage() float64
	HitRatio() float64
}

// RedisStatsProvider provides statistics about Redis cache usage.
type RedisStatsProvider interface {
	PendingWrites() int
	DroppedWrites() int64
}

// MemoryCacheLayer combines all memory cache operations.
type MemoryCacheLayer interface {
	CacheInfo
	CacheReader
	CacheWriter
	CacheClearer
	CacheCloser
	MemoryStatsProvider
}

// RedisCacheLayer combines all Redis cache operations.
type RedisCacheLayer interface {
	CacheInfo
	CacheReader
	CacheWriter
	CacheClearer
	CacheCloser
	BatchReader
	BatchWriter
	RedisStatsProvider
}

// Serializer provides serialization and deserialization operations.
type Serializer interface {
	Marshal(v any) ([]byte, error)
	Unmarshal(data []byte, dest any) error
}

// MetricsRecorder provides operations for recording cache metrics.
type MetricsRecorder interface {
	RecordHit(layer string, key string, latency time.Duration)
	RecordMiss(layer string, key string, latency time.Duration)
	RecordSet(layer string, key string, size int, latency time.Duration)
	RecordDelete(layer string, key string, latency time.Duration)
	RecordError(layer string, operation string, err error)
	RecordCircuitBreakerStateChange(from, to string)
}

// Logger provides logging operations.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}
