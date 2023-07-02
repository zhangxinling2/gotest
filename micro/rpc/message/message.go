package message

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
