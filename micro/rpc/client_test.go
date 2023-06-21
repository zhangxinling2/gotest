package rpc

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSetFuncField(t *testing.T) {
	testCases := []struct {
		name    string
		service Service
		wantErr error
	}{
		{
			name:    "nil",
			service: nil,
			wantErr: errors.New("服务是空服务"),
		},
		{
			name:    "no pointer",
			service: UserService{},
			wantErr: errors.New("只接受一级结构体指针"),
		},
		{
			name:    "user service ",
			service: &UserService{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := setFuncField(tc.service)
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
