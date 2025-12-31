package rentfree

import (
	"time"

	"github.com/LavishGent/rentfree/internal/types"
)

type (
	Option         = types.Option
	ManagerOptions = types.ManagerOptions
)

func ApplyOptions(opts ...Option) *CacheOptions {
	return types.ApplyOptions(opts...)
}

func WithTTL(ttl time.Duration) Option {
	return func(o *CacheOptions) {
		o.TTL = ttl
	}
}

func WithLevel(level CacheLevel) Option {
	return func(o *CacheOptions) {
		o.Level = level
	}
}

func WithPriority(priority CachePriority) Option {
	return func(o *CacheOptions) {
		o.Priority = priority
	}
}

func WithFireAndForget() Option {
	return func(o *CacheOptions) {
		o.FireAndForget = true
	}
}

func WithSkipLocalCache() Option {
	return func(o *CacheOptions) {
		o.SkipLocalCache = true
	}
}

func WithMemoryOnly() Option {
	return func(o *CacheOptions) {
		o.Level = LevelMemoryOnly
	}
}

func WithRedisOnly() Option {
	return func(o *CacheOptions) {
		o.Level = LevelRedisOnly
	}
}

func WithHighPriority() Option {
	return func(o *CacheOptions) {
		o.Priority = PriorityHigh
	}
}

func WithLowPriority() Option {
	return func(o *CacheOptions) {
		o.Priority = PriorityLow
	}
}

type ManagerOption func(*ManagerOptions)

func WithLogger(logger Logger) ManagerOption {
	return func(o *ManagerOptions) {
		o.Logger = logger
	}
}

func WithMetrics(metrics MetricsRecorder) ManagerOption {
	return func(o *ManagerOptions) {
		o.Metrics = metrics
	}
}

func WithSerializer(serializer Serializer) ManagerOption {
	return func(o *ManagerOptions) {
		o.Serializer = serializer
	}
}

func WithRedisAddress(addr string) ManagerOption {
	return func(o *ManagerOptions) {
		o.RedisAddress = addr
	}
}

func WithRedisPassword(password string) ManagerOption {
	return func(o *ManagerOptions) {
		o.RedisPassword = types.NewSecretString(password)
	}
}

func WithRedisDB(db int) ManagerOption {
	return func(o *ManagerOptions) {
		o.RedisDB = db
	}
}

func WithoutRedis() ManagerOption {
	return func(o *ManagerOptions) {
		o.DisableRedis = true
	}
}

func WithoutResilience() ManagerOption {
	return func(o *ManagerOptions) {
		o.DisableResilience = true
	}
}
