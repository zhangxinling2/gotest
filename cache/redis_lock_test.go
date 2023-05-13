package cache

import (
	"context"
	"github.com/go-redis/redis/v9"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"gotest/cache/mocks"
	"log"
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
func TestLock_Refresh(t *testing.T) {
	ctrl := gomock.NewController(t)
	testCases := []struct {
		name       string
		lock       *Lock
		expiration time.Duration
		wantErr    error
	}{
		{
			name: "refresh err",
			lock: &Lock{
				client: func(ctrl *gomock.Controller) redis.Cmdable {
					cmd := mocks.NewMockCmdable(ctrl)
					res := redis.NewCmd(context.Background())
					res.SetErr(context.DeadlineExceeded)
					cmd.EXPECT().Eval(context.Background(), luaRefresh, []string{"key"}, "value", float64(1)).Return(res)
					return cmd
				}(ctrl),
				key:        "key",
				value:      "value",
				expiration: time.Second,
			},
			wantErr:    context.DeadlineExceeded,
			expiration: time.Second,
		},
		{
			name: "lock not hold",
			lock: &Lock{
				client: func(ctrl *gomock.Controller) redis.Cmdable {
					cmd := mocks.NewMockCmdable(ctrl)
					res := redis.NewCmd(context.Background())
					res.SetVal(int64(0))
					cmd.EXPECT().Eval(context.Background(), luaRefresh, []string{"key"}, "value", float64(1)).Return(res)
					return cmd
				}(ctrl),
				key:        "key",
				value:      "value",
				expiration: time.Second,
			},
			wantErr:    errLockNotHold,
			expiration: time.Second,
		},
		{
			name: "Refresh",
			lock: &Lock{
				client: func(ctrl *gomock.Controller) redis.Cmdable {
					cmd := mocks.NewMockCmdable(ctrl)
					res := redis.NewCmd(context.Background())
					res.SetVal(int64(1))
					cmd.EXPECT().Eval(context.Background(), luaRefresh, []string{"key"}, "value", float64(1)).Return(res)
					return cmd
				}(ctrl),
				key:        "key",
				value:      "value",
				expiration: time.Second,
			},
			expiration: time.Second,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.lock.Refresh(context.Background())
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
		})
	}
}

func ExampleLock_Refresh() {
	//加锁成功
	var l *Lock
	//终止续约的channel
	stopChan := make(chan struct{})
	//出现错误的channel
	errChan := make(chan error)
	//可以挽回的error 比如超时的error chan,放入值后需要继续运行，所以设置缓冲为1
	timeoutChan := make(chan struct{}, 1)
	//一个goroutine用来续约
	go func() {
		ticker := time.NewTicker(time.Second * 10)
		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				err := l.Refresh(ctx)
				cancel()
				if err == context.DeadlineExceeded {
					timeoutChan <- struct{}{}
					continue
				}
				if err != nil {
					errChan <- err
					//自己选择在哪close
					//close(stopChan)
					//close(errChan)
					return
				}
			case <-timeoutChan:
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				err := l.Refresh(ctx)
				cancel()
				if err == context.DeadlineExceeded {
					timeoutChan <- struct{}{}
				}
				if err != nil {
					return
				}
			case <-stopChan:
				return
			}
		}

	}()
	//执行业务
	//在业务执行过程中检测error
	//循环中的业务
	for i := 0; i < 100; i++ {
		select {
		//续约失败
		case <-errChan:
			break
		default:
			//正常业务逻辑
		}
	}
	//非循环的业务
	//只能每个步骤都要检测error
	select {
	case err := <-errChan:
		log.Fatalln(err)
		return
	default:
		//业务步骤1
	}
	select {
	case err := <-stopChan:
		log.Fatalln(err)
		return
	default:
		//业务步骤2
	}
	//执行完业务，终止续约
	stopChan <- struct{}{}
	// l.Unlock(context.Background())
}
func TestClient_Lock(t *testing.T) {
	testCases := []struct {
		name string
		mock func(ctrl *gomock.Controller) redis.Cmdable
		key  string
		//key 过期时间
		expiration time.Duration
		//重试间隔
		timeout time.Duration
		//重试策略
		retry   RetryStrategy
		wantErr error
	}{
		{
			name: "locked",
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				cmd := mocks.NewMockCmdable(ctrl)
				res := redis.NewCmd(context.Background())
				res.SetVal(int64(1))
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				cmd.EXPECT().Eval(ctx, luaLock, []string{"lock_key1"}, gomock.Any(), float64(60)).Return(res)
				cancel()
				return cmd
			},
			key:        "lock_key1",
			expiration: time.Minute,
			timeout:    time.Second * 3,
			retry: &FixedIntervalRetryStrategy{
				Interval: time.Second * 3,
				MaxCnt:   10,
				cnt:      0,
			},
		},
	}
	ctrl := gomock.NewController(t)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := NewClient(tc.mock(ctrl))
			wantLock := &Lock{
				client:     NewClient(tc.mock(ctrl)).client,
				key:        tc.key,
				expiration: tc.expiration,
			}
			lock, err := client.Lock(context.Background(), tc.key, tc.expiration, tc.timeout, tc.retry)
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			assert.Equal(t, wantLock.key, lock.key)
			//无法得到准确的value只能通过判断是否有值做粗略的检查
			assert.NotEmpty(t, lock.value)
			assert.NotNil(t, wantLock.client)
		})
	}
}
