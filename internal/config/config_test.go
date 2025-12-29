package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	t.Run("memory defaults", func(t *testing.T) {
		if !cfg.Memory.Enabled {
			t.Error("Memory.Enabled = false, want true")
		}
		if cfg.Memory.MaxSizeMB != 256 {
			t.Errorf("Memory.MaxSizeMB = %d, want 256", cfg.Memory.MaxSizeMB)
		}
		if cfg.Memory.DefaultTTL != 5*time.Minute {
			t.Errorf("Memory.DefaultTTL = %v, want 5m", cfg.Memory.DefaultTTL)
		}
		if cfg.Memory.Shards != 1024 {
			t.Errorf("Memory.Shards = %d, want 1024", cfg.Memory.Shards)
		}
	})

	t.Run("redis defaults", func(t *testing.T) {
		if cfg.Redis.Enabled {
			t.Error("Redis.Enabled = true, want false")
		}
		if cfg.Redis.Address != "localhost:6379" {
			t.Errorf("Redis.Address = %s, want localhost:6379", cfg.Redis.Address)
		}
		if cfg.Redis.KeyPrefix != "rentfree:" {
			t.Errorf("Redis.KeyPrefix = %s, want rentfree:", cfg.Redis.KeyPrefix)
		}
		if cfg.Redis.PoolSize != 100 {
			t.Errorf("Redis.PoolSize = %d, want 100", cfg.Redis.PoolSize)
		}
	})

	t.Run("circuit breaker defaults", func(t *testing.T) {
		if !cfg.CircuitBreaker.Enabled {
			t.Error("CircuitBreaker.Enabled = false, want true")
		}
		if cfg.CircuitBreaker.FailureThreshold != 5 {
			t.Errorf("CircuitBreaker.FailureThreshold = %d, want 5", cfg.CircuitBreaker.FailureThreshold)
		}
		if cfg.CircuitBreaker.OpenDuration != 30*time.Second {
			t.Errorf("CircuitBreaker.OpenDuration = %v, want 30s", cfg.CircuitBreaker.OpenDuration)
		}
	})

	t.Run("retry defaults", func(t *testing.T) {
		if !cfg.Retry.Enabled {
			t.Error("Retry.Enabled = false, want true")
		}
		if cfg.Retry.MaxAttempts != 3 {
			t.Errorf("Retry.MaxAttempts = %d, want 3", cfg.Retry.MaxAttempts)
		}
		if cfg.Retry.Multiplier != 2.0 {
			t.Errorf("Retry.Multiplier = %f, want 2.0", cfg.Retry.Multiplier)
		}
	})

	t.Run("bulkhead defaults", func(t *testing.T) {
		if !cfg.Bulkhead.Enabled {
			t.Error("Bulkhead.Enabled = false, want true")
		}
		if cfg.Bulkhead.MaxConcurrent != 100 {
			t.Errorf("Bulkhead.MaxConcurrent = %d, want 100", cfg.Bulkhead.MaxConcurrent)
		}
		if cfg.Bulkhead.MaxQueue != 50 {
			t.Errorf("Bulkhead.MaxQueue = %d, want 50", cfg.Bulkhead.MaxQueue)
		}
	})

	t.Run("metrics defaults", func(t *testing.T) {
		if !cfg.Metrics.Enabled {
			t.Error("Metrics.Enabled = false, want true")
		}
		if cfg.Metrics.PublishInterval != 10*time.Second {
			t.Errorf("Metrics.PublishInterval = %v, want 10s", cfg.Metrics.PublishInterval)
		}
	})
}

func TestForTesting(t *testing.T) {
	cfg := ForTesting()

	t.Run("has smaller resource limits", func(t *testing.T) {
		if cfg.Memory.MaxSizeMB != 16 {
			t.Errorf("Memory.MaxSizeMB = %d, want 16", cfg.Memory.MaxSizeMB)
		}
		if cfg.Redis.PoolSize != 10 {
			t.Errorf("Redis.PoolSize = %d, want 10", cfg.Redis.PoolSize)
		}
	})

	t.Run("resilience features disabled", func(t *testing.T) {
		if cfg.CircuitBreaker.Enabled {
			t.Error("CircuitBreaker.Enabled = true, want false")
		}
		if cfg.Retry.Enabled {
			t.Error("Retry.Enabled = true, want false")
		}
		if cfg.Bulkhead.Enabled {
			t.Error("Bulkhead.Enabled = true, want false")
		}
	})

	t.Run("metrics disabled", func(t *testing.T) {
		if cfg.Metrics.Enabled {
			t.Error("Metrics.Enabled = true, want false")
		}
	})

	t.Run("redis disabled", func(t *testing.T) {
		if cfg.Redis.Enabled {
			t.Error("Redis.Enabled = true, want false")
		}
	})
}

