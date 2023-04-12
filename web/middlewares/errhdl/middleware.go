package errhdl

import "gotest/web"

type MiddlewareBuild struct {
	//这种设计只能返回固定的值，不能做到动态渲染
	resp map[int][]byte
}

func NewMiddlewareBuild()*MiddlewareBuild{
	return &MiddlewareBuild{
		resp: map[int][]byte{},
	}
}
func (m *MiddlewareBuild)AddCode(status int,data []byte)*MiddlewareBuild{
	m.resp[status]=data
	return m
}
func (m MiddlewareBuild)Builder()web.Middleware  {
	return func(next web.HandleFunc) web.HandleFunc {
		return func(ctx *web.Context) {
			next(ctx)
			resp,ok:=m.resp[ctx.RespStatusCode]
			if ok{
				//篡改结果
				ctx.RespData=resp
			}
		}
	}
}
