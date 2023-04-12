package memory

import (
	"context"
	"errors"
	"gotest/web/session"
	"sync"
	cache "github.com/patrickmn/go-cache"
	"time"
)

var(
	// ErrKeyNotFound sentinel error:与定义错误,在这定义独属于memory
	ErrKeyNotFound = errors.New("session:找不到key")
	errorSessionNotFound = errors.New("")
)

//Store 管理Session本身
type Store struct {
	mutex sync.RWMutex
	sessions *cache.Cache
	expiration time.Duration
}

func NewStore(expiration time.Duration)*Store{
	return &Store{
		sessions: cache.New(expiration,time.Second),
		expiration:expiration,
	}
}

func (s *Store) Generate(ctx context.Context, id string) (session.Session, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	sess:= &Session{
		id:id,
		values: sync.Map{},
	}
	s.sessions.Set(id,sess,s.expiration)
	return sess,nil
}

func (s *Store) Refresh(ctx context.Context, id string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	val,ok:=s.sessions.Get(id)
	if !ok{
		return errors.New("session: 该id对应的session不存在")
	}
	s.sessions.Set(id,val,s.expiration)
	return nil
}

func (s *Store) Remove(ctx context.Context, id string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.sessions.Delete(id)
	return nil
}

func (s *Store) Get(ctx context.Context, id string) (session.Session, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	sess,ok:= s.sessions.Get(id)
	if !ok{
		return nil,errorSessionNotFound
	}
	return sess.(*Session),nil
}

type Session struct {
	//控制性非常强
	// mutex sync.RWMutex
	// values map[string]any

	//控制性弱，支持简单操作
	id string
	values sync.Map
}

func (s *Session) Get(ctx context.Context, key string) (any, error) {
	val,ok:=s.values.Load(key)
	if !ok{
		//return nil,fmt.Errorf("%w, key %s",ErrKeyNotFound,key)
		return nil,ErrKeyNotFound
	}
	return val,nil
}

func (s *Session) Set(ctx context.Context, key string, val string) error {
	s.values.Store(key,val)
	return nil
}

func (s *Session) ID() string {
	return s.id
}
