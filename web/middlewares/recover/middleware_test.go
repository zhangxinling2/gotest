package recover

import (
	"fmt"
	"gotest/web"
	"testing"
)

func TestMiddlewareBuilder_Build(t *testing.T) {
	builder := MiddlewareBuilder{
		StatusCode: 500,
		Data: []byte("你 panic 了"),
		Log: func(ctx *web.Context) {
			fmt.Printf("panic 路径: %s", ctx.Req.URL.String())
		},
	}

	server := web.NewHTTPServer(web.ServerWithMiddleware(builder.Build()))
	server.Get("/user", func(ctx *web.Context) {
		panic("发生panic 了")
	})
	server.Start(":8081")
}
