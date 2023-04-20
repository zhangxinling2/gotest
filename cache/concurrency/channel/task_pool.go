package channel

import "context"

type Task func()

type TaskPool struct {
	tasks chan Task
	//close *atomic.Bool
	//一般用这个
	close chan struct{}

	//closeOnce sync.Once
}

// NewTaskPool nunG就是goroutine的数量，capacity是缓存的容量
func NewTaskPool(numG int, capacity int) {
	res := &TaskPool{
		tasks: make(chan Task, capacity),
		close: make(chan struct{}),
	}

	for i := 0; i < numG; i++ {
		go func() {
			for {
				select {
				case <-res.close:
					return
				case t := <-res.tasks:
					t()
				}
			}
			//for t:=range res.tasks{
			//	if res.close.Load(){
			//		return
			//	}
			//	t()
			//}
		}()
	}
}

//Submit 提交任务 task满了会被阻塞
func (p *TaskPool) Submit(ctx context.Context, t Task) error {
	select {
	case p.tasks <- t:
	case <-ctx.Done():
		//让用户自己判断是超时还是取消
		return ctx.Err()
	}
	return nil
}

// Close 开了goroutine，channel一定要设置一个Close方法迎来控制
func (p *TaskPool) Close() error {
	//p.close.Store(true)
	//这种写法是不行的
	//p.close<-struct{}{}
	//直接关闭channel，这种实现又有一种缺陷，重复调用close会panic
	close(p.close)
	//不建议，不需要考虑这么周全，可以在方法注释中直接告诉用户不要重复调用
	//p.closeOnce.Do(func() {
	//	close(p.close)
	//})
	return nil
}
