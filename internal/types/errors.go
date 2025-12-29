package types

import (
	"errors"
	"fmt"
)

var (
	ErrCacheMiss           = errors.New("cache: key not found")
	ErrRedisUnavailable    = errors.New("cache: redis unavailable")
	ErrCircuitOpen         = errors.New("cache: circuit breaker open")
	ErrClosed              = errors.New("cache: manager closed")
	ErrWriteQueueFull      = errors.New("cache: write queue full")
	ErrBulkheadFull        = errors.New("cache: bulkhead at capacity")
	ErrBulkheadTimeout     = errors.New("cache: bulkhead timeout")
	ErrSerializationFailed = errors.New("cache: serialization failed")
	ErrInvalidKey          = errors.New("cache: invalid key")
	ErrShutdownTimeout     = errors.New("cache: shutdown timeout waiting for background operations")
)

type CacheError struct {
	Op    string
	Key   string
	Layer string
	Err   error
}

func (e *CacheError) Error() string {
	if e.Key != "" {
		return fmt.Sprintf("cache %s on %s [%s]: %v", e.Op, e.Layer, e.Key, e.Err)
	}
	return fmt.Sprintf("cache %s on %s: %v", e.Op, e.Layer, e.Err)
}

func (e *CacheError) Unwrap() error {
	return e.Err
}

func NewCacheError(op, key, layer string, err error) *CacheError {
	return &CacheError{
		Op:    op,
		Key:   key,
		Layer: layer,
		Err:   err,
	}
}

func IsCacheMiss(err error) bool {
	return errors.Is(err, ErrCacheMiss)
}

func IsRedisUnavailable(err error) bool {
	return errors.Is(err, ErrRedisUnavailable)
}

func IsCircuitOpen(err error) bool {
	return errors.Is(err, ErrCircuitOpen)
}

func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Cache misses are not retryable - the key doesn't exist
	if IsCacheMiss(err) {
		return false
	}

	// Circuit open is not retryable - need to wait for recovery
	if IsCircuitOpen(err) {
		return false
	}

	// Closed manager is not retryable
	if errors.Is(err, ErrClosed) {
		return false
	}

	// Invalid key is not retryable
	if errors.Is(err, ErrInvalidKey) {
		return false
	}

	// Most other errors (network, timeout) are retryable
	return true
}
