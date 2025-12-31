package types

import (
	"errors"
	"fmt"
)

var (
	// ErrCacheMiss indicates that a requested key was not found in the cache.
	ErrCacheMiss = errors.New("cache: key not found")
	// ErrRedisUnavailable indicates that the Redis server is not available.
	ErrRedisUnavailable = errors.New("cache: redis unavailable")
	// ErrCircuitOpen indicates that the circuit breaker is open.
	ErrCircuitOpen = errors.New("cache: circuit breaker open")
	// ErrClosed indicates that the cache manager has been closed.
	ErrClosed = errors.New("cache: manager closed")
	// ErrWriteQueueFull indicates that the write queue is full.
	ErrWriteQueueFull = errors.New("cache: write queue full")
	// ErrBulkheadFull indicates that the bulkhead is at capacity.
	ErrBulkheadFull = errors.New("cache: bulkhead at capacity")
	// ErrBulkheadTimeout indicates that the bulkhead acquisition timed out.
	ErrBulkheadTimeout = errors.New("cache: bulkhead timeout")
	// ErrSerializationFailed indicates that serialization failed.
	ErrSerializationFailed = errors.New("cache: serialization failed")
	// ErrInvalidKey indicates that a cache key is invalid.
	ErrInvalidKey = errors.New("cache: invalid key")
	// ErrShutdownTimeout indicates that shutdown timed out waiting for background operations.
	ErrShutdownTimeout = errors.New("cache: shutdown timeout waiting for background operations")
)

// CacheError wraps an error with cache operation context.
type CacheError struct {
	Err   error
	Op    string
	Key   string
	Layer string
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

// NewCacheError creates a new CacheError with the given operation, key, layer, and underlying error.
func NewCacheError(op, key, layer string, err error) *CacheError {
	return &CacheError{
		Op:    op,
		Key:   key,
		Layer: layer,
		Err:   err,
	}
}

// IsCacheMiss returns true if the error is or wraps ErrCacheMiss.
func IsCacheMiss(err error) bool {
	return errors.Is(err, ErrCacheMiss)
}

// IsRedisUnavailable returns true if the error is or wraps ErrRedisUnavailable.
func IsRedisUnavailable(err error) bool {
	return errors.Is(err, ErrRedisUnavailable)
}

// IsCircuitOpen returns true if the error is or wraps ErrCircuitOpen.
func IsCircuitOpen(err error) bool {
	return errors.Is(err, ErrCircuitOpen)
}

// IsRetryable returns true if the error can be retried.
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
