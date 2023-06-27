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
	HeadLength  uint32
	BodyLength  uint32
	RequestID   uint32
	Version     uint8
	Compresser  uint8
	Serializer  uint8
	ServiceName string
	MethodName  string
	Meta        map[string]string
	Data        []byte
}

func (r *Request) Name() string {
	//TODO implement me
	panic("implement me")
}

type Response struct {
	HeadLength uint32
	BodyLength uint32
	RequestID  uint32
	Version    uint8
	Compresser uint8
	Serializer uint8
	Error      []byte
	Data       []byte //存储接收到的响应
}
