package sync

import "sync"

//SafeMap map安全的一个封装
type SafeMap[K comparable, V any] struct {
	data  map[K]V //K一定是comparable
	mutex sync.RWMutex
}

func (s *SafeMap[K, V]) Put(key K, val V) {
	s.mutex.Lock()
	//使用defer 就算s.data[key]=val发生了panic，锁还是会释放
	defer s.mutex.Unlock()
	s.data[key] = val
}

func (s *SafeMap[K, V]) Get(key K) (any, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	res, ok := s.data[key]
	return res, ok
}

func (s *SafeMap[K, V]) LoadOrStore(key K, newVal V) (val V, loaded bool) {
	s.mutex.RLock()
	res, ok := s.data[key]
	s.mutex.RUnlock()
	if ok {
		return res, true
	}

	s.mutex.Lock()
	//假如两个goroutine同时进来，那么都是false,那么都会跑下来其中一个goroutine会覆盖掉另一个
	//只有这样才能得到预期的结果，这种写法就是double-check
	res, ok = s.data[key]
	if ok {
		return res, true
	}
	defer s.mutex.Unlock()
	s.data[key] = newVal
	return newVal, false
}
