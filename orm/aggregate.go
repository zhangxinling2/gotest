package orm

// Aggregate 代表聚合函数
// 聚合函数就是一个函数名一个字段名AVG("age"),SUM("age"),COUNT("age"),MAX("age"),MIN("age")
type Aggregate struct {
	fn string
	arg string
	alias string
}

func(a Aggregate)selectable(){

}

func (a Aggregate)As(alias string)Aggregate{
	return Aggregate{
		fn: a.fn,
		arg: a.arg,
		alias: alias,
	}
}
func Avg(col string)Aggregate{
	return Aggregate{
		fn: "AVG",
		arg: col,
	}
}
func Sum(col string)Aggregate{
	return Aggregate{
		fn: "SUM",
		arg: col,
	}
}
func Count(col string)Aggregate{
	return Aggregate{
		fn: "COUNT",
		arg: col,
	}
}
func Max(col string)Aggregate{
	return Aggregate{
		fn: "MAX",
		arg: col,
	}
}
func Min(col string)Aggregate{
	return Aggregate{
		fn: "MIN",
		arg: col,
	}
}