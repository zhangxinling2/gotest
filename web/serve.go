package web

import (
	"fmt"
	"net"
	"net/http"
)

type HandleFunc func(ctx *Context)

//确保某个结构体一定实现了某个接口
var _ Server = &HTTPServer{}

type Server interface {
	http.Handler
	Start(addr string) error
	//Star1() error

	//AddRoute 增加路由注册功能
	//method 是HTTP方法
	//path 是路由
	//handlerFunc是你的业务逻辑
//	addRoute(method string,path string,handleFunc HandleFunc)
	//不采取这种
	//AddRoute(method string,path string,handleFunc... HandleFunc)

}


//type HTTPSServer struct {
//	HTTPServer
//}
type HTTPServerOption func(server *HTTPServer)
type HTTPServer struct {
	//addr string 创建的时候传递
	router

	mdls []Middleware
	log func(msg string,args ...any)

	tplEngine TemplateEngine
}

//缺乏扩展性
//func NewHTTPServer(mdls ...Middleware) *HTTPServer{
//	return &HTTPServer{
//		router:newRouter(),
//		mdls: mdls,
//	}
//}

func NewHTTPServer(opts ...HTTPServerOption) *HTTPServer{
	res:= &HTTPServer{
		router:newRouter(),
		log: func(msg string, args ...any) {
			fmt.Printf(msg,args...)
		},
	}
	for _,opt:=range opts{
		opt(res)
	}
	return res
}

func ServerWithTemplateEngine(tplEngine TemplateEngine) HTTPServerOption{
	return func(server *HTTPServer) {
		server.tplEngine=tplEngine
	}
}


func ServerWithMiddleware(mdls ...Middleware) HTTPServerOption{
	return func(server *HTTPServer) {
		server.mdls=mdls
	}
}

//ServeHTTP 处理请求的入口
func (h *HTTPServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	//框架代码就在这
	//首先构建ctx
	ctx:=&Context{
		Req: request,
		Resp: writer,
		tplEngine: h.tplEngine,
	}

	//最后一个是这个
	root :=h.serve

	//然后这里就是利用最后一个不断往前回溯组装链条
	//从后往前
	//把后一个作为前一个的next，构造好链条
	for i:=len(h.mdls)-1;i>=0;i--{
		root =h.mdls[i](root)
	}
	//接下来就是查找路由，并执行命中的业务逻辑
	//这里执行的时候就是从前往后了

	//这里，最后的一个步骤就是把respData和respStatusCode刷新到响应里
	var m Middleware= func(next HandleFunc) HandleFunc {
		return func(ctx *Context) {
			//就设置好了respData和respStatusCode
			next(ctx)

		}
	}
	root=m(root)

	root(ctx)
}
func (h *HTTPServer)flashResp(ctx *Context)  {
	if ctx.RespStatusCode!=0{
		ctx.Resp.WriteHeader(ctx.RespStatusCode)
	}
	n,err:=ctx.Resp.Write(ctx.RespData)
	if err!=nil||n!=len(ctx.RespData){
		h.log("写入响应失败 %v",err)
	}
}

func (h *HTTPServer) serve(ctx *Context){
	info,ok:=h.findRoute(ctx.Req.Method,ctx.Req.URL.Path)
	if !ok|| info.n.handler==nil{
		//路由没有命中 404
		ctx.RespStatusCode=404
		ctx.RespData=[]byte("NOT FOUND")
		//ctx.Resp.WriteHeader(404)
		//ctx.Resp.Write([]byte("NOT FOUND"))
		return
	}
	fmt.Println("server 执行")
	ctx.PathParams= info.pathParams
	ctx.MatchRoute=info.n.route
	info.n.handler(ctx)
}

//func (h *HTTPServer) AddRoute(method string,path string,handleFunc HandleFunc){
//
//}

func (h *HTTPServer) Get(path string, handleFunc HandleFunc) {
	h.addRoute(http.MethodGet, path, handleFunc)
}

func (h *HTTPServer) Post(path string, handleFunc HandleFunc) {
	h.addRoute(http.MethodPost, path, handleFunc)
}

func (h *HTTPServer) Options(path string, handleFunc HandleFunc) {
	h.addRoute(http.MethodOptions, path, handleFunc)
}

func (h *HTTPServer) Start(addr string) error {
	l,err:=net.Listen("tcp",addr)
	if err!=nil{
		return err
	}
	//在这可以让用户注册所谓的 after start 回调
	//在这执行一些业务所需的前置条件
	return http.Serve(l,h)
	//也可以自己创建Server
	//http.Server{}
}

