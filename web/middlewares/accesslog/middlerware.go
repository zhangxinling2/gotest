package accesslog

import (
	"encoding/json"
	"gotest/web"
)

type MiddlewareBuilder struct {
	logFunc func(log string)
}

func (m *MiddlewareBuilder)LogFunc(fn func(log string))*MiddlewareBuilder{
	m.logFunc=fn
	return m
}
func (m MiddlewareBuilder) Build() web.Middleware {
	return func(next web.HandleFunc) web.HandleFunc {
		return func(ctx *web.Context) {
			//要记录这个请求
			//一般使用defer 因为next可能会panic掉
			defer func() {
				l:=accesslog{
					Host: ctx.Req.Host,
					Route:ctx.MatchRoute,
					HTTPMethod: ctx.Req.Method,
					Path: ctx.Req.URL.Path,
				}
				data,_:=json.Marshal(l)
				m.logFunc(string(data))
			}()
			next(ctx)
		}
	}
}

type accesslog struct {
	Host       string `json:"host,omitempty"`
	//代表命中的路由
	Route      string `json:"route,omitempty"`
	HTTPMethod string `json:"http_method,omitempty"`
	Path       string `json:"path,omitempty"`
}
