package rpc

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestInitClientProxy(t *testing.T) {
	//先启动服务
	server := NewServer()
	go func() {
		server.Start("tcp", ":8081")
	}()
	time.Sleep(time.Second * 3)
	//注册上服务
	server.RegisterService(&UserServiceServer{})
	usClient := &UserService{}
	//初始化客户端
	err := InitClientProxy(usClient, ":8081")
	require.NoError(t, err)
	resp, err := usClient.GetById(context.Background(), &GetByIdReq{Id: 123})
	require.NoError(t, err)
	assert.Equal(t, &GetByIdResp{data: "hello rpc"}, resp)
}
