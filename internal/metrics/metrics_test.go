package metrics

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LavishGent/rentfree/pkg/rentfree"
)

func TestNewTracker(t *testing.T) {
	tracker := NewTracker()

	if tracker == nil {
		t.Fatal("NewTracker() returned nil")
	}

	snapshot := tracker.Snapshot()
	if snapshot.GetCount != 0 {
		t.Errorf("initial GetCount = %d, want 0", snapshot.GetCount)
	}
}

func TestTrackerRecordHit(t *testing.T) {
	tracker := NewTracker()

	t.Run("memory layer", func(t *testing.T) {
		tracker.Reset()
		tracker.RecordHit("memory", "key1", 10*time.Millisecond)

		snapshot := tracker.Snapshot()
		if snapshot.MemoryHits != 1 {
			t.Errorf("MemoryHits = %d, want 1", snapshot.MemoryHits)
		}
		if snapshot.GetCount != 1 {
			t.Errorf("GetCount = %d, want 1", snapshot.GetCount)
		}
	})

	t.Run("redis layer", func(t *testing.T) {
		tracker.Reset()
		tracker.RecordHit("redis", "key1", 10*time.Millisecond)

		snapshot := tracker.Snapshot()
		if snapshot.RedisHits != 1 {
			t.Errorf("RedisHits = %d, want 1", snapshot.RedisHits)
		}
	})
}

func TestTrackerRecordMiss(t *testing.T) {
	tracker := NewTracker()

	t.Run("memory layer", func(t *testing.T) {
		tracker.Reset()
		tracker.RecordMiss("memory", "key1", 5*time.Millisecond)

		snapshot := tracker.Snapshot()
		if snapshot.MemoryMisses != 1 {
			t.Errorf("MemoryMisses = %d, want 1", snapshot.MemoryMisses)
		}
		if snapshot.GetCount != 1 {
			t.Errorf("GetCount = %d, want 1", snapshot.GetCount)
		}
	})

	t.Run("redis layer", func(t *testing.T) {
		tracker.Reset()
		tracker.RecordMiss("redis", "key1", 5*time.Millisecond)

		snapshot := tracker.Snapshot()
		if snapshot.RedisMisses != 1 {
			t.Errorf("RedisMisses = %d, want 1", snapshot.RedisMisses)
		}
	})
}

func TestTrackerRecordSet(t *testing.T) {
	tracker := NewTracker()
	tracker.RecordSet("memory", "key1", 100, 15*time.Millisecond)

	snapshot := tracker.Snapshot()
	if snapshot.SetCount != 1 {
		t.Errorf("SetCount = %d, want 1", snapshot.SetCount)
	}
}

func TestTrackerRecordDelete(t *testing.T) {
	tracker := NewTracker()
	tracker.RecordDelete("memory", "key1", 5*time.Millisecond)

	snapshot := tracker.Snapshot()
	if snapshot.DeleteCount != 1 {
		t.Errorf("DeleteCount = %d, want 1", snapshot.DeleteCount)
	}
}

func TestTrackerRecordError(t *testing.T) {
	tracker := NewTracker()
	tracker.RecordError("redis", "get", errors.New("connection refused"))

	snapshot := tracker.Snapshot()
	if snapshot.ErrorCount != 1 {
		t.Errorf("ErrorCount = %d, want 1", snapshot.ErrorCount)
	}
}

func TestTrackerRecordCircuitBreakerStateChange(t *testing.T) {
	tracker := NewTracker()
	tracker.RecordCircuitBreakerStateChange("closed", "open")
	tracker.RecordCircuitBreakerStateChange("open", "half-open")

	// cbStateChanges is internal, verify no panic
}

