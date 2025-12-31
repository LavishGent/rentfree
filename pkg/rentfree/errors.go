package rentfree

import (
	"github.com/LavishGent/rentfree/internal/types"
)

// CacheError represents a cache operation error.
type CacheError = types.CacheError

var (
	// ErrCacheMiss indicates that a requested key was not found in the cache.
	ErrCacheMiss = types.ErrCacheMiss
	// ErrRedisUnavailable indicates that the Redis server is not available.
	ErrRedisUnavailable = types.ErrRedisUnavailable
	// ErrCircuitOpen indicates that the circuit breaker is open.
	ErrCircuitOpen = types.ErrCircuitOpen
	// ErrClosed indicates that the cache manager has been closed.
	ErrClosed = types.ErrClosed
	// ErrWriteQueueFull indicates that the write queue is full.
	ErrWriteQueueFull = types.ErrWriteQueueFull
	// ErrBulkheadFull indicates that the bulkhead is at capacity.
	ErrBulkheadFull = types.ErrBulkheadFull
	// ErrBulkheadTimeout indicates that the bulkhead acquisition timed out.
	ErrBulkheadTimeout = types.ErrBulkheadTimeout
	// ErrSerializationFailed indicates that serialization failed.
	ErrSerializationFailed = types.ErrSerializationFailed
	// ErrInvalidKey indicates that a cache key is invalid.
	ErrInvalidKey = types.ErrInvalidKey
)

// NewCacheError creates a new cache error with operation, key, layer, and underlying error.
func NewCacheError(op, key, layer string, err error) *CacheError {
	return types.NewCacheError(op, key, layer, err)
}

// IsCacheMiss returns true if the error is a cache miss.
func IsCacheMiss(err error) bool {
	return types.IsCacheMiss(err)
}

// IsRedisUnavailable returns true if the error indicates Redis is unavailable.
func IsRedisUnavailable(err error) bool {
	return types.IsRedisUnavailable(err)
}

// IsCircuitOpen returns true if the error indicates the circuit breaker is open.
func IsCircuitOpen(err error) bool {
	return types.IsCircuitOpen(err)
}

// IsRetryable returns true if the error can be retried.
func IsRetryable(err error) bool {
	return types.IsRetryable(err)
}
