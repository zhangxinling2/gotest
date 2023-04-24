package cache

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	errNoValue = errors.New("cache : 没有对应key的值")
)

type BuildInMapCacheOption func(b *BuildInMapCache)

type BuildInMapCache struct {
	data map[string]*item
	// 加锁保护数据
	mutex     sync.RWMutex
	close     chan struct{}
	onEvicted func(key string, val any)
}

// NewBuildInMapCache 新建cache,并且建立一个goroutine轮询
func NewBuildInMapCache(interval time.Duration, opts ...BuildInMapCacheOption) *BuildInMapCache {
	res := &BuildInMapCache{
		data:  make(map[string]*item, 100),
		close: make(chan struct{}),
		onEvicted: func(key string, val any) {

		},
	}
	for _, opt := range opts {
		opt(res)
	}
	//如何关闭这个goroutine？在结构体中维护一个channel来关闭
	go func() {
		//创建定时器
		ticker := time.NewTicker(interval)
		//轮询
		//for t:=range ticker.C{
		//	i:=0
		//	for k,v:=range res.data{
		//		//要是过期时间不为0并且在t之前，那么就代表Key过期
		//		if v.deadlineBefore(t){
		//			delete(res.data,k)
		//		}
		//		i++
		//		//轮询一千个数后开始下一次轮询
		//		if i>1000{
		//			break
		//		}
		//	}
		//}
		//为了能够关闭goroutine，使用select
		for {
			select {
			case t := <-ticker.C:
				res.mutex.Lock()
				i := 0
				for k, v := range res.data {
					//轮询一千个数后开始下一次轮询
					if i > 1000 {
						break
					}
					//要是过期时间不为0并且在t之前，那么就代表Key过期
					if v.deadlineBefore(t) {
						res.delete(k)
					}
					i++

				}
				res.mutex.Unlock()
			case <-res.close:
				return
			}
		}
	}()
	return res
}
func (b *BuildInMapCache) delete(key string) {
	//	b.mutex.RLock() 在外部调用delete时都已经加了锁，所以在这里加锁会导致程序卡死
	val, ok := b.data[key]
	//	b.mutex.RUnlock()
	if !ok {
		return
	}
	b.onEvicted(key, val.val)
	delete(b.data, key)
	return
}
func BuildInMapCacheWithEvictCallBack(fn func(key string, val any)) BuildInMapCacheOption {
	return func(b *BuildInMapCache) {
		b.onEvicted = fn
	}
}
func (b *BuildInMapCache) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.set(key,val,expiration)
}
func (b *BuildInMapCache) set( key string, val any, expiration time.Duration) error {
	var dl time.Time
	//如果expiration = 0那么它就是没有过期时间
	if expiration > 0 {
		dl = time.Now().Add(expiration)
	}
	b.data[key] = &item{
		val:        val,
		expiration: dl,
	}
	return nil
}
// Get 在Get时判断key有没有超时
func (b *BuildInMapCache) Get(ctx context.Context, key string) (any, error) {
	b.mutex.RLock()
	val, ok := b.data[key]
	b.mutex.RUnlock()
	if !ok {
		return nil, errNoValue
	}
	b.mutex.Lock()
	defer b.mutex.Unlock()
	//使用double-check，防止在加写锁之前有goroutine更新
	val, ok = b.data[key]
	if !ok {
		return nil, errNoValue
	}
	if val.deadlineBefore(time.Now()) {
		b.delete(key)
	}
	return val.val, nil
}

// Delete 删除map值,同时也返回值
func (b *BuildInMapCache) Delete(ctx context.Context, key string) (any, error) {
	b.mutex.RLock()
	defer b.mutex.Unlock()
	res, ok := b.data[key]
	b.mutex.RUnlock()
	if !ok {
		return nil, errNoValue
	}
	b.mutex.Lock()
	b.delete(key)
	return res.val, nil
}
func (b *BuildInMapCache) Close() error {
	b.close <- struct{}{}
	//使用select测试时会跑到default
	//select {
	//case b.close<- struct{}{}:
	//default:
	//	return errors.New("cache 重复关闭")
	//}
	return nil
}

// item 为值加上超时控制
type item struct {
	val any
	//expiration 超时时间
	expiration time.Time
}

func (i *item) deadlineBefore(t time.Time) bool {
	return !i.expiration.IsZero() && i.expiration.Before(t)
}
