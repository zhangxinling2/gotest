package reflectTest

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func CheckType(v interface{}){
	t:=reflect.TypeOf(v)
	switch t.Kind() {
	case reflect.Float32,reflect.Float64:
		fmt.Println("float")
	case reflect.Int,reflect.Int32,reflect.Int64:
		fmt.Println("Int")
	}
}
func TestType(t *testing.T) {
	CheckType(1.234)
}

type Employee struct {
	EmployeeId string
	Name string `format:"normal"`//结构体的tag 可用于内置json的解析
	Age int
}
func (e *Employee)UpdateAge(newVal int){
	e.Age=newVal
}

type Costumer struct {
	CookieId string
	Name string
	Age int
}
func TestInvokeByName(t *testing.T){
	e:=&Employee{"1","Mike",30}
	//按名字获取成员
	t.Logf("Name:Value(%[1]v),Type(%[1]T)",reflect.ValueOf(*e).FieldByName("Name"))
	if nameField,ok:=reflect.TypeOf(*e).FieldByName("Name");!ok{
		t.Error("Failed to get 'Name' field")
	}else{
		t.Log("Tag:format",nameField.Tag.Get("format"))
	}
	reflect.ValueOf(e).MethodByName("UpdateAge").Call([]reflect.Value{reflect.ValueOf(1)})
	t.Log("update age:",e)
}
func fillBySettings(st interface{},settings map[string]interface{})error{
	if reflect.TypeOf(st).Kind()!=reflect.Ptr{
		if reflect.TypeOf(st).Elem().Kind()!=reflect.Struct{
			return errors.New("the first param must be a pointer or a struct ")
		}
	}
	var (
		field reflect.StructField
		ok bool
	)
	for k,v:=range settings{
		if field,ok =(reflect.ValueOf(st)).Type().FieldByName(k);!ok{
			continue
		}
		if field.Type==reflect.TypeOf(v){
			vstr:=reflect.ValueOf(st)
			vstr=vstr.Elem()
			vstr.FieldByName(k).Set(reflect.ValueOf(v))
		}
	}
	return nil
}
func TestFillNameAndAge(t *testing.T) {
	settings := map[string]interface{}{"Name":"Mike","Age":40}
	e:=Employee{}
	if err:=fillBySettings(&e,settings);err!=nil{
		t.Fatal(err)
	}
	t.Log(e)
	c:=new(Costumer)
	if err:=fillBySettings(c,settings);err!=nil{
		t.Fatal(err)
	}
	t.Log(*c)
}