func TestForTestingWithRedis(t *testing.T) {
	addr := "redis.test.local:6380"
	cfg := ForTestingWithRedis(addr)

	if !cfg.Redis.Enabled {
		t.Error("Redis.Enabled = false, want true")
	}
	if cfg.Redis.Address != addr {
		t.Errorf("Redis.Address = %s, want %s", cfg.Redis.Address, addr)
	}
	if cfg.Defaults.Level != "memory-then-redis" {
		t.Errorf("Defaults.Level = %s, want memory-then-redis", cfg.Defaults.Level)
	}
}

func TestLoad(t *testing.T) {
	t.Run("empty path returns defaults", func(t *testing.T) {
		cfg, err := Load("")
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		// Should have default values
		if cfg.Memory.MaxSizeMB != 256 {
			t.Errorf("Memory.MaxSizeMB = %d, want 256", cfg.Memory.MaxSizeMB)
		}
	})

	t.Run("non-existent file returns defaults", func(t *testing.T) {
		cfg, err := Load("/non/existent/path/config.json")
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		// Should have default values
		if cfg.Memory.MaxSizeMB != 256 {
			t.Errorf("Memory.MaxSizeMB = %d, want 256", cfg.Memory.MaxSizeMB)
		}
	})

	t.Run("loads valid JSON file", func(t *testing.T) {
		// Create temp file
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		jsonContent := `{
			"memory": {
				"enabled": true,
				"maxSizeMB": 512,
				"shards": 512
			},
			"redis": {
				"enabled": true,
				"address": "redis.prod:6379",
				"poolSize": 200
			}
		}`

		if err := os.WriteFile(configPath, []byte(jsonContent), 0644); err != nil {
			t.Fatalf("failed to write test config: %v", err)
		}

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg.Memory.MaxSizeMB != 512 {
			t.Errorf("Memory.MaxSizeMB = %d, want 512", cfg.Memory.MaxSizeMB)
		}
		if cfg.Memory.Shards != 512 {
			t.Errorf("Memory.Shards = %d, want 512", cfg.Memory.Shards)
		}
		if cfg.Redis.Address != "redis.prod:6379" {
			t.Errorf("Redis.Address = %s, want redis.prod:6379", cfg.Redis.Address)
		}
		if cfg.Redis.PoolSize != 200 {
			t.Errorf("Redis.PoolSize = %d, want 200", cfg.Redis.PoolSize)
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "invalid.json")

		if err := os.WriteFile(configPath, []byte("not valid json"), 0644); err != nil {
			t.Fatalf("failed to write test config: %v", err)
		}

		_, err := Load(configPath)
		if err == nil {
			t.Error("Load() error = nil, want error")
		}
	})

	t.Run("returns error for invalid config values", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "invalid-values.json")

		// Invalid: shards not power of 2
		jsonContent := `{
			"memory": {
				"enabled": true,
				"maxSizeMB": 100,
				"shards": 100
			}
		}`

		if err := os.WriteFile(configPath, []byte(jsonContent), 0644); err != nil {
			t.Fatalf("failed to write test config: %v", err)
		}

		_, err := Load(configPath)
		if err == nil {
			t.Error("Load() error = nil, want validation error")
		}
	})
}

