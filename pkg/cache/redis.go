package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrCacheMiss = errors.New("cache miss")

type RedisKV struct {
	client *redis.Client
}

func NewRedisKV(addr, password string, db int) *RedisKV {
	return &RedisKV{
		client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
	}
}

func (r *RedisKV) Ping(ctx context.Context) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("redis client is not initialized")
	}
	return r.client.Ping(ctx).Err()
}

func (r *RedisKV) Close() error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Close()
}

func (r *RedisKV) Get(ctx context.Context, key string) (string, error) {
	value, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", ErrCacheMiss
		}
		return "", err
	}
	return value, nil
}

func (r *RedisKV) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisKV) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}
