package notion

import (
	"context"
	"strconv"
	"sync"
	"time"
)

// RateLimiter implements token bucket rate limiting for Notion API requests.
// It allows bursts up to the bucket size and refills at a steady rate.
// Designed for parallel requests: multiple goroutines can call Wait() concurrently.
type RateLimiter struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
	retryAfter time.Time // time to wait if we got a 429

	// Adaptive rate limiting: track recent 429s and back off
	consecutiveThrottles int
	lastThrottleTime     time.Time
}

// NewRateLimiter creates a rate limiter that allows requestsPerSecond average rate.
// burstSize determines how many requests can be made in quick succession.
// This limiter is safe for concurrent use by multiple goroutines.
func NewRateLimiter(requestsPerSecond float64, burstSize int) *RateLimiter {
	return &RateLimiter{
		tokens:     float64(burstSize),
		maxTokens:  float64(burstSize),
		refillRate: requestsPerSecond,
		lastRefill: time.Now(),
	}
}

// DefaultRateLimiter creates a rate limiter configured for Notion's API limits.
// Notion allows ~3 requests per second with bursts. We use a larger burst size (10)
// to enable parallel fetching, since Notion has no penalty for hitting 429s.
func DefaultRateLimiter() *RateLimiter {
	return NewRateLimiter(3.0, 10)
}

// Wait blocks until a request can be made without exceeding rate limits.
// It respects any Retry-After time set by SetRetryAfter.
// Safe for concurrent use by multiple goroutines.
func (r *RateLimiter) Wait(ctx context.Context) error {
	r.mu.Lock()

	now := time.Now()

	// If we have a retry-after time from a 429, wait until then
	if now.Before(r.retryAfter) {
		waitDuration := r.retryAfter.Sub(now)
		r.mu.Unlock()
		select {
		case <-time.After(waitDuration):
		case <-ctx.Done():
			return ctx.Err()
		}
		r.mu.Lock()
		now = time.Now()
	}

	// Refill tokens based on time elapsed
	elapsed := now.Sub(r.lastRefill).Seconds()
	r.tokens += elapsed * r.refillRate
	if r.tokens > r.maxTokens {
		r.tokens = r.maxTokens
	}
	r.lastRefill = now

	// If we don't have tokens, calculate wait time and release lock while waiting
	if r.tokens < 1 {
		waitDuration := time.Duration((1 - r.tokens) / r.refillRate * float64(time.Second))
		r.mu.Unlock()
		select {
		case <-time.After(waitDuration):
		case <-ctx.Done():
			return ctx.Err()
		}
		r.mu.Lock()
		// Recalculate tokens after waiting
		now = time.Now()
		elapsed = now.Sub(r.lastRefill).Seconds()
		r.tokens += elapsed * r.refillRate
		if r.tokens > r.maxTokens {
			r.tokens = r.maxTokens
		}
		r.lastRefill = now
	}

	// Consume a token
	r.tokens--
	r.mu.Unlock()

	return nil
}

// SetRetryAfter sets a time to wait before making more requests.
// Call this when receiving a 429 response with a Retry-After header.
// Implements adaptive backoff: consecutive 429s increase the wait time.
func (r *RateLimiter) SetRetryAfter(duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	// Track consecutive throttles for adaptive backoff
	if now.Sub(r.lastThrottleTime) < 30*time.Second {
		r.consecutiveThrottles++
	} else {
		r.consecutiveThrottles = 1
	}
	r.lastThrottleTime = now

	// Apply exponential backoff multiplier for consecutive throttles
	// 1st: 1x, 2nd: 2x, 3rd: 4x, capped at 8x
	multiplier := 1 << min(r.consecutiveThrottles-1, 3)
	adjustedDuration := duration * time.Duration(multiplier)

	// Cap at 30 seconds max
	if adjustedDuration > 30*time.Second {
		adjustedDuration = 30 * time.Second
	}

	r.retryAfter = now.Add(adjustedDuration)
	// Clear tokens to prevent burst after retry
	r.tokens = 0
}

// ResetThrottleState resets the consecutive throttle counter.
// Call this after a successful request following a throttle period.
func (r *RateLimiter) ResetThrottleState() {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Only reset if enough time has passed since last throttle
	if time.Since(r.lastThrottleTime) > 10*time.Second {
		r.consecutiveThrottles = 0
	}
}

// ParseRetryAfter parses the Retry-After header value.
// It handles both delta-seconds and HTTP-date formats.
func ParseRetryAfter(value string) time.Duration {
	if value == "" {
		return time.Second // Default to 1 second if not specified
	}

	// Try parsing as seconds
	if seconds, err := strconv.Atoi(value); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// Try parsing as HTTP-date
	if t, err := time.Parse(time.RFC1123, value); err == nil {
		return time.Until(t)
	}

	// Default fallback
	return time.Second
}
