package sql_demo

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type JsonColumn[T any] struct {
	Val T
	//NULL 的问题
	Valid bool
}
//非指针
func (j JsonColumn[T]) Value() (driver.Value, error) {
	if !j.Valid{
		return nil,nil
	}
	return json.Marshal(j.Val)
}
//指针
func (j *JsonColumn[T])Scan(src any)error  {
	var bs []byte
	switch data:=src.(type) {
	case string:
		//可以考虑额外处理空字符串
		bs=[]byte(data)
	case []byte:
		bs=data
	case nil:
		return nil
	default:
		return errors.New("不支持类型")
	}
	err:=json.Unmarshal(bs,&j.Val)
	if err!=nil{
		j.Valid=true
	}
	return err
}
