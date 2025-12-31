package rentfree

import (
	"time"

	"github.com/LavishGent/rentfree/internal/types"
)

type (
	// Option is a function that modifies CacheOptions.
	Option = types.Option
	// ManagerOptions contains options for creating a cache manager.
	ManagerOptions = types.ManagerOptions
)

// ApplyOptions applies all option functions and returns the resulting CacheOptions.
func ApplyOptions(opts ...Option) *CacheOptions {
	return types.ApplyOptions(opts...)
}

// WithTTL sets the time-to-live for a cache entry.
func WithTTL(ttl time.Duration) Option {
	return func(o *CacheOptions) {
		o.TTL = ttl
	}
}

// WithLevel sets the cache level for an operation.
func WithLevel(level CacheLevel) Option {
	return func(o *CacheOptions) {
		o.Level = level
	}
}

// WithPriority sets the cache priority for an entry.
func WithPriority(priority CachePriority) Option {
	return func(o *CacheOptions) {
		o.Priority = priority
	}
}

// WithFireAndForget enables fire-and-forget mode for asynchronous operations.
func WithFireAndForget() Option {
	return func(o *CacheOptions) {
		o.FireAndForget = true
	}
}

// WithSkipLocalCache skips the local memory cache for an operation.
func WithSkipLocalCache() Option {
	return func(o *CacheOptions) {
		o.SkipLocalCache = true
	}
}

// WithMemoryOnly sets the cache level to memory-only.
func WithMemoryOnly() Option {
	return func(o *CacheOptions) {
		o.Level = LevelMemoryOnly
	}
}

// WithRedisOnly sets the cache level to Redis-only.
func WithRedisOnly() Option {
	return func(o *CacheOptions) {
		o.Level = LevelRedisOnly
	}
}

// WithHighPriority sets the cache priority to high.
func WithHighPriority() Option {
	return func(o *CacheOptions) {
		o.Priority = PriorityHigh
	}
}

// WithLowPriority sets the cache priority to low.
func WithLowPriority() Option {
	return func(o *CacheOptions) {
		o.Priority = PriorityLow
	}
}

// ManagerOption is a function that configures a cache manager.
type ManagerOption func(*ManagerOptions)

// WithLogger sets a custom logger for the cache manager.
func WithLogger(logger Logger) ManagerOption {
	return func(o *ManagerOptions) {
		o.Logger = logger
	}
}

// WithMetrics sets a custom metrics recorder for the cache manager.
func WithMetrics(metrics MetricsRecorder) ManagerOption {
	return func(o *ManagerOptions) {
		o.Metrics = metrics
	}
}

// WithSerializer sets a custom serializer for the cache manager.
func WithSerializer(serializer Serializer) ManagerOption {
	return func(o *ManagerOptions) {
		o.Serializer = serializer
	}
}

// WithRedisAddress sets the Redis server address.
func WithRedisAddress(addr string) ManagerOption {
	return func(o *ManagerOptions) {
		o.RedisAddress = addr
	}
}

// WithRedisPassword sets the Redis server password.
func WithRedisPassword(password string) ManagerOption {
	return func(o *ManagerOptions) {
		o.RedisPassword = types.NewSecretString(password)
	}
}

// WithRedisDB sets the Redis database number.
func WithRedisDB(db int) ManagerOption {
	return func(o *ManagerOptions) {
		o.RedisDB = db
	}
}

// WithoutRedis disables Redis caching.
func WithoutRedis() ManagerOption {
	return func(o *ManagerOptions) {
		o.DisableRedis = true
	}
}

// WithoutResilience disables all resilience patterns.
func WithoutResilience() ManagerOption {
	return func(o *ManagerOptions) {
		o.DisableResilience = true
	}
}
