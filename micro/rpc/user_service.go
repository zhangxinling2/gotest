package rpc

import "context"

type UserService struct {
	// 用反射来赋值
	// 类型是函数的字段，它不是方法，(它不是定义在 UserService 上的方法)
	GetById func(ctx context.Context, req *GetByIdReq) (*GetByIdResp, error)
}

func (u UserService) Name() string {
	return "UserService"
}

type GetByIdReq struct {
	Id int
}
type GetByIdResp struct {
	data string
}

// 给UserService一个真正的实现
type UserServiceServer struct {
}

func (u *UserServiceServer) GetById(ctx context.Context, req *GetByIdReq) (*GetByIdResp, error) {
	return &GetByIdResp{data: "hello rpc"}, nil
}
func (u UserServiceServer) Name() string {
	return "UserService"
}
