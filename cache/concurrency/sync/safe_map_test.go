package sync

import (
	"sync"
	"testing"
	"time"
)

func TestSafeMap_LoadOrStore(t *testing.T) {
	s := &SafeMap[string, string]{
		data:  make(map[string]string),
		mutex: sync.RWMutex{},
	}
	go func() {
		val, ok := s.LoadOrStore("key1", "value1")
		t.Log("goroutine1", val, ok)
	}()
	go func() {
		val, ok := s.LoadOrStore("key1", "value1")
		t.Log("goroutine2", val, ok)
	}()
	//有可能两个都false
	time.Sleep(time.Second)
}
