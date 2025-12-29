// Package config provides configuration management for rentfree.
package config

import (
	"time"

	"github.com/darrell-green/rentfree/internal/types"
)

type SecretString = types.SecretString

func NewSecretString(value string) SecretString {
	return types.NewSecretString(value)
}

type Config struct {
	Memory         MemoryConfig         `json:"memory"`
	Redis          RedisConfig          `json:"redis"`
	Defaults       DefaultsConfig       `json:"defaults"`
	CircuitBreaker CircuitBreakerConfig `json:"circuitBreaker"`
	Retry          RetryConfig          `json:"retry"`
	Bulkhead       BulkheadConfig       `json:"bulkhead"`
	Metrics        MetricsConfig        `json:"metrics"`
	KeyValidation  KeyValidationConfig  `json:"keyValidation"`
}

type KeyValidationConfig struct {
	Enabled           bool     `json:"enabled"`
	MaxKeyLength      int      `json:"maxKeyLength"`
	AllowEmpty        bool     `json:"allowEmpty"`
	AllowControlChars bool     `json:"allowControlChars"`
	AllowWhitespace   bool     `json:"allowWhitespace"`
	ReservedPatterns  []string `json:"reservedPatterns"`
}

func (c KeyValidationConfig) ToTypesConfig() types.KeyValidationConfig {
	return types.KeyValidationConfig{
		MaxKeyLength:      c.MaxKeyLength,
		AllowEmpty:        c.AllowEmpty,
		AllowControlChars: c.AllowControlChars,
		AllowWhitespace:   c.AllowWhitespace,
		ReservedPatterns:  c.ReservedPatterns,
	}
}

type MemoryConfig struct {
	Enabled          bool          `json:"enabled"`
	MaxSizeMB        int           `json:"maxSizeMB"`
	DefaultTTL       time.Duration `json:"defaultTTL"`
	CleanupInterval  time.Duration `json:"cleanupInterval"`
	Shards           int           `json:"shards"`
	MaxEntrySize     int           `json:"maxEntrySize"`
	HardMaxCacheSize bool          `json:"hardMaxCacheSize"`
}

type RedisConfig struct {
	Enabled       bool          `json:"enabled"`
	Address       string        `json:"address"`
	Password      SecretString  `json:"password"`
	DB            int           `json:"db"`
	KeyPrefix     string        `json:"keyPrefix"`
	DefaultTTL    time.Duration `json:"defaultTTL"`
	PoolSize      int           `json:"poolSize"`
	MinIdleConns  int           `json:"minIdleConns"`
	DialTimeout   time.Duration `json:"dialTimeout"`
	ReadTimeout   time.Duration `json:"readTimeout"`
	WriteTimeout  time.Duration `json:"writeTimeout"`
	PoolTimeout   time.Duration `json:"poolTimeout"`
	MaxPendingWrites    int           `json:"maxPendingWrites"`
	EnableTLS           bool          `json:"enableTLS"`
	TLSSkipVerify       bool          `json:"tlsSkipVerify"`
	HealthCheckInterval time.Duration `json:"healthCheckInterval"`
}

type DefaultsConfig struct {
	TTL      time.Duration `json:"ttl"`
	Level    string        `json:"level"`
	Priority string        `json:"priority"`
	// FireAndForget enables async Redis writes. When true, SET operations
	// are queued and may be dropped if the queue is full.
	FireAndForget bool `json:"fireAndForget"`
}

type CircuitBreakerConfig struct {
	Enabled             bool          `json:"enabled"`
	FailureThreshold    int           `json:"failureThreshold"`
	SuccessThreshold    int           `json:"successThreshold"`
	OpenDuration        time.Duration `json:"openDuration"`
	HalfOpenMaxRequests int           `json:"halfOpenMaxRequests"`
}

type RetryConfig struct {
	Enabled        bool          `json:"enabled"`
	MaxAttempts    int           `json:"maxAttempts"`
	InitialBackoff time.Duration `json:"initialBackoff"`
	MaxBackoff     time.Duration `json:"maxBackoff"`
	Multiplier     float64       `json:"multiplier"`
	Jitter         bool          `json:"jitter"`
}

type BulkheadConfig struct {
	Enabled        bool          `json:"enabled"`
	MaxConcurrent  int           `json:"maxConcurrent"`
	MaxQueue       int           `json:"maxQueue"`
	AcquireTimeout time.Duration `json:"acquireTimeout"`
}

type MetricsConfig struct {
	Enabled         bool          `json:"enabled"`
	PublishInterval time.Duration `json:"publishInterval"`
	DataDog         DataDogConfig `json:"datadog"`
}

type DataDogConfig struct {
	Enabled                bool     `json:"enabled"`
	AgentHost              string   `json:"agentHost"`
	Port                   int      `json:"port"`
	Prefix                 string   `json:"prefix"`
	Tags                   []string `json:"tags"`
	PublishIntervalSeconds int      `json:"publishIntervalSeconds"`
}
