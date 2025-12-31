package metrics

import (
	"time"

	"github.com/LavishGent/rentfree/pkg/rentfree"
)

// NoOpTracker is a no-operation metrics tracker for testing.
type NoOpTracker struct{}

// NewNoOpTracker creates a new no-op tracker.
func NewNoOpTracker() *NoOpTracker {
	return &NoOpTracker{}
}

// RecordHit does nothing.
func (t *NoOpTracker) RecordHit(layer string, key string, latency time.Duration) {}

// RecordMiss does nothing.
func (t *NoOpTracker) RecordMiss(layer string, key string, latency time.Duration) {}

// RecordSet does nothing.
func (t *NoOpTracker) RecordSet(layer string, key string, size int, latency time.Duration) {}

// RecordDelete does nothing.
func (t *NoOpTracker) RecordDelete(layer string, key string, latency time.Duration) {}

// RecordError does nothing.
func (t *NoOpTracker) RecordError(layer string, operation string, err error) {}

// RecordCircuitBreakerStateChange does nothing.
func (t *NoOpTracker) RecordCircuitBreakerStateChange(from, to string) {}

// Snapshot returns empty metrics.
func (t *NoOpTracker) Snapshot() rentfree.MetricsSnapshot { return rentfree.MetricsSnapshot{} }

// Reset does nothing.
func (t *NoOpTracker) Reset() {}

// NoOpPublisher is a no-operation metrics publisher for testing or when disabled.
type NoOpPublisher struct{}

// NewNoOpPublisher creates a new no-op publisher.
func NewNoOpPublisher() *NoOpPublisher {
	return &NoOpPublisher{}
}

// Gauge does nothing.
func (p *NoOpPublisher) Gauge(name string, value float64, tags ...string) {}

// Incr does nothing.
func (p *NoOpPublisher) Incr(name string, tags ...string) {}

// Count does nothing.
func (p *NoOpPublisher) Count(name string, value int64, tags ...string) {}

// Histogram does nothing.
func (p *NoOpPublisher) Histogram(name string, value float64, tags ...string) {}

// Timing does nothing.
func (p *NoOpPublisher) Timing(name string, duration time.Duration, tags ...string) {}

// Event does nothing.
func (p *NoOpPublisher) Event(title, text, alertType string, tags ...string) {}

// PublishHealthMetrics does nothing.
func (p *NoOpPublisher) PublishHealthMetrics(metrics *rentfree.PublisherHealthMetrics) {}

// Close does nothing.
func (p *NoOpPublisher) Close() error { return nil }

// Ensure interfaces are implemented
var _ rentfree.MetricsRecorder = (*NoOpTracker)(nil)
var _ rentfree.Publisher = (*NoOpPublisher)(nil)
