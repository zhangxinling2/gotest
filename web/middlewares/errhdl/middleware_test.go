package errhdl

import (
	"gotest/web"
	"net/http"
	"testing"
)

func TestNewMiddlewareBuild(t *testing.T) {
	builder:=NewMiddlewareBuild()
	builder.AddCode(http.StatusNotFound,[]byte(`
<html>
	<body>
		<h1>哈哈哈走失了</h1>
	</body>
</html>
`)).AddCode(http.StatusBadRequest,[]byte(`
<html>
	<body>
		<h1>请求不对</h1>
	</body>
</html>
`))
	server:=web.NewHTTPServer(web.ServerWithMiddleware())
	server.Start(":8081")
}

