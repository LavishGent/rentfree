package metrics

import (
	"time"

	"gitlab.com/appliedsystems/experimental/users/ddavis/stuff/rentfree/pkg/rentfree"
)

// Timer is a helper for measuring operation latency.
type Timer struct {
	publisher rentfree.Publisher
	name      string
	tags      []string
	start     time.Time
}

// NewTimer creates a new timer that will record to the publisher when stopped.
func NewTimer(publisher rentfree.Publisher, name string, tags ...string) *Timer {
	return &Timer{
		publisher: publisher,
		name:      name,
		tags:      tags,
		start:     time.Now(),
	}
}

// Stop records the elapsed time as a timing metric and returns the duration.
func (t *Timer) Stop() time.Duration {
	duration := time.Since(t.start)
	t.publisher.Timing(t.name, duration, t.tags...)
	return duration
}

// Elapsed returns the time since the timer was started without recording.
func (t *Timer) Elapsed() time.Duration {
	return time.Since(t.start)
}
