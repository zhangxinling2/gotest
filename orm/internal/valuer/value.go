package valuer

import (
	"database/sql"
	"gotest/orm/model"
)

type Value interface {
	//Field 使用unsafe读取
	Field(name string)(any,error)
	SetColumns(rows *sql.Rows)error
}

type Creator func(model *model.Model,entity any) Value
