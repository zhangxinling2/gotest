package encapsulation

import (
	"fmt"
	"testing"
	"unsafe"
)

type Employee struct {
	Id string
	Name string
	Age int
}
func (e Employee)String()string{
	fmt.Printf("%x",unsafe.Pointer(&e.Name))
	return fmt.Sprintf("%s %s %d",e.Id,e.Name,e.Age)
}

func (e *Employee)String2()string{
	fmt.Printf("%x",unsafe.Pointer(&e.Name))
	return fmt.Sprintf("%s %s %d",e.Id,e.Name,e.Age)
}
func TestCreateEmployee(t *testing.T){
	e:=Employee{"0","Bob",20}
	e1:=Employee{Id: "1",Name: "Mike",Age: 20}
	e2:=new(Employee)
	e2.Id="2"
	e2.Name="Rose"
	e2.Age=20
	fmt.Printf("%x",unsafe.Pointer(&e.Name))
	t.Log(e1)
	t.Log(e)
	t.Log(e2)
	t.Log(e.String())
	t.Log(e.String2())
}