package web

import (
	"bytes"
	"context"
	"html/template"
)
// Engine只关注渲染，添加之类的交给具体的模板
type TemplateEngine interface {
	//Render 渲染页面
	//tplName 模板的名字，按名索引
	//data 渲染页面所需要的数据
	Render(ctx context.Context,tplName string,data any)([]byte,error)

	// 渲染页面，数据写入到Writer中
	// Render(ctx,"aa",map[]{},responseWriter)
	// 不太好测试，直接使用writer会写进tcp中
	//Render(ctx context.Context,tplName string,data any,write io.Write)error
	//用这个context会使engine和context耦合在一起
	//Render(ctx context.Context)
}

type GoTemplateEngine struct{
	T *template.Template
}

func (g *GoTemplateEngine)Render(ctx context.Context,tplName string,data any)([]byte,error){
	bs:=&bytes.Buffer{}
	err:=g.T.ExecuteTemplate(bs,tplName,data)
	return bs.Bytes(),err
}