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

func (t *NoOpTracker) RecordHit(layer string, key string, latency time.Duration)           {}
func (t *NoOpTracker) RecordMiss(layer string, key string, latency time.Duration)          {}
func (t *NoOpTracker) RecordSet(layer string, key string, size int, latency time.Duration) {}
func (t *NoOpTracker) RecordDelete(layer string, key string, latency time.Duration)        {}
func (t *NoOpTracker) RecordError(layer string, operation string, err error)               {}
func (t *NoOpTracker) RecordCircuitBreakerStateChange(from, to string)                     {}
func (t *NoOpTracker) Snapshot() rentfree.MetricsSnapshot                                  { return rentfree.MetricsSnapshot{} }
func (t *NoOpTracker) Reset()                                                              {}

// NoOpPublisher is a no-operation metrics publisher for testing or when disabled.
type NoOpPublisher struct{}

// NewNoOpPublisher creates a new no-op publisher.
func NewNoOpPublisher() *NoOpPublisher {
	return &NoOpPublisher{}
}

func (p *NoOpPublisher) Gauge(name string, value float64, tags ...string)              {}
func (p *NoOpPublisher) Incr(name string, tags ...string)                              {}
func (p *NoOpPublisher) Count(name string, value int64, tags ...string)                {}
func (p *NoOpPublisher) Histogram(name string, value float64, tags ...string)          {}
func (p *NoOpPublisher) Timing(name string, duration time.Duration, tags ...string)    {}
func (p *NoOpPublisher) Event(title, text, alertType string, tags ...string)           {}
func (p *NoOpPublisher) PublishHealthMetrics(metrics *rentfree.PublisherHealthMetrics) {}
func (p *NoOpPublisher) Close() error                                                  { return nil }

// Ensure interfaces are implemented
var _ rentfree.MetricsRecorder = (*NoOpTracker)(nil)
var _ rentfree.Publisher = (*NoOpPublisher)(nil)
