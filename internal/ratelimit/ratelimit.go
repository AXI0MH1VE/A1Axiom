package ratelimit

import (
	"fmt"
	"sync"
	"time"
)

// TokenBucket implements a token bucket rate limiter.
type TokenBucket struct {
	mu        sync.Mutex
	capacity  int64
	tokens    int64
	refillRate int64 // tokens per second
	lastRefill time.Time
}

// NewTokenBucket creates a new token bucket with the specified capacity and refill rate.
func NewTokenBucket(capacity int64, refillRate int64) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request should be allowed based on the current token count.
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}
	return false
}

// refill adds tokens to the bucket based on elapsed time.
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)
	tokensToAdd := int64(elapsed.Seconds()) * tb.refillRate

	if tokensToAdd > 0 {
		tb.tokens = min(tb.capacity, tb.tokens+tokensToAdd)
		tb.lastRefill = now
	}
}

// GetTokens returns the current number of available tokens.
func (tb *TokenBucket) GetTokens() int64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.refill()
	return tb.tokens
}

// RateLimiter provides HTTP middleware for rate limiting.
type RateLimiter struct {
	bucket *TokenBucket
}

// NewRateLimiter creates a new rate limiter with the specified requests per second.
func NewRateLimiter(requestsPerSecond int64, burstSize int64) *RateLimiter {
	if burstSize <= 0 {
		burstSize = requestsPerSecond
	}
	return &RateLimiter{
		bucket: NewTokenBucket(burstSize, requestsPerSecond),
	}
}

// Middleware returns an HTTP middleware function for rate limiting.
func (rl *RateLimiter) Middleware(next func()) func() bool {
	return func() bool {
		if !rl.bucket.Allow() {
			return false // Request denied
		}
		next()
		return true // Request allowed and processed
	}
}

// GetStats returns current rate limiter statistics.
func (rl *RateLimiter) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"available_tokens": rl.bucket.GetTokens(),
		"capacity":        rl.bucket.capacity,
		"refill_rate":     rl.bucket.refillRate,
	}
}

// Allow checks if a request should be allowed based on the current token count.
func (rl *RateLimiter) Allow() bool {
	if rl == nil || rl.bucket == nil {
		return true // Allow if rate limiter is not configured
	}
	return rl.bucket.Allow()
}

// min returns the minimum of two int64 values.
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}