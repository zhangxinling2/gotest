package cache

import (
	"context"
	"errors"
	"sync/atomic"
	"time"
)

var (
	errNoCapacity = errors.New("cache:超过容量限制")
)

type MaxCntCache struct {
	*BuildInMapCache
	maxCnt int32
	cnt    int32
}

func NewMaxCntCache(b *BuildInMapCache, max int32) *MaxCntCache {
	res := &MaxCntCache{
		BuildInMapCache: b,
		maxCnt:          max,
		cnt:             0,
	}
	evict := b.onEvicted
	b.onEvicted = func(key string, val any) {
		atomic.AddInt32(&res.cnt, -1)
		if evict != nil {
			evict(key, val)
		}
	}
	return res
}

func (m *MaxCntCache) Set(ctx context.Context, key string, val item, expiration time.Duration) error {
	// 这种写法，如果 key 已经存在，你这计数就不准了
	//cnt := atomic.AddInt32(&m.cnt, 1)
	//if cnt > m.maxCnt {
	//	atomic.AddInt32(&m.cnt, -1)
	//	return errOverCapacity
	//}
	//return m.BuildInMapCache.Set(ctx, key, val, expiration)
	//这样写在unlock后如果有goroutine拿到锁后再次进行set而此goroutine中的Set还未结束的话仍然会导致计数错误
	//m.mutex.Lock()
	//_,ok:=m.data[key]
	//if !ok{
	//	m.cnt++
	//}
	//if m.cnt>m.maxCnt{
	//	m.mutex.Unlock()
	//	return errNoCapacity
	//}
	//m.mutex.Unlock()
	//return m.BuildInMapCache.Set(ctx,key,val,expiration)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	_, ok := m.data[key]
	if !ok {
		if m.cnt+1 > m.maxCnt {
			return errNoCapacity
		}
		m.cnt++
	}
	return m.set(key, val, expiration)
}
