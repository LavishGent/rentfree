package rentfree

import (
	"github.com/LavishGent/rentfree/internal/types"
)

// CacheError represents a cache operation error.
type CacheError = types.CacheError

var (
	ErrCacheMiss           = types.ErrCacheMiss
	ErrRedisUnavailable    = types.ErrRedisUnavailable
	ErrCircuitOpen         = types.ErrCircuitOpen
	ErrClosed              = types.ErrClosed
	ErrWriteQueueFull      = types.ErrWriteQueueFull
	ErrBulkheadFull        = types.ErrBulkheadFull
	ErrBulkheadTimeout     = types.ErrBulkheadTimeout
	ErrSerializationFailed = types.ErrSerializationFailed
	ErrInvalidKey          = types.ErrInvalidKey
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
