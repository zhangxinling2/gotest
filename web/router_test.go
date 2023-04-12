package web

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"reflect"
	"testing"
)

func TestRouter_AddRoute(t *testing.T) {
	//第一个步骤 构造路由树
	//第二个步骤 校验路由树
	testRoutes :=[]struct{
		method string
		path string
	}{
		{
			method: http.MethodGet,
			path:"/",
		},
		{
			method: http.MethodGet,
			path:   "/user",
		},
		{
			method: http.MethodGet,
			path:"/user/home",
		},
		{
			method: http.MethodGet,
			path:   "/order/detail",
		},
		{
			method: http.MethodGet,
			path:   "/order/detail/:id",
		},
		{
			method: http.MethodGet,
			path:   "/order/*",
		},
		{
			method: http.MethodPost,
			path:   "/order/create",
		},
		{
			method: http.MethodPost,
			path:   "/login",
		},
	}
	var mockHandler HandleFunc = func(ctx *Context) {

	}
	r:=newRouter()
	for _,route:=range testRoutes{
		r.addRoute(route.method,route.path,mockHandler)
	}

	//在这断言两者相等 不能使用assert.Equal，因为handlerFunc不可比
	//预期中的路由树
	wantRouter :=&router{
		trees: map[string]*node{
			http.MethodGet: &node{
				path: "/",
				handler: mockHandler,
				children: map[string]*node{
					"user":&node{
						path: "user",
						handler: mockHandler,
						children: map[string]*node{
							"home":&node{
								path:"home",
								handler: mockHandler,
							},
						},
					},
					"order":&node{
						path: "order",
						children: map[string]*node{
							"detail":&node{
								path: "detail",
								handler: mockHandler,
								paramChild: &node{
									path: ":id",
									handler: mockHandler,
								},
							},

						},
						starChild: &node{
							path: "*",
							handler: mockHandler,
						},
					},
				},
			},
			http.MethodPost: &node{
				path:"/",
				children: map[string]*node{
					"order":&node{
						path: "order",
						children: map[string]*node{
							"create":&node{
								path: "create",
								handler: mockHandler,
							},
						},
					},
					"login":&node{
						path: "login",
						handler: mockHandler,
					},
				},
			},
		},
	}
	msg,ok:=wantRouter.equal(&r)
	assert.True(t, ok,msg)
	r=newRouter()
	assert.Panics(t, func() {
		r.addRoute(http.MethodGet,"",mockHandler)
	})
	assert.Panicsf(t, func() {
		r.addRoute(http.MethodGet,"/a/b/c/",mockHandler)
	},"web:路径不能以/结尾")
	assert.Panicsf(t, func() {
		r.addRoute(http.MethodGet,"abc",mockHandler)
	},"web:路径必须以/开头")
	r=newRouter()
	r.addRoute(http.MethodGet,"/",mockHandler)
	assert.Panicsf(t, func() {
		r.addRoute(http.MethodGet,"/",mockHandler)
	},"web:重复注册")
	r=newRouter()
	r.addRoute(http.MethodGet,"/a/*",mockHandler)
	assert.Panicsf(t, func() {
		r.addRoute(http.MethodGet,"/a/:id",mockHandler)
	},"web:重复注册")
	r.addRoute(http.MethodGet,"/b/:id",mockHandler)
	assert.Panicsf(t, func() {
		r.addRoute(http.MethodGet,"/b/*",mockHandler)
	},"web:重复注册")
}
//equal string是一个错误信息 帮助排查问题
func(r *router)equal(y *router) (string,bool){
	for k,v:=range r.trees{
		dst,ok:=y.trees[k]
		if !ok{
			return fmt.Sprintf("找不到对应的http method"),false
		}
		//v ,dst 要相等
		msg,equal:=v.equal(dst)
		if !equal{
			return msg,false
		}
	}
	return "",true
}

