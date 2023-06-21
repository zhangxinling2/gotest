package rpc

import (
	"context"
	"errors"
	"fmt"
	"reflect"
)

func InitClientProxy(service Service) error {
	//在这里初始化一个Proxy
	return setFuncField(service, nil)
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
				//如何赋值？需要知道三个调用信息，服务名，方法名和参数
				req := &Request{
					//服务名怎么得到？让服务实现Name
					ServiceName: service.Name(),
					MethodName:  fieldTyp.Name,
					//因为我们已经知道第一个参数是ctx,第二个是req，context本身是不会传到服务端的
					Args: []any{args[1].Interface()},
				}
				//赋完了值，就该发起调用了
				//var p Proxy
				resp, err := p.Invoke(args[0].Interface().(context.Context), req)
				if err != nil {
					return []reflect.Value{reflect.Zero(fieldTyp.Type.Out(0)), reflect.ValueOf(err)}
				}
				fmt.Println(resp)
				return []reflect.Value{reflect.Zero(fieldTyp.Type.Out(0)), reflect.ValueOf((*error)(nil)).Elem()}
			}
			//创建方法，第一个type 自然就是字段的type 把func提取出去
			fnVal := reflect.MakeFunc(fieldTyp.Type, fn)
			//替换原方法
			fieldVal.Set(fnVal)
		}
	}

	return nil
}
