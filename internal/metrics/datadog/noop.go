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

func (p *NoOpPublisher) Gauge(name string, value float64, tags ...string)              {}
func (p *NoOpPublisher) Incr(name string, tags ...string)                              {}
func (p *NoOpPublisher) Count(name string, value int64, tags ...string)                {}
func (p *NoOpPublisher) Histogram(name string, value float64, tags ...string)          {}
func (p *NoOpPublisher) Timing(name string, value time.Duration, tags ...string)       {}
func (p *NoOpPublisher) Event(title, text, alertType string, tags ...string)           {}
func (p *NoOpPublisher) PublishHealthMetrics(metrics *rentfree.PublisherHealthMetrics) {}
func (p *NoOpPublisher) Close() error                                                  { return nil }

var _ rentfree.Publisher = (*NoOpPublisher)(nil)
