package reflect

import (
	"errors"
	"reflect"
)

// IterateFields 遍历字段
// 注释里面说明，这里只能接受XXX之类的数据
func IterateFields(entity any) (map[string]any, error) {
	if entity == nil {
		return nil, errors.New("不支持 nil")
	}
	typ := reflect.TypeOf(entity)
	val := reflect.ValueOf(entity)

	if val.IsZero() {
		return nil, errors.New("不支持零值")
	}
	//for为了多重指针
	for typ.Kind() == reflect.Pointer {
		//如果typ是个指针，那么Elem则是指针指向的，如果是Array,Chain,Slice则是里面的内容
		typ = typ.Elem()
		val = val.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil, errors.New("不支持类型")
	}

	numField := typ.NumField()
	res := make(map[string]any, numField)
	for i := 0; i < numField; i++ {
		// 字段类型
		fieldType := typ.Field(i)
		// 字段值
		fieldVal := val.Field(i)
		if fieldType.IsExported() {
			res[fieldType.Name] = fieldVal.Interface()
		} else {
			res[fieldType.Name] = reflect.Zero(fieldType.Type).Interface()
		}
	}
	return res, nil
}

func SetField(entity any, field string, newVal any) error {
	val := reflect.ValueOf(entity)
	for val.Type().Kind() == reflect.Pointer {
		val = val.Elem()
	}

	fieldVal := val.FieldByName(field)
	if !fieldVal.CanSet() {
		return errors.New("不可修改字段")
	}
	fieldVal.Set(reflect.ValueOf(newVal))
	return nil
}
