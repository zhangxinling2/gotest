package querylog

import (
	"context"
	"gotest/orm"
	"log"
	"time"
)

type MiddlewareBuilder struct {
	//慢查询阈值
	threshold time.Duration
	logFunc func(query string,args []any)
}

func NewMiddlewareBuilder(threshold time.Duration)*MiddlewareBuilder{
	return &MiddlewareBuilder{
		logFunc: func(query string, args []any) {
			log.Printf("sql: %s ,args: %v \n",query,args)
		},
		threshold: threshold,
	}
}
func (m *MiddlewareBuilder)LogFunc(fn func(query string,args []any))*MiddlewareBuilder  {
	m.logFunc=fn
	return m
}
func(m MiddlewareBuilder)Build()orm.Middleware{
	return func(next orm.Handler) orm.Handler {
		return func(ctx context.Context, qc *orm.QueryContext) *orm.QueryResult {
			startTime:=time.Now()
			defer func() {
				duration:=time.Since(startTime)
				if duration<m.threshold{
					return
				}
				q,err:=qc.Builder.Build()
				if err==nil{
					m.logFunc(q.SQL,q.Args)
				}
			}()

			res:=next(ctx,qc)
			return res
		}
	}
}
