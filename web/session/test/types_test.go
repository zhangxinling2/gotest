package test

import (
	"gotest/web"
	"gotest/web/session"
	"gotest/web/session/memory"
	"net/http"
	"testing"
	"time"
)

func TestSession(t *testing.T){
	//var p Propagator
	//var s Store
	m :=&session.Manager{
		Store:memory.NewStore(15*time.Microsecond),
	}
	//非常简单的登录校验
	server:=web.NewHTTPServer(web.ServerWithMiddleware(func(next web.HandleFunc) web.HandleFunc {
		return func(ctx *web.Context) {
			if ctx.Req.URL.Path=="/login"{
				//放过去,让用户准备登录
				next(ctx)
				return
			}
			_,err:=m.GetSession(ctx)
			if err!=nil{
				ctx.RespStatusCode=http.StatusUnauthorized
				ctx.RespData=[]byte("请重新登录")
				return
			}
			//拿到session说明登陆成功

			//刷新session的过期时间
			_=m.RefreshSession(ctx)
			next(ctx)
		}
	}))
	//登录
	server.Post("/login", func(ctx *web.Context) {
		//要在这之前校验用户名和密码
		sess,err:=m.InitSession(ctx)
		if err!=nil{
			ctx.RespStatusCode=http.StatusInternalServerError
			ctx.RespData=[]byte("登陆异常")
			return
		}
		err=sess.Set(ctx.Req.Context(),"nickname","xiao")
		if err!=nil{
			ctx.RespStatusCode=http.StatusInternalServerError
			ctx.RespData=[]byte("登陆异常")
			return
		}
		ctx.RespData=[]byte("登陆成功")
		ctx.RespStatusCode=http.StatusOK
		return
	})
	server.Post("/loginOut", func(ctx *web.Context){
		err:=m.RemoveSession(ctx)
		if err!=nil{
			ctx.RespStatusCode=http.StatusInternalServerError
			ctx.RespData=[]byte("退出失败")
			return
		}
		ctx.RespStatusCode=http.StatusOK
		ctx.RespData=[]byte("退出登录")
	})
	server.Get("/user", func(ctx *web.Context) {
		sess,_:=m.GetSession(ctx)

		sess.Get(ctx.Req.Context(),"nickname")
	})
	server.Start(":8081")
}