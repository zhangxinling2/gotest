//go:build e2e
package web

import (
	"fmt"
	"net/http"
	"testing"
)

func TestServer(t *testing.T) {
	h := NewHTTPServer()
	//用户自己来集成业务逻辑
	//h.addRoute(http.MethodGet,"/user", func(ctx *Context) {
	//	fmt.Println("处理第一件事")
	//	fmt.Println("处理第二件事")
	//})
	handler1:=func (ctx *Context){
		fmt.Println("处理第一件事")
	}
	handler2:=func (ctx *Context){
		fmt.Println("处理第二件事")
	}
	h.addRoute(http.MethodGet,"/user", func(ctx *Context) {
		handler1(ctx)
		handler2(ctx)
	})

	//h.Get("/user", func(ctx *Context) {
	//
	//})

	h.addRoute(http.MethodGet,"/order/detail", func(ctx *Context) {
		ctx.Resp.Write([]byte("hello,detail"))
	})
	h.Post("/form", func(ctx *Context) {
		ctx.Req.ParseForm()
		ctx.Resp.Write([]byte(fmt.Sprintf("sss%d",1)))
	})
	//用法1
	//这个Hanler 就是我们跟http包的连接点。
	//http.ListenAndServe(":8081",h)

	//用法二 自己手动管
	h.Start(":8081")


}
