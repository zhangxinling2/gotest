package sync

import (
	"sync"
	"testing"
)

func TestPool(t *testing.T) {
	p := sync.Pool{
		New: func() any {
			t.Log("创建资源")
			//最好永远不要反回nil
			return "hello"
		},
	}
	//拿出来要伴随还回去
	str := p.Get().(string)
	p.Put(str)
	t.Log(str)
	str = p.Get().(string)
	defer p.Put(str)
	t.Log(str)
}
