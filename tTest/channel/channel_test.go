package channel

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func Service() string{
	time.Sleep(50*time.Millisecond)
	return "Done"
}

func otherTask(){
	fmt.Println("working on other task")
	time.Sleep(time.Millisecond*100)
	fmt.Println("other task is done")
}
func AsyncService() chan string{
	retCh := make(chan string,1)
	go func() {
		ret := Service()
		fmt.Println("return result")
		retCh <- ret //不加缓冲区会阻塞在这知道被取出
		fmt.Println("service exited")
	}()
	return retCh
}
func TestService(t *testing.T) {
	fmt.Println(Service())
	otherTask()
}
func TestAsyncService(t *testing.T){
	retCh := AsyncService()
	otherTask()
	fmt.Println(<-retCh)
}






func dateProducer(ch chan int,wg *sync.WaitGroup){
	go func() {
		for i:=0;i<10;i++{
			ch<-i
		}
		close(ch)//向关闭的channel发送数据会产生panic
		wg.Done()
	}()
}

func dataReceiver(ch chan int,wg *sync.WaitGroup){
	go func() {
		for i:=0;i<10;i++{
			if data,ok:=<-ch;ok{
				fmt.Println(data)
			}else{
				break
			}
		}
		wg.Done()
	}()
}

func TestCloseChannel(t *testing.T) {
	var wg sync.WaitGroup
	ch:=make(chan int)
	wg.Add(1)
	dateProducer(ch,&wg)
	wg.Add(1)
	dataReceiver(ch,&wg)

	wg.Wait()
}