package service

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"
)

// 典型的 Option 设计模式
type Option func(*App)

// ShutdownCallback 采用 context.Context 来控制超时，而不是用 time.After 是因为
// - 超时本质上是使用这个回调的人控制的
// - 我们还希望用户知道，他的回调必须要在一定时间内处理完毕，而且他必须显式处理超时错误
type ShutdownCallback func(ctx context.Context)

// 你需要实现这个方法
func WithShutdownCallbacks(cbs ...ShutdownCallback) Option {
	return func(app *App) {
		app.cbs = cbs
	}
}
func AppWithShutdownTimeout(shutdownTimeout time.Duration) Option {
	return func(app *App) {
		app.shutdownTimeout = shutdownTimeout
	}
}
func AppWithWaitTimeout(waitTimeout time.Duration) Option {
	return func(app *App) {
		app.waitTime = waitTimeout
	}
}
func AppWithCBTimeout(cbTimeout time.Duration) Option {
	return func(app *App) {
		app.cbTimeout = cbTimeout
	}
}

// 这里我已经预先定义好了各种可配置字段
type App struct {
	servers []*Server

	// 优雅退出整个超时时间，默认30秒
	shutdownTimeout time.Duration

	// 优雅退出时候等待处理已有请求时间，默认10秒钟
	waitTime time.Duration
	// 自定义回调超时时间，默认三秒钟
	cbTimeout time.Duration

	cbs []ShutdownCallback
}

// NewApp 创建 App 实例，注意设置默认值，同时使用这些选项
func NewApp(servers []*Server, opts ...Option) *App {
	app := &App{
		servers:         servers,
		shutdownTimeout: time.Second * 30,
		waitTime:        time.Second * 10,
		cbTimeout:       time.Second * 3,
		cbs:             nil,
	}
	for _, opt := range opts {
		opt(app)
	}
	return app
}

// StartAndServe 你主要要实现这个方法
func (app *App) StartAndServe() {
	for _, s := range app.servers {
		srv := s
		go func() {
			if err := srv.Start(); err != nil {
				if err == http.ErrServerClosed {
					log.Printf("服务器%s已关闭", srv.name)
				} else {
					log.Printf("服务器%s异常退出", srv.name)
				}
			}
		}()
	}
	// 从这里开始优雅退出监听系统信号，强制退出以及超时强制退出。
	// 优雅退出的具体步骤在 shutdown 里面实现
	// 所以你需要在这里恰当的位置，调用 shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
	go func() {
		select {
		case <-sig:
			log.Println("再次接收信号，强制退出")
			os.Exit(1)
		case <-time.After(app.shutdownTimeout):
			log.Println("超时，强制退出")
			os.Exit(1)
		}
	}()
	app.shutdown()
}

// shutdown 你要设计这里面的执行步骤。
//http.Server#Shutdown的逻辑是：先关闭所有的 listeners， 然后等待所有的连接（connection）都处理完，再关闭服务。如果传入的 context 超时，则会直接返回。
//而这个shutdown就实现设计中的五个步骤
func (app *App) shutdown() {
	log.Println("开始关闭应用，停止接收新请求")
	// 你需要在这里让所有的 server 拒绝新请求
	//在serverMux中设置了reject标记位，所以直接设置每个server的标记位即可
	for _, ser := range app.servers {
		ser.rejectReq()
	}

	log.Println("等待正在执行请求完结")
	// 在这里等待一段时间
	//设置ctx来控制关闭服务超时时间
	serverCtx, cancel1 := context.WithTimeout(context.Background(), app.waitTime)
	defer cancel1()
	//使用 waitGroup来等待所有的goroutine关闭服务，或者超时才能继续向下执行
	var wg sync.WaitGroup
	for _, ser := range app.servers {
		wg.Add(1)
		go func(ser *Server, ctx context.Context) {
			//select {
			//case <-ctx.Done():
			//	log.Println("关闭服务超时")
			//	wg.Done()
			//	return
			//default:
			//	if err := ser.stop(ctx); err != nil {
			//		log.Println("关闭服务失败")
			//	}
			//	wg.Done()
			//}
			//使用了http包中的shutdown就直接使用ERROR,就不需要上面这个select
			if err := ser.stop(ctx); err != nil {
				log.Println("关闭服务失败,error:", err)
			}
			wg.Done()
		}(ser, serverCtx)
	}
	wg.Wait()

	log.Println("开始执行自定义回调")
	// 并发执行回调，要注意协调所有的回调都执行完才会步入下一个阶段
	//设置ctx来控制回调超时时间
	cbsCtx, cancel2 := context.WithTimeout(context.Background(), app.cbTimeout)
	defer cancel2()
	wg.Add(len(app.cbs))
	for _, cb := range app.cbs {
		callback := cb
		go func(ctx context.Context) {
			select {
			case <-ctx.Done():
				log.Println("执行回调超时")
				wg.Done()
				return
			default:
				callback(ctx)
				wg.Done()
			}
		}(cbsCtx)
	}
	wg.Wait()

	// 释放资源
	log.Println("开始释放资源")
	// 这一个步骤不需要你干什么，这是假装我们整个应用自己要释放一些资源
	app.close()
}

func (app *App) close() {
	// 在这里释放掉一些可能的资源
	time.Sleep(time.Second)
	log.Println("应用关闭")
}

// Server 本身可以是很多种 Server，例如 http server
// 或者 RPC server
// 理论上来说，如果你设计一个脚手架的框架，那么 Server 应该是一个接口
type Server struct {
	srv  *http.Server
	name string
	mux  *serverMux
}

// serverMux 既可以看做是装饰器模式，也可以看做委托模式
type serverMux struct {
	reject bool
	*http.ServeMux
}

func (s *serverMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 只是在考虑到 CPU 高速缓存的时候，会存在短时间的不一致性
	if s.reject {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("服务已关闭"))
		return
	}
	s.ServeMux.ServeHTTP(w, r)
}

func NewServer(name string, addr string) *Server {
	mux := &serverMux{ServeMux: http.NewServeMux()}
	return &Server{
		name: name,
		mux:  mux,
		srv: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
	}
}

func (s *Server) Handle(pattern string, handler http.Handler) {
	s.mux.Handle(pattern, handler)
}

func (s *Server) Start() error {
	return s.srv.ListenAndServe()
}

func (s *Server) rejectReq() {
	s.mux.reject = true
}

//stop
func (s *Server) stop(ctx context.Context) error {
	//这就是HTTP包中的服务优雅退出，不过我们实现在app下的shutdown中
	return s.srv.Shutdown(ctx)
}
