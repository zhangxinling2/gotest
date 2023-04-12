package session

import (
	"github.com/google/uuid"
	"gotest/web"
)

type Manager struct {
	Store
	Propagator
	SessCtxKey string
}

func(m *Manager)GetSession(ctx *web.Context)(Session,error){
	//在这里缓存session到context
	if ctx.UserValues==nil{
		ctx.UserValues=make(map[string]any,1)
	}
	val,ok:=ctx.UserValues[m.SessCtxKey]
	if ok{
		return val.(Session),nil
	}
	sessId,err:=m.Extract(ctx.Req)
	if err!=nil{
		return nil,err
	}
	sess,err:= m.Get(ctx.Req.Context(),sessId)
	if err!=nil{
		return nil, err
	}
	ctx.UserValues[m.SessCtxKey]=sess
	return sess,err
}

func(m *Manager)InitSession(ctx *web.Context)(Session,error) {
	id:=uuid.New().String()
	sess,err:= m.Generate(ctx.Req.Context(),id)
	if err!=nil{
		return nil, err
	}
	//注入进HTTP响应
	err=m.Inject(id,ctx.Resp)
	return sess,err
}
func(m *Manager)RemoveSession(ctx *web.Context)error{
	sess,err:=m.GetSession(ctx)
	if err!=nil{
		return err
	}
	err=m.Store.Remove(ctx.Req.Context(),sess.ID())
	if err!=nil{
		return err
	}
	return m.Propagator.Remove(ctx.Resp)
}

func(m *Manager)RefreshSession(ctx *web.Context)error{
	sess,err:=m.GetSession(ctx)
	if err!=nil{
		return err
	}
	return m.Refresh(ctx.Req.Context(),sess.ID())
}