func TestTrackerSnapshot(t *testing.T) {
	tracker := NewTracker()

	// Record various operations
	tracker.RecordHit("memory", "key1", 10*time.Millisecond)
	tracker.RecordHit("memory", "key2", 20*time.Millisecond)
	tracker.RecordMiss("redis", "key3", 30*time.Millisecond)
	tracker.RecordSet("memory", "key4", 256, 15*time.Millisecond)
	tracker.RecordDelete("redis", "key5", 5*time.Millisecond)
	tracker.RecordError("redis", "get", errors.New("timeout"))

	snapshot := tracker.Snapshot()

	if snapshot.MemoryHits != 2 {
		t.Errorf("MemoryHits = %d, want 2", snapshot.MemoryHits)
	}
	if snapshot.RedisMisses != 1 {
		t.Errorf("RedisMisses = %d, want 1", snapshot.RedisMisses)
	}
	if snapshot.GetCount != 3 {
		t.Errorf("GetCount = %d, want 3", snapshot.GetCount)
	}
	if snapshot.SetCount != 1 {
		t.Errorf("SetCount = %d, want 1", snapshot.SetCount)
	}
	if snapshot.DeleteCount != 1 {
		t.Errorf("DeleteCount = %d, want 1", snapshot.DeleteCount)
	}
	if snapshot.ErrorCount != 1 {
		t.Errorf("ErrorCount = %d, want 1", snapshot.ErrorCount)
	}
	if snapshot.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestTrackerLatencyPercentiles(t *testing.T) {
	tracker := NewTracker()

	// Record operations with varying latencies
	latencies := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
		40 * time.Millisecond,
		50 * time.Millisecond,
		60 * time.Millisecond,
		70 * time.Millisecond,
		80 * time.Millisecond,
		90 * time.Millisecond,
		100 * time.Millisecond,
	}

	for _, lat := range latencies {
		tracker.RecordHit("memory", "key", lat)
	}

	snapshot := tracker.Snapshot()

	// Average should be around 55ms
	if snapshot.AvgLatencyMs < 50 || snapshot.AvgLatencyMs > 60 {
		t.Errorf("AvgLatencyMs = %f, want ~55", snapshot.AvgLatencyMs)
	}

	// P50 should be around 50ms
	if snapshot.P50LatencyMs < 40 || snapshot.P50LatencyMs > 60 {
		t.Errorf("P50LatencyMs = %f, want ~50", snapshot.P50LatencyMs)
	}

	// P95 should be around 90-100ms
	if snapshot.P95LatencyMs < 80 || snapshot.P95LatencyMs > 110 {
		t.Errorf("P95LatencyMs = %f, want ~90-100", snapshot.P95LatencyMs)
	}
}

func TestTrackerReset(t *testing.T) {
	tracker := NewTracker()

	// Record some data
	tracker.RecordHit("memory", "key1", 10*time.Millisecond)
	tracker.RecordMiss("redis", "key2", 20*time.Millisecond)
	tracker.RecordSet("memory", "key3", 100, 15*time.Millisecond)
	tracker.RecordError("redis", "get", errors.New("error"))

	// Reset
	tracker.Reset()

	snapshot := tracker.Snapshot()
	if snapshot.MemoryHits != 0 {
		t.Errorf("after reset MemoryHits = %d, want 0", snapshot.MemoryHits)
	}
	if snapshot.RedisMisses != 0 {
		t.Errorf("after reset RedisMisses = %d, want 0", snapshot.RedisMisses)
	}
	if snapshot.SetCount != 0 {
		t.Errorf("after reset SetCount = %d, want 0", snapshot.SetCount)
	}
	if snapshot.ErrorCount != 0 {
		t.Errorf("after reset ErrorCount = %d, want 0", snapshot.ErrorCount)
	}
	// Latency stats should be zero
	if snapshot.AvgLatencyMs != 0 {
		t.Errorf("after reset AvgLatencyMs = %f, want 0", snapshot.AvgLatencyMs)
	}
}

func TestTrackerLatencyCircularBuffer(t *testing.T) {
	tracker := NewTracker()

	// Record more than the buffer size
	// The buffer size is defaultLatencyBufferSize (10000)
	// Record many entries to test circular buffer behavior
	for i := 0; i < 150; i++ {
		tracker.RecordHit("memory", "key", time.Duration(i)*time.Millisecond)
	}

	// Should have exactly 150 entries (buffer not full yet)
	tracker.latencyMu.RLock()
	count := tracker.latencyCount
	tracker.latencyMu.RUnlock()

	if count != 150 {
		t.Errorf("latencies count = %d, want 150", count)
	}

	// Verify snapshot works correctly
	snapshot := tracker.Snapshot()
	if snapshot.AvgLatencyMs == 0 {
		t.Error("AvgLatencyMs should not be zero")
	}
}

