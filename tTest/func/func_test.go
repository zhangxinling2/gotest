package _func

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)
func returnMultivalues()(int,int){
	return rand.Intn(10),rand.Intn(20)
}
func timeSpent(inner func(op int)int)func (op int)int{
	return func(n int) int {
		start:=time.Now()
		ret:=inner(n)
		fmt.Println("time spent:",time.Since(start).Seconds())
		return ret
	}
}

func slowFun(op int)int{
	time.Sleep(time.Second*1)
	return op
}
func TestFn(t *testing.T) {
	a,b:=returnMultivalues()
	t.Log(a,b)
	tsSf:=timeSpent(slowFun)
	t.Log(tsSf(10))
}


func Sum(ops ...int)int{
	ret:=0
	for _,op:=range ops{
		ret+=op
	}
	return ret
}
func TestVarParam(t *testing.T){
	t.Log((Sum(1,2,3,5)))
}

func TestDefer(t *testing.T){
	defer func() {
		t.Log("defer")
	}()
	t.Log("Start")
	//panic("err")
	t.Log("end")
}