package ast

import (
	"github.com/stretchr/testify/require"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestPrintVisitor_Visit(t *testing.T) {
	//就是编译原理中的token,高级程序语言中的最小的单元,分隔符(界限符) 关键字和保留字 标识符 操作符 字面值
	fset:=token.NewFileSet()
	//解析源代码
	f,err:=parser.ParseFile(fset,"src.go",`
package ast

import (
	"fmt"
	"go/ast"
	"log"
	"reflect"
)

type PrintVisitor struct {
	
}

func (p PrintVisitor) Visit(node ast.Node) (w ast.Visitor) {
	if node==nil{
		fmt.Println(nil)
		return p
	}
	typ:=reflect.TypeOf(node)
	val:=reflect.ValueOf(node)
	for typ.Kind()==reflect.Ptr{
		typ=typ.Elem()
		val=val.Elem()
	}

	log.Printf("val: %+v, typ %s,\n",val.Interface(),typ.Name())
	return p
}
`,parser.ParseComments)
	require.NoError(t, err)
	v:=&PrintVisitor{}
	ast.Walk(v,f)
}