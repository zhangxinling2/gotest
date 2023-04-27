package cache

import (
	"context"
	"fmt"
	"golang.org/x/sync/singleflight"
	"time"
)

type SingleFlightCache struct {
	ReadThroughCache
}

//NewSingleFlightCache 中传入cache,loadfunc,expiration复写掉他们。与在readThrough直接放入一个singflight不同，这样是非侵入式的设计。
func NewSingleFlightCache(cache Cache, loadFunc func(ctx context.Context, key string) (any, error), expiration time.Duration) *SingleFlightCache {
	return &SingleFlightCache{ReadThroughCache{
		Cache: cache,
		//只关注loadfunc而不关注同步异步
		LoadFunc: func(ctx context.Context, key string) (any, error) {
			g := &singleflight.Group{}
			val, err, _ := g.Do(key, func() (interface{}, error) {
				return loadFunc(ctx, key)
			})
			return val, err
		},
		expiration: expiration,
	}}
}

type SingleFlightCacheV1 struct {
	ReadThroughCache
	g *singleflight.Group
}

func (r *SingleFlightCacheV1) Get(ctx context.Context, key string) (any, error) {
	val, err := r.Cache.Get(ctx, key)
	if err == errNoValue {
		val, err, _ = r.g.Do(key, func() (interface{}, error) {
			v, er := r.LoadFunc(ctx, key)
			if er == nil {
				//_ = r.Cache.Set(ctx, key, val, r.Expiration)
				er = r.Cache.Set(ctx, key, val, r.expiration)
				if er != nil {
					return v, fmt.Errorf("%w, 原因：%s", errFailToRefreshCache, er.Error())
				}
			}
			return v, er
		})
	}
	return val, err
}