func TestLoadWithEnv(t *testing.T) {
	t.Run("applies environment overrides", func(t *testing.T) {
		// Set environment variables
		os.Setenv("RENTFREE_REDIS_ADDRESS", "redis.env:6380")
		os.Setenv("RENTFREE_REDIS_ENABLED", "true")
		os.Setenv("RENTFREE_BULKHEAD_MAX_CONCURRENT", "200")
		defer func() {
			os.Unsetenv("RENTFREE_REDIS_ADDRESS")
			os.Unsetenv("RENTFREE_REDIS_ENABLED")
			os.Unsetenv("RENTFREE_BULKHEAD_MAX_CONCURRENT")
		}()

		cfg, err := LoadWithEnv("")
		if err != nil {
			t.Fatalf("LoadWithEnv() error = %v", err)
		}

		if cfg.Redis.Address != "redis.env:6380" {
			t.Errorf("Redis.Address = %s, want redis.env:6380", cfg.Redis.Address)
		}
		if !cfg.Redis.Enabled {
			t.Error("Redis.Enabled = false, want true")
		}
		if cfg.Bulkhead.MaxConcurrent != 200 {
			t.Errorf("Bulkhead.MaxConcurrent = %d, want 200", cfg.Bulkhead.MaxConcurrent)
		}
	})

	t.Run("env overrides JSON file values", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		jsonContent := `{
			"redis": {
				"enabled": true,
				"address": "redis.json:6379",
				"poolSize": 100
			}
		}`

		if err := os.WriteFile(configPath, []byte(jsonContent), 0644); err != nil {
			t.Fatalf("failed to write test config: %v", err)
		}

		// Environment should override JSON
		os.Setenv("RENTFREE_REDIS_ADDRESS", "redis.override:6380")
		defer os.Unsetenv("RENTFREE_REDIS_ADDRESS")

		cfg, err := LoadWithEnv(configPath)
		if err != nil {
			t.Fatalf("LoadWithEnv() error = %v", err)
		}

		if cfg.Redis.Address != "redis.override:6380" {
			t.Errorf("Redis.Address = %s, want redis.override:6380", cfg.Redis.Address)
		}
	})
}

func TestValidate(t *testing.T) {
	t.Run("valid config passes", func(t *testing.T) {
		cfg := DefaultConfig()
		if err := cfg.Validate(); err != nil {
			t.Errorf("Validate() error = %v, want nil", err)
		}
	})

	t.Run("memory.maxSizeMB must be positive", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Memory.MaxSizeMB = 0

		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() error = nil, want error")
		}
	})

	t.Run("memory.shards must be power of 2", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Memory.Shards = 100

		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() error = nil, want error")
		}
	})

	t.Run("redis.address required when enabled", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Redis.Enabled = true
		cfg.Redis.Address = ""

		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() error = nil, want error")
		}
	})

	t.Run("redis.poolSize must be positive", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Redis.Enabled = true
		cfg.Redis.PoolSize = 0

		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() error = nil, want error")
		}
	})

	t.Run("circuitBreaker.failureThreshold must be positive", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.CircuitBreaker.FailureThreshold = 0

		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() error = nil, want error")
		}
	})

	t.Run("circuitBreaker.openDuration must be positive", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.CircuitBreaker.OpenDuration = 0

		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() error = nil, want error")
		}
	})

	t.Run("retry.maxAttempts must be positive", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Retry.MaxAttempts = 0

		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() error = nil, want error")
		}
	})

	t.Run("bulkhead.maxConcurrent must be positive", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Bulkhead.MaxConcurrent = 0

		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() error = nil, want error")
		}
	})

	t.Run("disabled components skip validation", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Memory.Enabled = false
		cfg.Memory.MaxSizeMB = 0 // Would fail if enabled
		cfg.Redis.Enabled = false
		cfg.Redis.Address = "" // Would fail if enabled
		cfg.CircuitBreaker.Enabled = false
		cfg.CircuitBreaker.FailureThreshold = 0 // Would fail if enabled
		cfg.Retry.Enabled = false
		cfg.Retry.MaxAttempts = 0 // Would fail if enabled
		cfg.Bulkhead.Enabled = false
		cfg.Bulkhead.MaxConcurrent = 0 // Would fail if enabled

		if err := cfg.Validate(); err != nil {
			t.Errorf("Validate() error = %v, want nil", err)
		}
	})
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"1", true},
		{"yes", true},
		{"YES", true},
		{"on", true},
		{"ON", true},
		{"false", false},
		{"0", false},
		{"no", false},
		{"off", false},
		{"invalid", false},
		{"", false},
		{"  true  ", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseBool(tt.input)
			if result != tt.expected {
				t.Errorf("parseBool(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		input      string
		defaultVal int
		expected   int
	}{
		{"42", 0, 42},
		{"0", 10, 0},
		{"-5", 0, -5},
		{"invalid", 99, 99},
		{"", 99, 99},
		{"  100  ", 0, 100},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseInt(tt.input, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("parseInt(%q, %d) = %d, want %d", tt.input, tt.defaultVal, result, tt.expected)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	defaultDur := 5 * time.Second

	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"30s", 30 * time.Second},
		{"5m", 5 * time.Minute},
		{"1h", 1 * time.Hour},
		{"100ms", 100 * time.Millisecond},
		{"60", 60 * time.Second},   // Plain number as seconds
		{"120", 120 * time.Second}, // Plain number as seconds
		{"invalid", defaultDur},
		{"", defaultDur},
		{"  30s  ", 30 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseDuration(tt.input, defaultDur)
			if result != tt.expected {
				t.Errorf("parseDuration(%q, %v) = %v, want %v", tt.input, defaultDur, result, tt.expected)
			}
		})
	}
}

