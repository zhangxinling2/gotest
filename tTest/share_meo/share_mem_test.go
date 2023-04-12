package share_meo

import (
	"sync"
	"testing"
	"time"
)

func TestCounter(t *testing.T) {
	count:=0
	for t:=0;t<5000;t++{//存在竞争，需要进行一个锁的保护
		go func(){
			count++
		}()
	}
	time.Sleep(1*time.Second)
	t.Logf("counter : %d",count)
}

func TestCounterThreadSafe(t *testing.T) {
	var mut sync.Mutex
	count:=0
	for t:=0;t<5000;t++{//使用mutex保证线程安全
		go func(){
			defer func(){
				mut.Unlock()
			}()
			mut.Lock()
			count++
		}()
	}
	time.Sleep(1*time.Second)//不加sleep 主协程运行完可能其他协程还没有运行
	t.Logf("counter : %d",count)
}

func TestCounterThreadSafeWaitGroup(t *testing.T) {
	var wg sync.WaitGroup
	var mut sync.Mutex
	count:=0
	for t:=0;t<5000;t++{//使用wait group所有现场执行完之后，才会执行wait,不需要知道运行时间
		wg.Add(1)
		go func(){
			defer func(){
				mut.Unlock()
			}()
			mut.Lock()
			count++
			wg.Done()
		}()
	}
	wg.Wait()
	//wait group不需要加sleep
	t.Logf("counter : %d",count)
}