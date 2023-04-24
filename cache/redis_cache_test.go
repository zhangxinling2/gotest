package cache

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v9"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"gotest/cache/mocks"
	"testing"
	"time"
)

func TestRedisCache_Set(t *testing.T) {
	testCases := []struct {
		name       string
		mock       func(ctrl *gomock.Controller) redis.Cmdable
		key        string
		val        string
		expiration time.Duration
		wantErr    error
	}{
		{
			name: "set val",
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				cmd := mocks.NewMockCmdable(ctrl)
				status := redis.NewStatusCmd(context.Background())
				status.SetVal("OK")
				cmd.EXPECT().Set(context.Background(), "key1", "value1", time.Second).Return(status)
				return cmd
			},
			key:        "key1",
			val:        "value1",
			expiration: time.Second,
		},
		{
			name: "expiration",
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				cmd := mocks.NewMockCmdable(ctrl)
				status := redis.NewStatusCmd(context.Background())
				status.SetErr(context.DeadlineExceeded)
				cmd.EXPECT().Set(context.Background(), "key1", "value1", time.Second).Return(status)
				return cmd
			},
			key:        "key1",
			val:        "value1",
			expiration: time.Second,
			wantErr:    context.DeadlineExceeded,
		},
		{
			name: "unexpected msg",
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				cmd := mocks.NewMockCmdable(ctrl)
				status := redis.NewStatusCmd(context.Background())
				status.SetVal("un ok")
				cmd.EXPECT().Set(context.Background(), "key1", "value1", time.Second).Return(status)
				return cmd
			},
			key:        "key1",
			val:        "value1",
			expiration: time.Second,
			wantErr:    errors.New(fmt.Sprintf("%v ,res: %s", errFailedToSetCache, "un ok")),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			rdb := NewRedisCache(tc.mock(ctrl))
			err := rdb.Set(context.Background(), tc.key, tc.val, tc.expiration)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}

func TestRedisCache_Get(t *testing.T) {
	testCases := []struct {
		name    string
		mock    func(ctrl *gomock.Controller) redis.Cmdable
		key     string
		wantErr error
		wantVal string
	}{
		{
			name: "get val",
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				cmd := mocks.NewMockCmdable(ctrl)
				status := redis.NewStringCmd(context.Background())
				status.SetVal("val1")
				cmd.EXPECT().Get(context.Background(), "key1").Return(status)
				return cmd
			},
			key:     "key1",
			wantVal: "val1", //这个value就是status设置的
		},
		{
			name: "expiration",
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				cmd := mocks.NewMockCmdable(ctrl)
				status := redis.NewStringCmd(context.Background())
				status.SetErr(context.DeadlineExceeded)
				cmd.EXPECT().Get(context.Background(), "key1").Return(status)
				return cmd
			},
			key:     "key1",
			wantErr: context.DeadlineExceeded,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			rdb := NewRedisCache(tc.mock(ctrl))
			val, err := rdb.Get(context.Background(), tc.key)
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			assert.Equal(t, tc.wantVal, val)
		})
	}
}
