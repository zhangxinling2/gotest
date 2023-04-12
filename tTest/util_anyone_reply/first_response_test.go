package util_anyone_reply

import (
	"fmt"
	"runtime"
	"testing"
	"time"
)

func RunTast(id int) string{
	time.Sleep(time.Microsecond*10)
	return fmt.Sprintf("the result is %d",id)
}

func FirstResponse()string{
	numOfRunner :=10
	ch:=make(chan string,numOfRunner)
	for i:=0;i<numOfRunner;i++{
		go func(i int) {
			ret:=RunTast(i)
			ch<-ret
		}(i)
	}
	return <-ch
}



func AllResponse()string{
	numOfRunner :=10
	ch:=make(chan string,numOfRunner)
	for i:=0;i<numOfRunner;i++{
		go func(i int) {
			ret:=RunTast(i)
			ch<-ret
		}(i)
	}
	final:=""
	for j:=0;j<numOfRunner;j++{
		final += <-ch +"\n"
	}
	return final

}
func TestFirstResponse(t *testing.T) {
	t.Log(AllResponse())
	t.Log(runtime.NumGoroutine())
}