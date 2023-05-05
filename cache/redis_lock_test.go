package cache

import (
	"context"
	"github.com/go-redis/redis/v9"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"gotest/cache/mocks"
	"testing"
	"time"
)

func TestClient_TryLock(t *testing.T) {
	testCases := []struct {
		name       string
		mock       func(ctrl *gomock.Controller) redis.Cmdable
		key        string
		expiration time.Duration
		wantErr    error
		wantVal    *Lock
	}{
		{
			name: "set nx error",
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				cmd := mocks.NewMockCmdable(ctrl)
				//setNx返回的就是Bool
				res := redis.NewBoolResult(false, context.DeadlineExceeded)
				cmd.EXPECT().SetNX(context.Background(), "key1", gomock.Any(), time.Second).Return(res)
				return cmd
			},
			key:        "key1",
			expiration: time.Second,
			wantErr:    context.DeadlineExceeded,
		},
		{
			name: "fail to preempt lock",
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				cmd := mocks.NewMockCmdable(ctrl)
				res := redis.NewBoolResult(false, errFailToPreemptLock)
				cmd.EXPECT().SetNX(context.Background(), "key1", gomock.Any(), time.Second).Return(res)
				return cmd
			},
			key:        "key1",
			expiration: time.Second,
			wantErr:    errFailToPreemptLock,
		},
		{
			name: "lock",
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				cmd := mocks.NewMockCmdable(ctrl)
				res := redis.NewBoolResult(true, nil)
				cmd.EXPECT().SetNX(context.Background(), "key1", gomock.Any(), time.Second).Return(res)
				return cmd
			},
			key:        "key1",
			expiration: time.Second,
			wantVal: &Lock{
				key:        "key1",
				expiration: time.Second,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			lock, err := NewClient(tc.mock(ctrl)).TryLock(context.Background(), tc.key, tc.expiration)
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			assert.Equal(t, tc.wantVal.key, lock.key)
			//无法得到准确的value只能通过判断是否有值做粗略的检查
			assert.NotEmpty(t, lock.value)
		})
	}
}

func TestLock_Unlock(t *testing.T) {
	//在测试用例中使用Lock，那么里面的client就需要ctrl，只能定义一个总的ctrl来复用
	//可以使用下面的定义，这样就可以在每个测试用例中单独创建ctrl，再把lock创建起来即可
	//testCases := []struct{
	//	name string
	//	mock func(ctrl *gomock.Controller) redis.Cmdable
	//	key string
	//	value string
	//	wantErr error
	//}
	ctrl := gomock.NewController(t)
	testCases := []struct {
		name    string
		lock    *Lock
		wantErr error
	}{
		{
			name: "unlock err",
			lock: &Lock{
				client: func(ctrl *gomock.Controller) redis.Cmdable {
					cmd := mocks.NewMockCmdable(ctrl)
					res := redis.NewCmd(context.Background())
					res.SetErr(context.DeadlineExceeded)
					cmd.EXPECT().Eval(context.Background(), luaUnlock, []string{"key"}, "value").Return(res)
					return cmd
				}(ctrl),
				key:        "key",
				value:      "value",
				expiration: time.Second,
			},
			wantErr: context.DeadlineExceeded,
		},
		{
			name: "lock not hold",
			lock: &Lock{
				client: func(ctrl *gomock.Controller) redis.Cmdable {
					cmd := mocks.NewMockCmdable(ctrl)
					res := redis.NewCmd(context.Background())
					res.SetVal(int64(0))
					cmd.EXPECT().Eval(context.Background(), luaUnlock, []string{"key"}, "value").Return(res)
					return cmd
				}(ctrl),
				key:        "key",
				value:      "value",
				expiration: time.Second,
			},
			wantErr: errLockNotHold,
		},
		{
			name: "unlock",
			lock: &Lock{
				client: func(ctrl *gomock.Controller) redis.Cmdable {
					cmd := mocks.NewMockCmdable(ctrl)
					res := redis.NewCmd(context.Background())
					res.SetVal(int64(1))
					cmd.EXPECT().Eval(context.Background(), luaUnlock, []string{"key"}, "value").Return(res)
					return cmd
				}(ctrl),
				key:        "key",
				value:      "value",
				expiration: time.Second,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.lock.Unlock(context.Background())
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
		})
	}
}