func TestTrackerConcurrency(t *testing.T) {
	tracker := NewTracker()
	var wg sync.WaitGroup

	// Run concurrent operations
	for i := 0; i < 100; i++ {
		wg.Add(4)
		go func() {
			defer wg.Done()
			tracker.RecordHit("memory", "key", 10*time.Millisecond)
		}()
		go func() {
			defer wg.Done()
			tracker.RecordMiss("redis", "key", 20*time.Millisecond)
		}()
		go func() {
			defer wg.Done()
			tracker.RecordSet("memory", "key", 100, 15*time.Millisecond)
		}()
		go func() {
			defer wg.Done()
			tracker.Snapshot()
		}()
	}

	wg.Wait()

	// Should have recorded all operations
	snapshot := tracker.Snapshot()
	if snapshot.MemoryHits != 100 {
		t.Errorf("MemoryHits = %d, want 100", snapshot.MemoryHits)
	}
	if snapshot.RedisMisses != 100 {
		t.Errorf("RedisMisses = %d, want 100", snapshot.RedisMisses)
	}
	if snapshot.SetCount != 100 {
		t.Errorf("SetCount = %d, want 100", snapshot.SetCount)
	}
}

func TestLoggingPublisher(t *testing.T) {
	t.Run("creates with default logger", func(t *testing.T) {
		publisher := NewLoggingPublisher(nil)
		if publisher == nil {
			t.Fatal("NewLoggingPublisher(nil) returned nil")
		}
	})

	t.Run("creates with custom logger", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		publisher := NewLoggingPublisher(logger)
		if publisher == nil {
			t.Fatal("NewLoggingPublisher() returned nil")
		}
	})

	t.Run("publishes health metrics", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		publisher := NewLoggingPublisher(logger)

		health := &rentfree.PublisherHealthMetrics{
			MemoryUsedBytes:       1024 * 1024 * 50,
			MemoryLimitBytes:      1024 * 1024 * 100,
			MemoryUsagePercentage: 50.0,
			TotalEntries:          1000,
			HitRatio:              0.85,
			AverageLatencyMs:      5.5,
			IsConnected:           true,
		}

		publisher.PublishHealthMetrics(health)

		output := buf.String()
		if output == "" {
			t.Error("expected log output, got empty string")
		}
	})

	t.Run("gauge metric", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
		publisher := NewLoggingPublisher(logger)

		publisher.Gauge("test.metric", 42.5, "tag1:value1")

		output := buf.String()
		if output == "" {
			t.Error("expected log output for gauge")
		}
	})

	t.Run("incr metric", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
		publisher := NewLoggingPublisher(logger)

		publisher.Incr("test.counter", "operation:get")

		output := buf.String()
		if output == "" {
			t.Error("expected log output for incr")
		}
	})

	t.Run("timing metric", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
		publisher := NewLoggingPublisher(logger)

		publisher.Timing("test.latency", 100*time.Millisecond, "layer:memory")

		output := buf.String()
		if output == "" {
			t.Error("expected log output for timing")
		}
	})

	t.Run("event", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		publisher := NewLoggingPublisher(logger)

		publisher.Event("Test Event", "This is a test event", "info", "source:test")

		output := buf.String()
		if output == "" {
			t.Error("expected log output for event")
		}
	})

	t.Run("close returns nil", func(t *testing.T) {
		publisher := NewLoggingPublisher(nil)
		if err := publisher.Close(); err != nil {
			t.Errorf("Close() error = %v, want nil", err)
		}
	})
}

