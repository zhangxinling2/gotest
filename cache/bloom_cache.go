package cache

import (
	"context"
	"fmt"
)

type BloomFilter interface {
	HasKey(ctx context.Context, key string) bool
}

// BloomFilterCache 直接组合ReadThroughCache
type BloomFilterCache struct {
	ReadThroughCache
}

func NewBloomFilterCache(cache Cache, filter BloomFilter, LoadFunc func(ctx context.Context, key string) (any, error)) *BloomFilterCache {
	return &BloomFilterCache{ReadThroughCache{
		Cache: cache,
		LoadFunc: func(ctx context.Context, key string) (any, error) {
			if filter.HasKey(ctx, key) {
				return LoadFunc(ctx, key)
			}
			return nil, errNoValue
		},
	}}
}

//BloomFilterCacheV1 组合ReadThroughCache,持有一个BloomFilter，直接修改Get方法
type BloomFilterCacheV1 struct {
	ReadThroughCache
	bf BloomFilter
}

func (b *BloomFilterCacheV1) Get(ctx context.Context, key string) (any, error) {
	val, err := b.Cache.Get(ctx, key)
	if err != nil && b.bf.HasKey(ctx, key) {
		val, err = b.LoadFunc(ctx, key)
		if err == nil {
			er := b.Cache.Set(ctx, key, val, b.expiration)
			if er != nil {
				return val, fmt.Errorf("%w, 原因：%s", errFailToRefreshCache, er.Error())
			}
		}
	}
	return val, err
}
