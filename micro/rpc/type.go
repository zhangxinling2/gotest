package rpc

import "context"

type Service interface {
	Name() string //要求服务要实现服务名
}
type Proxy interface {
	//跟在Service中的字段结构相同
	Invoke(ctx context.Context, req *Request) (*Response, error)
}

type Request struct {
	ServiceName string
	MethodName  string
	Args        []any
}

func (r *Request) Name() string {
	//TODO implement me
	panic("implement me")
}

type Response struct {
}
