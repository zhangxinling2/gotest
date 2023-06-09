package cache

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v9"
	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
	"time"
)

var (
	errFailToPreemptLock = errors.New("redis-lock: 抢锁失败")
	errLockNotHold       = errors.New("redis-lock: 没有持有锁")

	//go:embed lua/unlock.lua
	luaUnlock string
	//go:embed lua/refresh.lua
	luaRefresh string
	//go:embed lua/lock.lua
	luaLock string
)

//Client 用于加锁
type Client struct {
	client redis.Cmdable
	g      *singleflight.Group
}

func NewClient(client redis.Cmdable) *Client {
	return &Client{client: client,
		g: &singleflight.Group{}}
}
func (c *Client) SingleLock(ctx context.Context, key string, expiration, timeout time.Duration, retry RetryStrategy) (*Lock, error) {
	for {
		flag := false
		resCh := c.g.DoChan(key, func() (interface{}, error) {
			lock, err := c.Lock(ctx, key, expiration, timeout, retry)
			if err != nil {
				return nil, err
			}
			flag = true
			return lock, nil
		})
		select {
		case res := <-resCh:
			if flag {
				c.g.Forget(key)
				if res.Err != nil {
					return nil, res.Err
				}
				return res.Val.(*Lock), nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

//Lock 与tryLock不同的是它是重试加锁,timeout时重试的超时时间
func (c *Client) Lock(ctx context.Context, key string, expiration, timeout time.Duration, retry RetryStrategy) (*Lock, error) {
	val := uuid.New().String()
	//重试的计时器
	var timer *time.Timer
	for {
		tctx, cancel := context.WithTimeout(ctx, timeout)
		res, err := c.client.Eval(tctx, luaLock, []string{key}, val, expiration.Seconds()).Result()
		cancel()
		if err != nil && err != context.DeadlineExceeded {
			return nil, err
		}
		//加锁ok了
		if res == "OK" {
			return &Lock{
				client:     c.client,
				key:        key,
				value:      val,
				expiration: expiration,
				unlockCh:   make(chan struct{}, 1),
			}, nil
		}
		//加锁失败，进行重试
		interval, ok := retry.Next()
		//超出重试次数
		if !ok {
			return nil, fmt.Errorf("redis-lock: 超出重试限制, %w", errFailToPreemptLock)
		}
		//没有超出，则重置计时器
		if timer == nil {
			timer = time.NewTimer(interval)
		} else {
			timer.Reset(interval)
		}
		select {
		case <-timer.C:
			//什么都不用干，步入下一个循环
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
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
		client:   c.client,
		key:      key,
		value:    val,
		unlockCh: make(chan struct{}, 1),
	}, nil
}

//Lock 代表锁
type Lock struct {
	client     redis.Cmdable
	key        string
	value      string
	expiration time.Duration
	unlockCh   chan struct{}
}

// AutoRefresh 自动续约 传入超时时间
func (l *Lock) AutoRefresh(interval time.Duration, timeout time.Duration) error {
	//可以挽回的error 比如超时的error chan,放入值后需要继续运行，所以设置缓冲为1
	timeoutChan := make(chan struct{}, 1)
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			err := l.Refresh(ctx)
			cancel()
			if err == context.DeadlineExceeded {
				timeoutChan <- struct{}{}
				continue
			}
			if err != nil {
				return err
			}
		case <-timeoutChan:
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			err := l.Refresh(ctx)
			cancel()
			if err == context.DeadlineExceeded {
				timeoutChan <- struct{}{}
			}
			if err != nil {
				return err
			}
		case <-l.unlockCh:
			return nil
		}
	}
}
func (l *Lock) Unlock(ctx context.Context) error {
	res, err := l.client.Eval(ctx, luaUnlock, []string{l.key}, l.value).Int64()
	defer func() {
		select {
		case l.unlockCh <- struct{}{}:
		default:
			//说明没有人调用AutoRefresh
		}
	}()
	if err != nil {
		return err
	}
	if res != 1 {
		return errLockNotHold
	}
	return nil
}

func (l *Lock) Refresh(ctx context.Context) error {
	res, err := l.client.Eval(ctx, luaRefresh, []string{l.key}, l.value, l.expiration.Seconds()).Int64()
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
