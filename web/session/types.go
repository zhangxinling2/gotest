package session

import (
	"context"
	"net/http"
)

//这里是让用户知道的错误，用户关心的错误
//var(
//	// ErrKeyNotFound sentinel error:与定义错误,在这定义独属于memory
//	ErrKeyNotFound = errors.New("")
//)
//管理session本身
type Store interface {
	//有与第三方打交道的地方一般都带上Context
	// session对应的ID谁来指定？让用户指定
	Generate(ctx context.Context,id string)(Session,error)
	Refresh(ctx context.Context,id string)error
	Remove(ctx context.Context,id string)error
	Get(ctx context.Context,id string)(Session,error)
}

type Session interface {
	Get(ctx context.Context,key string)(any,error)
	Set(ctx context.Context,key string,val string)error
	ID()string
}

type Propagator interface {
	Inject(id string,write http.ResponseWriter)error
	Extract(req *http.Request)(string,error)
	Remove(writer http.ResponseWriter)error
}