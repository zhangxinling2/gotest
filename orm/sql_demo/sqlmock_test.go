package sql_demo

import (
	"context"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"log"

	"testing"
)

func TestSQLMock(t *testing.T){
	db,mock,err:=sqlmock.New()
	defer db.Close()
	require.NoError(t, err)
	mockRows:=sqlmock.NewRows([]string{"id","first_name"})
	mockRows.AddRow(1,"Tom")
	mock.ExpectQuery("SELECT id,first_name FROM `user`.*").WillReturnRows(mockRows)
	mock.ExpectQuery("SELECT id, FROM `user`.*").WillReturnError(errors.New("mock error"))

	rows,err:=db.QueryContext(context.Background(),"SELECT id,first_name FROM `user` WHERE id =?",1)
	require.NoError(t, err)
	for rows.Next(){
		tm:=&TestModel{}
		err:=rows.Scan(&tm.Id,&tm.FirstName)
		require.NoError(t, err)
		log.Println(tm)
	}

	_,err=db.QueryContext(context.Background(),"SELECT id,first_name FROM `user` WHERE id =?",1)
	require.Error(t, err)
}
