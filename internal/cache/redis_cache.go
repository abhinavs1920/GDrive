package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"

)

var ctx = context.Background()

// RedisCache struct
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache initializes a Redis client
func NewRedisCache(addr string) *RedisCache {
	client := redis.NewClient(&redis.Options{Addr: addr})
	return &RedisCache{client: client}
}

// SetCache stores data in Redis
func (r *RedisCache) SetCache(key string, value string, ttl time.Duration) {
	err := r.client.Set(ctx, key, value, ttl).Err()
	if err != nil {
		fmt.Println("Error setting cache:", err)
	}
}

// GetCache retrieves data from Redis
func (r *RedisCache) GetCache(key string) string {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		fmt.Println("Cache miss for:", key)
		return ""
	}
	return val
}
