package config

import "time"

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Memory: MemoryConfig{
			Enabled:          true,
			MaxSizeMB:        256,
			DefaultTTL:       5 * time.Minute,
			CleanupInterval:  10 * time.Second,
			Shards:           1024,
			MaxEntrySize:     10 * 1024 * 1024, // 10MB
			HardMaxCacheSize: false,
		},
		Redis: RedisConfig{
			Enabled:             false,
			Address:             "localhost:6379",
			Password:            SecretString{},
			DB:                  0,
			KeyPrefix:           "rentfree:",
			DefaultTTL:          15 * time.Minute,
			PoolSize:            100,
			MinIdleConns:        10,
			DialTimeout:         5 * time.Second,
			ReadTimeout:         3 * time.Second,
			WriteTimeout:        3 * time.Second,
			PoolTimeout:         4 * time.Second,
			MaxPendingWrites:    500,
			EnableTLS:           false,
			TLSSkipVerify:       false,
			HealthCheckInterval: 5 * time.Second,
		},
		Defaults: DefaultsConfig{
			TTL:           5 * time.Minute,
			Level:         "memory-then-redis",
			Priority:      "normal",
			FireAndForget: false,
		},
		CircuitBreaker: CircuitBreakerConfig{
			Enabled:             true,
			FailureThreshold:    5,
			SuccessThreshold:    2,
			OpenDuration:        30 * time.Second,
			HalfOpenMaxRequests: 3,
		},
		Retry: RetryConfig{
			Enabled:        true,
			MaxAttempts:    3,
			InitialBackoff: 100 * time.Millisecond,
			MaxBackoff:     2 * time.Second,
			Multiplier:     2.0,
			Jitter:         true,
		},
		Bulkhead: BulkheadConfig{
			Enabled:        true,
			MaxConcurrent:  100,
			MaxQueue:       50,
			AcquireTimeout: 100 * time.Millisecond,
		},
		Metrics: MetricsConfig{
			Enabled:         true,
			PublishInterval: 10 * time.Second,
			DataDog: DataDogConfig{
				Enabled:                false,
				AgentHost:              "127.0.0.1",
				Port:                   8125,
				Prefix:                 "rentfree",
				Tags:                   []string{},
				PublishIntervalSeconds: 30,
			},
		},
		KeyValidation: KeyValidationConfig{
			Enabled:           true,
			MaxKeyLength:      1024,
			AllowEmpty:        false,
			AllowControlChars: false,
			AllowWhitespace:   true,
		},
	}
}

// ForTesting returns a minimal configuration suitable for unit tests.
func ForTesting() *Config {
	return &Config{
		Memory: MemoryConfig{
			Enabled:          true,
			MaxSizeMB:        16,
			DefaultTTL:       1 * time.Minute,
			CleanupInterval:  1 * time.Second,
			Shards:           64,
			MaxEntrySize:     1024 * 1024, // 1MB
			HardMaxCacheSize: false,
		},
		Redis: RedisConfig{
			Enabled:             false, // Disabled for unit tests
			Address:             "localhost:6379",
			KeyPrefix:           "test:",
			DefaultTTL:          1 * time.Minute,
			PoolSize:            10,
			MinIdleConns:        1,
			DialTimeout:         1 * time.Second,
			ReadTimeout:         1 * time.Second,
			WriteTimeout:        1 * time.Second,
			PoolTimeout:         1 * time.Second,
			MaxPendingWrites:    50,
			HealthCheckInterval: 0,
		},
		Defaults: DefaultsConfig{
			TTL:           1 * time.Minute,
			Level:         "memory-only",
			Priority:      "normal",
			FireAndForget: false,
		},
		CircuitBreaker: CircuitBreakerConfig{
			Enabled:             false,
			FailureThreshold:    3,
			SuccessThreshold:    1,
			OpenDuration:        1 * time.Second,
			HalfOpenMaxRequests: 1,
		},
		Retry: RetryConfig{
			Enabled:        false,
			MaxAttempts:    1,
			InitialBackoff: 10 * time.Millisecond,
			MaxBackoff:     100 * time.Millisecond,
			Multiplier:     2.0,
			Jitter:         false,
		},
		Bulkhead: BulkheadConfig{
			Enabled:        false,
			MaxConcurrent:  10,
			MaxQueue:       5,
			AcquireTimeout: 50 * time.Millisecond,
		},
		Metrics: MetricsConfig{
			Enabled:         false,
			PublishInterval: 1 * time.Second,
		},
		KeyValidation: KeyValidationConfig{
			Enabled:           true,
			MaxKeyLength:      1024,
			AllowEmpty:        false,
			AllowControlChars: false,
			AllowWhitespace:   true,
		},
	}
}

// ForTestingWithRedis returns a test config with Redis enabled.
func ForTestingWithRedis(addr string) *Config {
	cfg := ForTesting()
	cfg.Redis.Enabled = true
	cfg.Redis.Address = addr
	cfg.Defaults.Level = "memory-then-redis"
	return cfg
}
