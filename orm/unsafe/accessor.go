package unsafe

import (
	"errors"
	"reflect"
	"unsafe"
)

type UnsafeAccessor struct {
	fields map[string]fieldMeta
	address unsafe.Pointer
}

//NewUnsafeAccessor entity是结构体指针
func NewUnsafeAccessor(entity any)*UnsafeAccessor{
	typ:=reflect.TypeOf(entity)
	typ=typ.Elem()
	numField:=typ.NumField()
	fields:=make(map[string]fieldMeta,numField)
	for i := 0; i < numField; i++ {
		fd:=typ.Field(i)
		fields[fd.Name]=fieldMeta{
			Offset: fd.Offset,
			typ: fd.Type,
		}
	}

	val:=reflect.ValueOf(entity)
	return &UnsafeAccessor{
		fields: fields,
		//不直接用UnsafeAddr，因为它对应的地址不是稳定的，Gc之后地址会变化
		//UnsafePointer会帮助维持指针
		address:val.UnsafePointer(),
	}
}
//读一个
func(a *UnsafeAccessor)Field(field string)(any,error){
	//起始地址+字段偏移量
	fd,ok:=a.fields[field]
	if !ok{
		return nil, errors.New("非法字段")
	}
	//字段起始地址
	fdAddress:=unsafe.Pointer(uintptr(a.address)+fd.Offset)
	//读取任意类型
	return reflect.NewAt(fd.typ,fdAddress).Elem().Interface(),nil
	//如果知道类型
	//return *(*int)(fdAddress),nil


}


func(a *UnsafeAccessor)SetField(field string,val any)error{
	//起始地址+字段偏移量
	fd,ok:=a.fields[field]
	if !ok{
		return errors.New("非法字段")
	}
	//字段起始地址
	fdAddress:=unsafe.Pointer(uintptr(a.address)+fd.Offset)
	//不知道类型
	reflect.NewAt(fd.typ,fdAddress).Elem().Set(reflect.ValueOf(val))

	//知道类型
	//*(*int)(fdAddress)=val.(int)
	return nil
}
type fieldMeta struct {

	Offset uintptr
	typ reflect.Type
}