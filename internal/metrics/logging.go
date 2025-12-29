package metrics

import (
	"log/slog"
	"time"

	"github.com/darrell-green/rentfree/pkg/rentfree"
)

// LoggingPublisher logs metrics using slog.
type LoggingPublisher struct {
	logger   *slog.Logger
	baseTags []string
}

// NewLoggingPublisher creates a new logging publisher.
func NewLoggingPublisher(logger *slog.Logger, baseTags ...string) *LoggingPublisher {
	if logger == nil {
		logger = slog.Default()
	}
	return &LoggingPublisher{
		logger:   logger.With("component", "metrics"),
		baseTags: baseTags,
	}
}

// Gauge logs a gauge metric.
func (p *LoggingPublisher) Gauge(name string, value float64, tags ...string) {
	p.logger.Debug("gauge",
		"name", name,
		"value", value,
		"tags", p.mergeTags(tags),
	)
}

// Incr logs an increment metric.
func (p *LoggingPublisher) Incr(name string, tags ...string) {
	p.logger.Debug("incr",
		"name", name,
		"tags", p.mergeTags(tags),
	)
}

// Count logs a count metric.
func (p *LoggingPublisher) Count(name string, value int64, tags ...string) {
	p.logger.Debug("count",
		"name", name,
		"value", value,
		"tags", p.mergeTags(tags),
	)
}

// Histogram logs a histogram metric.
func (p *LoggingPublisher) Histogram(name string, value float64, tags ...string) {
	p.logger.Debug("histogram",
		"name", name,
		"value", value,
		"tags", p.mergeTags(tags),
	)
}

// Timing logs a timing metric.
func (p *LoggingPublisher) Timing(name string, duration time.Duration, tags ...string) {
	p.logger.Debug("timing",
		"name", name,
		"duration_ms", duration.Milliseconds(),
		"tags", p.mergeTags(tags),
	)
}

// Event logs an event.
func (p *LoggingPublisher) Event(title, text, alertType string, tags ...string) {
	p.logger.Info("event",
		"title", title,
		"text", text,
		"alert_type", alertType,
		"tags", p.mergeTags(tags),
	)
}

// PublishHealthMetrics logs a batch of health metrics.
func (p *LoggingPublisher) PublishHealthMetrics(m *rentfree.PublisherHealthMetrics) {
	if m == nil {
		return
	}

	connected := 0
	if m.IsConnected {
		connected = 1
	}

	p.logger.Info("health_metrics",
		"memory_used_bytes", m.MemoryUsedBytes,
		"memory_limit_bytes", m.MemoryLimitBytes,
		"memory_usage_pct", m.MemoryUsagePercentage,
		"total_entries", m.TotalEntries,
		"hit_ratio", m.HitRatio,
		"avg_latency_ms", m.AverageLatencyMs,
		"is_connected", connected,
	)
}

// Close does nothing for logging publisher.
func (p *LoggingPublisher) Close() error {
	return nil
}

func (p *LoggingPublisher) mergeTags(tags []string) []string {
	if len(tags) == 0 {
		return p.baseTags
	}
	if len(p.baseTags) == 0 {
		return tags
	}
	return append(p.baseTags, tags...)
}

// Ensure LoggingPublisher implements Publisher
var _ rentfree.Publisher = (*LoggingPublisher)(nil)
