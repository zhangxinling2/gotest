package cache

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/sync/singleflight"
	"log"
	"time"
)

var (
	errFailToRefreshCache = errors.New("cache: 刷新缓存失败")
)

type ReadThroughCache struct {
	Cache
	LoadFunc   func(ctx context.Context, key string) (any, error)
	expiration time.Duration
	g          *singleflight.Group
}

// Get 同步刷新缓存
func (r *ReadThroughCache) Get(ctx context.Context, key string) (any, error) {
	//先从cache中取得值
	val, err := r.Cache.Get(ctx, key)
	//没有值就可以进行LoadFunc
	if err == errNoValue {
		v, err := r.LoadFunc(ctx, key)
		//LoadFunc成功
		if err == nil {
			//取得值后就刷新缓存
			er := r.Set(ctx, key, v, r.expiration)
			if er != nil {
				return nil, errors.New(fmt.Sprintf("%v,res: %s", errFailToRefreshCache, er))
			}
		}
	}
	return val, err
}

// GetV1 半异步刷新缓存，就是在取得值后异步的刷新缓存，在LoadFunc后开个goroutine即可
func (r *ReadThroughCache) GetV1(ctx context.Context, key string) (any, error) {
	//先从cache中取得值
	val, err := r.Cache.Get(ctx, key)
	//没有值就可以进行LoadFunc
	if err == errNoValue {
		v, err := r.LoadFunc(ctx, key)
		//LoadFunc成功
		if err == nil {
			go func() {
				//取得值后就刷新缓存
				er := r.Set(ctx, key, v, r.expiration)
				if er != nil {
					//由于是goroutine，所以只能log记录一下
					log.Printf("%v,res: %s", errFailToRefreshCache, er)
				}
			}()
		}
	}
	return val, err
}

// GetV2 异步刷新缓存，就是在缓存为空后异步的读取值和刷新缓存，在Get后开个goroutine即可
func (r *ReadThroughCache) GetV2(ctx context.Context, key string) (any, error) {
	//先从cache中取得值
	val, err := r.Cache.Get(ctx, key)
	//没有值就可以进行LoadFunc
	if err == errNoValue {
		go func() {
			v, err := r.LoadFunc(ctx, key)
			//LoadFunc成功
			if err == nil {
				//取得值后就刷新缓存
				er := r.Set(ctx, key, v, r.expiration)
				if er != nil {
					//由于是goroutine，所以只能log记录一下
					log.Printf("%v,res: %s", errFailToRefreshCache, er)
				}
			}
		}()
	}
	return val, err
}

// GetV3 Singleflight 在ReadThroughCache中再放一个g *singleflight.Group
func (r *ReadThroughCache) GetV3(ctx context.Context, key string) (any, error) {
	//先从cache中取得值
	val, err := r.Cache.Get(ctx, key)
	//没有值就可以进行LoadFunc
	if err == errNoValue {
		val, err, _ = r.g.Do(key, func() (interface{}, error) {
			v, er := r.LoadFunc(ctx, key)
			//LoadFunc成功
			if er == nil {
				//取得值后就刷新缓存
				er = r.Set(ctx, key, v, r.expiration)
				if er != nil {
					return nil, errors.New(fmt.Sprintf("%v,res: %s", errFailToRefreshCache, er))
				}
			}
			return v, er
		})
	}
	return val, err
}

type ReadThroughCacheV1[T any] struct {
	Cache
	LoadFunc   func(ctx context.Context, key string) (any, error)
	expiration time.Duration
	g          *singleflight.Group
}

// Get 并没有实现Cache中的Get方法，因为返回的是一个泛型数据
func (r *ReadThroughCacheV1[T]) Get(ctx context.Context, key string) (T, error) {
	//先从cache中取得值
	val, err := r.Cache.Get(ctx, key)
	//没有值就可以进行LoadFunc
	if err == errNoValue {
		v, err := r.LoadFunc(ctx, key)
		//LoadFunc成功
		if err == nil {
			//取得值后就刷新缓存
			er := r.Set(ctx, key, v, r.expiration)
			if er != nil {
				return v.(T), errors.New(fmt.Sprintf("%v,res: %s", errFailToRefreshCache, er))
			}
		}
	}
	return val.(T), err
}
