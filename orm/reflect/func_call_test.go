package reflect

import (
	"github.com/stretchr/testify/assert"
	"gotest/orm/reflect/types"
	"reflect"
	"testing"
)


func TestIterateFunc(t *testing.T) {
	testCases:=[]struct{
		name string
		entity any
		wantRes map[string]FunInfo
		wantErr error
	}{
		{
			name: "struct",
			entity: types.NewUser("Tom",18),
			wantRes: map[string]FunInfo{
				"GetAge":{
					Name: "GetAge",
					//InputTypes: nil,
					//因为在go看来func (u User)GetAge()等价于func GetAge(u User)
					InputTypes: []reflect.Type{reflect.TypeOf(types.User{})},
					OutputTypes:[]reflect.Type{reflect.TypeOf(0)},
					Result: []any{18},
				},
				//"ChangeName":{
				//	Name: "ChangeName",
				//	InputTypes: []reflect.Type{reflect.TypeOf("")},
				//},
			},

		},

		{
			name: "pointer",
			entity: types.NewUserPtr("Tom",18),
			wantRes: map[string]FunInfo{
				"GetAge":{
					Name: "GetAge",
					//InputTypes: nil,
					//因为在go看来func (u User)GetAge()等价于func GetAge(u User)
					InputTypes: []reflect.Type{reflect.TypeOf(&types.User{})},
					OutputTypes:[]reflect.Type{reflect.TypeOf(0)},
					Result: []any{18},
				},
				"ChangeName":{
					Name: "ChangeName",
					InputTypes: []reflect.Type{reflect.TypeOf(&types.User{}),reflect.TypeOf("")},
					OutputTypes:[]reflect.Type{},
					Result: []any{},
				},
			},

		},
	}
	for _,tc:=range testCases{
		t.Run(tc.name, func(t *testing.T) {
			res,err:=IterateFunc(tc.entity)
			assert.Equal(t, tc.wantErr,err)
			if err!=nil{
				return
			}
			assert.Equal(t, tc.wantRes,res)
		})
	}
}