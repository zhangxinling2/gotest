package obj_cache

import (
	"fmt"
	"sync"
	"testing"
)

func TestSyncPool(t *testing.T) {
	pool:=sync.Pool{
		New: func() interface {}{
			fmt.Println("Create a new obj")
			return 100
		},
	}
	v:=pool.Get().(int)
	fmt.Println(v)
	pool.Put(3)
	//runtime.GC()//1.13后一次GC不会回收对象
	v1,_:=pool.Get().(int)
	fmt.Println(v1)
	v2,_:=pool.Get().(int)
	fmt.Println(v2)
}
func TestSyncPoolInMutiGroutine(t *testing.T) {
	pool:=sync.Pool{
		New: func()interface {}{
			fmt.Println("create a new obj")
			return 10
		},
	}
	pool.Put(100)
	pool.Put(100)
	pool.Put(100)
	var wg sync.WaitGroup
	for i:=0;i<10;i++{
		wg.Add(1)
		go func(id int) {
			t.Log(pool.Get())
			wg.Done()
		}(i)
	}
	wg.Wait()
}