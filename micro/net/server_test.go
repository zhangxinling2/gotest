package net

import (
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"gotest/micro/net/mocks"
	"net"
	"testing"
)

func TestHandleConn(t *testing.T) {
	testCases := []struct {
		name    string
		mock    func(ctrl *gomock.Controller) net.Conn
		wantErr error
	}{
		{
			name: "read error",
			mock: func(ctrl *gomock.Controller) net.Conn {
				res := mocks.NewMockConn(ctrl)
				res.EXPECT().Read(gomock.Any()).Return(0, errors.New("read error"))
				return res
			},
			wantErr: errors.New("read error"),
		},
		{
			name: "write error",
			mock: func(ctrl *gomock.Controller) net.Conn {
				res := mocks.NewMockConn(ctrl)
				data := make([]byte, 128)
				res.EXPECT().Read(data).Return(0, nil)
				res.EXPECT().Write(gomock.Any()).Return(0, errors.New("write error"))
				return res
			},
			wantErr: errors.New("write error"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			err := handleConn(tc.mock(ctrl))
			assert.Equal(t, tc.wantErr, err)
		})
	}
}
