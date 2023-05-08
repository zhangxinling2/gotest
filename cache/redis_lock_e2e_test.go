//go:build e2e

package cache

import (
	"context"
	"github.com/go-redis/redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestClient_TryLock2(t *testing.T) {
	//before和after都要使用，所以放到外面
	rdb := NewClient(redis.NewClient(&redis.Options{Addr: "localhost:6379"}))
	testCases := []struct {
		name       string
		before     func(t *testing.T)
		after      func(t *testing.T)
		key        string
		expiration time.Duration
		wantErr    error
		wantLock   *Lock
	}{
		{
			name: "locked",
			before: func(t *testing.T) {
				_, err := rdb.client.SetNX(context.Background(), "key1", "value1", time.Second*10).Result()
				if err != nil {
					return
				}
			},
			after: func(t *testing.T) {
				res, err := rdb.client.GetDel(context.Background(), "key1").Result()
				require.NoError(t, err)
				require.Equal(t, "value1", res)
			},
			key:        "key1",
			expiration: time.Second * 10,
			wantErr:    errFailToPreemptLock,
		},
		{
			name:   "set lock",
			before: func(t *testing.T) {},
			after: func(t *testing.T) {
				_, err := rdb.client.Del(context.Background(), "key2").Result()
				require.NoError(t, err)
			},
			key:        "key2",
			expiration: time.Second * 10,
			wantLock: &Lock{
				client:     rdb.client,
				key:        "key2",
				expiration: time.Second * 10,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before(t)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()
			lock, err := rdb.TryLock(ctx, tc.key, tc.expiration)
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			assert.Equal(t, tc.wantLock.key, lock.key)
			assert.NotEmpty(t, lock.value)
			tc.after(t)
		})
	}
}

func TestLock_Unlock2(t *testing.T) {
	//before和after都要使用，所以放到外面
	rdb := NewClient(redis.NewClient(&redis.Options{Addr: "localhost:6379"}))
	testCases := []struct {
		name    string
		lock    *Lock
		before  func(t *testing.T)
		after   func(t *testing.T)
		wantErr error
	}{
		{
			name: "no locked",
			lock: &Lock{
				client: rdb.client,
				key:    "unlock_key1",
			},
			before:  func(t *testing.T) {},
			after:   func(t *testing.T) {},
			wantErr: errLockNotHold,
		},
		{
			name: "other has locked",
			lock: &Lock{
				client: rdb.client,
				key:    "unlock_key2",
				value:  "unlock_value",
			},
			before: func(t *testing.T) {
				_, err := rdb.client.SetNX(context.Background(), "unlock_key2", "unlock_value2", time.Second*10).Result()
				require.NoError(t, err)
				if err != nil {
					return
				}
			},
			after: func(t *testing.T) {
				res, err := rdb.client.GetDel(context.Background(), "unlock_key2").Result()
				require.NoError(t, err)
				require.Equal(t, "unlock_value2", res)
			},
			wantErr: errLockNotHold,
		},
		{
			name: "unlocked",
			lock: &Lock{
				client: rdb.client,
				key:    "unlock_key3",
				value:  "unlock_value3",
			},
			before: func(t *testing.T) {
				_, err := rdb.client.SetNX(context.Background(), "unlock_key3", "unlock_value3", time.Second*10).Result()
				require.NoError(t, err)
				if err != nil {
					return
				}
			},
			after: func(t *testing.T) {
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before(t)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()
			err := tc.lock.Unlock(ctx)
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			tc.after(t)
		})
	}
}
