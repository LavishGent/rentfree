package rentfree

import (
	"github.com/darrell-green/rentfree/internal/cache"
	"github.com/darrell-green/rentfree/internal/config"
)

// New creates a new cache manager with default configuration.
func New(opts ...ManagerOption) (CacheManager, error) {
	cfg := config.DefaultConfig()
	return NewFromConfig(cfg, opts...)
}

// NewFromConfig creates a new cache manager from configuration.
func NewFromConfig(cfg *config.Config, opts ...ManagerOption) (CacheManager, error) {
	managerOpts := &ManagerOptions{}
	for _, opt := range opts {
		opt(managerOpts)
	}
	return cache.NewManager(cfg, managerOpts)
}

// NewFromFile creates a new cache manager from a JSON config file.
func NewFromFile(path string, opts ...ManagerOption) (CacheManager, error) {
	cfg, err := config.LoadWithEnv(path)
	if err != nil {
		return nil, err
	}
	return NewFromConfig(cfg, opts...)
}

// NewMemoryOnly creates a cache manager using only in-memory cache.
func NewMemoryOnly(opts ...ManagerOption) (CacheManager, error) {
	cfg := config.DefaultConfig()
	cfg.Redis.Enabled = false
	cfg.Defaults.Level = "memory-only"
	return NewFromConfig(cfg, opts...)
}

// Config returns a default configuration that can be modified before creating a manager.
func Config() *config.Config {
	return config.DefaultConfig()
}

// TestConfig returns a configuration suitable for unit tests.
func TestConfig() *config.Config {
	return config.ForTesting()
}
