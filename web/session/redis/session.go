package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v9"
	"gotest/web/session"
)

var(
	errSessionNotFound = errors.New("session not found")
)
type StoreOption func(s *Store)
// hset 就是一个两层map
//    sessid      key    value
// map[string]map[string]string
type Store struct {
	prefix string
	client redis.Cmdable
	expiration time.Duration
}
func StoreWithPrefix(prefix string)StoreOption{
	return func(s *Store) {
		s.prefix=prefix
	}
}
func NewStore(client redis.Cmdable,opts ...StoreOption)*Store{
	res:= &Store{
		expiration: time.Minute*15,
		client: client,
		prefix: "sessId",
	}
	for _,opt:=range opts{
		opt(res)
	}
	return res
}

//创建一个map和id绑在一起
func (s *Store) Generate(ctx context.Context, id string) (session.Session, error) {
	_,err:=s.client.HSet(ctx,redisKey(s.prefix,id),id,id).Result()
	if err!=nil{
		return nil,err
	}
	_,err=s.client.Expire(ctx,redisKey(s.prefix,id),s.expiration).Result()
	if err!=nil{
		return nil,err
	}
	return &Session{
		key: redisKey(s.prefix,id),
		id:id,
		client: s.client,
	},nil
}

func (s *Store) Refresh(ctx context.Context, id string) error {
	ok,err:=s.client.Expire(ctx,redisKey(s.prefix,id),s.expiration).Result()
	if err!=nil{
		return err
	}
	if !ok{
		return errSessionNotFound
	}
	return nil
}

func (s *Store) Remove(ctx context.Context, id string) error {
	_,err:=s.client.Del(ctx,redisKey(s.prefix,id)).Result()
	if err!=nil{
		return err
	}
	return nil
	//代表 id 对应的 session不存在，没有删任何东西
	//if cnt==0
}

func (s *Store) Get(ctx context.Context, id string) (session.Session, error) {
	//自由决策要不要提前把 session 存储的用户数据一并拿过来
	//1. 都不拿
	//2. 只拿高频数据
	//3. 都拿
	cnt,err:=s.client.Exists(ctx,redisKey(s.prefix,id)).Result()
	if err!=nil{
		return nil, err
	}
	if cnt!=1{
		return nil, errSessionNotFound
	}
	return &Session{
		key: redisKey(s.prefix,id),
		id: id,
		client: s.client,
	},nil
}




type Session struct {
	id string
	key string
	prefix string

	client redis.Cmdable
}

func (s *Session) Get(ctx context.Context, key string) (any, error) {
	val, err := s.client.HGet(ctx, key, key).Result()
	return val, err
}

func (s *Session) Set(ctx context.Context, key string, val string) error {
	const lua = `
if redis.call("exists", KEYS[1])
then
	return redis.call("hset", KEYS[1], ARGV[1], ARGV[2])
else
	return -1
end
`

	res, err := s.client.Eval(ctx, lua, []string{key}, key, val).Int()
	if err != nil {
		return err
	}
	if res < 0 {
		return errSessionNotFound
	}
	return nil
}

func (s *Session) ID() string {
	return s.id
}

func redisKey(prefix,id string)string{
	return fmt.Sprintf("%s-%s",prefix,id)
}