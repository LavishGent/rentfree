// Package config provides configuration management for rentfree.
package config

import (
	"time"

	"github.com/LavishGent/rentfree/internal/types"
)

// SecretString is a string type that redacts its value when marshaled to JSON.
type SecretString = types.SecretString

// NewSecretString creates a new SecretString with the provided value.
func NewSecretString(value string) SecretString {
	return types.NewSecretString(value)
}

// Config contains all configuration for the rentfree cache manager.
//
//nolint:govet // Configuration struct - logical grouping prioritized over alignment
type Config struct {
	Redis          RedisConfig          `json:"redis"`
	Metrics        MetricsConfig        `json:"metrics"`
	CircuitBreaker CircuitBreakerConfig `json:"circuitBreaker"`
	Defaults       DefaultsConfig       `json:"defaults"`
	Memory         MemoryConfig         `json:"memory"`
	Retry          RetryConfig          `json:"retry"`
	Bulkhead       BulkheadConfig       `json:"bulkhead"`
	KeyValidation  KeyValidationConfig  `json:"keyValidation"`
}

// KeyValidationConfig contains configuration for cache key validation.
type KeyValidationConfig struct {
	ReservedPatterns  []string `json:"reservedPatterns"`
	MaxKeyLength      int      `json:"maxKeyLength"`
	Enabled           bool     `json:"enabled"`
	AllowEmpty        bool     `json:"allowEmpty"`
	AllowControlChars bool     `json:"allowControlChars"`
	AllowWhitespace   bool     `json:"allowWhitespace"`
}

// ToTypesConfig converts this config to a types.KeyValidationConfig.
func (c KeyValidationConfig) ToTypesConfig() types.KeyValidationConfig {
	return types.KeyValidationConfig{
		MaxKeyLength:      c.MaxKeyLength,
		AllowEmpty:        c.AllowEmpty,
		AllowControlChars: c.AllowControlChars,
		AllowWhitespace:   c.AllowWhitespace,
		ReservedPatterns:  c.ReservedPatterns,
	}
}

// MemoryConfig contains configuration for the memory cache layer.
type MemoryConfig struct {
	DefaultTTL       time.Duration `json:"defaultTTL"`
	CleanupInterval  time.Duration `json:"cleanupInterval"`
	MaxSizeMB        int           `json:"maxSizeMB"`
	Shards           int           `json:"shards"`
	MaxEntrySize     int           `json:"maxEntrySize"`
	Enabled          bool          `json:"enabled"`
	HardMaxCacheSize bool          `json:"hardMaxCacheSize"`
}

// RedisConfig contains configuration for the Redis cache layer.
//
//nolint:govet // Configuration struct - logical grouping prioritized over alignment
type RedisConfig struct {
	DefaultTTL          time.Duration `json:"defaultTTL"`
	DialTimeout         time.Duration `json:"dialTimeout"`
	ReadTimeout         time.Duration `json:"readTimeout"`
	WriteTimeout        time.Duration `json:"writeTimeout"`
	PoolTimeout         time.Duration `json:"poolTimeout"`
	HealthCheckInterval time.Duration `json:"healthCheckInterval"`
	Password            SecretString  `json:"password"`
	Address             string        `json:"address"`
	KeyPrefix           string        `json:"keyPrefix"`
	DB                  int           `json:"db"`
	PoolSize            int           `json:"poolSize"`
	MinIdleConns        int           `json:"minIdleConns"`
	MaxPendingWrites    int           `json:"maxPendingWrites"`
	Enabled             bool          `json:"enabled"`
	EnableTLS           bool          `json:"enableTLS"`
	TLSSkipVerify       bool          `json:"tlsSkipVerify"`
}

// DefaultsConfig contains default values for cache operations.
//
//nolint:govet // Small config struct - minimal alignment benefit
type DefaultsConfig struct {
	TTL      time.Duration `json:"ttl"`
	Level    string        `json:"level"`
	Priority string        `json:"priority"`
	// FireAndForget enables async Redis writes. When true, SET operations
	// are queued and may be dropped if the queue is full.
	FireAndForget bool `json:"fireAndForget"`
}

// CircuitBreakerConfig contains configuration for the circuit breaker pattern.
type CircuitBreakerConfig struct {
	Enabled             bool          `json:"enabled"`
	FailureThreshold    int           `json:"failureThreshold"`
	SuccessThreshold    int           `json:"successThreshold"`
	OpenDuration        time.Duration `json:"openDuration"`
	HalfOpenMaxRequests int           `json:"halfOpenMaxRequests"`
}

// RetryConfig contains configuration for the retry pattern.
type RetryConfig struct {
	InitialBackoff time.Duration `json:"initialBackoff"`
	MaxBackoff     time.Duration `json:"maxBackoff"`
	Multiplier     float64       `json:"multiplier"`
	MaxAttempts    int           `json:"maxAttempts"`
	Enabled        bool          `json:"enabled"`
	Jitter         bool          `json:"jitter"`
}

// BulkheadConfig contains configuration for the bulkhead pattern.
type BulkheadConfig struct {
	Enabled        bool          `json:"enabled"`
	MaxConcurrent  int           `json:"maxConcurrent"`
	MaxQueue       int           `json:"maxQueue"`
	AcquireTimeout time.Duration `json:"acquireTimeout"`
}

// MetricsConfig contains configuration for metrics publishing.
//
//nolint:govet // Small config struct - minimal alignment benefit
type MetricsConfig struct {
	PublishInterval time.Duration `json:"publishInterval"`
	DataDog         DataDogConfig `json:"datadog"`
	Enabled         bool          `json:"enabled"`
}

// DataDogConfig contains configuration for DataDog metrics publishing.
//
//nolint:govet // Small config struct - minimal alignment benefit
type DataDogConfig struct {
	Tags                   []string `json:"tags"`
	AgentHost              string   `json:"agentHost"`
	Prefix                 string   `json:"prefix"`
	Port                   int      `json:"port"`
	PublishIntervalSeconds int      `json:"publishIntervalSeconds"`
	Enabled                bool     `json:"enabled"`
}