func TestBackgroundPublisher(t *testing.T) {
	t.Run("creates with nil logger", func(t *testing.T) {
		publisher := NewNoOpPublisher()
		bg := NewBackgroundPublisher(publisher, 10*time.Millisecond, func() *rentfree.PublisherHealthMetrics {
			return &rentfree.PublisherHealthMetrics{}
		}, nil)
		if bg == nil {
			t.Fatal("NewBackgroundPublisher() returned nil")
		}
	})

	t.Run("start and stop", func(t *testing.T) {
		publisher := &trackingPublisher{}
		bg := NewBackgroundPublisher(publisher, 10*time.Millisecond, func() *rentfree.PublisherHealthMetrics {
			return &rentfree.PublisherHealthMetrics{
				MemoryUsedBytes: 1000,
				IsConnected:     true,
			}
		}, nil)

		ctx := context.Background()
		bg.Start(ctx)
		time.Sleep(50 * time.Millisecond) // Let it publish a few times
		bg.Stop()

		if publisher.publishCount.Load() < 1 {
			t.Error("expected at least one publish before stop")
		}
	})

	t.Run("publishes on stop", func(t *testing.T) {
		publisher := &trackingPublisher{}
		bg := NewBackgroundPublisher(publisher, 1*time.Hour, func() *rentfree.PublisherHealthMetrics {
			return &rentfree.PublisherHealthMetrics{}
		}, nil) // Long interval

		ctx := context.Background()
		bg.Start(ctx)
		countBefore := publisher.publishCount.Load()
		bg.Stop()
		countAfter := publisher.publishCount.Load()

		if countAfter <= countBefore {
			t.Error("expected publish on stop")
		}
	})

	t.Run("publish now", func(t *testing.T) {
		publisher := &trackingPublisher{}
		bg := NewBackgroundPublisher(publisher, 1*time.Hour, func() *rentfree.PublisherHealthMetrics {
			return &rentfree.PublisherHealthMetrics{}
		}, nil)

		ctx := context.Background()
		bg.Start(ctx)
		bg.PublishNow()
		bg.Stop()

		if publisher.publishCount.Load() < 2 {
			t.Error("expected at least 2 publishes (PublishNow + Stop)")
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		publisher := &trackingPublisher{}
		bg := NewBackgroundPublisher(publisher, 10*time.Millisecond, func() *rentfree.PublisherHealthMetrics {
			return &rentfree.PublisherHealthMetrics{}
		}, nil)

		ctx, cancel := context.WithCancel(context.Background())
		bg.Start(ctx)
		time.Sleep(30 * time.Millisecond)
		cancel() // Cancel context
		bg.Stop()

		// Should have published at least once
		if publisher.publishCount.Load() < 1 {
			t.Error("expected at least one publish")
		}
	})
}

func TestBackgroundService(t *testing.T) {
	t.Run("creates with tracker", func(t *testing.T) {
		tracker := NewTracker()
		publisher := NewNoOpPublisher()
		service := NewBackgroundService(tracker, publisher, 10*time.Millisecond, nil)
		if service == nil {
			t.Fatal("NewBackgroundService() returned nil")
		}
	})

	t.Run("start and stop with tracker", func(t *testing.T) {
		tracker := NewTracker()
		tracker.RecordHit("memory", "key", 10*time.Millisecond)

		publisher := &trackingPublisher{}
		service := NewBackgroundService(tracker, publisher, 10*time.Millisecond, nil)

		ctx := context.Background()
		service.Start(ctx)
		time.Sleep(50 * time.Millisecond)
		service.Stop()

		if publisher.publishCount.Load() < 1 {
			t.Error("expected at least one publish before stop")
		}
	})
}

func TestNoOpTracker(t *testing.T) {
	tracker := NewNoOpTracker()

	// All methods should be no-ops
	tracker.RecordHit("memory", "key", 10*time.Millisecond)
	tracker.RecordMiss("redis", "key", 10*time.Millisecond)
	tracker.RecordSet("memory", "key", 100, 10*time.Millisecond)
	tracker.RecordDelete("redis", "key", 10*time.Millisecond)
	tracker.RecordError("redis", "get", errors.New("error"))
	tracker.RecordCircuitBreakerStateChange("closed", "open")
	tracker.Reset()

	snapshot := tracker.Snapshot()
	if snapshot.GetCount != 0 {
		t.Errorf("NoOp GetCount = %d, want 0", snapshot.GetCount)
	}
}

func TestNoOpPublisher(t *testing.T) {
	publisher := NewNoOpPublisher()

	// All methods should be no-ops without error
	publisher.Gauge("test", 1.0, "tag:value")
	publisher.Incr("test", "tag:value")
	publisher.Count("test", 10, "tag:value")
	publisher.Histogram("test", 1.5, "tag:value")
	publisher.Timing("test", time.Second, "tag:value")
	publisher.Event("title", "text", "info", "tag:value")
	publisher.PublishHealthMetrics(&rentfree.PublisherHealthMetrics{})

	err := publisher.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestAvgDuration(t *testing.T) {
	tests := []struct {
		name      string
		durations []time.Duration
		expected  time.Duration
	}{
		{"empty", []time.Duration{}, 0},
		{"single", []time.Duration{10 * time.Millisecond}, 10 * time.Millisecond},
		{"multiple", []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 30 * time.Millisecond}, 20 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := avgDuration(tt.durations)
			if result != tt.expected {
				t.Errorf("avgDuration() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPercentile(t *testing.T) {
	tests := []struct {
		name      string
		durations []time.Duration
		p         int
		expected  time.Duration
	}{
		{"empty", []time.Duration{}, 50, 0},
		{"single_p50", []time.Duration{10 * time.Millisecond}, 50, 10 * time.Millisecond},
		{"ten_values_p50", []time.Duration{
			1 * time.Millisecond,
			2 * time.Millisecond,
			3 * time.Millisecond,
			4 * time.Millisecond,
			5 * time.Millisecond,
			6 * time.Millisecond,
			7 * time.Millisecond,
			8 * time.Millisecond,
			9 * time.Millisecond,
			10 * time.Millisecond,
		}, 50, 5 * time.Millisecond},
		{"ten_values_p90", []time.Duration{
			1 * time.Millisecond,
			2 * time.Millisecond,
			3 * time.Millisecond,
			4 * time.Millisecond,
			5 * time.Millisecond,
			6 * time.Millisecond,
			7 * time.Millisecond,
			8 * time.Millisecond,
			9 * time.Millisecond,
			10 * time.Millisecond,
		}, 90, 9 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := percentile(tt.durations, tt.p)
			if result != tt.expected {
				t.Errorf("percentile(%d) = %v, want %v", tt.p, result, tt.expected)
			}
		})
	}
}

func TestTagHelpers(t *testing.T) {
	tests := []struct {
		name     string
		fn       func() string
		expected string
	}{
		{"Tag", func() string { return Tag("key", "value") }, "key:value"},
		{"LevelTag", func() string { return LevelTag("memory") }, "level:memory"},
		{"OperationTag", func() string { return OperationTag("get") }, "operation:get"},
		{"PatternTag", func() string { return PatternTag("user:*") }, "pattern:user:*"},
		{"StatusTag", func() string { return StatusTag("hit") }, "status:hit"},
		{"LayerTag", func() string { return LayerTag("redis") }, "layer:redis"},
		{"CircuitStateTag", func() string { return CircuitStateTag("open") }, "circuit_state:open"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn()
			if result != tt.expected {
				t.Errorf("%s() = %q, want %q", tt.name, result, tt.expected)
			}
		})
	}
}

func TestTimer(t *testing.T) {
	publisher := &trackingPublisher{}

	timer := NewTimer(publisher, "test.operation", "layer:memory")

	// Simulate some work
	time.Sleep(10 * time.Millisecond)

	elapsed := timer.Elapsed()
	if elapsed < 10*time.Millisecond {
		t.Errorf("Elapsed() = %v, want >= 10ms", elapsed)
	}

	duration := timer.Stop()
	if duration < 10*time.Millisecond {
		t.Errorf("Stop() = %v, want >= 10ms", duration)
	}

	if publisher.timingCount.Load() != 1 {
		t.Errorf("timingCount = %d, want 1", publisher.timingCount.Load())
	}
}

// Helper for testing publishers
type trackingPublisher struct {
	publishCount atomic.Int64
	timingCount  atomic.Int64
}

func (p *trackingPublisher) Gauge(name string, value float64, tags ...string)     {}
func (p *trackingPublisher) Incr(name string, tags ...string)                     {}
func (p *trackingPublisher) Count(name string, value int64, tags ...string)       {}
func (p *trackingPublisher) Histogram(name string, value float64, tags ...string) {}
func (p *trackingPublisher) Timing(name string, duration time.Duration, tags ...string) {
	p.timingCount.Add(1)
}
func (p *trackingPublisher) Event(title, text, alertType string, tags ...string) {}
func (p *trackingPublisher) PublishHealthMetrics(metrics *rentfree.PublisherHealthMetrics) {
	p.publishCount.Add(1)
}
func (p *trackingPublisher) Close() error { return nil }

var _ rentfree.Publisher = (*trackingPublisher)(nil)
