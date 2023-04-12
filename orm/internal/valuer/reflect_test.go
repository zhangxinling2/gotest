package valuer

import (
	"database/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gotest/orm/model"
	"testing"
)

func Test_reflectValue_SetColumns(t *testing.T) {
	testSetColumns(t,NewReflectValue)
}


func testSetColumns(t *testing.T,creator Creator){
	testcases:= []struct {
		name string
		//一定是指针
		entity any
		rows func() *sqlmock.Rows
		//就算没出error也可能出错
		wantErr error

		//对比一下数据有没有改成功
		wantEntity any
	}{
		{
			name: "set columns",
			entity:&TestModel{},
			rows: func() *sqlmock.Rows {
				rows:=sqlmock.NewRows([]string{"id","first_name","age","last_name"})
				rows.AddRow("1","Tom","18","Jerry")
				return rows
			},
			wantEntity: &TestModel{
				Id:1,
				FirstName: "Tom",
				Age: 18,
				LastName: &sql.NullString{Valid:true,String: "Jerry"},
			},
		},
		{
			name: "partial columns",
			entity:&TestModel{},
			rows: func() *sqlmock.Rows {
				rows:=sqlmock.NewRows([]string{"id","first_name"})
				rows.AddRow("1","Tom")
				return rows
			},
			wantEntity: &TestModel{
				Id:1,
				FirstName: "Tom",

			},
		},
	}
	r:= model.NewRegistry()
	mockDB,mock,err:=sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()
	for _,tc:=range testcases{
		t.Run(tc.name, func(t *testing.T) {

			//mock构造sql.Rows
			mockRows:=tc.rows()
			mock.ExpectQuery("SELECT XX").WillReturnRows(mockRows)
			rows,err:=mockDB.Query("SELECT XX")
			require.NoError(t, err)

			rows.Next()

			m,err:=r.Get(tc.entity)
			require.NoError(t, err)
			val:=creator(m,tc.entity)
			err=val.SetColumns(rows)
			assert.Equal(t, tc.wantErr,err)
			if err!=nil{
				return
			}
			assert.Equal(t,tc.wantEntity,tc.entity )
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