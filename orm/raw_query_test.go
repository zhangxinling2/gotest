package orm

import (
	"context"
	"database/sql"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRawQuerier_Get(t *testing.T){
	mockDB,mock,err:=sqlmock.New()
	require.NoError(t, err)
	db,err:=OpenDB(mockDB)
	require.NoError(t, err)

	mock.ExpectQuery("SELECT .*").WillReturnError(errors.New("query error"))

	rows:=sqlmock.NewRows([]string{"id","first_name","last_name","age"})
	mock.ExpectQuery("SELECT .*").WillReturnRows(rows)

	rows=sqlmock.NewRows([]string{"id","first_name","last_name","age"})
	rows.AddRow("1","Tom","Jerry","18")
	mock.ExpectQuery("SELECT .*").WillReturnRows(rows)

	testCases:=[]struct {
		name      string
		s *RawQuerier[TestModel]
		wantErr error
		wantRes *TestModel
	}{
		{
			name:"Query error",
			s:RawQuery[TestModel](db,"SELECT * FROM `test_model`"),
			wantErr: errors.New("query error"),
		},
		{
			name:"no rows",
			s:RawQuery[TestModel](db,"SELECT * FROM `test_model` WHERE `id` = ?",-1),
			wantErr: ErrNoRows,
		},
		{
			name:"data",
			s:RawQuery[TestModel](db,"SELECT * FROM `test_model` WHERE `id` = ?",1),
			wantRes:&TestModel{
				Id:1,
				FirstName: "Tom",
				Age: 18,
				LastName: &sql.NullString{Valid: true,String: "Jerry"},
			},
		},
	}
	for _,tc:=range testCases{
		t.Run(tc.name, func(t *testing.T) {
			res,err:=tc.s.Get(context.Background())
			assert.Equal(t, tc.wantErr,err)
			if err!=nil{
				return
			}
			assert.Equal(t, tc.wantRes,res)
		})
	}
}
