package cache

import (
	"context"
	_ "embed"
	"errors"
	"github.com/go-redis/redis/v9"
	"github.com/google/uuid"
	"time"
)

var (
	errFailToPreemptLock = errors.New("redis-lock: 抢锁失败")
	errLockNotHold       = errors.New("redis-lock: 没有持有锁")

	//go:embed lua/unlock.lua
	luaUnlock string
)

//Client 用于加锁
type Client struct {
	client redis.Cmdable
}

func NewClient(client redis.Cmdable) *Client {
	return &Client{client: client}
}

//TryLock 传入上下文，key和过期时间，返回一个Lock，即锁
func (c *Client) TryLock(ctx context.Context, key string, expiration time.Duration) (*Lock, error) {
	val := uuid.New().String()
	ok, err := c.client.SetNX(ctx, key, val, expiration).Result()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errFailToPreemptLock
	}
	return &Lock{
		client: c.client,
		key:    key,
		value:  val,
	}, nil
}

//Lock 代表锁
type Lock struct {
	client     redis.Cmdable
	key        string
	value      string
	expiration time.Duration
}

func (l *Lock) Unlock(ctx context.Context) error {
	res, err := l.client.Eval(ctx, luaUnlock, []string{l.key}, l.value).Int64()
	if err != nil {
		return err
	}
	if res != 1 {
		return errLockNotHold
	}
	return nil
}

//Unlock Lock删除key以释放锁
//func(l *Lock)Unlock(ctx context.Context)error{
//	//检查锁是否是自己的
//	val,err:=l.client.Get(ctx,l.key).Result()
//	if err!=nil{
//		return err
//	}
//	if val!=l.value{
//		return errors.New("不是自己的锁")
//	}
//	//上面check，下面do something,在中间这里的空缺，键值对就可能被删除了
//	cnt,err:=l.client.Del(ctx,l.key).Result()
//	if err!=nil{
//		return err
//	}
//	if cnt!=1{
//		return errLockNotHold
//	}
//	return nil
//}
