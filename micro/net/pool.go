package net

import (
	"context"
	"net"
	"sync"
	"time"
)

//Pool 线程池
type Pool struct {
	lock sync.RWMutex
	//最大连接数
	maxCnt int
	//当前连接数
	cnt int
	//请求队列,可以用channel,用channel要确认最大请求数
	connReq []connReq
	//空闲队列
	idleConn chan *connInfo
	//最大空闲连接数
	maxIdleCnt int
	//初始连接数
	initCnt int
	//建立连接的工厂
	factory func() (net.Conn, error)
	//空闲连接过期时间
	expiration time.Duration
}

type connInfo struct {
	conn     net.Conn
	backTime time.Time
}
type connReq struct {
	conn chan net.Conn
}

//NewPool 创建一个线程池
func NewPool(maxCnt, maxIdleCnt, initCnt int, expiration time.Duration, factory func() (net.Conn, error)) (*Pool, error) {
	pool := &Pool{
		lock:       sync.RWMutex{},
		maxCnt:     maxCnt,
		cnt:        0,
		idleConn:   make(chan *connInfo, maxIdleCnt),
		maxIdleCnt: maxIdleCnt,
		initCnt:    initCnt,
		factory:    factory,
		expiration: expiration,
	}
	//创建初始等待队列
	for i := 0; i < initCnt; i++ {
		conn, err := factory()
		if err != nil {
			return nil, err
		}
		newConnInfo := &connInfo{
			conn:     conn,
			backTime: time.Now(),
		}
		pool.idleConn <- newConnInfo
	}
	return pool, nil
}

//Get 获得一个连接
func (p *Pool) Get(ctx context.Context) (net.Conn, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	//先判断空闲队列中有无空闲连接
	select {
	//有空闲连接就从空闲连接中取
	case conn := <-p.idleConn:
		//取出来要判断连接是否超时
		if conn.backTime.Add(p.expiration).Before(time.Now()) {
			//超时了就关闭连接
			conn.conn.Close()
		}
		return conn.conn, nil
	default:
		p.lock.Lock()
		//没有空闲连接
		//看连接满了没有，满了就加入等待队列
		if p.cnt >= p.maxCnt {
			newReq := connReq{conn: make(chan net.Conn, 1)}
			p.connReq = append(p.connReq, newReq)
			p.lock.Unlock()
			//等待别人归还
			select {
			case <-ctx.Done():
				//要么删除，要么转发
				go func() {
					c := <-newReq.conn
					_ = p.Put(context.Background(), c)
				}()
				return nil, ctx.Err()
			case c := <-newReq.conn:
				return c, nil
			}

		}
		c, err := p.factory()
		if err != nil {
			return nil, err
		}
		p.cnt++
		p.lock.Unlock()
		return c, err
	}
}
func (p *Pool) Put(ctx context.Context, c net.Conn) error {
	p.lock.Lock()
	//有阻塞的请求,直接把连接给它
	if len(p.connReq) > 0 {
		req := p.connReq[0]
		p.connReq = p.connReq[1:]
		p.lock.Unlock()
		req.conn <- c
		return nil
	}
	defer p.lock.Unlock()
	idle := &connInfo{
		conn:     c,
		backTime: time.Now(),
	}
	select {
	//连接放入空闲队列
	case p.idleConn <- idle:
	default:
		c.Close()
	}
	return nil
}
