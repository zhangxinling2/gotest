package cancel_by_close

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func isCancelled(cancelChan chan struct{}) bool  {
	select {
	case ret,ok:=<-cancelChan:
		fmt.Println(ret,ok)
		return true
	default:
		return false
	}
}

func cancel_1(cancelChan chan struct{})  {
	cancelChan<- struct{}{}
}

func cancel_2(cancelChan chan struct{}){
	close(cancelChan)
}

func TestCancel(t *testing.T) {
	cancelChan:=make(chan struct{},0)
	for i:=0;i<5;i++{
		go func(i int,cancelCh chan struct{}) {
			for{
				if isCancelled(cancelCh){
					break
				}
				time.Sleep(time.Millisecond*5)
			}
			fmt.Println(i,"canceled")
		}(i,cancelChan)
	}
	cancel_2(cancelChan)
	time.Sleep(time.Second*1)
}




func isCancelled2(ctx context.Context) bool  {
	select {
	case ret,ok:=<-ctx.Done():
		fmt.Println(ret,ok)
		return true
	default:
		return false
	}
}
func TestCancel2(t *testing.T) {
	ctx,cancel:=context.WithCancel(context.Background())
	for i:=0;i<5;i++{
		go func(i int,ctx context.Context) {
			for{
				if isCancelled2(ctx){
					break
				}
				//time.Sleep(time.Millisecond*5)
			}
			fmt.Println(i,"canceled")
		}(i,ctx)
	}
	cancel()
	time.Sleep(time.Second*1)
}