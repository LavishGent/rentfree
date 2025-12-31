package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Load loads configuration from a JSON file.
// If the file doesn't exist, returns default configuration.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, use defaults
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// LoadWithEnv loads configuration from a JSON file and applies environment overrides.
func LoadWithEnv(path string) (*Config, error) {
	cfg, err := Load(path)
	if err != nil {
		return nil, err
	}

	applyEnvOverrides(cfg)
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

//nolint:gocyclo // Environment variable parsing requires many conditional checks
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("RENTFREE_MEMORY_ENABLED"); v != "" {
		cfg.Memory.Enabled = parseBool(v)
	}
	if v := os.Getenv("RENTFREE_MEMORY_MAX_SIZE_MB"); v != "" {
		cfg.Memory.MaxSizeMB = parseInt(v, cfg.Memory.MaxSizeMB)
	}
	if v := os.Getenv("RENTFREE_MEMORY_DEFAULT_TTL"); v != "" {
		cfg.Memory.DefaultTTL = parseDuration(v, cfg.Memory.DefaultTTL)
	}

	if v := os.Getenv("RENTFREE_REDIS_ENABLED"); v != "" {
		cfg.Redis.Enabled = parseBool(v)
	}
	if v := os.Getenv("RENTFREE_REDIS_ADDRESS"); v != "" {
		cfg.Redis.Address = v
	}
	if v := os.Getenv("RENTFREE_REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = NewSecretString(v)
	}
	if v := os.Getenv("RENTFREE_REDIS_DB"); v != "" {
		cfg.Redis.DB = parseInt(v, cfg.Redis.DB)
	}
	if v := os.Getenv("RENTFREE_REDIS_KEY_PREFIX"); v != "" {
		cfg.Redis.KeyPrefix = v
	}
	if v := os.Getenv("RENTFREE_REDIS_DEFAULT_TTL"); v != "" {
		cfg.Redis.DefaultTTL = parseDuration(v, cfg.Redis.DefaultTTL)
	}
	if v := os.Getenv("RENTFREE_REDIS_POOL_SIZE"); v != "" {
		cfg.Redis.PoolSize = parseInt(v, cfg.Redis.PoolSize)
	}
	if v := os.Getenv("RENTFREE_REDIS_ENABLE_TLS"); v != "" {
		cfg.Redis.EnableTLS = parseBool(v)
	}
	if v := os.Getenv("RENTFREE_REDIS_TLS_SKIP_VERIFY"); v != "" {
		cfg.Redis.TLSSkipVerify = parseBool(v)
	}

	if v := os.Getenv("RENTFREE_DEFAULTS_TTL"); v != "" {
		cfg.Defaults.TTL = parseDuration(v, cfg.Defaults.TTL)
	}
	if v := os.Getenv("RENTFREE_DEFAULTS_LEVEL"); v != "" {
		cfg.Defaults.Level = v
	}
	if v := os.Getenv("RENTFREE_DEFAULTS_FIRE_AND_FORGET"); v != "" {
		cfg.Defaults.FireAndForget = parseBool(v)
	}

	if v := os.Getenv("RENTFREE_CIRCUIT_BREAKER_ENABLED"); v != "" {
		cfg.CircuitBreaker.Enabled = parseBool(v)
	}
	if v := os.Getenv("RENTFREE_CIRCUIT_BREAKER_FAILURE_THRESHOLD"); v != "" {
		cfg.CircuitBreaker.FailureThreshold = parseInt(v, cfg.CircuitBreaker.FailureThreshold)
	}
	if v := os.Getenv("RENTFREE_CIRCUIT_BREAKER_OPEN_DURATION"); v != "" {
		cfg.CircuitBreaker.OpenDuration = parseDuration(v, cfg.CircuitBreaker.OpenDuration)
	}

	if v := os.Getenv("RENTFREE_RETRY_ENABLED"); v != "" {
		cfg.Retry.Enabled = parseBool(v)
	}
	if v := os.Getenv("RENTFREE_RETRY_MAX_ATTEMPTS"); v != "" {
		cfg.Retry.MaxAttempts = parseInt(v, cfg.Retry.MaxAttempts)
	}

	if v := os.Getenv("RENTFREE_BULKHEAD_ENABLED"); v != "" {
		cfg.Bulkhead.Enabled = parseBool(v)
	}
	if v := os.Getenv("RENTFREE_BULKHEAD_MAX_CONCURRENT"); v != "" {
		cfg.Bulkhead.MaxConcurrent = parseInt(v, cfg.Bulkhead.MaxConcurrent)
	}

	if v := os.Getenv("RENTFREE_METRICS_ENABLED"); v != "" {
		cfg.Metrics.Enabled = parseBool(v)
	}

	if v := os.Getenv("DD_AGENT_HOST"); v != "" {
		cfg.Metrics.DataDog.AgentHost = v
		cfg.Metrics.DataDog.Enabled = true
	}
	if v := os.Getenv("DD_DOGSTATSD_PORT"); v != "" {
		cfg.Metrics.DataDog.Port = parseInt(v, cfg.Metrics.DataDog.Port)
	}
	if v := os.Getenv("DD_SERVICE"); v != "" {
		cfg.Metrics.DataDog.Prefix = v
	}
	if v := os.Getenv("DD_ENV"); v != "" {
		cfg.Metrics.DataDog.Tags = append(cfg.Metrics.DataDog.Tags, "env:"+v)
	}
	if v := os.Getenv("DD_VERSION"); v != "" {
		cfg.Metrics.DataDog.Tags = append(cfg.Metrics.DataDog.Tags, "version:"+v)
	}

	if v := os.Getenv("RENTFREE_DATADOG_ENABLED"); v != "" {
		if os.Getenv("DD_AGENT_HOST") == "" {
			cfg.Metrics.DataDog.Enabled = parseBool(v)
		}
	}
	if v := os.Getenv("RENTFREE_DATADOG_PREFIX"); v != "" {
		if os.Getenv("DD_SERVICE") == "" {
			cfg.Metrics.DataDog.Prefix = v
		}
	}
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Memory.Enabled {
		if c.Memory.MaxSizeMB <= 0 {
			return fmt.Errorf("memory.maxSizeMB must be positive")
		}
		if c.Memory.Shards <= 0 || (c.Memory.Shards&(c.Memory.Shards-1)) != 0 {
			return fmt.Errorf("memory.shards must be a positive power of 2")
		}
	}

	if c.Redis.Enabled {
		if c.Redis.Address == "" {
			return fmt.Errorf("redis.address is required when redis is enabled")
		}
		if c.Redis.PoolSize <= 0 {
			return fmt.Errorf("redis.poolSize must be positive")
		}
	}

	if c.CircuitBreaker.Enabled {
		if c.CircuitBreaker.FailureThreshold <= 0 {
			return fmt.Errorf("circuitBreaker.failureThreshold must be positive")
		}
		if c.CircuitBreaker.OpenDuration <= 0 {
			return fmt.Errorf("circuitBreaker.openDuration must be positive")
		}
	}

	if c.Retry.Enabled {
		if c.Retry.MaxAttempts <= 0 {
			return fmt.Errorf("retry.maxAttempts must be positive")
		}
	}

	if c.Bulkhead.Enabled {
		if c.Bulkhead.MaxConcurrent <= 0 {
			return fmt.Errorf("bulkhead.maxConcurrent must be positive")
		}
	}

	return nil
}

func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "1" || s == "yes" || s == "on"
}

func parseInt(s string, defaultVal int) int {
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return defaultVal
	}
	return v
}

func parseDuration(s string, defaultVal time.Duration) time.Duration {
	s = strings.TrimSpace(s)

	if d, err := time.ParseDuration(s); err == nil {
		return d
	}

	if secs, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Duration(secs) * time.Second
	}

	return defaultVal
}
