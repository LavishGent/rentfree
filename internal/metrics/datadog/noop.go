package datadog

import (
	"time"

	"github.com/LavishGent/rentfree/pkg/rentfree"
)

// NoOpPublisher is a Publisher that does nothing.
// Used when DataDog is disabled.
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
func (p *NoOpPublisher) Timing(name string, value time.Duration, tags ...string) {}

// Event does nothing.
func (p *NoOpPublisher) Event(title, text, alertType string, tags ...string) {}

// PublishHealthMetrics does nothing.
func (p *NoOpPublisher) PublishHealthMetrics(metrics *rentfree.PublisherHealthMetrics) {}

// Close does nothing.
func (p *NoOpPublisher) Close() error { return nil }

var _ rentfree.Publisher = (*NoOpPublisher)(nil)
