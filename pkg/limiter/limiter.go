package limiter

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

// RateLimiter 限流器接口
type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}

// TokenBucketLimiter 令牌桶限流器
type TokenBucketLimiter struct {
	client *redis.Client
	rate   int           // 每秒生成的令牌数
	burst  int           // 桶容量
	window time.Duration // 时间窗口
}

// NewTokenBucketLimiter 创建令牌桶限流器
func NewTokenBucketLimiter(client *redis.Client, rate, burst int) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		client: client,
		rate:   rate,
		burst:  burst,
		window: time.Second,
	}
}

// Allow 检查是否允许请求
func (l *TokenBucketLimiter) Allow(ctx context.Context, key string) (bool, error) {
	script := `
		local key = KEYS[1]
		local rate = tonumber(ARGV[1])
		local burst = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		local window = tonumber(ARGV[4])

		local tokens_key = key .. ":tokens"
		local timestamp_key = key .. ":timestamp"

		local last_tokens = tonumber(redis.call("get", tokens_key)) or burst
		local last_timestamp = tonumber(redis.call("get", timestamp_key)) or now

		local delta = math.max(0, now - last_timestamp)
		local new_tokens = math.min(burst, last_tokens + delta * rate / window)

		if new_tokens >= 1 then
			new_tokens = new_tokens - 1
			redis.call("setex", tokens_key, window * 2, new_tokens)
			redis.call("setex", timestamp_key, window * 2, now)
			return 1
		else
			return 0
		end
	`

	result, err := l.client.Eval(ctx, script, []string{key},
		l.rate, l.burst, time.Now().Unix(), int(l.window.Seconds())).Int()

	if err != nil {
		return false, err
	}

	return result == 1, nil
}

// LocalRateLimiter 本地限流器（基于 golang.org/x/time/rate）
type LocalRateLimiter struct {
	limiter *rate.Limiter
}

// NewLocalRateLimiter 创建本地限流器
func NewLocalRateLimiter(r rate.Limit, burst int) *LocalRateLimiter {
	return &LocalRateLimiter{
		limiter: rate.NewLimiter(r, burst),
	}
}

// Allow 检查是否允许请求
func (l *LocalRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	return l.limiter.Allow(), nil
}

// Wait 等待直到允许请求
func (l *LocalRateLimiter) Wait(ctx context.Context) error {
	return l.limiter.Wait(ctx)
}

// IPRateLimiter IP 限流器
type IPRateLimiter struct {
	limiter *TokenBucketLimiter
}

// NewIPRateLimiter 创建 IP 限流器
func NewIPRateLimiter(client *redis.Client, rate, burst int) *IPRateLimiter {
	return &IPRateLimiter{
		limiter: NewTokenBucketLimiter(client, rate, burst),
	}
}

// Allow 检查 IP 是否允许请求
func (l *IPRateLimiter) Allow(ctx context.Context, ip string) (bool, error) {
	key := "ratelimit:ip:" + ip
	return l.limiter.Allow(ctx, key)
}

// UserRateLimiter 用户限流器
type UserRateLimiter struct {
	limiter *TokenBucketLimiter
}

// NewUserRateLimiter 创建用户限流器
func NewUserRateLimiter(client *redis.Client, rate, burst int) *UserRateLimiter {
	return &UserRateLimiter{
		limiter: NewTokenBucketLimiter(client, rate, burst),
	}
}

// Allow 检查用户是否允许请求
func (l *UserRateLimiter) Allow(ctx context.Context, userID string) (bool, error) {
	key := "ratelimit:user:" + userID
	return l.limiter.Allow(ctx, key)
}

// APIRateLimiter API 限流器
type APIRateLimiter struct {
	limiter *TokenBucketLimiter
}

// NewAPIRateLimiter 创建 API 限流器
func NewAPIRateLimiter(client *redis.Client, rate, burst int) *APIRateLimiter {
	return &APIRateLimiter{
		limiter: NewTokenBucketLimiter(client, rate, burst),
	}
}

// Allow 检查 API 是否允许请求
func (l *APIRateLimiter) Allow(ctx context.Context, api string) (bool, error) {
	key := "ratelimit:api:" + api
	return l.limiter.Allow(ctx, key)
}
