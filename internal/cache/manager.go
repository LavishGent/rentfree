package cache

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/LavishGent/rentfree/internal/config"
	"github.com/LavishGent/rentfree/internal/resilience"
	"github.com/LavishGent/rentfree/internal/types"
)

// DefaultShutdownTimeout is the default timeout for shutting down the cache manager.
const DefaultShutdownTimeout = 30 * time.Second

// DefaultBackgroundOpTimeout is the default timeout for background operations.
const DefaultBackgroundOpTimeout = 5 * time.Second

// Manager is the main cache manager that coordinates memory and Redis caches.
type Manager struct {
	memory         types.MemoryCacheLayer
	redis          types.RedisCacheLayer
	policy         *resilience.Policy
	serializer     types.Serializer
	config         *config.Config
	metrics        types.MetricsRecorder
	logger         *slog.Logger
	keyValidator   *types.KeyValidator
	shutdownCancel context.CancelFunc
	shutdownCtx    context.Context
	sfGroup        singleflight.Group
	bgWg           sync.WaitGroup
	bgMu           sync.Mutex
	closed         atomic.Bool
}

// NewManager creates a new cache manager with the given configuration and options.
//
//nolint:gocyclo // Configuration initialization requires multiple conditional checks
func NewManager(cfg *config.Config, opts *types.ManagerOptions) (*Manager, error) {
	logger := slog.Default()
	if opts != nil && opts.Logger != nil {
		logger = slog.New(slogAdapter{logger: opts.Logger})
	}
	logger = logger.With("component", "cache-manager")

	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())

	m := &Manager{
		config:         cfg,
		logger:         logger,
		serializer:     NewJSONSerializer(),
		shutdownCtx:    shutdownCtx,
		shutdownCancel: shutdownCancel,
	}

	if opts != nil {
		if opts.Serializer != nil {
			m.serializer = opts.Serializer
		}
		if opts.Metrics != nil {
			m.metrics = opts.Metrics
		}
		if opts.RedisAddress != "" {
			cfg.Redis.Address = opts.RedisAddress
		}
		if !opts.RedisPassword.IsEmpty() {
			cfg.Redis.Password = opts.RedisPassword
		}
		if opts.RedisDB != 0 {
			cfg.Redis.DB = opts.RedisDB
		}
		if opts.DisableRedis {
			cfg.Redis.Enabled = false
		}
		if opts.DisableResilience {
			cfg.CircuitBreaker.Enabled = false
			cfg.Retry.Enabled = false
			cfg.Bulkhead.Enabled = false
		}
	}

	if cfg.KeyValidation.Enabled {
		m.keyValidator = types.NewKeyValidator(cfg.KeyValidation.ToTypesConfig())
	}

	if cfg.Memory.Enabled {
		memCache, err := NewMemoryCache(cfg.Memory, logger)
		if err != nil {
			return nil, err
		}
		m.memory = memCache
	} else {
		m.memory = NewDisabledMemoryCache()
	}

	if cfg.Redis.Enabled {
		redisCache, err := NewRedisCache(&cfg.Redis, logger)
		if err != nil {
			logger.Warn("Failed to create Redis cache, using memory-only mode", "error", err)
			m.redis = NewDisabledRedisCache()
		} else {
			m.redis = redisCache
		}
	} else {
		m.redis = NewDisabledRedisCache()
	}

	m.policy = resilience.NewPolicy(cfg)

	m.policy.SetOnCircuitStateChange(func(from, to resilience.State) {
		logger.Info("Circuit breaker state changed",
			"from", from.String(),
			"to", to.String(),
		)
		if m.metrics != nil {
			m.metrics.RecordCircuitBreakerStateChange(from.String(), to.String())
		}
	})

	return m, nil
}

