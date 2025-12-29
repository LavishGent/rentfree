// Package datadog provides a DataDog StatsD metrics publisher.
package datadog

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
	"github.com/darrell-green/rentfree/internal/config"
	"github.com/darrell-green/rentfree/pkg/rentfree"
)

// Publisher implements rentfree.Publisher using the DataDog StatsD client.
type Publisher struct {
	client   *statsd.Client
	config   *config.DataDogConfig
	baseTags []string
	logger   *slog.Logger
}

// NewPublisher creates a new DataDog publisher from config.
// If DataDog is not enabled, returns a NoOpPublisher instead.
func NewPublisher(cfg *config.DataDogConfig, logger *slog.Logger) (rentfree.Publisher, error) {
	if !cfg.Enabled {
		return &NoOpPublisher{}, nil
	}

	if logger == nil {
		logger = slog.Default()
	}

	addr := fmt.Sprintf("%s:%d", cfg.AgentHost, cfg.Port)

	client, err := statsd.New(addr,
		statsd.WithNamespace(cfg.Prefix+"."),
		statsd.WithTags(cfg.Tags),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create statsd client: %w", err)
	}

	logger.Info("DataDog publisher initialized",
		"address", addr,
		"prefix", cfg.Prefix,
		"tags", cfg.Tags,
	)

	return &Publisher{
		client:   client,
		config:   cfg,
		baseTags: cfg.Tags,
		logger:   logger.With("component", "datadog"),
	}, nil
}

// Gauge records a gauge metric (value at a point in time).
func (p *Publisher) Gauge(name string, value float64, tags ...string) {
	allTags := p.mergeTags(tags)
	_ = p.client.Gauge(name, value, allTags, 1)
}

// Incr increments a counter by 1.
func (p *Publisher) Incr(name string, tags ...string) {
	allTags := p.mergeTags(tags)
	_ = p.client.Incr(name, allTags, 1)
}

// Count increments a counter by a specified amount.
func (p *Publisher) Count(name string, value int64, tags ...string) {
	allTags := p.mergeTags(tags)
	_ = p.client.Count(name, value, allTags, 1)
}

// Histogram records a distribution of values.
func (p *Publisher) Histogram(name string, value float64, tags ...string) {
	allTags := p.mergeTags(tags)
	_ = p.client.Histogram(name, value, allTags, 1)
}

// Timing records a timing metric.
func (p *Publisher) Timing(name string, duration time.Duration, tags ...string) {
	allTags := p.mergeTags(tags)
	_ = p.client.Timing(name, duration, allTags, 1)
}

// Event sends a DataDog event.
func (p *Publisher) Event(title, text, alertType string, tags ...string) {
	allTags := p.mergeTags(tags)
	event := &statsd.Event{
		Title:     title,
		Text:      text,
		AlertType: statsd.EventAlertType(alertType),
		Tags:      allTags,
	}
	_ = p.client.Event(event)
}

// PublishHealthMetrics publishes a batch of health metrics.
func (p *Publisher) PublishHealthMetrics(m *rentfree.PublisherHealthMetrics) {
	if m == nil {
		return
	}

	p.Gauge("memory.used_bytes", float64(m.MemoryUsedBytes))
	p.Gauge("memory.limit_bytes", float64(m.MemoryLimitBytes))
	p.Gauge("memory.usage_percentage", clamp(m.MemoryUsagePercentage, 0, 100))
	p.Gauge("entries.total", float64(m.TotalEntries))
	p.Gauge("performance.hit_ratio", clamp(m.HitRatio, 0, 1))
	p.Gauge("performance.average_latency_ms", max(0, m.AverageLatencyMs))

	connected := 0.0
	if m.IsConnected {
		connected = 1.0
	}
	p.Gauge("connection.status", connected)
}

// Close releases resources held by the publisher.
func (p *Publisher) Close() error {
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}

func (p *Publisher) mergeTags(tags []string) []string {
	if len(tags) == 0 {
		return p.baseTags
	}
	if len(p.baseTags) == 0 {
		return tags
	}
	return append(p.baseTags, tags...)
}

func clamp(val, min, max float64) float64 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// Ensure Publisher implements the interface
var _ rentfree.Publisher = (*Publisher)(nil)
