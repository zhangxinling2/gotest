package valuer

import (
	"database/sql"
	"gotest/orm/internal/errs"
	"gotest/orm/model"
	"reflect"
	"unsafe"
)

type unsafeValue struct {
	model *model.Model

	//对应于T的指针
	//val any
	//基准地址
	address unsafe.Pointer
}
func NewUnsafeValue(model *model.Model,val any)Value{
	//起始地址
	address:=reflect.ValueOf(val).UnsafePointer()
	return unsafeValue{
		model: model,
		//val: val,
		address: address,
	}
}

var _ Creator=NewUnsafeValue

func(u unsafeValue)Field(name string)(any,error){
	fd,ok:=u.model.FieldMap[name]
	if !ok{
		return nil,errs.NewUnknownField(name)
	}
	//要计算字段的地址
	fdAddress:=unsafe.Pointer(uintptr(u.address)+fd.Offset)
	val:=reflect.NewAt(fd.Type,fdAddress)
	return val.Elem().Interface(),nil
}
func (u unsafeValue) SetColumns(rows *sql.Rows) error {
	cs,err:=rows.Columns()
	if err!=nil{
		return  err
	}

	var vals []any


	for _,c:=range cs{
		//c是列名
		fd,ok:=u.model.ColumnMap[c]
		if !ok{
			return errs.NewUnknownColumn(c)
		}

		//要计算字段的地址
		fdAddress:=unsafe.Pointer(uintptr(u.address)+fd.Offset)
		val:=reflect.NewAt(fd.Type,fdAddress)
		vals=append(vals, val.Interface())
	}
	err=rows.Scan(vals...)
	return nil
}
