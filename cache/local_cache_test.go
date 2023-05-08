package cache

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestBuildInMapCache_Get(t *testing.T) {
	testCases := []struct {
		name    string
		key     string
		cache   func() *BuildInMapCache //为了向里面加key设置为函数类型
		wantVal any
		wantErr error
	}{
		{
			name: "key not found",
			key:  "not exist key",
			cache: func() *BuildInMapCache {
				return NewBuildInMapCache(10 * time.Second)
			},
			wantErr: errNoValue,
		},
		{
			name: "get value",
			key:  "key1",
			cache: func() *BuildInMapCache {
				res := NewBuildInMapCache(10 * time.Second)
				err := res.Set(context.Background(), "key1", 123, time.Minute)
				require.NoError(t, err)
				return res
			},
			wantVal: 123,
		},
		{
			name: "expiration",
			key:  "expiration key",
			cache: func() *BuildInMapCache {
				res := NewBuildInMapCache(10 * time.Second)
				err := res.Set(context.Background(), "expiration  key1", 123, time.Second)
				require.NoError(t, err)
				time.Sleep(time.Second)
				return res
			},
			wantErr: errNoValue,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := tc.cache().Get(context.Background(), tc.key)
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			assert.Equal(t, tc.wantVal, res)
		})
	}
}

//测试我们的轮询起效果
func TestNewBuildInMapCache(t *testing.T) {
	//探针
	cnt := 0
	c := NewBuildInMapCache(time.Second, BuildInMapCacheWithEvictCallBack(func(key string, val any) {
		cnt++
	}))
	err := c.Set(context.Background(), "key1", 123, time.Millisecond)
	require.NoError(t, err)
	time.Sleep(3 * time.Second)
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	var _, ok = c.data["key1"]
	require.False(t, ok)
	require.Equal(t, cnt, 1)
}

//测试每轮轮询数据的个数，临时修改一下代码里判断的值
func TestNewBuildInMapCacheTicker(t *testing.T) {
	cnt := 0
	c := NewBuildInMapCache(time.Second, BuildInMapCacheWithEvictCallBack(func(key string, val any) {
		cnt++
	}))
	err := c.Set(context.Background(), "key1", 123, time.Second)
	require.NoError(t, err)
	err = c.Set(context.Background(), "key2", 123, time.Second)
	require.NoError(t, err)
	err = c.Set(context.Background(), "key3", 123, time.Second)
	require.NoError(t, err)
	err = c.Set(context.Background(), "key4", 123, time.Second)
	require.NoError(t, err)
	time.Sleep(time.Second * 2)
	require.Equal(t, 4, cnt)
}

func TestBuildInMapCache_Close(t *testing.T) {
	c := NewBuildInMapCache(time.Second)
	c.Close()
	err := c.Set(context.Background(), "key1", 123, time.Second*2)
	require.NoError(t, err)
	time.Sleep(2 * time.Second)
	v, err := c.Get(context.Background(), "key1")
	require.NoError(t, err)
	assert.Equal(t, 123, v)
}
