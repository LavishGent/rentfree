package rentfree

import (
	"context"
	"time"
)

type CacheManager interface {
	Get(ctx context.Context, key string, dest interface{}, opts ...Option) error
	Set(ctx context.Context, key string, value interface{}, opts ...Option) error
	GetOrCreate(ctx context.Context, key string, dest interface{}, factory func() (interface{}, error), opts ...Option) error
	Delete(ctx context.Context, key string, opts ...Option) error
	DeleteMany(ctx context.Context, keys []string, opts ...Option) error
	Contains(ctx context.Context, key string, opts ...Option) (bool, error)
	GetMany(ctx context.Context, keys []string, opts ...Option) (map[string][]byte, error)
	SetMany(ctx context.Context, items map[string]interface{}, opts ...Option) error
	Clear(ctx context.Context, level CacheLevel) error
	ClearByPattern(ctx context.Context, pattern string, level CacheLevel) error
	Health(ctx context.Context) (*HealthMetrics, error)
	IsHealthy(ctx context.Context) bool
	IsRedisAvailable() bool
	IsMemoryAvailable() bool
	Close() error
}

type Publisher interface {
	Gauge(name string, value float64, tags ...string)
	Incr(name string, tags ...string)
	Count(name string, value int64, tags ...string)
	Histogram(name string, value float64, tags ...string)
	Timing(name string, duration time.Duration, tags ...string)
	Event(title, text string, alertType string, tags ...string)
	PublishHealthMetrics(metrics *PublisherHealthMetrics)
	Close() error
}

type PublisherHealthMetrics struct {
	MemoryUsedBytes       int64
	MemoryLimitBytes      int64
	MemoryUsagePercentage float64
	TotalEntries          int64
	HitRatio              float64
	AverageLatencyMs      float64
	IsConnected           bool
}
