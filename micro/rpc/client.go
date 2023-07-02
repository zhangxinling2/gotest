package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/silenceper/pool"
	"gotest/micro/rpc/message"
	"net"
	"reflect"
	"time"
)

const (
	numOfLengthByte = 8
)

func InitClientProxy(service Service, addr string) error {
	//初始化Client
	client := NewClient(addr)
	//在这里初始化一个Proxy
	return setFuncField(service, client)
}

func setFuncField(service Service, p Proxy) error {
	if service == nil {
		return errors.New("服务是空服务")
	}

	//判断是否是结构体指针
	val := reflect.ValueOf(service)
	typ := val.Type()
	if val.Type().Kind() != reflect.Pointer || val.Elem().Type().Kind() != reflect.Struct {
		return errors.New("只接受一级结构体指针")
	}
	//是结构体指针就可以开始赋值
	val = val.Elem()
	typ = typ.Elem()
	numField := val.NumField()
	//给每一个方法赋值
	for i := 0; i < numField; i++ {
		//得到每个字段的 typ和val
		fieldTyp := typ.Field(i)
		fieldVal := val.Field(i)
		if fieldVal.CanSet() {
			fn := func(args []reflect.Value) (results []reflect.Value) {
				//resp反序列化进ret
				ret := reflect.New(fieldTyp.Type.Out(0).Elem())
				//如何赋值？需要知道三个调用信息，服务名，方法名和参数
				reqData, err := json.Marshal(args[1].Interface())
				if err != nil {
					return []reflect.Value{ret, reflect.ValueOf(err)}
				}
				req := &message.Request{
					//服务名怎么得到？让服务实现Name
					ServiceName: service.Name(),
					MethodName:  fieldTyp.Name,
					//因为我们已经知道第一个参数是ctx,第二个是req，context本身是不会传到服务端的
					Data: reqData,
				}
				//赋完了值，就该发起调用了
				//var p Proxy
				resp, err := p.Invoke(args[0].Interface().(context.Context), req)

				err = json.Unmarshal(resp.Data, ret.Interface())
				if err != nil {
					return []reflect.Value{ret, reflect.ValueOf(err)}
				}

				return []reflect.Value{ret, reflect.Zero(reflect.TypeOf(new(error)).Elem())}
			}
			//创建方法，第一个type 自然就是字段的type 把func提取出去
			fnVal := reflect.MakeFunc(fieldTyp.Type, fn)
			//替换原方法
			fieldVal.Set(fnVal)
		}
	}

	return nil
}

type Client struct {
	pool pool.Pool
}

func (c *Client) Invoke(ctx context.Context, req *message.Request) (*message.Response, error) {
	//发送请求到服务器
	//新建一个连接来发送请求
	//直接把net中的send拷过来使用
	//编码发送请求
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	res, err := c.Send(data)
	if err != nil {
		return nil, err
	}
	return &message.Response{Data: res}, nil
}
func (c *Client) Send(data []byte) ([]byte, error) {
	val, err := c.pool.Get()
	if err != nil {
		return nil, err
	}
	conn := val.(net.Conn)
	if err != nil {
		return nil, err
	}
	defer func() {
		conn.Close()
	}()
	reqData, err := EncodeMsg(data)
	if err != nil {
		return nil, err
	}
	_, err = conn.Write(reqData)
	if err != nil {
		return nil, err
	}
	resData, err := ReadMsg(conn)
	if err != nil {
		return nil, err
	}
	return resData, nil
}
func NewClient(addr string) *Client {
	config := &pool.Config{
		InitialCap:  1,
		MaxCap:      30,
		MaxIdle:     10,
		IdleTimeout: time.Minute,
		Factory: func() (interface{}, error) {
			return net.DialTimeout("tcp", addr, time.Second*3)
		},
		Close: func(i interface{}) error {
			return i.(net.Conn).Close()
		},
	}
	p, err := pool.NewChannelPool(config)
	if err != nil {
		return nil
	}
	return &Client{
		pool: p,
	}
}
