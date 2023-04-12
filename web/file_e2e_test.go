//go:build e2e
package web

import (
	"github.com/stretchr/testify/require"
	"html/template"
	"log"
	"mime/multipart"
	"path/filepath"
	"testing"
)

func TestUpLoad(t *testing.T){
	//tpl:=template.New("template-demo")
	tpl,err:=template.ParseGlob("testdata/tpls/*.gohtml")
	require.NoError(t, err)
	engine:=&GoTemplateEngine{
		T:tpl,
	}
	h:=NewHTTPServer(ServerWithTemplateEngine(engine))
	h.Get("/upload", func(ctx *Context) {
		err:=ctx.Render("upload.gohtml",nil)
		if err!=nil{
			log.Println(err)
		}
	})
	fu:=FileUpLoader{
		FileField: "myfile",
		DstPathFunc: func(header *multipart.FileHeader) string {
			return filepath.Join("testdata","upload",header.Filename)
		},
	}
	h.Post("/upload", fu.Handle())
	h.Start(":8081")
}
func TestDownLoad(t *testing.T){
	h:=NewHTTPServer()

	fu:=FileDownLoader{
		Dir: filepath.Join("testdata","download"),
	}
	h.Get("/download", fu.Handle())
	h.Start(":8081")
}

func TestStaticResourceHandler_Handle(t *testing.T){
	h:=NewHTTPServer()

	s,err:=NewStaticResourceHandler(filepath.Join("testdata","static"))
	require.NoError(t, err)
	//localhost:8081/static/xxx.jpg
	h.Get("/static/:file", s.Handle)
	h.Start(":8081")
}