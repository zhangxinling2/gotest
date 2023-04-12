package orm

import (
	"context"
	"gotest/orm/model"
)

type QueryContext struct {
	// 查询类型，标记增删改查
	Type string

	//代表的是查询本身,大多数情况下需要转化到具体的类型才能篡改查询
	Builder QueryBuilder
	//一般都会暴露出来给用户做高级处理
	Model *model.Model
}

type QueryResult struct {
	//Result 在不同查询下类型不同
	//SELECT 可以是*T也可以是[]*T
	//其他就是类型Result
	Result any
	//查询本身出的问题
	Err error
}
type Handler func(ctx context.Context,qc *QueryContext)*QueryResult
type Middleware func(next Handler)Handler