// Get retrieves a value from the cache.
func (m *Manager) Get(ctx context.Context, key string, dest any, opts ...types.Option) error {
	if m.closed.Load() {
		return types.ErrClosed
	}

	if err := m.validateKey(key); err != nil {
		return err
	}

	start := time.Now()
	options := m.applyDefaults(opts...)

	var data []byte
	var err error
	var layer string

	switch options.Level {
	case types.LevelMemoryOnly:
		data, err = m.memory.Get(ctx, key)
		layer = "memory"

	case types.LevelRedisOnly:
		data, err = m.getFromRedis(ctx, key)
		layer = "redis"

	case types.LevelMemoryThenRedis, types.LevelAll:
		data, layer, err = m.getFromBothLayers(ctx, key)

	default:
		return types.ErrCacheMiss
	}

	latency := time.Since(start)

	if err != nil {
		if types.IsCacheMiss(err) && m.metrics != nil {
			m.metrics.RecordMiss(layer, key, latency)
		}
		return err
	}

	if err := m.serializer.Unmarshal(data, dest); err != nil {
		m.logger.Debug("Deserialization failed", "key", key, "error", err)
		return err
	}

	if m.metrics != nil {
		m.metrics.RecordHit(layer, key, latency)
	}

	return nil
}

// getFromBothLayers tries memory first, then Redis.
func (m *Manager) getFromBothLayers(ctx context.Context, key string) ([]byte, string, error) {
	data, err := m.memory.Get(ctx, key)
	if err == nil {
		return data, "memory", nil
	}

	if !types.IsCacheMiss(err) {
		m.logger.Debug("Memory cache error", "key", key, "error", err)
	}

	data, err = m.getFromRedis(ctx, key)
	if err != nil {
		return nil, "redis", err
	}

	m.runBackground(func(ctx context.Context) {
		if setErr := m.memory.Set(ctx, key, data, nil); setErr != nil {
			m.logger.Debug("Failed to populate memory from Redis", "key", key, "error", setErr)
		}
	})

	return data, "redis", nil
}

// getFromRedis gets data from Redis with resilience.
func (m *Manager) getFromRedis(ctx context.Context, key string) ([]byte, error) {
	if !m.redis.IsAvailable() {
		return nil, types.ErrRedisUnavailable
	}

	result, err := m.policy.ExecuteWithResult(ctx, func(ctx context.Context) (any, error) {
		return m.redis.Get(ctx, key)
	})

	if err != nil {
		return nil, err
	}

	data, ok := result.([]byte)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}

	return data, nil
}

// Set stores a value in the cache.
func (m *Manager) Set(ctx context.Context, key string, value any, opts ...types.Option) error {
	if m.closed.Load() {
		return types.ErrClosed
	}

	if err := m.validateKey(key); err != nil {
		return err
	}

	start := time.Now()
	options := m.applyDefaults(opts...)

	data, err := m.serializer.Marshal(value)
	if err != nil {
		return err
	}

	var setErr error

	switch options.Level {
	case types.LevelMemoryOnly:
		setErr = m.memory.Set(ctx, key, data, options)

	case types.LevelRedisOnly:
		setErr = m.setToRedis(ctx, key, data, options)

	case types.LevelMemoryThenRedis, types.LevelAll:
		memErr := m.memory.Set(ctx, key, data, options)
		redisErr := m.setToRedis(ctx, key, data, options)

		if memErr != nil {
			setErr = memErr
		} else if redisErr != nil && !options.FireAndForget {
			m.logger.Warn("Redis SET failed, wrote to memory only", "key", key, "error", redisErr)
		}
	}

	latency := time.Since(start)

	if m.metrics != nil {
		m.metrics.RecordSet(options.Level.String(), key, len(data), latency)
	}

	return setErr
}

// setToRedis sets data to Redis with resilience.
func (m *Manager) setToRedis(ctx context.Context, key string, data []byte, opts *types.CacheOptions) error {
	if !m.redis.IsAvailable() && !opts.FireAndForget {
		return types.ErrRedisUnavailable
	}

	return m.policy.Execute(ctx, func(ctx context.Context) error {
		return m.redis.Set(ctx, key, data, opts)
	})
}

