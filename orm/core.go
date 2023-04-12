package orm

import (
	"context"
	"gotest/orm/internal/valuer"
	"gotest/orm/model"
)

type core struct {
	model *model.Model
	dialect Dialect
	creator valuer.Creator
	r model.Registry
	mdls []Middleware
}
func get[T any](ctx context.Context,sess Session,c core,qc  *QueryContext)*QueryResult{


	//var root Handler = getHandler[T](ctx,s.sess,s.core,&QueryContext{
	//	Type: "RAW",
	//	Builder: s,
	//	Model: s.model,
	//})
	var root Handler = func(ctx context.Context, qc *QueryContext) *QueryResult {
		return getHandler[T](ctx,sess,c,qc)
	}
	for i:=len(c.mdls)-1;i>=0;i--{
		root=c.mdls[i](root)
	}
	//return root(ctx,&QueryContext{
	//	Type: "RAW",
	//	Builder: builder,
	//	//问题在于s.model在Build时才会赋值，1.在Get初始化s.model 2.专门设置一个middleware来设置model
	//	Model: c.model,
	//})
	return root(ctx,qc)
}
func getHandler[T any](ctx context.Context,sess Session,c core,qc *QueryContext) *QueryResult{
	q,err:=qc.Builder.Build()
	if err!=nil{
		return &QueryResult{
			Err: err,
		}
	}
	//在这里发起查询并处理结果集
	rows,err:=sess.queryContext(ctx,q.SQL,q.Args...)
	//这是查询错误，数据库返回的
	if err!=nil{
		return &QueryResult{
			Err: err,
		}
	}
	//将row 转化成*T
	//在这里处理结果集
	if !rows.Next(){
		//要不要返回error
		//返回error,和sql包语义保持一致 sql.ErrNoRows
		//return nil, ErrNoRows
		return &QueryResult{
			Err: ErrNoRows,
		}
	}
	tp:=new(T)
	creator:=c.creator
	val:=creator(c.model,tp)
	err=val.SetColumns(rows)
	//接口定义好后，一个是用新接口的方法改造上层，一个就是提供不同实现
	return &QueryResult{
		Err: err,
		Result: tp,
	}
}

func exec(ctx context.Context,sess Session,c core,qc  *QueryContext)*QueryResult{


	//var root Handler = getHandler[T](ctx,s.sess,s.core,&QueryContext{
	//	Type: "RAW",
	//	Builder: s,
	//	Model: s.model,
	//})
	var root Handler = func(ctx context.Context, qc *QueryContext) *QueryResult {
		return execHandler(ctx,sess,c,qc)
	}
	for i:=len(c.mdls)-1;i>=0;i--{
		root=c.mdls[i](root)
	}
	//return root(ctx,&QueryContext{
	//	Type: "RAW",
	//	Builder: builder,
	//	//问题在于s.model在Build时才会赋值，1.在Get初始化s.model 2.专门设置一个middleware来设置model
	//	Model: c.model,
	//})
	return root(ctx,qc)
}
func execHandler(ctx context.Context,sess Session,c core,qc *QueryContext) *QueryResult{
	q,err:=qc.Builder.Build()
	if err!=nil{
		return &QueryResult{
			Err: err,
			Result: Result{
				err: err,
			},
		}
	}
	res,err:=sess.execContext(ctx,q.SQL,q.Args...)
	return &QueryResult{
		Err: err,
		Result: Result{
			res: res,
			err: err,
		},
	}
}