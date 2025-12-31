package cache

import (
	"context"
	"fmt"
	"testing"

	"github.com/LavishGent/rentfree/internal/config"
	"github.com/LavishGent/rentfree/internal/types"
)

func BenchmarkMemoryCache_Set(b *testing.B) {
	cfg := config.MemoryConfig{
		Enabled:      true,
		MaxSizeMB:    256,
		DefaultTTL:   config.DefaultConfig().Memory.DefaultTTL,
		Shards:       1024,
		MaxEntrySize: 10 * 1024 * 1024,
	}
	cache, err := NewMemoryCache(cfg, nil)
	if err != nil {
		b.Fatal(err)
	}
	defer cache.Close()

	ctx := context.Background()
	value := []byte("test-value-with-some-data")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key:%d", i)
		_ = cache.Set(ctx, key, value, nil)
	}
}

func BenchmarkMemoryCache_Get(b *testing.B) {
	cfg := config.MemoryConfig{
		Enabled:      true,
		MaxSizeMB:    256,
		DefaultTTL:   config.DefaultConfig().Memory.DefaultTTL,
		Shards:       1024,
		MaxEntrySize: 10 * 1024 * 1024,
	}
	cache, err := NewMemoryCache(cfg, nil)
	if err != nil {
		b.Fatal(err)
	}
	defer cache.Close()

	ctx := context.Background()
	value := []byte("test-value-with-some-data")

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key:%d", i)
		_ = cache.Set(ctx, key, value, nil)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key:%d", i%1000)
		_, _ = cache.Get(ctx, key)
	}
}

func BenchmarkMemoryCache_Delete(b *testing.B) {
	cfg := config.MemoryConfig{
		Enabled:      true,
		MaxSizeMB:    256,
		DefaultTTL:   config.DefaultConfig().Memory.DefaultTTL,
		Shards:       1024,
		MaxEntrySize: 10 * 1024 * 1024,
	}
	cache, err := NewMemoryCache(cfg, nil)
	if err != nil {
		b.Fatal(err)
	}
	defer cache.Close()

	ctx := context.Background()
	value := []byte("test-value-with-some-data")

	// Pre-populate cache
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key:%d", i)
		_ = cache.Set(ctx, key, value, nil)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key:%d", i)
		_ = cache.Delete(ctx, key)
	}
}

func BenchmarkMemoryCache_SetParallel(b *testing.B) {
	cfg := config.MemoryConfig{
		Enabled:      true,
		MaxSizeMB:    256,
		DefaultTTL:   config.DefaultConfig().Memory.DefaultTTL,
		Shards:       1024,
		MaxEntrySize: 10 * 1024 * 1024,
	}
	cache, err := NewMemoryCache(cfg, nil)
	if err != nil {
		b.Fatal(err)
	}
	defer cache.Close()

	ctx := context.Background()
	value := []byte("test-value-with-some-data")

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key:%d", i)
			_ = cache.Set(ctx, key, value, nil)
			i++
		}
	})
}

func BenchmarkMemoryCache_GetParallel(b *testing.B) {
	cfg := config.MemoryConfig{
		Enabled:      true,
		MaxSizeMB:    256,
		DefaultTTL:   config.DefaultConfig().Memory.DefaultTTL,
		Shards:       1024,
		MaxEntrySize: 10 * 1024 * 1024,
	}
	cache, err := NewMemoryCache(cfg, nil)
	if err != nil {
		b.Fatal(err)
	}
	defer cache.Close()

	ctx := context.Background()
	value := []byte("test-value-with-some-data")

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key:%d", i)
		_ = cache.Set(ctx, key, value, nil)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key:%d", i%1000)
			_, _ = cache.Get(ctx, key)
			i++
		}
	})
}

func BenchmarkMemoryCache_Contains(b *testing.B) {
	cfg := config.MemoryConfig{
		Enabled:      true,
		MaxSizeMB:    256,
		DefaultTTL:   config.DefaultConfig().Memory.DefaultTTL,
		Shards:       1024,
		MaxEntrySize: 10 * 1024 * 1024,
	}
	cache, err := NewMemoryCache(cfg, nil)
	if err != nil {
		b.Fatal(err)
	}
	defer cache.Close()

	ctx := context.Background()
	value := []byte("test-value")

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key:%d", i)
		_ = cache.Set(ctx, key, value, nil)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key:%d", i%1000)
		_, _ = cache.Contains(ctx, key)
	}
}

func BenchmarkSerializer_Marshal(b *testing.B) {
	serializer := NewJSONSerializer()
	data := types.CacheEntry{
		Value: []byte("test-data-with-some-content"),
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = serializer.Marshal(data)
	}
}

func BenchmarkSerializer_Unmarshal(b *testing.B) {
	serializer := NewJSONSerializer()
	data := types.CacheEntry{
		Value: []byte("test-data-with-some-content"),
	}
	serialized, _ := serializer.Marshal(data)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var entry types.CacheEntry
		_ = serializer.Unmarshal(serialized, &entry)
	}
}

// Benchmark with different payload sizes
func BenchmarkMemoryCache_Set_1KB(b *testing.B) {
	benchmarkMemoryCacheSetBySize(b, 1024)
}

func BenchmarkMemoryCache_Set_10KB(b *testing.B) {
	benchmarkMemoryCacheSetBySize(b, 10240)
}

func BenchmarkMemoryCache_Set_100KB(b *testing.B) {
	benchmarkMemoryCacheSetBySize(b, 102400)
}

func benchmarkMemoryCacheSetBySize(b *testing.B, size int) {
	cfg := config.MemoryConfig{
		Enabled:      true,
		MaxSizeMB:    256,
		DefaultTTL:   config.DefaultConfig().Memory.DefaultTTL,
		Shards:       1024,
		MaxEntrySize: size * 2, // Ensure it fits
	}
	cache, err := NewMemoryCache(cfg, nil)
	if err != nil {
		b.Fatal(err)
	}
	defer cache.Close()

	ctx := context.Background()
	value := make([]byte, size)
	for i := range value {
		value[i] = byte(i % 256)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key:%d", i)
		_ = cache.Set(ctx, key, value, nil)
	}
}
