package unsafe

import (
	"fmt"
	"reflect"
)

func PrintFieldOffset(entity any){
	typ:=reflect.TypeOf(entity)
	numField:=typ.NumField()
	for i := 0; i < numField; i++ {
		field:=typ.Field(i)
		fmt.Println(field.Offset)
	}
}
