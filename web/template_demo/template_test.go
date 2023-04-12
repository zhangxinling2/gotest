package template_demo

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"html/template"
	"testing"
)

func TestHelloWorld(t *testing.T){
	type User struct {
		Name string
	}
	//使用{{}}来包裹模板语法
	//使用.访问数据 .代表的是当前作用域的当前对象，类似于java的this,python的self
	//.可以是普通结构体，指针，map,切片
	//去除空格和换行: -，注意和别的元素用空格分开
	//声明遍历：如同go语言，但是用$来表示。$xxx:= some_value
	//执行方法调用:形式"调用者.方法 参数1 参数2"
	//创建一个模板实例
	tpl:=template.New("hello world")
	//解析模板(预编译模板):传入的参数是模板的具体内容
	tpl,err:=tpl.Parse(`Hello, {{.Name}}`)
	require.NoError(t, err)
	buffer:=&bytes.Buffer{}
	//传入数据:参数作为模板渲染所使用的数据
	err=tpl.Execute(buffer,User{Name: "Tom"})
	assert.Equal(t, `Hello, Tom`,buffer.String())
}

func TestSlice(t *testing.T){
	type User struct {
		Name string
	}
	//使用{{}}来包裹模板语法
	//使用.访问数据 .代表的是当前作用域的当前对象，类似于java的this,python的self
	//.可以是普通结构体，指针，map,切片
	//创建一个模板实例
	tpl:=template.New("hello world")
	//解析模板(预编译模板):传入的参数是模板的具体内容
	tpl,err:=tpl.Parse(`Hello, {{index . 0}}`)
	require.NoError(t, err)
	buffer:=&bytes.Buffer{}
	//传入数据:参数作为模板渲染所使用的数据
	err=tpl.Execute(buffer,[]string{ "Tom"})
	assert.Equal(t, `Hello, Tom`,buffer.String())
}

func TestBasic(t *testing.T){
	type User struct {
		Name string
	}
	//使用{{}}来包裹模板语法
	//使用.访问数据 .代表的是当前作用域的当前对象，类似于java的this,python的self
	//.可以是普通结构体，指针，map,切片
	//创建一个模板实例
	tpl:=template.New("hello world")
	//解析模板(预编译模板):传入的参数是模板的具体内容
	tpl,err:=tpl.Parse(`Hello, {{. }}`)
	require.NoError(t, err)
	buffer:=&bytes.Buffer{}
	//传入数据:参数作为模板渲染所使用的数据
	err=tpl.Execute(buffer,123)
	assert.Equal(t, `Hello, 123`,buffer.String())
}

func TestFuncCall(t *testing.T){

	tpl:=template.New("hello world")
	//解析模板(预编译模板):传入的参数是模板的具体内容
	tpl,err:=tpl.Parse(`
切片长度：{{len .Slice}}
{{printf "%.2f" 1.2345}}
Hello, {{.Hello "Tom" "Jerry"}}`)
	require.NoError(t, err)
	buffer:=&bytes.Buffer{}
	//传入数据:参数作为模板渲染所使用的数据
	err=tpl.Execute(buffer,FuncCall{
		Slice: []string{"a","b"},
	})
	assert.Equal(t, `
切片长度：2
1.23
Hello, Tom·Jerry`,buffer.String())
}

type FuncCall struct {
	Slice []string
}

func(f FuncCall)Hello(firstname,lastname string)string{
	return fmt.Sprintf("%s·%s",firstname,lastname)
}

func TestForLoop(t *testing.T){
	tpl:=template.New("hello world")
	//解析模板(预编译模板):传入的参数是模板的具体内容
	tpl,err:=tpl.Parse(`
{{- range $idx, $elem := .Slice}}
{{- .}}
{{$idx}}-{{$elem}}
{{end}}
`)
	require.NoError(t, err)
	buffer:=&bytes.Buffer{}
	//传入数据:参数作为模板渲染所使用的数据
	err=tpl.Execute(buffer,FuncCall{
		Slice: []string{"a","b"},
	})
	assert.Equal(t, `a
0-a
b
1-b

`,buffer.String())
}


func TestIfElse(t *testing.T){
	type User struct {
		Age int
	}
	tpl:=template.New("hello world")
	//解析模板(预编译模板):传入的参数是模板的具体内容
	tpl,err:=tpl.Parse(`
{{- if and (gt .Age 0) (le .Age 6)}}
我是儿童: (0,6]
{{ else if and (gt .Age 6) (le .Age 18)}}
我是少年: (6,18]
{{ else }}
我是成人: >18
{{end -}}
`)
	require.NoError(t, err)
	buffer:=&bytes.Buffer{}
	//传入数据:参数作为模板渲染所使用的数据
	err=tpl.Execute(buffer,User{
		Age: 13,
	})
	assert.Equal(t, `\n我是少年: (6,18]\n"`,buffer.String())
}