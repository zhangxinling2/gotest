package valuer

import (
	"database/sql"
	"gotest/orm/internal/errs"
	"gotest/orm/model"
	"reflect"
)

type reflectValue struct {
	model *model.Model

	//对应于T的指针
	//val any
	val reflect.Value
}

func NewReflectValue(model *model.Model, val any) Value {
	return reflectValue{
		model: model,
		val:   reflect.ValueOf(val).Elem(),
	}
}

func (r reflectValue) Field(name string) (any, error) {
	//检测 name 是否合法
	//_,ok:=r.val.Type().FieldByName(name)
	//if !ok{
	//	//报错
	//}
	val := r.val.FieldByName(name).Interface()
	//if val==(reflect.Value{}){
	//
	//}
	return val, nil
}

var _ Creator = NewReflectValue

func (r reflectValue) SetColumns(rows *sql.Rows) error {
	cs, err := rows.Columns()
	if err != nil {
		return err
	}
	vals := make([]any, 0, len(cs))

	valElem := make([]reflect.Value, 0, len(cs))
	for _, c := range cs {
		//c是列名
		fd, ok := r.model.ColumnMap[c]
		if !ok {
			return errs.NewUnknownColumn(c)
		}
		//反射创建新的实例
		//这里创建的实例是原本类型的指针
		//例如 fd.type=int 那么val是*int
		val := reflect.New(fd.Type)
		//这样scan就不用取地址了
		vals = append(vals, val.Interface())

		valElem = append(valElem, val.Elem())

		//for _,fd:=range s.model.fieldMap{
		//	if fd.colName==c{
		//		//反射创建新的实例
		//		//这里创建的实例是原本类型的指针
		//		//例如 fd.type=int 那么val是*int
		//		val:=reflect.New(fd.typ)
		//		//这样scan就不用取地址了
		//		vals=append(vals, val.Interface())
		//	}
		//}
	}
	err = rows.Scan(vals...)
	if err != nil {
		return err
	}
	tpValue := r.val
	for i, c := range cs {
		fd, ok := r.model.ColumnMap[c]
		if !ok {
			return errs.NewUnknownColumn(c)
		}
		tpValue.FieldByName(fd.GoName).Set(valElem[i])
		//for _,fd:=range s.model.fieldMap{
		//	if fd.colName==c{
		//		tpValue.Elem().FieldByName(fd.goName).Set(reflect.ValueOf(vals[i]).Elem())
		//	}
		//}

	}

	return nil
}
