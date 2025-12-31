package rentfree_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/LavishGent/rentfree/pkg/rentfree"
)

type BenchUser struct {
	ID    string
	Name  string
	Email string
	Age   int
}

func BenchmarkMemoryOnly_Set(b *testing.B) {
	manager, err := rentfree.NewMemoryOnly()
	if err != nil {
		b.Fatal(err)
	}
	defer manager.Close()

	ctx := context.Background()
	user := BenchUser{ID: "123", Name: "Alice", Email: "alice@example.com", Age: 30}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("user:%d", i)
		_ = manager.Set(ctx, key, user)
	}
}

func BenchmarkMemoryOnly_Get(b *testing.B) {
	manager, err := rentfree.NewMemoryOnly()
	if err != nil {
		b.Fatal(err)
	}
	defer manager.Close()

	ctx := context.Background()
	user := BenchUser{ID: "123", Name: "Alice", Email: "alice@example.com", Age: 30}

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("user:%d", i)
		_ = manager.Set(ctx, key, user)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("user:%d", i%1000)
		var result BenchUser
		_ = manager.Get(ctx, key, &result)
	}
}

func BenchmarkMemoryOnly_GetOrCreate(b *testing.B) {
	manager, err := rentfree.NewMemoryOnly()
	if err != nil {
		b.Fatal(err)
	}
	defer manager.Close()

	ctx := context.Background()
	factory := func() (any, error) {
		return BenchUser{ID: "456", Name: "Bob", Email: "bob@example.com", Age: 25}, nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("user:%d", i%100) // Reuse keys to test cache hits
		var result BenchUser
		_ = manager.GetOrCreate(ctx, key, &result, factory)
	}
}

func BenchmarkMemoryOnly_Delete(b *testing.B) {
	manager, err := rentfree.NewMemoryOnly()
	if err != nil {
		b.Fatal(err)
	}
	defer manager.Close()

	ctx := context.Background()
	user := BenchUser{ID: "123", Name: "Alice", Email: "alice@example.com", Age: 30}

	// Pre-populate cache
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("user:%d", i)
		_ = manager.Set(ctx, key, user)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("user:%d", i)
		_ = manager.Delete(ctx, key)
	}
}

func BenchmarkMemoryOnly_SetParallel(b *testing.B) {
	manager, err := rentfree.NewMemoryOnly()
	if err != nil {
		b.Fatal(err)
	}
	defer manager.Close()

	ctx := context.Background()
	user := BenchUser{ID: "123", Name: "Alice", Email: "alice@example.com", Age: 30}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("user:%d", i)
			_ = manager.Set(ctx, key, user)
			i++
		}
	})
}

func BenchmarkMemoryOnly_GetParallel(b *testing.B) {
	manager, err := rentfree.NewMemoryOnly()
	if err != nil {
		b.Fatal(err)
	}
	defer manager.Close()

	ctx := context.Background()
	user := BenchUser{ID: "123", Name: "Alice", Email: "alice@example.com", Age: 30}

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("user:%d", i)
		_ = manager.Set(ctx, key, user)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("user:%d", i%1000)
			var result BenchUser
			_ = manager.Get(ctx, key, &result)
			i++
		}
	})
}

func BenchmarkMemoryOnly_GetOrCreateParallel(b *testing.B) {
	manager, err := rentfree.NewMemoryOnly()
	if err != nil {
		b.Fatal(err)
	}
	defer manager.Close()

	ctx := context.Background()
	factory := func() (any, error) {
		return BenchUser{ID: "456", Name: "Bob", Email: "bob@example.com", Age: 25}, nil
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("user:%d", i%100)
			var result BenchUser
			_ = manager.GetOrCreate(ctx, key, &result, factory)
			i++
		}
	})
}

// Benchmark with different payload sizes
func BenchmarkMemoryOnly_Set_SmallPayload(b *testing.B) {
	benchmarkSetBySize(b, 10) // 10 bytes
}

func BenchmarkMemoryOnly_Set_MediumPayload(b *testing.B) {
	benchmarkSetBySize(b, 1024) // 1KB
}

func BenchmarkMemoryOnly_Set_LargePayload(b *testing.B) {
	benchmarkSetBySize(b, 10240) // 10KB
}

func benchmarkSetBySize(b *testing.B, size int) {
	manager, err := rentfree.NewMemoryOnly()
	if err != nil {
		b.Fatal(err)
	}
	defer manager.Close()

	ctx := context.Background()
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("data:%d", i)
		_ = manager.Set(ctx, key, data)
	}
}
