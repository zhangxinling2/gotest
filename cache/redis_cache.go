package cache

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v9"
	"time"
)

var (
	errFailedToSetCache = errors.New("cache: 写入 redis 失败")
)

type RedisCache struct {
	client redis.Cmdable
}

func NewRedisCache(client redis.Cmdable) *RedisCache {
	return &RedisCache{client: client}
}

func (r *RedisCache) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
	val, err := r.client.Set(ctx, key, val, expiration).Result()
	if err != nil {
		return err
	}
	if val != "OK" {
		return errors.New(fmt.Sprintf("%v ,res: %s", errFailedToSetCache, val))
	}
	return nil
}

func (r *RedisCache) Get(ctx context.Context, key string) (any, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *RedisCache) Delete(ctx context.Context, key string) (any, error) {
	return r.client.GetDel(ctx, key).Result()
}
