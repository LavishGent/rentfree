package resilience

import (
	"errors"
	"net"
	"os"
	"syscall"

	"github.com/darrell-green/rentfree/internal/types"
)

// Re-export errors from types package for convenience within the resilience package.
// This allows existing code to continue using resilience.ErrCircuitOpen etc.
var (
	ErrCircuitOpen     = types.ErrCircuitOpen
	ErrBulkheadFull    = types.ErrBulkheadFull
	ErrBulkheadTimeout = types.ErrBulkheadTimeout
)

// IsCircuitOpen returns true if the error is a circuit open error.
func IsCircuitOpen(err error) bool {
	return errors.Is(err, types.ErrCircuitOpen)
}

// IsBulkheadError returns true if the error is a bulkhead error.
func IsBulkheadError(err error) bool {
	return errors.Is(err, types.ErrBulkheadFull) || errors.Is(err, types.ErrBulkheadTimeout)
}

// IsRetryable determines if an error is transient and worth retrying.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Circuit open errors should not be retried
	if errors.Is(err, types.ErrCircuitOpen) {
		return false
	}

	// Bulkhead errors should not be retried
	if errors.Is(err, types.ErrBulkheadFull) || errors.Is(err, types.ErrBulkheadTimeout) {
		return false
	}

	// Check for temporary network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}

	// Check for connection refused, reset, etc.
	if errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ETIMEDOUT) {
		return true
	}

	// Check for temporary OS errors
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}

	// By default, assume errors are retryable for resilience
	return true
}
