package notion

import (
	"context"
	"testing"
	"time"
)

func TestNewWorkerPool(t *testing.T) {
	client := &Client{limiter: NewRateLimiter(100, 20)}

	tests := []struct {
		name        string
		concurrency int
		want        int
	}{
		{"normal concurrency", 5, 5},
		{"zero defaults to 1", 0, 1},
		{"negative defaults to 1", -1, 1},
		{"capped at 20", 50, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewWorkerPool(client, tt.concurrency)
			if pool.concurrency != tt.want {
				t.Errorf("NewWorkerPool(%d) concurrency = %d, want %d", tt.concurrency, pool.concurrency, tt.want)
			}
		})
	}
}

func TestDefaultWorkerPool(t *testing.T) {
	client := &Client{limiter: NewRateLimiter(100, 20)}
	pool := DefaultWorkerPool(client)

	if pool.concurrency != 5 {
		t.Errorf("DefaultWorkerPool() concurrency = %d, want 5", pool.concurrency)
	}
}

func TestWorkerPool_FetchBlocksParallel_ContextCanceled(t *testing.T) {
	client := &Client{limiter: NewRateLimiter(100, 20)}
	pool := NewWorkerPool(client, 3)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	pageIDs := []string{"page1", "page2", "page3"}
	results := pool.FetchBlocksParallel(ctx, pageIDs)

	// Should complete quickly without blocking
	select {
	case <-time.After(1 * time.Second):
		t.Fatal("FetchBlocksParallel did not respect canceled context")
	case _, ok := <-results:
		// Channel should be closed or return quickly
		if ok {
			// Drain any remaining results
			for range results {
			}
		}
	}
}

func TestWorkerPool_SemaphoreLimit(t *testing.T) {
	// This test verifies that the semaphore properly limits concurrency
	client := &Client{limiter: NewRateLimiter(1000, 100)} // High rate to not interfere
	pool := NewWorkerPool(client, 2)                      // Only 2 concurrent

	// Verify semaphore channel has correct capacity
	if cap(pool.semaphore) != 2 {
		t.Errorf("semaphore capacity = %d, want 2", cap(pool.semaphore))
	}
}
