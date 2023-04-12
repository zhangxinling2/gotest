package model

import (
	"database/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gotest/orm/internal/errs"
	"reflect"
	"testing"
)

func  Test_register_Register(t *testing.T){
	testCases:=[]struct{
		name string
		entity any
		wantModel *Model

		wantErr error
	}{
		{
			name:    "test Model",
			entity:  TestModel{},
			wantErr: errs.ErrPointOnly,
		},
		{
			name: "map",
			entity: map[string]string{},
			wantErr: errs.ErrPointOnly,
		},

		{
			name: "pointer",
			entity: &TestModel{},
			wantModel: &Model{
				TableName: "test_model",
				Fields: []*Field{
					{
						ColName: "id",
						GoName:  "Id",
						Type:    reflect.TypeOf(int64(0)),
						Offset: 0,
					},
					{
						ColName: "first_name",
						GoName:  "FirstName",
						Type:    reflect.TypeOf(""),
						Offset: 8,
					},
					{
						ColName: "age",
						GoName:  "Age",
						Type:    reflect.TypeOf(int8(0)),
						Offset: 24,
					},
					{
						ColName: "last_name",
						GoName:  "LastName",
						Type:    reflect.TypeOf(&sql.NullString{}),
						Offset: 32,
					},
				},
			},

		},

	}
	r:=&registry{}
	for _,tc:=range testCases{
		t.Run(tc.name, func(t *testing.T) {
			m,err:= r.Register(tc.entity)
			assert.Equal(t, tc.wantErr,err)
			if err!=nil{
				return
			}
			fieldMap:=make(map[string]*Field)
			columnMap:=make(map[string]*Field)
			for _,f:=range tc.wantModel.Fields{
				fieldMap[f.GoName]=f
				columnMap[f.ColName]=f
			}
			tc.wantModel.FieldMap =fieldMap
			tc.wantModel.ColumnMap =columnMap
			assert.EqualValues(t, tc.wantModel,m)
		})
	}
}
func TestRegister_get(t *testing.T){
	testCases:=[]struct{
		name string
		entity any
		wantModel *Model
		wantErr error
		cacheSize int
	}{
		{
			name: "pointer",
			entity: &TestModel{},
			wantModel: &Model{
				TableName: "test_model",
				Fields: []*Field{
					{
						ColName: "id",
						GoName:  "Id",
						Type:    reflect.TypeOf(int64(0)),
						Offset: 0,
					},
					{
						ColName: "first_name",
						GoName:  "FirstName",
						Type:    reflect.TypeOf(""),
						Offset: 8,
					},
					{
						ColName: "age",
						GoName:  "Age",
						Type:    reflect.TypeOf(int8(0)),
						Offset: 24,
					},
					{
						ColName: "last_name",
						GoName:  "LastName",
						Type:    reflect.TypeOf(&sql.NullString{}),
						Offset: 32,
					},
				},
			},

			cacheSize: 1,
		},
		{
			name: "tag",
			entity: func() any{
				type TagTable struct {
					FirstName string `orm:"column=first_name_t"`
				}
				return &TagTable{}
			}(),
			wantModel: &Model{
				TableName: "tag_table",
				Fields: []*Field{
					{
						ColName: "first_name_t",
						GoName:  "FirstName",
						Type:    reflect.TypeOf(""),
					},
				},
			},

		},
		{
			name: "empty column",
			entity: func() any{
				type TagTable struct {
					FirstName string `orm:"column="`
				}
				return &TagTable{}
			}(),
			wantModel: &Model{
				TableName: "tag_table",
				Fields: []*Field{
					{
						ColName: "first_name",
						GoName:  "FirstName",
						Type:    reflect.TypeOf(""),
					},
				},
			},

		},
		{
			name: "column only",
			entity: func() any{
				type TagTable struct {
					FirstName string `orm:"column"`
				}
				return &TagTable{}
			}(),
			wantErr: errs.NewErrInvalidTagContext("column"),
		},
		{
			name: "invalid column",
			entity: func() any{
				type TagTable struct {
					FirstName string `orm:"abc=abc"`
				}
				return &TagTable{}
			}(),
			wantModel: &Model{
				TableName: "tag_table",
				Fields: []*Field{
					{
						ColName: "first_name",
						GoName:  "FirstName",
						Type:    reflect.TypeOf(""),
					},
				},
			},

		},
		{
			name: "custom table name",
			entity: &CustomTableName{},
			wantModel: &Model{
				TableName: "custom_table_name_t",
				Fields:[]*Field{
					{
						ColName: "first_name",
						GoName:  "FirstName",
						Type:    reflect.TypeOf(""),
					},
				},
			},

		},
		{
			name: "custom table name ptr",
			entity: &CustomTableNamePtr{},
			wantModel: &Model{
				TableName: "custom_table_name_ptr_t",
				Fields: []*Field{
					{
						ColName: "first_name",
						GoName:  "FirstName",
						Type:    reflect.TypeOf(""),
					},
				},
			},

		},
	}
	r:= NewRegistry()
	for _,tc:=range testCases{
		t.Run(tc.name, func(t *testing.T) {
			m,err:=r.Get(tc.entity)
			assert.Equal(t, tc.wantErr,err)
			if err!=nil{
				return
			}




			fieldMap:=make(map[string]*Field)
			columnMap:=make(map[string]*Field)
			for _,f:=range tc.wantModel.Fields{
				fieldMap[f.GoName]=f
				columnMap[f.ColName]=f
			}
			tc.wantModel.FieldMap =fieldMap
			tc.wantModel.ColumnMap =columnMap
			assert.Equal(t, tc.wantModel, m)
			typ := reflect.TypeOf(tc.entity)
			cache, ok := r.(*registry).models.Load(typ)
			assert.True(t, ok)
			assert.Equal(t, tc.wantModel,cache)
		})
	}
}

type CustomTableName struct {
	FirstName string
}
func(c CustomTableName)TableName()string{
	return "custom_table_name_t"
}
type CustomTableNamePtr struct {
	FirstName string
}
func(c *CustomTableNamePtr)TableName()string{
	return "custom_table_name_ptr_t"
}
//type TestModel struct {
//	Id int64
//	FirstName string
//	Age int8
//	LastName *sql.NullString
//}

func TestModelWithTableName(t *testing.T) {
	r:= NewRegistry()
	m,err:=r.Register(&TestModel{}, WithTableName("test_model_ttt"))
	require.NoError(t, err)
	assert.Equal(t, "test_model_ttt",m.TableName)
}

func TestModelWithColumnName(t *testing.T) {

	testCases := []struct {
		name string
		field string
		colName string
		wantColName string
		wantErr error
	}{
		{
			name: "column name",
			field: "FirstName",
			colName: "first_name_ccc",
			wantColName: "first_name_ccc",
		},
		{
			name: "invalid column name",
			field: "XXX",
			colName: "first_name_ccc",
			wantErr: errs.NewUnknownField("XXX"),
		},
	}
		for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			r:= NewRegistry()
			m,err:=r.Register(&TestModel{}, WithColumnName(tt.field,tt.colName))
			assert.Equal(t, tt.wantErr,err)
			if err!=nil{
				return
			}
			fd,ok:=m.FieldMap[tt.field]
			require.True(t, ok)
			assert.Equal(t, tt.wantColName,fd.ColName)
		})
	}
}
type TestModel struct {
	Id int64
	// 空为""
	FirstName string
	Age int8
	LastName *sql.NullString
}