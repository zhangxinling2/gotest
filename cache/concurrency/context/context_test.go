package context

import (
	"context"
	"testing"
	"time"
)

//一般都是这样定义key,官方推荐
type mykey struct{}

//衍生类型也可以
type mykey2 int

func TestContext(t *testing.T) {
	//一般是链路起点，或者调用的起点
	ctx := context.Background()
	//在不确定context该用什么的时候，用TODO() 用的较少
	//ctx:=context.TODO()

	ctx = context.WithValue(ctx, mykey{}, "my-value")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	val := ctx.Value(mykey{}).(string)
	t.Log(val)
	//不使用ok直接使用会panic掉，拿到的是nil，除非非常确定，不然不要使用连调
	newVal := ctx.Value("不存在的key")
	val, ok := newVal.(string)
	if !ok {
		t.Log("类型不对")
		return
	}
	t.Log(val)
}

func TestContext_WithCancel(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	//defer cancel()
	go func() {
		time.Sleep(time.Second)
		cancel()
	}()
	//用 ctx,如果不调用cancel,这里会阻塞到死
	<-ctx.Done()
	t.Log("hello,cancel:", ctx.Err())
}

func TestContext_WithDeadline(t *testing.T) {
	ctx := context.Background()
	//当前时间戳加上三秒
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(time.Second*3))
	//第二个值是bool，表示deadline有没有设置
	deadline, _ := ctx.Deadline()
	t.Log("deadline:", deadline)
	defer cancel()
	//不需要另开一个goroutine来cancel，因为有超时控制
	<-ctx.Done()
	t.Log("hello,deadline:", ctx.Err())
}

func TestContext_WithTimeOut(t *testing.T) {
	ctx := context.Background()
	//WithTimeout其实就是调用了WithDeadline
	ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	deadline, _ := ctx.Deadline()
	t.Log("deadline:", deadline)
	defer cancel()
	//不需要另开一个goroutine来cancel，因为有超时控制
	<-ctx.Done()
	t.Log("hello,timeout:", ctx.Err())
}

func TestContext_Parent(t *testing.T) {
	ctx := context.Background()
	parent := context.WithValue(ctx, "my-key", "my-value")
	child := context.WithValue(ctx, "my-key", "my new value")

	t.Log("parent my-key:", parent.Value("my-key"))
	//子ctx设置相同key，那么会覆盖父ctx
	t.Log("child my-key:", child.Value("my-key"))

	child2, cancel := context.WithTimeout(parent, time.Second)
	defer cancel()
	t.Log("child2 my-key", child2.Value("my-key"))

	child3 := context.WithValue(parent, "new-key", "child3 value")
	//父ctx拿不到子ctx新设置的key
	t.Log("parent new-key:", parent.Value("new-key"))
	t.Log("child new-key:", child3.Value("new-key"))

	parent1 := context.WithValue(ctx, "map", map[string]string{})
	child4, cancel := context.WithTimeout(parent1, time.Second)
	defer cancel()
	m := child4.Value("map").(map[string]string)
	m["key1"] = "value1"
	nm := parent1.Value("map").(map[string]string)
	t.Log("parent1:key1", nm["key1"])
}
