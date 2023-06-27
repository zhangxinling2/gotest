package rpc

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"net"
	"reflect"
)

type Server struct {
	service map[string]Service
}

func NewServer() *Server {
	return &Server{
		service: make(map[string]Service, 16),
	}
}
func (s *Server) RegisterService(service Service) {
	s.service[service.Name()] = service
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
		data, err := ReadMsg(conn)
		if err != nil {
			return err
		}
		res, err := s.handleMsg(data)
		if err != nil {
			return err
		}
		//写数据
		respData := make([]byte, len(res)+numOfLengthByte)
		binary.BigEndian.PutUint64(respData[:numOfLengthByte], uint64(len(res)))
		copy(respData[numOfLengthByte:], res)
		_, err = conn.Write(respData)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) handleMsg(reqData []byte) ([]byte, error) {
	req := &Request{}
	//还原请求
	err := json.Unmarshal(reqData, req)
	if err != nil {
		return nil, err
	}
	//找到服务
	ser, ok := s.service[req.ServiceName]
	if !ok {
		return nil, errors.New("没有这个服务")
	}
	//有服务就反射发起调用
	serVal := reflect.ValueOf(ser)
	//找到方法
	m := serVal.MethodByName(req.MethodName)
	//设置输入hash = {uint32} 1552704771
	in := make([]reflect.Value, 2)
	in[0] = reflect.ValueOf(context.Background())
	inReq := reflect.New(m.Type().In(1).Elem())
	json.Unmarshal(req.Args, inReq.Interface())
	in[1] = inReq
	//执行方法
	res := m.Call(in)
	if res[1].Interface() != nil {
		return nil, res[1].Interface().(error)
	}
	resp, err := json.Marshal(res[0].Interface())
	return resp, err
}
