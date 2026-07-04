package daraja

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisTokenCache implements TokenCache using Redis.
type RedisTokenCache struct {
	client *redis.Client
}

// NewRedisTokenCache creates a new RedisTokenCache.
func NewRedisTokenCache(client *redis.Client) *RedisTokenCache {
	return &RedisTokenCache{client: client}
}

func (c *RedisTokenCache) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

func (c *RedisTokenCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}