func(n *node) equal(y *node) (string,bool){
	if n.path!=y.path{
		return fmt.Sprintf("节点路径不匹配"),false
	}
	if len(n.children)!=len(y.children){
		return fmt.Sprintf("子节点数量不相等"),false
	}
	if n.starChild!=nil{
		msg,ok:= n.starChild.equal(y.starChild)
		if !ok{
			return msg,ok
		}
	}
	if n.paramChild!=nil{
		msg,ok:= n.paramChild.equal(y.paramChild)
		if !ok{
			return msg,ok
		}
	}
	//比较 handler 需要使用反射
	nHandler:=reflect.ValueOf(n.handler)
	yHandler:=reflect.ValueOf(y.handler)
	if nHandler!=yHandler{
		return fmt.Sprintf("handler 不相等"),false
	}
	for path,c:=range n.children{
		dst,ok:=y.children[path]
		if !ok{
			return fmt.Sprintf("子节点%s不存在",path),false
		}
		msg,ok:=c.equal(dst)
		if !ok{
			return msg,false
		}
	}
	return "",true
}

func TestRoute_findRoute(t *testing.T){
	testRoute :=[]struct{
		method string
		path string
	}{
		{
			method: http.MethodDelete,
			path:   "/",
		},
		{
			method: http.MethodGet,
			path:   "/",
		},
		{
			method: http.MethodGet,
			path:   "/*/*",
		},
		{
			method: http.MethodGet,
			path:   "/user",
		},
		{
			method: http.MethodGet,
			path:   "/user/home",
		},
		{
			method: http.MethodGet,
			path:   "/order/detail",
		},

		{
			method: http.MethodPost,
			path:   "/login",
		},
		{
			method: http.MethodPost,
			path:   "/login/:username",
		},
	}
	r:=newRouter()
	var mockHanlder HandleFunc= func(ctx *Context) {

	}
	for _,route:=range testRoute{
		r.addRoute(route.method,route.path,mockHanlder)
	}

	testCases:=[]struct{
		name string
		method string
		path string

		wantFound bool
		info *matchInfo
	}{
		{
			//方法都不存在
			name:"method not found",
			method: http.MethodOptions,
			path: "/order/detail",
		},
		{
			//根节点
			name:"root",
			method: http.MethodDelete,
			path: "/",
			wantFound: true,
			info: &matchInfo{
				n:&node{
					path: "/",
					handler: mockHanlder,
				},
				//children: map[string]*node{
				//	"order":&node{
				//		path: "order",
				//		children: map[string]*node{
				//			"detail":&node{
				//				path: "detail",
				//				handler: mockHanlder,
				//			},
				//		},
				//	},
				//},
			},
		},
		{
			//命中但没有handler
			name:"命中但没有handler",
			method: http.MethodGet,
			path: "/order",
			wantFound: true,
			info : &matchInfo{
				n: &node{
					path: "order",
					children: map[string]*node{
						"detail":&node{
							path: "detail",
							handler: mockHanlder,
						},
					},
				},
			},
		},
		{
			//完全命中
			name:"order detail",
			method: http.MethodGet,
			path: "/order/detail",
			wantFound: true,
			info:&matchInfo{
				n: &node{
					path: "detail",
					handler: mockHanlder,
				},
			},
		},
		{
			//username路径参数匹配
			name:"login username",
			method: http.MethodPost,
			path: "/login/daming",
			wantFound: true,
			info:&matchInfo{
				n: &node{
					path: ":username",
					handler: mockHanlder,
				},
				pathParams: map[string]string{
					"username":"daming",
				},
			},
		},
	}
	for _,tc:=range testCases{
		t.Run(tc.name, func(t *testing.T) {
			info,found:=r.findRoute(tc.method,tc.path)
			assert.Equal(t, tc.wantFound,found)
			if !found{
				return
			}
			//因为node 里有func所以不能直接比
			assert.Equal(t, tc.info.pathParams,info.pathParams)
			msg,ok:=tc.info.n.equal(info.n)
			assert.True(t,ok,msg)
		})
	}
}
