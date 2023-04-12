package orm
//Column 此种形态链式调用
type Column struct {
	table TableReference//表示代表的是哪个table
	name string
	alias string
}
func C(name string)Column{
	return Column{
		name: name,
	}
}
func(c Column)assigns(){

}
//As 此种是不可变设计(不使用指针，Column是不可变对象)可稍稍减少内存逃逸的概率
func (c Column)As(alias string)Column{
	return Column{
		name: c.name,
		alias: alias,
		table: c.table,
	}
}
//Eq C("id").Eq(12)
func(c Column)Eq(arg any)Predicate{
	return Predicate{
		left: c,
		op: opEq,
		right: ValueOf(arg),
	}
}
func ValueOf(arg any)Expression{
	switch val:=arg.(type) {
	case Expression:
		return val
	default:
		return value{val:val}
	}
}
func(c Column)Lt(arg any)Predicate{
	return Predicate{
		left: c,
		op: opLT,
		right: value{val: arg},
	}
}
func(c Column)Gt(arg any)Predicate{
	return Predicate{
		left: c,
		op: opGT,
		right: value{val: arg},
	}
}
//标记Column是一个表达式
func(c Column) expr(){}
//标记Column是可选列
func(c Column) selectable(){}
