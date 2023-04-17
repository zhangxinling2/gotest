package sync

import "sync/atomic"

const (
	UNLOCK int32 = 0
	LOCKED int32 = 1
)

type Lock struct {
	state int32
}

func (l *Lock) Lock() {
	i := 0
	var locked = false
	//归根结底锁就是一个状态位，自旋就是for 加上CAS操作
	for locked := atomic.CompareAndSwapInt32(&l.state, UNLOCK, LOCKED); !locked && i < 10; i++ {

	}
	if locked {
		return
	}

	//加入队列  没办法模拟，因为要用到runtime中的一些私有方法
}
