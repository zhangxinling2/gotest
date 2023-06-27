package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"reflect"
)

type Server struct {
	service map[string]reflectStub
}

func NewServer() *Server {
	return &Server{
		service: make(map[string]reflectStub, 16),
	}
}

func (s *Server) RegisterService(service Service) {
	s.service[service.Name()] = reflectStub{
		service: service,
		value:   reflect.ValueOf(service),
	}
}
func (s *Server) Start(network, addr string) error {
	listener, err := net.Listen(network, addr)
	if err != nil {
		return err
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go func() {
			if err = s.handleConn(conn); err != nil {
				conn.Close()
			}
		}()
	}
}

func (s *Server) handleConn(conn net.Conn) error {
	for {
		reqData, err := ReadMsg(conn)
		if err != nil {
			return err
		}
		req := &Request{}
		//还原请求
		err = json.Unmarshal(reqData, req)
		if err != nil {
			return err
		}
		res, err := s.Invoke(context.Background(), req)
		if err != nil {
			return err
		}
		//写数据
		respData, err := EncodeMsg(res.data)
		if err != nil {
			return err
		}
		_, err = conn.Write(respData)
		if err != nil {
			return err
		}
	}
	return nil
}
func (s *Server) Invoke(ctx context.Context, req *Request) (*Response, error) {
	//找到服务
	ser, ok := s.service[req.ServiceName]
	if !ok {
		return nil, errors.New("没有这个服务")
	}
	resp, err := ser.Invoke(ctx, req.MethodName, req.Args)
	if err != nil {
		return nil, err
	}
	return &Response{data: resp}, nil
}

type reflectStub struct {
	service Service
	//value存储的就是reflect.ValueOf(service)
	value reflect.Value
}

func (r *reflectStub) Invoke(ctx context.Context, methodName string, data []byte) ([]byte, error) {
	//有服务就反射发起调用
	//serVal := reflect.ValueOf(ser)
	//找到方法
	m := r.value.MethodByName(methodName)
	//设置输入hash = {uint32} 1552704771
	in := make([]reflect.Value, 2)
	in[0] = reflect.ValueOf(context.Background())
	inReq := reflect.New(m.Type().In(1).Elem())
	json.Unmarshal(data, inReq.Interface())
	in[1] = inReq
	//执行方法
	res := m.Call(in)
	if res[1].Interface() != nil {
		return nil, res[1].Interface().(error)
	}
	resp, err := json.Marshal(res[0].Interface())
	if err != nil {
		return nil, err
	}
	return resp, nil
}
