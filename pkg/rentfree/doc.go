// Package rentfree provides a multi-layer caching library with minimal dependencies.
//
// rentfree offers a unified API for managing cache operations across multiple backends
// (in-memory and Redis) with built-in resilience patterns, graceful degradation,
// and observability features.
//
// # Features
//
//   - Multi-backend Support: In-memory (bigcache) and Redis with intelligent fallback
//   - Resilience Patterns: Circuit breaker, retry with exponential backoff, and bulkhead
//   - Cache-Aside Pattern: Built-in GetOrCreate for lazy loading with factory functions
//   - Graceful Degradation: Continues working when Redis fails by falling back to memory
//   - Observability: Metrics tracking with pluggable publishers
//   - Minimal Dependencies: Only bigcache and go-redis required
//
// # Quick Start
//
// Create a cache manager with default configuration (memory-only):
//
//	manager, err := rentfree.New()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer manager.Close()
//
// # Cache Operations
//
// Basic set and get operations:
//
//	ctx := context.Background()
//	user := User{ID: "123", Name: "Alice"}
//
//	// Set a value
//	err := manager.Set(ctx, "user:123", user)
//
//	// Get a value
//	var cached User
//	err = manager.Get(ctx, "user:123", &cached)
//
// Cache-aside pattern with GetOrCreate:
//
//	var result User
//	err := manager.GetOrCreate(ctx, "user:456", &result, func() (any, error) {
//	    // This function only runs on cache miss
//	    return fetchUserFromDB("456")
//	})
//
// # Cache Levels
//
// The library supports multiple cache levels for different use cases:
//
//   - MemoryOnly: High-frequency reads, session data
//   - RedisOnly: Shared state across instances
//   - MemoryThenRedis: Optimal performance with distribution (L1: memory, L2: Redis)
//   - All: Maximum availability
//
// Configure cache level when creating the manager:
//
//	cfg := rentfree.Config()
//	cfg.Defaults.Level = "memory-then-redis"
//	cfg.Redis.Enabled = true
//	cfg.Redis.Address = "localhost:6379"
//	manager, err := rentfree.NewFromConfig(cfg)
//
// # Options
//
// Use functional options to customize behavior per operation:
//
//	// Set with TTL
//	manager.Set(ctx, "key", value, rentfree.WithTTL(5*time.Minute))
//
//	// Set at specific cache level
//	manager.Set(ctx, "key", value, rentfree.WithLevel(rentfree.LevelRedisOnly))
//
// # Observability
//
// The library provides built-in metrics tracking. Use WithMetrics to enable:
//
//	manager, err := rentfree.New(
//	    rentfree.WithMetrics(rentfree.NewLoggingMetrics()),
//	)
//
// # Health Checks
//
// Check the health status of cache backends:
//
//	health := manager.Health(ctx)
//	if health.Status == "healthy" {
//	    fmt.Println("All backends operational")
//	}
//
// # Configuration
//
// Load configuration from a JSON file:
//
//	manager, err := rentfree.NewFromFile("config.json")
//
// Or use the default configuration:
//
//	cfg := rentfree.Config()
//	// Customize cfg...
//	manager, err := rentfree.NewFromConfig(cfg)
//
// For testing, use the test configuration:
//
//	cfg := rentfree.TestConfig()
//
// # Thread Safety
//
// All cache operations are thread-safe and can be used concurrently from multiple goroutines.
package rentfree
