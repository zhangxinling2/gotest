package TestType
type User2 struct {
	Name string
	age int
}

func NewUser(Name string,Age int)User2{
	return User2{
		Name: Name,
		age: Age,
	}
}
