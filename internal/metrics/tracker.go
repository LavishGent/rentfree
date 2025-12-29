// Package metrics provides cache operation metrics collection and publishing.
package metrics

import (
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"gitlab.com/appliedsystems/experimental/users/ddavis/stuff/rentfree/pkg/rentfree"
)

const (
	defaultLatencyBufferSize = 10000
)

type Tracker struct {
	memoryHits   atomic.Int64
	memoryMisses atomic.Int64
	redisHits    atomic.Int64
	redisMisses  atomic.Int64

	getCount    atomic.Int64
	setCount    atomic.Int64
	deleteCount atomic.Int64

	errorCount atomic.Int64

	latencyMu     sync.RWMutex
	latencyBuffer []time.Duration
	latencyIndex  int
	latencyCount  int

	totalBytesWritten atomic.Int64

	cbStateChanges atomic.Int64
}

func NewTracker() *Tracker {
	return &Tracker{
		latencyBuffer: make([]time.Duration, defaultLatencyBufferSize),
	}
}

func (t *Tracker) RecordHit(layer string, key string, latency time.Duration) {
	switch layer {
	case "memory":
		t.memoryHits.Add(1)
	case "redis":
		t.redisHits.Add(1)
	}
	t.getCount.Add(1)
	t.recordLatency(latency)
}

func (t *Tracker) RecordMiss(layer string, key string, latency time.Duration) {
	switch layer {
	case "memory":
		t.memoryMisses.Add(1)
	case "redis":
		t.redisMisses.Add(1)
	}
	t.getCount.Add(1)
	t.recordLatency(latency)
}

func (t *Tracker) RecordSet(layer string, key string, size int, latency time.Duration) {
	t.setCount.Add(1)
	t.totalBytesWritten.Add(int64(size))
	t.recordLatency(latency)
}

// RecordDelete records a delete operation.
func (t *Tracker) RecordDelete(layer string, key string, latency time.Duration) {
	t.deleteCount.Add(1)
	t.recordLatency(latency)
}

// RecordError records an error.
func (t *Tracker) RecordError(layer string, operation string, err error) {
	t.errorCount.Add(1)
}

// RecordCircuitBreakerStateChange records circuit breaker state transitions.
func (t *Tracker) RecordCircuitBreakerStateChange(from, to string) {
	t.cbStateChanges.Add(1)
}

// recordLatency adds a latency measurement using a circular buffer.
// This is O(1) time complexity with no memory allocations.
func (t *Tracker) recordLatency(latency time.Duration) {
	t.latencyMu.Lock()
	t.latencyBuffer[t.latencyIndex] = latency
	t.latencyIndex = (t.latencyIndex + 1) % len(t.latencyBuffer)
	if t.latencyCount < len(t.latencyBuffer) {
		t.latencyCount++
	}
	t.latencyMu.Unlock()
}

// Snapshot returns current metrics snapshot.
func (t *Tracker) Snapshot() rentfree.MetricsSnapshot {
	// Use RLock for reading - allows concurrent snapshots
	t.latencyMu.RLock()
	count := t.latencyCount
	latencyCopy := make([]time.Duration, count)
	// Copy from circular buffer in correct order
	if count > 0 {
		if count < len(t.latencyBuffer) {
			// Buffer not full yet - data starts at 0
			copy(latencyCopy, t.latencyBuffer[:count])
		} else {
			// Buffer is full - oldest data starts at latencyIndex
			firstPart := len(t.latencyBuffer) - t.latencyIndex
			copy(latencyCopy[:firstPart], t.latencyBuffer[t.latencyIndex:])
			copy(latencyCopy[firstPart:], t.latencyBuffer[:t.latencyIndex])
		}
	}
	t.latencyMu.RUnlock()

	snapshot := rentfree.MetricsSnapshot{
		Timestamp:    time.Now(),
		MemoryHits:   t.memoryHits.Load(),
		MemoryMisses: t.memoryMisses.Load(),
		RedisHits:    t.redisHits.Load(),
		RedisMisses:  t.redisMisses.Load(),
		GetCount:     t.getCount.Load(),
		SetCount:     t.setCount.Load(),
		DeleteCount:  t.deleteCount.Load(),
		ErrorCount:   t.errorCount.Load(),
	}

	// Calculate latency percentiles
	if len(latencyCopy) > 0 {
		snapshot.AvgLatencyMs = float64(avgDuration(latencyCopy).Milliseconds())
		snapshot.P50LatencyMs = float64(percentile(latencyCopy, 50).Milliseconds())
		snapshot.P95LatencyMs = float64(percentile(latencyCopy, 95).Milliseconds())
		snapshot.P99LatencyMs = float64(percentile(latencyCopy, 99).Milliseconds())
	}

	return snapshot
}

// Reset clears all metrics.
func (t *Tracker) Reset() {
	t.memoryHits.Store(0)
	t.memoryMisses.Store(0)
	t.redisHits.Store(0)
	t.redisMisses.Store(0)
	t.getCount.Store(0)
	t.setCount.Store(0)
	t.deleteCount.Store(0)
	t.errorCount.Store(0)
	t.totalBytesWritten.Store(0)
	t.cbStateChanges.Store(0)

	t.latencyMu.Lock()
	t.latencyIndex = 0
	t.latencyCount = 0
	t.latencyMu.Unlock()
}

// Helper functions for latency calculations

func avgDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	return total / time.Duration(len(durations))
}

func percentile(durations []time.Duration, p int) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	// Sort a copy
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)
	slices.Sort(sorted)

	idx := (len(sorted) - 1) * p / 100
	return sorted[idx]
}

// Ensure Tracker implements MetricsRecorder
var _ rentfree.MetricsRecorder = (*Tracker)(nil)
