package synthesis

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
)

// RateLimiter implements token bucket algorithm for synthesis rate limiting
type RateLimiter struct {
	mu sync.RWMutex

	// Per-namespace rate limiting
	namespaceTokens map[string]*TokenBucket

	// Global configuration
	maxSynthesisPerNamespacePerHour int
	log                             logr.Logger
}

// TokenBucket represents a token bucket for rate limiting
type TokenBucket struct {
	tokens     float64
	capacity   float64
	refillRate float64 // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxPerHour int, log logr.Logger) *RateLimiter {
	return &RateLimiter{
		namespaceTokens:                 make(map[string]*TokenBucket),
		maxSynthesisPerNamespacePerHour: maxPerHour,
		log:                             log,
	}
}

// NewTokenBucket creates a new token bucket with given capacity and refill rate
func NewTokenBucket(capacity, refillRate float64) *TokenBucket {
	return &TokenBucket{
		tokens:     capacity,
		capacity:   capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// CheckAndConsume checks if synthesis is allowed and consumes a token if so
func (rl *RateLimiter) CheckAndConsume(ctx context.Context, namespace string) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Get or create token bucket for namespace
	bucket, exists := rl.namespaceTokens[namespace]
	if !exists {
		// Create new bucket: capacity = max per hour, refill rate = max per hour / 3600 seconds
		refillRate := float64(rl.maxSynthesisPerNamespacePerHour) / 3600.0
		bucket = NewTokenBucket(float64(rl.maxSynthesisPerNamespacePerHour), refillRate)
		rl.namespaceTokens[namespace] = bucket
	}

	// Try to consume a token
	if err := bucket.Consume(1.0); err != nil {
		rl.log.Info("Synthesis rate limit exceeded",
			"namespace", namespace,
			"limit", rl.maxSynthesisPerNamespacePerHour,
			"availableTokens", bucket.tokens)
		return fmt.Errorf("synthesis rate limit exceeded for namespace %s: %d per hour (retry in %.0f seconds)",
			namespace, rl.maxSynthesisPerNamespacePerHour, bucket.TimeUntilTokens(1.0).Seconds())
	}

	rl.log.V(1).Info("Synthesis rate limit check passed",
		"namespace", namespace,
		"remainingTokens", bucket.tokens)

	return nil
}

// Consume attempts to consume tokens from the bucket
func (tb *TokenBucket) Consume(tokens float64) error {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// Refill tokens based on elapsed time
	tb.refill()

	// Check if enough tokens available
	if tb.tokens < tokens {
		return fmt.Errorf("insufficient tokens: have %.2f, need %.2f", tb.tokens, tokens)
	}

	// Consume tokens
	tb.tokens -= tokens
	return nil
}

// refill adds tokens to the bucket based on elapsed time
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()

	// Add tokens based on refill rate
	tb.tokens += elapsed * tb.refillRate

	// Cap at capacity
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}

	tb.lastRefill = now
}

// TimeUntilTokens returns the time until the specified number of tokens will be available
func (tb *TokenBucket) TimeUntilTokens(tokens float64) time.Duration {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens >= tokens {
		return 0
	}

	tokensNeeded := tokens - tb.tokens
	secondsNeeded := tokensNeeded / tb.refillRate

	return time.Duration(secondsNeeded * float64(time.Second))
}

// GetNamespaceStats returns current statistics for a namespace
func (rl *RateLimiter) GetNamespaceStats(namespace string) (availableTokens, capacity float64, exists bool) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	bucket, exists := rl.namespaceTokens[namespace]
	if !exists {
		return 0, 0, false
	}

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	bucket.refill()
	return bucket.tokens, bucket.capacity, true
}

// Reset clears all rate limit state (useful for testing)
func (rl *RateLimiter) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.namespaceTokens = make(map[string]*TokenBucket)
}
