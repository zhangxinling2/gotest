package rpc

import (
	"context"
	"gotest/micro/rpc/message"
)

type Service interface {
	Name() string //要求服务要实现服务名
}
type Proxy interface {
	//跟在Service中的字段结构相同
	Invoke(ctx context.Context, req *message.Request) (*message.Response, error)
}
