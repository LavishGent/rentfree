package metrics

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/LavishGent/rentfree/pkg/rentfree"
)

// BackgroundPublisher publishes health metrics at regular intervals
// with context-based cancellation support.
type BackgroundPublisher struct {
	publisher rentfree.Publisher
	logger    *slog.Logger
	getHealth func() *rentfree.PublisherHealthMetrics
	cancel    context.CancelFunc
	ctx       context.Context
	wg        sync.WaitGroup
	interval  time.Duration
}

// NewBackgroundPublisher creates a new background publisher.
// The healthFn is called on each interval to get the current health metrics.
func NewBackgroundPublisher(
	publisher rentfree.Publisher,
	interval time.Duration,
	healthFn func() *rentfree.PublisherHealthMetrics,
	logger *slog.Logger,
) *BackgroundPublisher {
	if logger == nil {
		logger = slog.Default()
	}

	return &BackgroundPublisher{
		publisher: publisher,
		interval:  interval,
		logger:    logger.With("component", "metrics-background"),
		getHealth: healthFn,
	}
}

// Start begins the background publishing loop.
// The provided context controls the lifecycle of the background goroutine.
func (b *BackgroundPublisher) Start(ctx context.Context) {
	b.ctx, b.cancel = context.WithCancel(ctx)
	b.wg.Add(1)
	go b.run()
	b.logger.Info("Background metrics publisher started", "interval", b.interval)
}

// Stop cancels the background context and waits for shutdown.
func (b *BackgroundPublisher) Stop() {
	if b.cancel != nil {
		b.cancel()
	}
	b.wg.Wait()
	b.logger.Info("Background metrics publisher stopped")
}

func (b *BackgroundPublisher) run() {
	defer b.wg.Done()

	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()

	for {
		select {
		case <-b.ctx.Done():
			// Final publish before stopping
			b.publish()
			return
		case <-ticker.C:
			b.publish()
		}
	}
}

func (b *BackgroundPublisher) publish() {
	defer func() {
		if r := recover(); r != nil {
			b.logger.Error("Recovered from panic in metrics publisher", "panic", r)
		}
	}()

	if b.getHealth == nil {
		return
	}

	metrics := b.getHealth()
	if metrics != nil {
		b.publisher.PublishHealthMetrics(metrics)
	}
}

// PublishNow triggers an immediate metrics publish.
func (b *BackgroundPublisher) PublishNow() {
	b.publish()
}

// BackgroundService is an alias for BackgroundPublisher for backward compatibility.
// Deprecated: Use BackgroundPublisher instead.
type BackgroundService = BackgroundPublisher

// NewBackgroundService creates a new background service.
// Deprecated: Use NewBackgroundPublisher instead.
func NewBackgroundService(
	tracker *Tracker,
	publisher rentfree.Publisher,
	interval time.Duration,
	logger *slog.Logger,
) *BackgroundService {
	return NewBackgroundPublisher(
		publisher,
		interval,
		func() *rentfree.PublisherHealthMetrics {
			snapshot := tracker.Snapshot()
			return &rentfree.PublisherHealthMetrics{
				MemoryUsedBytes:       snapshot.MemorySizeBytes,
				MemoryLimitBytes:      snapshot.MemoryMaxBytes,
				MemoryUsagePercentage: snapshot.MemoryUsageRatio * 100,
				TotalEntries:          snapshot.MemoryEntries,
				HitRatio:              snapshot.TotalHitRatio(),
				AverageLatencyMs:      snapshot.AvgLatencyMs,
				IsConnected:           snapshot.RedisConnected,
			}
		},
		logger,
	)
}
