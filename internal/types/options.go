package types

// Option is a functional option for configuring cache operations.
type Option func(*CacheOptions)

// ApplyOptions applies functional options to create CacheOptions.
func ApplyOptions(opts ...Option) *CacheOptions {
	options := DefaultOptions()
	for _, opt := range opts {
		opt(options)
	}
	return options
}

// ManagerOptions holds configuration for the cache manager.
type ManagerOptions struct {
	// Logger is the structured logger to use.
	Logger Logger

	// Metrics is the metrics recorder.
	Metrics MetricsRecorder

	// Serializer is the value serializer.
	Serializer Serializer

	// RedisAddress overrides the Redis address from config.
	RedisAddress string

	// RedisPassword overrides the Redis password from config.
	// Uses SecretString to prevent accidental logging of sensitive values.
	RedisPassword SecretString

	// RedisDB overrides the Redis database from config.
	RedisDB int

	// DisableRedis disables the Redis layer entirely.
	DisableRedis bool

	// DisableResilience disables circuit breaker and retry patterns.
	DisableResilience bool
}
