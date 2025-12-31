package rentfree

import (
	"github.com/LavishGent/rentfree/internal/types"
)

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

func NewCacheError(op, key, layer string, err error) *CacheError {
	return types.NewCacheError(op, key, layer, err)
}

func IsCacheMiss(err error) bool {
	return types.IsCacheMiss(err)
}

func IsRedisUnavailable(err error) bool {
	return types.IsRedisUnavailable(err)
}

func IsCircuitOpen(err error) bool {
	return types.IsCircuitOpen(err)
}

func IsRetryable(err error) bool {
	return types.IsRetryable(err)
}
