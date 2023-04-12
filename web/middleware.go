package web

//Midddleware 函数式的责任链模式
//此种go多一点
type Middleware func(next HandleFunc)HandleFunc

//type MiddlewareV1 interface {
//	Invoke(next HandleFunc)HandleFunc
//}
//
////此种java多
//type Interceptor interface {
//	Before(ctx *Context)
//	After(ctx *Context)
//	Surround(ctx *Context)
//}
//type Chain []HandleFunc
//
//type HandleFuncV1 func(ctx *Context)(next bool)
//
//type ChainV1 struct {
//	handlers []HandleFunc
//}
//
//func(c ChainV1)Run(ctx *Context){
//	for _,h:=range c.handlers{
//		next:=h(ctx)
//		//这种是中断执行
//		if !next{
//			return
//		}
//	}
//}
