package rentfree

import (
	"github.com/darrell-green/rentfree/internal/types"
)

// Re-export health types from internal/types.
type (
	// HealthStatus represents the overall health state.
	HealthStatus = types.HealthStatus

	// HealthMetrics contains overall cache health information.
	HealthMetrics = types.HealthMetrics

	// MemoryHealthMetrics contains memory cache health details.
	MemoryHealthMetrics = types.MemoryHealthMetrics

	// RedisHealthMetrics contains Redis cache health details.
	RedisHealthMetrics = types.RedisHealthMetrics

	// MetricsSnapshot contains a point-in-time view of cache metrics.
	MetricsSnapshot = types.MetricsSnapshot
)

// Re-export health status constants.
const (
	HealthStatusHealthy   = types.HealthStatusHealthy
	HealthStatusDegraded  = types.HealthStatusDegraded
	HealthStatusUnhealthy = types.HealthStatusUnhealthy
)
