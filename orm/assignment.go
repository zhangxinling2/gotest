package orm

type Assignment struct {
	col string
	val any
}
func(a Assignment)assigns(){}
func Assign(col string,val any)Assignment{
	return Assignment{
		col: col,
		val: val,
	}
}