package main
//"html/template"会帮你转义引号导致产生&#34;，所以更改为text的
import (
	_ "embed"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"text/template"
)

func main(){
	//f:=os.OpenFile("testdata/user.gen.go",)
	//gen(f,"testdata/user.go")
}
//go:embed tpl.gohtml
var genOrm string
// gen 调用这个方法来生成代码
//放入io.Writer,不必生成.gen.go，可以使用io.Write直接进行测试
func gen(w io.Writer,srcFile string)error{
	//ast语法树解析
	fset:=token.NewFileSet()
	f,err:=parser.ParseFile(fset,srcFile,nil,parser.ParseComments)
	if err!=nil{
		return err
	}
	s:=&SingleFileEntryVisitor{}
	ast.Walk(s,f)
	file:=s.Get()


	//操作模板
	tpl:=template.New("gen-orm")
	tpl,err=tpl.Parse(genOrm)
	if err!=nil{
		return err
	}
	return tpl.Execute(w,Data{
		File:file,
		Ops: []string{"Lt","Gt"},
	})

}

type Data struct {
	*File
	Ops []string
}
