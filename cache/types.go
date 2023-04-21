package cache

import (
	"context"
	"time"
)

type Cache interface {
	// Set 方法会设置一个过期时间
	Set(ctx context.Context, key string, val any, expiration time.Duration) error
	Get(ctx context.Context, key string) (any, error)
	Delete(ctx context.Context, key string) (any, error)
}
