package notion

import (
	"context"
	"strconv"
	"sync"
	"time"
)

// RateLimiter implements token bucket rate limiting for Notion API requests.
// It allows bursts up to the bucket size and refills at a steady rate.
type RateLimiter struct {
	mu          sync.Mutex
	tokens      float64
	maxTokens   float64
	refillRate  float64 // tokens per second
	lastRefill  time.Time
	retryAfter  time.Time // time to wait if we got a 429
	minInterval time.Duration
	lastRequest time.Time
}

// NewRateLimiter creates a rate limiter that allows requestsPerSecond average rate.
// burstSize determines how many requests can be made in quick succession.
func NewRateLimiter(requestsPerSecond float64, burstSize int) *RateLimiter {
	return &RateLimiter{
		tokens:      float64(burstSize),
		maxTokens:   float64(burstSize),
		refillRate:  requestsPerSecond,
		lastRefill:  time.Now(),
		minInterval: time.Duration(float64(time.Second) / requestsPerSecond),
	}
}

// DefaultRateLimiter creates a rate limiter configured for Notion's API limits.
// Notion allows ~3 requests per second with bursts.
func DefaultRateLimiter() *RateLimiter {
	return NewRateLimiter(3.0, 5)
}

// Wait blocks until a request can be made without exceeding rate limits.
// It respects any Retry-After time set by SetRetryAfter.
func (r *RateLimiter) Wait(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	// If we have a retry-after time from a 429, wait until then
	if now.Before(r.retryAfter) {
		waitDuration := r.retryAfter.Sub(now)
		r.mu.Unlock()
		select {
		case <-time.After(waitDuration):
		case <-ctx.Done():
			r.mu.Lock()
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

	// Ensure minimum interval between requests
	timeSinceLastRequest := now.Sub(r.lastRequest)
	if timeSinceLastRequest < r.minInterval {
		waitDuration := r.minInterval - timeSinceLastRequest
		r.mu.Unlock()
		select {
		case <-time.After(waitDuration):
		case <-ctx.Done():
			r.mu.Lock()
			return ctx.Err()
		}
		r.mu.Lock()
	}

	// If we don't have tokens, wait for one to be available
	if r.tokens < 1 {
		waitDuration := time.Duration((1 - r.tokens) / r.refillRate * float64(time.Second))
		r.mu.Unlock()
		select {
		case <-time.After(waitDuration):
		case <-ctx.Done():
			r.mu.Lock()
			return ctx.Err()
		}
		r.mu.Lock()
		r.tokens = 1
	}

	// Consume a token
	r.tokens--
	r.lastRequest = time.Now()

	return nil
}

// SetRetryAfter sets a time to wait before making more requests.
// Call this when receiving a 429 response with a Retry-After header.
func (r *RateLimiter) SetRetryAfter(duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.retryAfter = time.Now().Add(duration)
	// Also clear tokens to prevent burst after retry
	r.tokens = 0
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
