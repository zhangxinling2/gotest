package web

import "strings"

//用来支持对路由树的操作
//代表路由树
type router struct {
	//Beego Gin Http Method对应一棵树


	//http method=>路由树根节点
	trees map[string]*node
}
func newRouter() router{
	return router{
		trees: map[string]*node{},
	}
}

//AddRoute 加一些限制：
//path必须以/开头，不能以/结尾，中间也不能有连续的//
func (r *router) addRoute(method string,path string,handleFunc HandleFunc)  {
	if path==""{
		panic("web 路径不能为空")
	}
	//首先找到树
	root ,ok:=r.trees[method]
	if!ok{
		//说明还没有根节点
		root=&node{
			path: "/",
		}
		r.trees[method]=root
	}
	// 开头不能没有/
	if path[0]!='/'{
		panic("web:路径必须以/开头")
	}
	//结尾
	if path!="/"&&path[len(path)-1]=='/'{
		panic("web:路径不能以/结尾")
	}
	//中间连续 可以用strings.contains("//")

	//根节点特殊处理
	if path == "/"{
		if root.handler!=nil{
			panic("web:根节点重复注册")
		}
		root.handler=handleFunc
		root.route="/"
		return
	}
	//切割这个path
	segs := strings.Split(path[1:],"/")
	for _,seg:=range segs{
		if seg==""{
			panic("web:路径中间不能有连续的/")
		}
		//递归下去找准位置
		//如果中途有节点不存在则需要创建
		children:=root.childOrCreate(seg)
		root = children
	}
	if root.handler!=nil{
		panic("web:普通节点重复注册")
	}
	root.handler=handleFunc
	root.route=path
}

func (r *router) findRoute(method string,path string)(*matchInfo,bool){
	//一层一层遍历下去
	root,ok:=r.trees[method]
	if !ok{
		return nil,false
	}
	if path=="/"{
		return &matchInfo{
			n: root,
		},true
	}
	//去除前后/
	path=strings.Trim(path,"/")
	segs :=strings.Split(path,"/")
	var pathParams map[string]string
	for _,seg:=range segs{
		//找到child
		child,paramChild,found:=root.childOf(seg)
		if !found{
			return nil,false
		}
		//命中路由参数
		if paramChild{
			if pathParams==nil{
				pathParams=make(map[string]string)
			}
			//path是 :id 形式
			pathParams[child.path[1:]]=seg
		}
		root=child
	}

	//代表确实有这个节点，但是不是用户注册的不知道
	return &matchInfo{
		n:root,
		pathParams:pathParams,
	},true
	//return root,root.handler!=nil
}

//childOf 优先考虑静态匹配，匹配不上再考虑通配符匹配
//第一个返回值是子节点
//第二个是标记是否是路径参数
//第三个标记命中了没有
func(n *node)childOf(path string)(*node,bool,bool){
	if n.children==nil{
		if n.paramChild!=nil{
			return n.paramChild,true,true
		}
		return n.starChild,false,n.starChild!=nil
	}
	child,ok:=n.children[path]
	if !ok{
		if n.paramChild!=nil{
			return n.paramChild,true,true
		}
		return n.starChild,false,n.starChild!=nil
	}
	return child,false,ok
}
//第一个返回值是正确的子节点
func (n *node) childOrCreate(seg string) *node {
	if seg[0]==':'{
		if n.starChild!=nil{
			panic("web：不允许同时注册路径参数和通配符匹配，已有通配符匹配")
		}
		n.paramChild=&node{
			path: seg,
		}
		return n.paramChild
	}
	if seg=="*"{
		if n.paramChild!=nil{
			panic("web：不允许同时注册路径参数和通配符匹配，已有路径参数匹配")
		}
		n.starChild=&node{
			path: seg,
		}
		return n.starChild
	}
	if n.children==nil{
		n.children= map[string]*node{}
	}
	res,ok:=n.children[seg]
	if !ok{
		//要新建一个
		res=&node{
			path: seg,
		}
		n.children[seg]=res
	}
	return res
}

type node struct {
	route string

	path string

	//静态匹配的节点
	//子path到子节点的映射
	children map[string]*node

	//加一个通配符“*”匹配
	starChild *node
	//路径参数
	paramChild *node
	//命中路由后的逻辑   代表用户注册的业务逻辑
	handler HandleFunc
}

type matchInfo struct {
	n *node
	pathParams map[string]string
}