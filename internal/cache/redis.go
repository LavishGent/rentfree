package cache

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/LavishGent/rentfree/internal/config"
	"github.com/LavishGent/rentfree/internal/types"
)

const (
	disconnectErrorThreshold = 5
)

type RedisCache struct {
	client *redis.Client
	config config.RedisConfig
	logger *slog.Logger

	mu            sync.RWMutex
	connected     atomic.Bool
	lastError     error
	lastErrorTime time.Time
	errorCount    atomic.Int64

	writeQueue    chan writeOp
	pendingWrites atomic.Int32
	droppedWrites atomic.Int64
	stopCh        chan struct{}
	wg            sync.WaitGroup

	healthCheckStopCh chan struct{}
	healthCheckWg     sync.WaitGroup

	hits    atomic.Int64
	misses  atomic.Int64
	sets    atomic.Int64
	deletes atomic.Int64
}

type writeOp struct {
	key   string
	value []byte
	ttl   time.Duration
}

func NewRedisCache(cfg config.RedisConfig, logger *slog.Logger) (*RedisCache, error) {
	if logger == nil {
		logger = slog.Default()
	}

	opts := &redis.Options{
		Addr:         cfg.Address,
		Password:     cfg.Password.Value(),
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolTimeout:  cfg.PoolTimeout,
	}

	if cfg.EnableTLS {
		opts.TLSConfig = &tls.Config{
			InsecureSkipVerify: cfg.TLSSkipVerify,
		}
		if cfg.TLSSkipVerify {
			logger.Warn("TLS certificate verification is disabled - this is insecure for production use")
		}
	}

	client := redis.NewClient(opts)

	rc := &RedisCache{
		client:            client,
		config:            cfg,
		logger:            logger.With("component", "redis-cache"),
		writeQueue:        make(chan writeOp, cfg.MaxPendingWrites),
		stopCh:            make(chan struct{}),
		healthCheckStopCh: make(chan struct{}),
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.DialTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		rc.logger.Warn("Redis initial connection failed", "error", err)
		rc.setError(err)
		// Don't return error - allow graceful degradation
	} else {
		rc.connected.Store(true)
		rc.logger.Info("Redis connected", "address", cfg.Address)
	}

	rc.wg.Add(1)
	go rc.asyncWriteWorker()

	if cfg.HealthCheckInterval > 0 {
		rc.healthCheckWg.Add(1)
		go rc.healthCheckWorker()
	}

	return rc, nil
}

func (c *RedisCache) Name() string {
	return "redis"
}

func (c *RedisCache) IsAvailable() bool {
	return c.connected.Load()
}

func (c *RedisCache) prefixKey(key string) string {
	return c.config.KeyPrefix + key
}

func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	if !c.connected.Load() {
		return nil, types.ErrRedisUnavailable
	}

	prefixedKey := c.prefixKey(key)

	data, err := c.client.Get(ctx, prefixedKey).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			c.misses.Add(1)
			return nil, types.ErrCacheMiss
		}
		c.handleError(err)
		return nil, types.NewCacheError("Get", key, "redis", err)
	}

	c.hits.Add(1)
	c.clearError()

	return data, nil
}

func (c *RedisCache) Set(ctx context.Context, key string, value []byte, opts *types.CacheOptions) error {
	if !c.connected.Load() {
		return types.ErrRedisUnavailable
	}

	if opts == nil {
		opts = types.DefaultOptions()
	}

	ttl := c.config.DefaultTTL
	if opts.TTL > 0 {
		ttl = opts.TTL
	}

	prefixedKey := c.prefixKey(key)

	if opts.FireAndForget {
		return c.setAsync(prefixedKey, value, ttl)
	}

	if err := c.client.Set(ctx, prefixedKey, value, ttl).Err(); err != nil {
		c.handleError(err)
		return types.NewCacheError("Set", key, "redis", err)
	}

	c.sets.Add(1)
	c.clearError()

	return nil
}

func (c *RedisCache) setAsync(key string, value []byte, ttl time.Duration) error {
	select {
	case c.writeQueue <- writeOp{key: key, value: value, ttl: ttl}:
		c.pendingWrites.Add(1)
		return nil
	default:
		c.droppedWrites.Add(1)
		c.logger.Warn("Write queue full, dropping SET",
			"key", key,
			"dropped_total", c.droppedWrites.Load(),
		)
		return types.ErrWriteQueueFull
	}
}

func (c *RedisCache) asyncWriteWorker() {
	defer c.wg.Done()

	for {
		select {
		case <-c.stopCh:
			for {
				select {
				case op := <-c.writeQueue:
					c.executeWrite(op)
				default:
					return
				}
			}
		case op := <-c.writeQueue:
			c.executeWrite(op)
		}
	}
}

func (c *RedisCache) executeWrite(op writeOp) {
	defer c.pendingWrites.Add(-1)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := c.client.Set(ctx, op.key, op.value, op.ttl).Err(); err != nil {
		c.handleError(err)
		c.logger.Debug("Async SET failed", "key", op.key, "error", err)
	} else {
		c.sets.Add(1)
		c.clearError()
	}
}

func (c *RedisCache) healthCheckWorker() {
	defer c.healthCheckWg.Done()

	ticker := time.NewTicker(c.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.healthCheckStopCh:
			return
		case <-ticker.C:
			c.performHealthCheck()
		}
	}
}

