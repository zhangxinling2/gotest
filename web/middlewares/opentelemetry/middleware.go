package opentelemetry

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"gotest/web"
)

const instrumentationName = "gotest/web/middlewares/opentelementry"

type MiddlewareBuilder struct {
	Tracer trace.Tracer
}
// NewMiddlewareBuilder 定义为私有的让用户传入
//func NewMiddlewareBuilder(trace trace.Tracer) *MiddlewareBuilder{
//	return &MiddlewareBuilder{
//		Tracer: trace,
//	}
//}

func (m MiddlewareBuilder)Build() web.Middleware {
	if m.Tracer==nil{
		m.Tracer=otel.GetTracerProvider().Tracer(instrumentationName)
	}
	return func(next web.HandleFunc) web.HandleFunc {
		return func(ctx *web.Context) {
			reqCtx:=ctx.Req.Context()
			//尝试和客户端的trace结合在一起
			reqCtx=otel.GetTextMapPropagator().Extract(reqCtx,propagation.HeaderCarrier(ctx.Req.Header))

			_,span:=m.Tracer.Start(reqCtx,"unknown")
			defer span.End()

			span.SetAttributes(attribute.String("http.method",ctx.Req.Method))
			span.SetAttributes(attribute.String("http.url",ctx.Req.URL.String()))
			span.SetAttributes(attribute.String("http.scheme",ctx.Req.URL.Scheme))
			span.SetAttributes(attribute.String("http.host",ctx.Req.Host))

			ctx.Req=ctx.Req.WithContext(reqCtx)
			//直接调用下一步
			next(ctx)
			//这个执行完next才可能有值
			span.SetName(ctx.MatchRoute)

			//把响应码加上去
			span.SetAttributes(attribute.Int("http.state",))
		}
	}
}

