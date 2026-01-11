package notion

import (
	"context"
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(3.0, 5)

	if limiter.maxTokens != 5 {
		t.Errorf("expected maxTokens 5, got %f", limiter.maxTokens)
	}

	if limiter.refillRate != 3.0 {
		t.Errorf("expected refillRate 3.0, got %f", limiter.refillRate)
	}
}

func TestDefaultRateLimiter(t *testing.T) {
	limiter := DefaultRateLimiter()

	if limiter.refillRate != 3.0 {
		t.Errorf("expected refillRate 3.0, got %f", limiter.refillRate)
	}

	if limiter.maxTokens != 5 {
		t.Errorf("expected maxTokens 5, got %f", limiter.maxTokens)
	}
}

func TestRateLimiter_Wait(t *testing.T) {
	limiter := NewRateLimiter(10.0, 3) // 10 req/s, burst of 3
	ctx := context.Background()

	// First 3 requests should be immediate (burst)
	for i := 0; i < 3; i++ {
		start := time.Now()
		if err := limiter.Wait(ctx); err != nil {
			t.Fatalf("Wait() error = %v", err)
		}
		elapsed := time.Since(start)
		// Should be very fast (< 200ms accounting for minimum interval)
		if elapsed > 200*time.Millisecond {
			t.Errorf("burst request %d took too long: %v", i, elapsed)
		}
	}
}

func TestRateLimiter_Wait_ContextCanceled(t *testing.T) {
	limiter := NewRateLimiter(0.1, 1) // Very slow: 1 req/10s
	ctx, cancel := context.WithCancel(context.Background())

	// Use up the token
	if err := limiter.Wait(ctx); err != nil {
		t.Fatalf("first Wait() error = %v", err)
	}

	// Cancel context immediately
	cancel()

	// Next wait should return context error
	err := limiter.Wait(ctx)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestRateLimiter_SetRetryAfter(t *testing.T) {
	limiter := NewRateLimiter(10.0, 5)
	ctx := context.Background()

	// Set a short retry-after
	limiter.SetRetryAfter(50 * time.Millisecond)

	start := time.Now()
	if err := limiter.Wait(ctx); err != nil {
		t.Fatalf("Wait() error = %v", err)
	}
	elapsed := time.Since(start)

	// Should have waited at least 50ms
	if elapsed < 50*time.Millisecond {
		t.Errorf("expected to wait at least 50ms, waited %v", elapsed)
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  time.Duration
	}{
		{
			name:  "seconds as integer",
			value: "5",
			want:  5 * time.Second,
		},
		{
			name:  "zero seconds",
			value: "0",
			want:  0,
		},
		{
			name:  "empty string",
			value: "",
			want:  time.Second,
		},
		{
			name:  "invalid string",
			value: "invalid",
			want:  time.Second,
		},
		{
			name:  "large value",
			value: "120",
			want:  120 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseRetryAfter(tt.value)
			if got != tt.want {
				t.Errorf("ParseRetryAfter(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestRateLimiter_TokenRefill(t *testing.T) {
	limiter := NewRateLimiter(100.0, 2) // 100 req/s for fast test
	ctx := context.Background()

	// Use all tokens
	for i := 0; i < 2; i++ {
		if err := limiter.Wait(ctx); err != nil {
			t.Fatalf("Wait() error = %v", err)
		}
	}

	// Wait for tokens to refill
	time.Sleep(50 * time.Millisecond) // Should refill ~5 tokens at 100/s

	// Should be able to make more requests
	start := time.Now()
	if err := limiter.Wait(ctx); err != nil {
		t.Fatalf("Wait() after refill error = %v", err)
	}
	elapsed := time.Since(start)

	// Should be relatively quick since tokens refilled
	if elapsed > 100*time.Millisecond {
		t.Errorf("request after refill took too long: %v", elapsed)
	}
}