// GetOrCreate retrieves a value or creates it using the factory function.
// It uses singleflight to prevent thundering herd: concurrent requests for the same key will share a single factory invocation.
func (m *Manager) GetOrCreate(ctx context.Context, key string, dest any, factory func() (any, error), opts ...types.Option) error {
	if m.closed.Load() {
		return types.ErrClosed
	}

	err := m.Get(ctx, key, dest, opts...)
	if err == nil {
		return nil
	}

	if !types.IsCacheMiss(err) {
		return err
	}

	result, err, _ := m.sfGroup.Do(key, func() (any, error) {
		var check any
		if checkErr := m.Get(ctx, key, &check, opts...); checkErr == nil {
			data, marshalErr := m.serializer.Marshal(check)
			if marshalErr != nil {
				return nil, marshalErr
			}
			return data, nil
		}

		value, factoryErr := factory()
		if factoryErr != nil {
			return nil, factoryErr
		}

		data, marshalErr := m.serializer.Marshal(value)
		if marshalErr != nil {
			return nil, marshalErr
		}

		if setErr := m.Set(ctx, key, value, opts...); setErr != nil {
			m.logger.Debug("Failed to cache factory result", "key", key, "error", setErr)
		}

		return data, nil
	})

	if err != nil {
		return err
	}

	data, ok := result.([]byte)
	if !ok {
		return fmt.Errorf("unexpected result type: %T", result)
	}

	return m.serializer.Unmarshal(data, dest)
}

// Delete removes a value from the cache.
func (m *Manager) Delete(ctx context.Context, key string, opts ...types.Option) error {
	if m.closed.Load() {
		return types.ErrClosed
	}

	if err := m.validateKey(key); err != nil {
		return err
	}

	start := time.Now()
	options := m.applyDefaults(opts...)

	var err error

	switch options.Level {
	case types.LevelMemoryOnly:
		err = m.memory.Delete(ctx, key)

	case types.LevelRedisOnly:
		err = m.redis.Delete(ctx, key)

	case types.LevelMemoryThenRedis, types.LevelAll:
		memErr := m.memory.Delete(ctx, key)
		redisErr := m.redis.Delete(ctx, key)
		if memErr != nil {
			err = memErr
		} else if redisErr != nil {
			err = redisErr
		}
	}

	if m.metrics != nil {
		m.metrics.RecordDelete(options.Level.String(), key, time.Since(start))
	}

	return err
}

