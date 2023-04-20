package channel

import (
	"errors"
	"sync"
)

type Broker struct {
	mutex sync.RWMutex
	chans []chan Msg
}

// Send 向消息队列发数据,Msg不用指针是因为如果在接受时修改数据，其他消费者也会受到影响
func (b *Broker) Send(m Msg) error {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	for _, ch := range b.chans {
		//ch <- m//这样写cap放满了这里会阻塞住
		select {
		case ch <- m:

		default:
			return errors.New("消息队列已满")
		}
	}
	return nil
}

// Subscribe 订阅    <-chan Msg  代表只读
func (b *Broker) Subscribe(cap int) (<-chan Msg, error) {
	//该给多少缓冲?设置cap让用户管
	res := make(chan Msg, cap)
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.chans = append(b.chans, res)
	return res, nil
}

func (b *Broker) Close() error {
	b.mutex.Lock()
	chans := b.chans
	//避开了b.chans被重复关闭的问题
	b.chans = nil
	b.mutex.Unlock()
	for _, ch := range chans {
		close(ch)
	}
	return nil
}

type Msg struct {
	Content string
}

type Listener func(m Msg)

type BrokerV2 struct {
	mutex     sync.RWMutex
	consumers []Listener
}

// 这种一发出就被消费了
func (b *BrokerV2) Send(m Msg) error {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	for _, c := range b.consumers {
		c(m)
	}
	return nil
}

func (b *BrokerV2) Subscribe(cb func(s Msg)) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.consumers = append(b.consumers, cb)
	return nil
}
