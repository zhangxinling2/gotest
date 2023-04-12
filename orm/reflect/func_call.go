package reflect

import "reflect"

func IterateFunc(entity any)(map[string]FunInfo,error){
	//先拿到它的类型信息
	typ:=reflect.TypeOf(entity)
	//拿到方法数量
	numMethod:=typ.NumMethod()
	res:= make(map[string]FunInfo,numMethod)
	//遍历方法
	for i := 0; i < numMethod; i++ {
		//方法是typ.Method(i).Func
		method:=typ.Method(i)
		fn:=method.Func
		//得到它有多少个参数
		numIn:=fn.Type().NumIn()
		input:=make([]reflect.Type,0,numIn)
		inputValue:=make([]reflect.Value,0,numIn)
		input=append(input,reflect.TypeOf(entity))
		inputValue=append(inputValue,reflect.ValueOf(entity))
		for j := 1; j< numIn; j++ {
			fnInType:=fn.Type().In(j)

			//实际上第0个指向的是user
			input=append(input,fnInType)
			//测试用，参数都用0值
			inputValue=append(inputValue,reflect.Zero(fnInType))
		}
		//得到输出
		numOut:=fn.Type().NumOut()
		output:=make([]reflect.Type,0,numOut)
		for j := 0; j< numOut; j++ {
			output=append(output,fn.Type().Out(j))
		}
		//发起调用
		resValue:=fn.Call(inputValue)
		//把结果从[]Value变成我们熟悉的
		result:=make([]any,0,len(resValue))
		for _,v:=range resValue{
			result=append(result,v.Interface())
		}
		res[method.Name]=FunInfo{
			Name: method.Name,
			InputTypes: input,
			OutputTypes: output,
			Result: result,
		}
	}
	return res, nil
}

type FunInfo struct {
	Name string
	InputTypes []reflect.Type
	OutputTypes []reflect.Type
	Result any
}
