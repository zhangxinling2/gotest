package accesslog

import (
	"fmt"
	"gotest/web"
	"net/http"
	"testing"
)

func TestMiddlewareBuilder(t *testing.T) {
	builder :=MiddlewareBuilder{}
	mdl:=builder.LogFunc(func(log string) {
		fmt.Println(log)
	}).Build()
	server:=web.NewHTTPServer(web.ServerWithMiddleware(mdl))
	server.Post("/a/b/*", func(ctx *web.Context) {
		fmt.Println("hello,it's me")
	})
	req,_:=http.NewRequest(http.MethodPost,"/a/b/c",nil)
	server.ServeHTTP(nil,req)
}