func TestApplyEnvOverrides(t *testing.T) {
	t.Run("memory overrides", func(t *testing.T) {
		os.Setenv("RENTFREE_MEMORY_ENABLED", "false")
		os.Setenv("RENTFREE_MEMORY_MAX_SIZE_MB", "128")
		os.Setenv("RENTFREE_MEMORY_DEFAULT_TTL", "10m")
		defer func() {
			os.Unsetenv("RENTFREE_MEMORY_ENABLED")
			os.Unsetenv("RENTFREE_MEMORY_MAX_SIZE_MB")
			os.Unsetenv("RENTFREE_MEMORY_DEFAULT_TTL")
		}()

		cfg := DefaultConfig()
		applyEnvOverrides(cfg)

		if cfg.Memory.Enabled {
			t.Error("Memory.Enabled = true, want false")
		}
		if cfg.Memory.MaxSizeMB != 128 {
			t.Errorf("Memory.MaxSizeMB = %d, want 128", cfg.Memory.MaxSizeMB)
		}
		if cfg.Memory.DefaultTTL != 10*time.Minute {
			t.Errorf("Memory.DefaultTTL = %v, want 10m", cfg.Memory.DefaultTTL)
		}
	})

	t.Run("redis overrides", func(t *testing.T) {
		os.Setenv("RENTFREE_REDIS_ENABLED", "true")
		os.Setenv("RENTFREE_REDIS_ADDRESS", "redis.custom:6380")
		os.Setenv("RENTFREE_REDIS_PASSWORD", "secret123")
		os.Setenv("RENTFREE_REDIS_DB", "5")
		os.Setenv("RENTFREE_REDIS_KEY_PREFIX", "custom:")
		os.Setenv("RENTFREE_REDIS_DEFAULT_TTL", "1h")
		os.Setenv("RENTFREE_REDIS_POOL_SIZE", "50")
		os.Setenv("RENTFREE_REDIS_ENABLE_TLS", "true")
		os.Setenv("RENTFREE_REDIS_TLS_SKIP_VERIFY", "true")
		defer func() {
			os.Unsetenv("RENTFREE_REDIS_ENABLED")
			os.Unsetenv("RENTFREE_REDIS_ADDRESS")
			os.Unsetenv("RENTFREE_REDIS_PASSWORD")
			os.Unsetenv("RENTFREE_REDIS_DB")
			os.Unsetenv("RENTFREE_REDIS_KEY_PREFIX")
			os.Unsetenv("RENTFREE_REDIS_DEFAULT_TTL")
			os.Unsetenv("RENTFREE_REDIS_POOL_SIZE")
			os.Unsetenv("RENTFREE_REDIS_ENABLE_TLS")
			os.Unsetenv("RENTFREE_REDIS_TLS_SKIP_VERIFY")
		}()

		cfg := DefaultConfig()
		applyEnvOverrides(cfg)

		if !cfg.Redis.Enabled {
			t.Error("Redis.Enabled = false, want true")
		}
		if cfg.Redis.Address != "redis.custom:6380" {
			t.Errorf("Redis.Address = %s, want redis.custom:6380", cfg.Redis.Address)
		}
		if cfg.Redis.Password.Value() != "secret123" {
			t.Errorf("Redis.Password.Value() = %s, want secret123", cfg.Redis.Password.Value())
		}
		if cfg.Redis.DB != 5 {
			t.Errorf("Redis.DB = %d, want 5", cfg.Redis.DB)
		}
		if cfg.Redis.KeyPrefix != "custom:" {
			t.Errorf("Redis.KeyPrefix = %s, want custom:", cfg.Redis.KeyPrefix)
		}
		if cfg.Redis.DefaultTTL != 1*time.Hour {
			t.Errorf("Redis.DefaultTTL = %v, want 1h", cfg.Redis.DefaultTTL)
		}
		if cfg.Redis.PoolSize != 50 {
			t.Errorf("Redis.PoolSize = %d, want 50", cfg.Redis.PoolSize)
		}
		if !cfg.Redis.EnableTLS {
			t.Error("Redis.EnableTLS = false, want true")
		}
		if !cfg.Redis.TLSSkipVerify {
			t.Error("Redis.TLSSkipVerify = false, want true")
		}
	})

	t.Run("defaults overrides", func(t *testing.T) {
		os.Setenv("RENTFREE_DEFAULTS_TTL", "30m")
		os.Setenv("RENTFREE_DEFAULTS_LEVEL", "redis-only")
		os.Setenv("RENTFREE_DEFAULTS_FIRE_AND_FORGET", "false")
		defer func() {
			os.Unsetenv("RENTFREE_DEFAULTS_TTL")
			os.Unsetenv("RENTFREE_DEFAULTS_LEVEL")
			os.Unsetenv("RENTFREE_DEFAULTS_FIRE_AND_FORGET")
		}()

		cfg := DefaultConfig()
		applyEnvOverrides(cfg)

		if cfg.Defaults.TTL != 30*time.Minute {
			t.Errorf("Defaults.TTL = %v, want 30m", cfg.Defaults.TTL)
		}
		if cfg.Defaults.Level != "redis-only" {
			t.Errorf("Defaults.Level = %s, want redis-only", cfg.Defaults.Level)
		}
		if cfg.Defaults.FireAndForget {
			t.Error("Defaults.FireAndForget = true, want false")
		}
	})

	t.Run("circuit breaker overrides", func(t *testing.T) {
		os.Setenv("RENTFREE_CIRCUIT_BREAKER_ENABLED", "false")
		os.Setenv("RENTFREE_CIRCUIT_BREAKER_FAILURE_THRESHOLD", "10")
		os.Setenv("RENTFREE_CIRCUIT_BREAKER_OPEN_DURATION", "1m")
		defer func() {
			os.Unsetenv("RENTFREE_CIRCUIT_BREAKER_ENABLED")
			os.Unsetenv("RENTFREE_CIRCUIT_BREAKER_FAILURE_THRESHOLD")
			os.Unsetenv("RENTFREE_CIRCUIT_BREAKER_OPEN_DURATION")
		}()

		cfg := DefaultConfig()
		applyEnvOverrides(cfg)

		if cfg.CircuitBreaker.Enabled {
			t.Error("CircuitBreaker.Enabled = true, want false")
		}
		if cfg.CircuitBreaker.FailureThreshold != 10 {
			t.Errorf("CircuitBreaker.FailureThreshold = %d, want 10", cfg.CircuitBreaker.FailureThreshold)
		}
		if cfg.CircuitBreaker.OpenDuration != 1*time.Minute {
			t.Errorf("CircuitBreaker.OpenDuration = %v, want 1m", cfg.CircuitBreaker.OpenDuration)
		}
	})

	t.Run("retry overrides", func(t *testing.T) {
		os.Setenv("RENTFREE_RETRY_ENABLED", "false")
		os.Setenv("RENTFREE_RETRY_MAX_ATTEMPTS", "5")
		defer func() {
			os.Unsetenv("RENTFREE_RETRY_ENABLED")
			os.Unsetenv("RENTFREE_RETRY_MAX_ATTEMPTS")
		}()

		cfg := DefaultConfig()
		applyEnvOverrides(cfg)

		if cfg.Retry.Enabled {
			t.Error("Retry.Enabled = true, want false")
		}
		if cfg.Retry.MaxAttempts != 5 {
			t.Errorf("Retry.MaxAttempts = %d, want 5", cfg.Retry.MaxAttempts)
		}
	})

	t.Run("bulkhead overrides", func(t *testing.T) {
		os.Setenv("RENTFREE_BULKHEAD_ENABLED", "false")
		os.Setenv("RENTFREE_BULKHEAD_MAX_CONCURRENT", "50")
		defer func() {
			os.Unsetenv("RENTFREE_BULKHEAD_ENABLED")
			os.Unsetenv("RENTFREE_BULKHEAD_MAX_CONCURRENT")
		}()

		cfg := DefaultConfig()
		applyEnvOverrides(cfg)

		if cfg.Bulkhead.Enabled {
			t.Error("Bulkhead.Enabled = true, want false")
		}
		if cfg.Bulkhead.MaxConcurrent != 50 {
			t.Errorf("Bulkhead.MaxConcurrent = %d, want 50", cfg.Bulkhead.MaxConcurrent)
		}
	})

	t.Run("metrics overrides", func(t *testing.T) {
		os.Setenv("RENTFREE_METRICS_ENABLED", "false")
		os.Setenv("DD_AGENT_HOST", "datadog.custom")
		os.Setenv("DD_DOGSTATSD_PORT", "8126")
		os.Setenv("DD_SERVICE", "myapp")
		os.Setenv("DD_ENV", "test")
		os.Setenv("DD_VERSION", "1.0.0")
		defer func() {
			os.Unsetenv("RENTFREE_METRICS_ENABLED")
			os.Unsetenv("DD_AGENT_HOST")
			os.Unsetenv("DD_DOGSTATSD_PORT")
			os.Unsetenv("DD_SERVICE")
			os.Unsetenv("DD_ENV")
			os.Unsetenv("DD_VERSION")
		}()

		cfg := DefaultConfig()
		applyEnvOverrides(cfg)

		if cfg.Metrics.Enabled {
			t.Error("Metrics.Enabled = true, want false")
		}
		if !cfg.Metrics.DataDog.Enabled {
			t.Error("Metrics.DataDog.Enabled = false, want true (auto-enabled by DD_AGENT_HOST)")
		}
		if cfg.Metrics.DataDog.AgentHost != "datadog.custom" {
			t.Errorf("DataDog.AgentHost = %s, want datadog.custom", cfg.Metrics.DataDog.AgentHost)
		}
		if cfg.Metrics.DataDog.Port != 8126 {
			t.Errorf("DataDog.Port = %d, want 8126", cfg.Metrics.DataDog.Port)
		}
		if cfg.Metrics.DataDog.Prefix != "myapp" {
			t.Errorf("DataDog.Prefix = %s, want myapp", cfg.Metrics.DataDog.Prefix)
		}
	})

	t.Run("legacy RENTFREE_DATADOG vars still work", func(t *testing.T) {
		os.Setenv("RENTFREE_DATADOG_ENABLED", "true")
		os.Setenv("RENTFREE_DATADOG_PREFIX", "legacyapp")
		defer func() {
			os.Unsetenv("RENTFREE_DATADOG_ENABLED")
			os.Unsetenv("RENTFREE_DATADOG_PREFIX")
		}()

		cfg := DefaultConfig()
		applyEnvOverrides(cfg)

		if !cfg.Metrics.DataDog.Enabled {
			t.Error("Metrics.DataDog.Enabled = false, want true")
		}
		if cfg.Metrics.DataDog.Prefix != "legacyapp" {
			t.Errorf("DataDog.Prefix = %s, want legacyapp", cfg.Metrics.DataDog.Prefix)
		}
	})

	t.Run("DD_* vars take precedence over RENTFREE_DATADOG vars", func(t *testing.T) {
		os.Setenv("DD_AGENT_HOST", "dd-agent")
		os.Setenv("DD_SERVICE", "new-app")
		os.Setenv("RENTFREE_DATADOG_ENABLED", "false")
		os.Setenv("RENTFREE_DATADOG_PREFIX", "old-app")
		defer func() {
			os.Unsetenv("DD_AGENT_HOST")
			os.Unsetenv("DD_SERVICE")
			os.Unsetenv("RENTFREE_DATADOG_ENABLED")
			os.Unsetenv("RENTFREE_DATADOG_PREFIX")
		}()

		cfg := DefaultConfig()
		applyEnvOverrides(cfg)

		// DD_AGENT_HOST should auto-enable, ignoring RENTFREE_DATADOG_ENABLED=false
		if !cfg.Metrics.DataDog.Enabled {
			t.Error("Metrics.DataDog.Enabled = false, want true (DD_AGENT_HOST takes precedence)")
		}
		// DD_SERVICE should take precedence over RENTFREE_DATADOG_PREFIX
		if cfg.Metrics.DataDog.Prefix != "new-app" {
			t.Errorf("DataDog.Prefix = %s, want new-app", cfg.Metrics.DataDog.Prefix)
		}
	})
}

