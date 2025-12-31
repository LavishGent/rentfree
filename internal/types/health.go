package types

import "time"

// HealthStatus represents the overall health state.
type HealthStatus int

const (
	// HealthStatusHealthy indicates all systems operating normally.
	HealthStatusHealthy HealthStatus = iota + 1
	// HealthStatusDegraded indicates partial functionality (e.g., Redis down).
	HealthStatusDegraded
	// HealthStatusUnhealthy indicates critical failure.
	HealthStatusUnhealthy
)

// String returns the string representation of health status.
func (s HealthStatus) String() string {
	switch s {
	case HealthStatusHealthy:
		return "healthy"
	case HealthStatusDegraded:
		return "degraded"
	case HealthStatusUnhealthy:
		return "unhealthy"
	default:
		return "unknown"
	}
}

// HealthMetrics contains overall cache health information.
type HealthMetrics struct {
	Timestamp time.Time
	Redis     RedisHealthMetrics
	Memory    MemoryHealthMetrics
	Status    HealthStatus
}

// MemoryHealthMetrics contains memory cache health details.
type MemoryHealthMetrics struct {
	Status          HealthStatus
	Available       bool
	EntryCount      int
	SizeBytes       int64
	MaxSizeBytes    int64
	UsagePercentage float64
	HitCount        int64
	MissCount       int64
	HitRatio        float64
	EvictionCount   int64
}

// RedisHealthMetrics contains Redis cache health details.
//
//nolint:govet // Metrics struct - logical grouping prioritized for readability
type RedisHealthMetrics struct {
	LastErrorTime       time.Time
	DroppedWrites       int64
	HitCount            int64
	MissCount           int64
	HitRatio            float64
	CircuitBreakerState string
	LastError           string
	PendingWrites       int
	Status              HealthStatus
	Available           bool
	Connected           bool
}

// MetricsSnapshot contains a point-in-time view of cache metrics.
//
//nolint:govet // Metrics struct with many counters - grouping by category improves readability
type MetricsSnapshot struct {
	Timestamp time.Time
	// Hit/miss counters
	MemoryHits   int64
	MemoryMisses int64
	RedisHits    int64
	RedisMisses  int64
	// Operation counters
	GetCount    int64
	SetCount    int64
	DeleteCount int64
	ErrorCount  int64

	// Latency metrics (milliseconds)
	AvgLatencyMs float64
	P50LatencyMs float64
	P95LatencyMs float64
	P99LatencyMs float64

	// Memory cache metrics
	MemorySizeBytes  int64
	MemoryEntries    int64
	MemoryEvictions  int64
	MemoryMaxBytes   int64
	MemoryUsageRatio float64

	// Redis metrics
	RedisConnected      bool
	RedisPendingWrites  int
	RedisDroppedWrites  int64
	CircuitBreakerState string

	// Resilience metrics
	RetryCount       int64
	BulkheadRejected int64
}

// MemoryHitRatio calculates the memory cache hit ratio.
func (s *MetricsSnapshot) MemoryHitRatio() float64 {
	total := s.MemoryHits + s.MemoryMisses
	if total == 0 {
		return 0
	}
	return float64(s.MemoryHits) / float64(total)
}

// RedisHitRatio calculates the Redis cache hit ratio.
func (s *MetricsSnapshot) RedisHitRatio() float64 {
	total := s.RedisHits + s.RedisMisses
	if total == 0 {
		return 0
	}
	return float64(s.RedisHits) / float64(total)
}

// TotalHitRatio calculates the overall cache hit ratio.
func (s *MetricsSnapshot) TotalHitRatio() float64 {
	totalHits := s.MemoryHits + s.RedisHits
	totalMisses := s.MemoryMisses + s.RedisMisses
	total := totalHits + totalMisses
	if total == 0 {
		return 0
	}
	return float64(totalHits) / float64(total)
}
