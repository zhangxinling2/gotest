package rpc

import (
	"context"
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSetFuncField(t *testing.T) {
	testCases := []struct {
		name    string
		service Service
		mock    func(ctrl *gomock.Controller) Proxy
		wantErr error
	}{
		{
			name:    "nil",
			service: nil,
			mock: func(ctrl *gomock.Controller) Proxy {
				return NewMockProxy(ctrl)
			},
			wantErr: errors.New("服务是空服务"),
		},
		{
			name:    "no pointer",
			service: &UserService{},
			mock: func(ctrl *gomock.Controller) Proxy {
				return NewMockProxy(ctrl)
			},
			wantErr: errors.New("只接受一级结构体指针"),
		},
		{
			name: "user service ",
			mock: func(ctrl *gomock.Controller) Proxy {
				p := NewMockProxy(ctrl)
				p.EXPECT().Invoke(gomock.Any(), &Request{
					ServiceName: "user-service",
					MethodName:  "GetById",
					Data:        []byte("Id:123"),
				}).Return(&Response{}, nil)
				return p
			},
			service: &UserService{},
		},
	}
	for _, tc := range testCases {

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			err := setFuncField(tc.service, tc.mock(ctrl))
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			resp, err := tc.service.(*UserService).GetById(context.Background(), &GetByIdReq{Id: 123})
			assert.Equal(t, tc.wantErr, err)
			t.Log(resp)
		})
	}
}
