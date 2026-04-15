package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache Redis 缓存
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache 创建 Redis 缓存
func NewRedisCache(addr, password string, db int) *RedisCache {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
		PoolSize: 100,
	})

	return &RedisCache{
		client: client,
	}
}

// Get 获取缓存
func (c *RedisCache) Get(ctx context.Context, key string, value interface{}) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}

	return json.Unmarshal(data, value)
}

// Set 设置缓存
func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, expiration).Err()
}

// Delete 删除缓存
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// Exists 检查缓存是否存在
func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, key).Result()
	return n > 0, err
}

// Expire 设置过期时间
func (c *RedisCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.client.Expire(ctx, key, expiration).Err()
}

// Close 关闭连接
func (c *RedisCache) Close() error {
	return c.client.Close()
}

// ProviderCache Provider 缓存
type ProviderCache struct {
	cache *RedisCache
}

// NewProviderCache 创建 Provider 缓存
func NewProviderCache(cache *RedisCache) *ProviderCache {
	return &ProviderCache{
		cache: cache,
	}
}

// GetProvider 获取 Provider
func (p *ProviderCache) GetProvider(ctx context.Context, appID, channel string) (interface{}, error) {
	key := p.makeKey(appID, channel)
	var provider interface{}
	err := p.cache.Get(ctx, key, &provider)
	return provider, err
}

// SetProvider 设置 Provider
func (p *ProviderCache) SetProvider(ctx context.Context, appID, channel string, provider interface{}) error {
	key := p.makeKey(appID, channel)
	// Provider 缓存 1 小时
	return p.cache.Set(ctx, key, provider, time.Hour)
}

// DeleteProvider 删除 Provider
func (p *ProviderCache) DeleteProvider(ctx context.Context, appID, channel string) error {
	key := p.makeKey(appID, channel)
	return p.cache.Delete(ctx, key)
}

// makeKey 生成缓存 key
func (p *ProviderCache) makeKey(appID, channel string) string {
	return "provider:" + appID + ":" + channel
}

// ConfigCache 配置缓存
type ConfigCache struct {
	cache *RedisCache
}

// NewConfigCache 创建配置缓存
func NewConfigCache(cache *RedisCache) *ConfigCache {
	return &ConfigCache{
		cache: cache,
	}
}

// GetConfig 获取配置
func (c *ConfigCache) GetConfig(ctx context.Context, appID, channel string) (interface{}, error) {
	key := c.makeKey(appID, channel)
	var config interface{}
	err := c.cache.Get(ctx, key, &config)
	return config, err
}

// SetConfig 设置配置
func (c *ConfigCache) SetConfig(ctx context.Context, appID, channel string, config interface{}) error {
	key := c.makeKey(appID, channel)
	// 配置缓存 30 分钟
	return c.cache.Set(ctx, key, config, 30*time.Minute)
}

// DeleteConfig 删除配置
func (c *ConfigCache) DeleteConfig(ctx context.Context, appID, channel string) error {
	key := c.makeKey(appID, channel)
	return c.cache.Delete(ctx, key)
}

// makeKey 生成缓存 key
func (c *ConfigCache) makeKey(appID, channel string) string {
	return "config:" + appID + ":" + channel
}
