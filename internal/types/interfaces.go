package types

import (
	"context"
	"time"
)

type CacheInfo interface {
	Name() string
	IsAvailable() bool
}

type CacheReader interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Contains(ctx context.Context, key string) (bool, error)
}

type CacheWriter interface {
	Set(ctx context.Context, key string, value []byte, opts *CacheOptions) error
	Delete(ctx context.Context, key string) error
}

type CacheClearer interface {
	Clear(ctx context.Context) error
	ClearByPattern(ctx context.Context, pattern string) error
}

type CacheCloser interface {
	Close() error
}

type BatchReader interface {
	GetMany(ctx context.Context, keys []string) (map[string][]byte, error)
}

type BatchWriter interface {
	SetMany(ctx context.Context, items map[string][]byte, opts *CacheOptions) error
}

type MemoryStatsProvider interface {
	Stats() MemoryCacheStats
	EntryCount() int
	Size() int64
	MaxSize() int64
	UsagePercentage() float64
	HitRatio() float64
}

type RedisStatsProvider interface {
	PendingWrites() int
	DroppedWrites() int64
}

type MemoryCacheLayer interface {
	CacheInfo
	CacheReader
	CacheWriter
	CacheClearer
	CacheCloser
	MemoryStatsProvider
}

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

type Serializer interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, dest interface{}) error
}

type MetricsRecorder interface {
	RecordHit(layer string, key string, latency time.Duration)
	RecordMiss(layer string, key string, latency time.Duration)
	RecordSet(layer string, key string, size int, latency time.Duration)
	RecordDelete(layer string, key string, latency time.Duration)
	RecordError(layer string, operation string, err error)
	RecordCircuitBreakerStateChange(from, to string)
}

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}
