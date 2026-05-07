package limiter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func TestLocalRateLimiter_Allow(t *testing.T) {
	limiter := NewLocalRateLimiter(rate.Limit(10), 10)

	// Should allow initial burst
	for i := 0; i < 10; i++ {
		allowed, err := limiter.Allow(context.Background(), "test")
		assert.NoError(t, err)
		assert.True(t, allowed, "request %d should be allowed", i)
	}

	// After burst exhausted, should deny
	allowed, err := limiter.Allow(context.Background(), "test")
	assert.NoError(t, err)
	assert.False(t, allowed)
}

func TestLocalRateLimiter_Wait(t *testing.T) {
	limiter := NewLocalRateLimiter(rate.Limit(1000), 1)

	err := limiter.Wait(context.Background())
	assert.NoError(t, err)
}

func TestLocalRateLimiter_Wait_CancelledContext(t *testing.T) {
	limiter := NewLocalRateLimiter(rate.Limit(0.001), 1)

	// Exhaust the burst
	limiter.Allow(context.Background(), "test")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := limiter.Wait(ctx)
	assert.Error(t, err)
}

func TestNewTokenBucketLimiter(t *testing.T) {
	// Just test construction (no Redis needed)
	limiter := NewTokenBucketLimiter(nil, 100, 200)
	assert.NotNil(t, limiter)
	assert.Equal(t, 100, limiter.rate)
	assert.Equal(t, 200, limiter.burst)
}

func TestNewIPRateLimiter(t *testing.T) {
	limiter := NewIPRateLimiter(nil, 100, 200)
	assert.NotNil(t, limiter)
}

func TestNewUserRateLimiter(t *testing.T) {
	limiter := NewUserRateLimiter(nil, 50, 100)
	assert.NotNil(t, limiter)
}

func TestNewAPIRateLimiter(t *testing.T) {
	limiter := NewAPIRateLimiter(nil, 200, 400)
	assert.NotNil(t, limiter)
}
