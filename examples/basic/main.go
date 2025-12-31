// Package main demonstrates basic usage of the rentfree cache library.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/LavishGent/rentfree/pkg/rentfree"
)

// User represents a sample data type to cache.
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func main() {
	ctx := context.Background()

	// Create a memory-only cache (no Redis required for this example)
	cache, err := rentfree.NewMemoryOnly()
	if err != nil {
		log.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	fmt.Println("=== Basic Cache Operations ===")

	user := User{ID: 1, Name: "Alice", Email: "alice@example.com"}
	err = cache.Set(ctx, "user:1", user, rentfree.WithTTL(5*time.Minute))
	if err != nil {
		log.Fatalf("Failed to set: %v", err)
	}
	fmt.Println("Set user:1")

	var retrieved User
	err = cache.Get(ctx, "user:1", &retrieved)
	if err != nil {
		log.Fatalf("Failed to get: %v", err)
	}
	fmt.Printf("Got user:1: %+v\n", retrieved)

	exists, _ := cache.Contains(ctx, "user:1")
	fmt.Printf("user:1 exists: %v\n", exists)

	fmt.Println("\n=== GetOrCreate (Cache-Aside Pattern) ===")

	var cachedUser User
	err = cache.GetOrCreate(ctx, "user:1", &cachedUser, func() (any, error) {
		fmt.Println("Factory called (should not see this)")
		return User{ID: 1, Name: "From Factory"}, nil
	})
	if err != nil {
		log.Fatalf("Failed GetOrCreate: %v", err)
	}
	fmt.Printf("GetOrCreate (cached): %+v\n", cachedUser)

	var newUser User
	err = cache.GetOrCreate(ctx, "user:2", &newUser, func() (any, error) {
		fmt.Println("Factory called - creating user:2")
		return User{ID: 2, Name: "Bob", Email: "bob@example.com"}, nil
	})
	if err != nil {
		log.Fatalf("Failed GetOrCreate: %v", err)
	}
	fmt.Printf("GetOrCreate (new): %+v\n", newUser)

	fmt.Println("\n=== Cache Miss ===")

	var missing User
	err = cache.Get(ctx, "user:999", &missing)
	if rentfree.IsCacheMiss(err) {
		fmt.Println("user:999 not found (expected)")
	} else if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}

	fmt.Println("\n=== Delete ===")

	err = cache.Delete(ctx, "user:1")
	if err != nil {
		log.Fatalf("Failed to delete: %v", err)
	}
	fmt.Println("Deleted user:1")

	exists, _ = cache.Contains(ctx, "user:1")
	fmt.Printf("user:1 exists after delete: %v\n", exists)

	fmt.Println("\n=== Health Check ===")

	health, err := cache.Health(ctx)
	if err != nil {
		log.Fatalf("Health check failed: %v", err)
	}
	fmt.Printf("Status: %s\n", health.Status)
	fmt.Printf("Memory Available: %v\n", health.Memory.Available)
	fmt.Printf("Memory Entries: %d\n", health.Memory.EntryCount)
	fmt.Printf("Redis Available: %v (expected false - not configured)\n", health.Redis.Available)

	fmt.Println("\n=== Done ===")
}
