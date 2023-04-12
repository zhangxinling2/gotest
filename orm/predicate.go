package orm


//衍生类型，由string衍生过来
type op string

const (
	opEq op="="
	opLT op="<"
	opGT op=">"
	opNot op="NOT"
	opAnd op="AND"
	opOr op="OR"
)
func(o op)String()string {
	return string(o)
}
//别名
//type op=string
type Predicate struct {
	left Expression
	op op
	right Expression
}



// Not(C("name")).Eq("Tom")
func Not(p Predicate)Predicate{
	return Predicate{
		op:opNot,
		right: p,
	}
}

// C("id").Eq(12).And(C("name")).Eq("Tom")
func (left Predicate)And(right Predicate)Predicate{
	return Predicate{
		left:left,//这里应该是column,所以引入一个顶级抽象expression,修改设计
		op:opAnd,
		right: right,
	}
}

// C("id").Eq(12).Or(C("name")).Eq("Tom")
func (left Predicate)Or(right Predicate)Predicate{
	return Predicate{
		left:left,
		op:opOr,
		right: right,
	}
}

//让Predicate实现expression,标记Predicate是一个表达式
func (p Predicate)expr(){}

type value struct {
	val any
}

func(value)expr(){}

















//type Predicate struct {
//	c Column
//	op op
//	arg any
//}
////Eq("id",12)
////func Eq(column Column,arg any)Predicate{
////	return Predicate{
////		c: column,
////		op: "=",
////		arg: arg,
////	}
////}
////此种形态链式调用
//type Column struct {
//	name string
//}
//func C(name string)Column{
//	return Column{
//		name: name,
//	}
//}
////C("id").Eq(12)
//func(c Column)Eq(arg any)Predicate{
//	return Predicate{
//		c: c,
//		op: opEq,
//		arg: arg,
//	}
//}
//// Not(C("name")).Eq("Tom")
//func Not(p Predicate)Predicate{
//	return Predicate{
//		op:opNot,
//		arg: p,
//	}
//}
//
//// C("id").Eq(12).And(C("name")).Eq("Tom")
//func (left Predicate)And(right Predicate)Predicate{
//	return Predicate{
//		c:left,//这里应该是column,所以引入一个顶级抽象expression,修改设计
//		op:opAnd,
//		arg: right,
//	}
//}
//
//// C("id").Eq(12).Or(C("name")).Eq("Tom")
//func (left Predicate)Or(right Predicate)Predicate{
//	return Predicate{
//		c:left,
//		op:opOr,
//		arg: right,
//	}
//}