func TestSecretString(t *testing.T) {
	t.Run("Value returns actual secret", func(t *testing.T) {
		secret := NewSecretString("my-password-123")
		if secret.Value() != "my-password-123" {
			t.Errorf("Value() = %s, want my-password-123", secret.Value())
		}
	})

	t.Run("String returns redacted for non-empty", func(t *testing.T) {
		secret := NewSecretString("my-password-123")
		if secret.String() != "[REDACTED]" {
			t.Errorf("String() = %s, want [REDACTED]", secret.String())
		}
	})

	t.Run("String returns empty for empty secret", func(t *testing.T) {
		secret := SecretString{}
		if secret.String() != "" {
			t.Errorf("String() = %s, want empty string", secret.String())
		}
	})

	t.Run("MarshalJSON returns redacted for non-empty", func(t *testing.T) {
		secret := NewSecretString("my-password-123")
		data, err := json.Marshal(secret)
		if err != nil {
			t.Fatalf("MarshalJSON failed: %v", err)
		}
		if string(data) != `"[REDACTED]"` {
			t.Errorf("MarshalJSON = %s, want \"[REDACTED]\"", string(data))
		}
	})

	t.Run("MarshalJSON returns empty string for empty secret", func(t *testing.T) {
		secret := SecretString{}
		data, err := json.Marshal(secret)
		if err != nil {
			t.Fatalf("MarshalJSON failed: %v", err)
		}
		if string(data) != `""` {
			t.Errorf("MarshalJSON = %s, want empty string", string(data))
		}
	})

	t.Run("UnmarshalJSON loads actual value", func(t *testing.T) {
		var secret SecretString
		err := json.Unmarshal([]byte(`"super-secret"`), &secret)
		if err != nil {
			t.Fatalf("UnmarshalJSON failed: %v", err)
		}
		if secret.Value() != "super-secret" {
			t.Errorf("Value() after unmarshal = %s, want super-secret", secret.Value())
		}
	})

	t.Run("IsEmpty returns true for empty secret", func(t *testing.T) {
		secret := SecretString{}
		if !secret.IsEmpty() {
			t.Error("IsEmpty() = false, want true")
		}
	})

	t.Run("IsEmpty returns false for non-empty secret", func(t *testing.T) {
		secret := NewSecretString("password")
		if secret.IsEmpty() {
			t.Error("IsEmpty() = true, want false")
		}
	})

	t.Run("config JSON marshal redacts password", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Redis.Password = NewSecretString("super-secret-password")

		data, err := json.Marshal(cfg)
		if err != nil {
			t.Fatalf("Marshal config failed: %v", err)
		}

		// The JSON should contain [REDACTED], not the actual password
		jsonStr := string(data)
		if contains(jsonStr, "super-secret-password") {
			t.Error("JSON contains actual password, should be redacted")
		}
		if !contains(jsonStr, "[REDACTED]") {
			t.Error("JSON should contain [REDACTED] for password")
		}
	})

	t.Run("fmt.Sprintf redacts password", func(t *testing.T) {
		secret := NewSecretString("super-secret-password")
		output := fmt.Sprintf("password: %s", secret)
		if contains(output, "super-secret-password") {
			t.Errorf("fmt.Sprintf leaked password: %s", output)
		}
		if !contains(output, "[REDACTED]") {
			t.Errorf("fmt.Sprintf should contain [REDACTED], got: %s", output)
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
