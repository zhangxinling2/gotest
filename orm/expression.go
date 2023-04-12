package orm
// Expression 是一个标记接口(因为expr()没有任何含义)，代表表达式
type Expression interface {
	expr()
}
//RawExpr 代表的是原生表达式
type RawExpr struct {
	raw  string
	args []any//考虑把RawExpr用到where的部分
}

func Raw(expr  string,args...any)RawExpr{
	return RawExpr{
		raw:  expr,
		args: args,
	}
}

func(r RawExpr)selectable(){}
func(r RawExpr)expr(){}

func(r RawExpr) AsPredicate()Predicate{
	return Predicate{
		left: r,
	}
}