func (c *RedisCache) performHealthCheck() {
	wasConnected := c.connected.Load()

	ctx, cancel := context.WithTimeout(context.Background(), c.config.DialTimeout)
	defer cancel()

	err := c.client.Ping(ctx).Err()
	if err != nil {
		if wasConnected {
			c.logger.Warn("Redis health check failed", "error", err)
			c.setError(err)
		}
		return
	}

	if !wasConnected {
		c.connected.Store(true)
		c.errorCount.Store(0)
		c.logger.Info("Redis connection restored via health check")
	}
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
	if !c.connected.Load() {
		return types.ErrRedisUnavailable
	}

	prefixedKey := c.prefixKey(key)

	if err := c.client.Del(ctx, prefixedKey).Err(); err != nil {
		c.handleError(err)
		return types.NewCacheError("Delete", key, "redis", err)
	}

	c.deletes.Add(1)
	c.clearError()

	return nil
}

func (c *RedisCache) Contains(ctx context.Context, key string) (bool, error) {
	if !c.connected.Load() {
		return false, types.ErrRedisUnavailable
	}

	prefixedKey := c.prefixKey(key)

	exists, err := c.client.Exists(ctx, prefixedKey).Result()
	if err != nil {
		c.handleError(err)
		return false, types.NewCacheError("Contains", key, "redis", err)
	}

	c.clearError()
	return exists > 0, nil
}

func (c *RedisCache) Clear(ctx context.Context) error {
	if !c.connected.Load() {
		return types.ErrRedisUnavailable
	}

	pattern := c.prefixKey("*")
	return c.clearByPatternInternal(ctx, pattern)
}

func (c *RedisCache) GetMany(ctx context.Context, keys []string) (map[string][]byte, error) {
	if !c.connected.Load() {
		return nil, types.ErrRedisUnavailable
	}

	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}

	prefixedKeys := make([]string, len(keys))
	for i, key := range keys {
		prefixedKeys[i] = c.prefixKey(key)
	}

	results, err := c.client.MGet(ctx, prefixedKeys...).Result()
	if err != nil {
		c.handleError(err)
		return nil, types.NewCacheError("GetMany", "", "redis", err)
	}

	resultMap := make(map[string][]byte, len(keys))
	for i, result := range results {
		if result != nil {
			if str, ok := result.(string); ok {
				resultMap[keys[i]] = []byte(str)
				c.hits.Add(1)
			}
		} else {
			c.misses.Add(1)
		}
	}

	c.clearError()
	return resultMap, nil
}

func (c *RedisCache) SetMany(ctx context.Context, items map[string][]byte, opts *types.CacheOptions) error {
	if !c.connected.Load() {
		return types.ErrRedisUnavailable
	}

	if len(items) == 0 {
		return nil
	}

	if opts == nil {
		opts = types.DefaultOptions()
	}

	ttl := c.config.DefaultTTL
	if opts.TTL > 0 {
		ttl = opts.TTL
	}

	pipe := c.client.Pipeline()

	for key, value := range items {
		prefixedKey := c.prefixKey(key)
		pipe.Set(ctx, prefixedKey, value, ttl)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		c.handleError(err)
		return types.NewCacheError("SetMany", "", "redis", err)
	}

	c.sets.Add(int64(len(items)))
	c.clearError()

	return nil
}

func (c *RedisCache) ClearByPattern(ctx context.Context, pattern string) error {
	if !c.connected.Load() {
		return types.ErrRedisUnavailable
	}

	fullPattern := c.prefixKey(pattern)
	return c.clearByPatternInternal(ctx, fullPattern)
}

func (c *RedisCache) clearByPatternInternal(ctx context.Context, pattern string) error {
	var cursor uint64
	var deleted int64

	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			c.handleError(err)
			return types.NewCacheError("ClearByPattern", pattern, "redis", err)
		}

		if len(keys) > 0 {
			if err := c.client.Del(ctx, keys...).Err(); err != nil {
				c.handleError(err)
				return types.NewCacheError("ClearByPattern", pattern, "redis", err)
			}
			deleted += int64(len(keys))
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	c.logger.Debug("Cleared keys by pattern", "pattern", pattern, "deleted", deleted)
	c.clearError()
	return nil
}

func (c *RedisCache) Close() error {
	c.connected.Store(false)

	close(c.healthCheckStopCh)
	c.healthCheckWg.Wait()

	close(c.stopCh)
	c.wg.Wait()

	return c.client.Close()
}

func (c *RedisCache) PendingWrites() int {
	return int(c.pendingWrites.Load())
}

func (c *RedisCache) DroppedWrites() int64 {
	return c.droppedWrites.Load()
}

func (c *RedisCache) handleError(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lastError = err
	c.lastErrorTime = time.Now()
	count := c.errorCount.Add(1)

	if count >= disconnectErrorThreshold {
		if c.connected.CompareAndSwap(true, false) {
			c.logger.Warn("Redis marked as disconnected after errors",
				"error_count", count,
				"last_error", err,
			)
		}
	}
}

func (c *RedisCache) clearError() {
	if c.errorCount.Swap(0) > 0 {
		if c.connected.CompareAndSwap(false, true) {
			c.logger.Info("Redis connection restored")
		}
	}
}

func (c *RedisCache) setError(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastError = err
	c.lastErrorTime = time.Now()
	c.connected.Store(false)
}

func (c *RedisCache) LastError() (error, time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastError, c.lastErrorTime
}

func (c *RedisCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *RedisCache) Reconnect(ctx context.Context) error {
	if err := c.client.Ping(ctx).Err(); err != nil {
		return err
	}
	c.connected.Store(true)
	c.errorCount.Store(0)
	c.logger.Info("Redis reconnected successfully")
	return nil
}

var _ types.RedisCacheLayer = (*RedisCache)(nil)
