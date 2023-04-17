package sync

import "sync"

type MyBiz struct {
	once sync.Once
}

// Init 接收器一定要是指针，因为结构体中不是指针，不然会引起复制。
func (m *MyBiz) Init() {
	m.once.Do(func() {

	})
}

//singleton 单例模式，确保只初始化一次,一般和接口结合在一起使用
type singleton struct {
}

func (s singleton) DoSomething() {
}

type MyBusiness interface {
	DoSomething()
}

var s *singleton

var singletonOnce = sync.Once{}

// 懒加载
func GetSingleton() MyBusiness {
	singletonOnce.Do(func() {
		s = &singleton{}
	})
	return s
}

// 饥饿
func init() {
	//用包初始化函数取代 once
	s = &singleton{}
}
