package opentelemetry

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gotest/orm"
)

const instrumentationName = "gotest/orm/middlewares/opentelemetry"
type MiddlewareBuilder struct {
	Tracer trace.Tracer
}

func (m MiddlewareBuilder)Build() orm.Middleware {
	if m.Tracer==nil{
		m.Tracer=otel.GetTracerProvider().Tracer(instrumentationName)
	}
	return func(next orm.Handler) orm.Handler {
		return func(ctx context.Context, qc *orm.QueryContext) *orm.QueryResult {

			// span name : select-test_model
			// insert-test_model
			tbl:=qc.Model.TableName
			spanCtx,span:=m.Tracer.Start(ctx,fmt.Sprintf("%s-%s",qc.Type,tbl))
			defer span.End()
			q,_:=qc.Builder.Build()
			if q!=nil{
				span.SetAttributes(attribute.String("sql",q.SQL))
			}
			//在tracing中就没必要追踪args,因为args可能非常大
			span.SetAttributes(attribute.String("table",tbl))
			span.SetAttributes(attribute.String("component","orm"))
			res:=next(spanCtx,qc)
			if res.Err!=nil{
				span.RecordError(res.Err)
			}
			return res
		}
	}
}