// DeleteMany attempts to delete all keys and returns a combined error if any deletions fail.
func (m *Manager) DeleteMany(ctx context.Context, keys []string, opts ...types.Option) error {
	if m.closed.Load() {
		return types.ErrClosed
	}

	if err := m.validateKeys(keys); err != nil {
		return err
	}

	var errs []error
	for _, key := range keys {
		if err := m.Delete(ctx, key, opts...); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// Contains checks if a key exists in the cache.
func (m *Manager) Contains(ctx context.Context, key string, opts ...types.Option) (bool, error) {
	if m.closed.Load() {
		return false, types.ErrClosed
	}

	if err := m.validateKey(key); err != nil {
		return false, err
	}

	options := m.applyDefaults(opts...)

	switch options.Level {
	case types.LevelMemoryOnly:
		return m.memory.Contains(ctx, key)

	case types.LevelRedisOnly:
		return m.redis.Contains(ctx, key)

	case types.LevelMemoryThenRedis, types.LevelAll:
		exists, err := m.memory.Contains(ctx, key)
		if err != nil {
			m.logger.Debug("Memory contains check failed", "key", key, "error", err)
		} else if exists {
			return true, nil
		}
		return m.redis.Contains(ctx, key)
	}

	return false, nil
}

// GetMany retrieves multiple values from the cache.
func (m *Manager) GetMany(ctx context.Context, keys []string, opts ...types.Option) (map[string][]byte, error) {
	if m.closed.Load() {
		return nil, types.ErrClosed
	}

	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}

	if err := m.validateKeys(keys); err != nil {
		return nil, err
	}

	options := m.applyDefaults(opts...)
	results := make(map[string][]byte)
	var missingKeys []string

	if options.Level.IncludesMemory() {
		for _, key := range keys {
			data, err := m.memory.Get(ctx, key)
			if err == nil {
				results[key] = data
			} else {
				missingKeys = append(missingKeys, key)
			}
		}
	} else {
		missingKeys = keys
	}

	if len(missingKeys) > 0 && options.Level.IncludesRedis() && m.redis.IsAvailable() {
		redisResults, err := m.redis.GetMany(ctx, missingKeys)
		if err == nil {
			for k, v := range redisResults {
				results[k] = v
				key, value := k, v
				m.runBackground(func(ctx context.Context) {
					if setErr := m.memory.Set(ctx, key, value, nil); setErr != nil {
						m.logger.Debug("Failed to populate memory from Redis GetMany", "key", key, "error", setErr)
					}
				})
			}
		}
	}

	return results, nil
}

// SetMany stores multiple values in the cache.
func (m *Manager) SetMany(ctx context.Context, items map[string]any, opts ...types.Option) error {
	if m.closed.Load() {
		return types.ErrClosed
	}

	if len(items) == 0 {
		return nil
	}

	if m.keyValidator != nil {
		for key := range items {
			if err := m.keyValidator.Validate(key); err != nil {
				return err
			}
		}
	}

	options := m.applyDefaults(opts...)

	serializedItems := make(map[string][]byte, len(items))
	for key, value := range items {
		data, err := m.serializer.Marshal(value)
		if err != nil {
			return err
		}
		serializedItems[key] = data
	}

	var memoryErrors []string

	if options.Level.IncludesMemory() {
		for key, data := range serializedItems {
			if err := m.memory.Set(ctx, key, data, options); err != nil {
				memoryErrors = append(memoryErrors, key)
				m.logger.Warn("Memory SetMany failed for key", "key", key, "error", err)
			}
		}
	}

	if options.Level.IncludesRedis() && m.redis.IsAvailable() {
		if err := m.redis.SetMany(ctx, serializedItems, options); err != nil {
			// Log at WARN for visibility, but don't fail since memory may have succeeded
			m.logger.Warn("Redis SetMany failed", "error", err, "keys_count", len(serializedItems))
		}
	}

	// If memory writes had errors, return an error indicating partial failure
	if len(memoryErrors) > 0 {
		return types.NewCacheError("SetMany", "", "memory",
			fmt.Errorf("failed to set %d/%d keys", len(memoryErrors), len(items)))
	}

	return nil
}

// Clear removes all entries from the specified cache level.
func (m *Manager) Clear(ctx context.Context, level types.CacheLevel) error {
	if m.closed.Load() {
		return types.ErrClosed
	}

	var err error

	switch level {
	case types.LevelMemoryOnly:
		err = m.memory.Clear(ctx)

	case types.LevelRedisOnly:
		err = m.redis.Clear(ctx)

	case types.LevelAll, types.LevelMemoryThenRedis:
		memErr := m.memory.Clear(ctx)
		redisErr := m.redis.Clear(ctx)
		if memErr != nil {
			err = memErr
		} else if redisErr != nil {
			err = redisErr
		}
	}

	return err
}

// ClearByPattern removes entries matching the given pattern.
func (m *Manager) ClearByPattern(ctx context.Context, pattern string, level types.CacheLevel) error {
	if m.closed.Load() {
		return types.ErrClosed
	}

	switch level {
	case types.LevelMemoryOnly:
		return m.memory.ClearByPattern(ctx, pattern)

	case types.LevelRedisOnly:
		return m.redis.ClearByPattern(ctx, pattern)

	case types.LevelAll, types.LevelMemoryThenRedis:
		memErr := m.memory.ClearByPattern(ctx, pattern)
		redisErr := m.redis.ClearByPattern(ctx, pattern)
		if memErr != nil {
			return memErr
		}
		return redisErr
	}

	return nil
}

// Health returns comprehensive health metrics for the cache.
func (m *Manager) Health(ctx context.Context) (*types.HealthMetrics, error) {
	metrics := &types.HealthMetrics{
		Timestamp: time.Now(),
	}

	memStats := m.memory.Stats()
	metrics.Memory = types.MemoryHealthMetrics{
		Status:          types.HealthStatusHealthy,
		Available:       m.memory.IsAvailable(),
		EntryCount:      m.memory.EntryCount(),
		SizeBytes:       m.memory.Size(),
		MaxSizeBytes:    m.memory.MaxSize(),
		UsagePercentage: m.memory.UsagePercentage(),
		HitCount:        memStats.Hits,
		MissCount:       memStats.Misses,
		HitRatio:        m.memory.HitRatio(),
		EvictionCount:   memStats.Evictions,
	}

	metrics.Redis = types.RedisHealthMetrics{
		Status:              types.HealthStatusHealthy,
		Available:           m.redis.IsAvailable(),
		Connected:           m.redis.IsAvailable(),
		CircuitBreakerState: m.policy.CircuitState().String(),
		PendingWrites:       m.redis.PendingWrites(),
		DroppedWrites:       m.redis.DroppedWrites(),
	}

	if !m.redis.IsAvailable() {
		metrics.Redis.Status = types.HealthStatusUnhealthy
	}

	switch {
	case metrics.Memory.Status == types.HealthStatusHealthy && metrics.Redis.Status == types.HealthStatusHealthy:
		metrics.Status = types.HealthStatusHealthy
	case metrics.Memory.Status == types.HealthStatusHealthy:
		metrics.Status = types.HealthStatusDegraded
	default:
		metrics.Status = types.HealthStatusUnhealthy
	}

	return metrics, nil
}

// IsHealthy returns true if the cache is functioning normally.
func (m *Manager) IsHealthy(ctx context.Context) bool {
	return m.memory.IsAvailable()
}

// IsRedisAvailable returns true if Redis is connected.
func (m *Manager) IsRedisAvailable() bool {
	return m.redis.IsAvailable() && !m.policy.IsCircuitOpen()
}

// IsMemoryAvailable returns true if memory cache is available.
func (m *Manager) IsMemoryAvailable() bool {
	return m.memory.IsAvailable()
}

// Close releases all resources using the default shutdown timeout.
// It waits for all in-flight background operations to complete before closing the underlying cache layers.
func (m *Manager) Close() error {
	return m.CloseWithTimeout(DefaultShutdownTimeout)
}

// CloseWithTimeout releases all resources with a configurable timeout.
// If background operations don't complete within the timeout, it returns ErrShutdownTimeout but still proceeds to close cache layers.
func (m *Manager) CloseWithTimeout(timeout time.Duration) error {
	// Acquire bgMu to prevent new background operations from starting.
	// This synchronizes with runBackground to ensure no Add calls happen after we set closed=true and before Wait completes.
	m.bgMu.Lock()
	if m.closed.Swap(true) {
		m.bgMu.Unlock()
		return nil
	}
	// Signal shutdown to all background operations
	m.shutdownCancel()
	m.bgMu.Unlock()

	m.logger.Info("Closing cache manager, waiting for background operations", "timeout", timeout)

	done := make(chan struct{})
	go func() {
		m.bgWg.Wait()
		close(done)
	}()

	var timedOut bool
	select {
	case <-done:
		m.logger.Info("Background operations complete, closing cache layers")
	case <-time.After(timeout):
		m.logger.Warn("Shutdown timeout exceeded, proceeding with close", "timeout", timeout)
		timedOut = true
	}

	var errs []error

	if timedOut {
		errs = append(errs, types.ErrShutdownTimeout)
	}

	if err := m.memory.Close(); err != nil {
		errs = append(errs, err)
	}

	if err := m.redis.Close(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// runBackground executes fn in a background goroutine that is tracked for graceful shutdown.
// The function receives a context derived from the shutdown context with a timeout.
// The goroutine will not be started if the manager is already closed.
func (m *Manager) runBackground(fn func(ctx context.Context)) {
	// Hold bgMu while checking closed and adding to WaitGroup to prevent a race with CloseWithTimeout where Add is called after Wait starts.
	m.bgMu.Lock()
	if m.closed.Load() {
		m.bgMu.Unlock()
		return
	}
	m.bgWg.Add(1)
	m.bgMu.Unlock()

	go func() {
		defer m.bgWg.Done()
		ctx, cancel := context.WithTimeout(m.shutdownCtx, DefaultBackgroundOpTimeout)
		defer cancel()
		fn(ctx)
	}()
}

func (m *Manager) validateKey(key string) error {
	if m.keyValidator == nil {
		return nil
	}
	return m.keyValidator.Validate(key)
}

func (m *Manager) validateKeys(keys []string) error {
	if m.keyValidator == nil {
		return nil
	}
	for _, key := range keys {
		if err := m.keyValidator.Validate(key); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) applyDefaults(opts ...types.Option) *types.CacheOptions {
	options := types.ApplyOptions(opts...)

	if options.TTL == 0 {
		options.TTL = m.config.Defaults.TTL
	}

	if options.Level == 0 {
		options.Level = parseCacheLevel(m.config.Defaults.Level)
	}

	if options.Priority == 0 {
		options.Priority = parsePriority(m.config.Defaults.Priority)
	}

	if m.config.Defaults.FireAndForget && !options.FireAndForget {
		options.FireAndForget = true
	}

	return options
}

func parseCacheLevel(s string) types.CacheLevel {
	switch s {
	case "memory-only":
		return types.LevelMemoryOnly
	case "redis-only":
		return types.LevelRedisOnly
	case "memory-then-redis":
		return types.LevelMemoryThenRedis
	case "all":
		return types.LevelAll
	default:
		return types.LevelMemoryThenRedis
	}
}

func parsePriority(s string) types.CachePriority {
	switch s {
	case "low":
		return types.PriorityLow
	case "normal":
		return types.PriorityNormal
	case "high":
		return types.PriorityHigh
	case "never-remove":
		return types.PriorityNeverRemove
	default:
		return types.PriorityNormal
	}
}

//nolint:govet // Simple adapter struct - alignment optimization minimal
type slogAdapter struct {
	attrs  []slog.Attr
	logger types.Logger
	group  string // current group prefix from WithGroup calls
}

// Enabled implements slog.Handler.
func (a slogAdapter) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

// Handle implements slog.Handler.
//
//nolint:gocritic // slog.Handler interface requires passing Record by value
func (a slogAdapter) Handle(ctx context.Context, r slog.Record) error {
	args := make([]any, 0, (len(a.attrs)+r.NumAttrs())*2)

	for _, attr := range a.attrs {
		key := attr.Key
		if a.group != "" {
			key = a.group + "." + key
		}
		args = append(args, key, attr.Value.Any())
	}

	r.Attrs(func(attr slog.Attr) bool {
		key := attr.Key
		if a.group != "" {
			key = a.group + "." + key
		}
		args = append(args, key, attr.Value.Any())
		return true
	})

	switch r.Level {
	case slog.LevelDebug:
		a.logger.Debug(r.Message, args...)
	case slog.LevelInfo:
		a.logger.Info(r.Message, args...)
	case slog.LevelWarn:
		a.logger.Warn(r.Message, args...)
	case slog.LevelError:
		a.logger.Error(r.Message, args...)
	}
	return nil
}

// WithAttrs implements slog.Handler.
func (a slogAdapter) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(a.attrs), len(a.attrs)+len(attrs))
	copy(newAttrs, a.attrs)
	newAttrs = append(newAttrs, attrs...)
	return slogAdapter{
		logger: a.logger,
		attrs:  newAttrs,
		group:  a.group,
	}
}

// WithGroup implements slog.Handler.
func (a slogAdapter) WithGroup(name string) slog.Handler {
	newGroup := name
	if a.group != "" {
		newGroup = a.group + "." + name
	}
	return slogAdapter{
		logger: a.logger,
		attrs:  a.attrs,
		group:  newGroup,
	}
}
