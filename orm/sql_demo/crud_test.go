package sql_demo

import (
	"context"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"log"
	"testing"
	"time"
)

type TestModel struct {
	Id int64 `eorm:"auto_increment,primary_key"`
	FirstName string
	Age int8
	LastName *sql.NullString
}
func TestDB(t *testing.T){
	db,err:=sql.Open("sqlite3","file:test.db?cache=shared&mode=memory")
	require.NoError(t, err)
	db.Ping()
	ctx,cancel:=context.WithTimeout(context.Background(),time.Second)
	defer cancel()
	//除了select,其他都用这个
	_,err=db.ExecContext(ctx,`
CREATE TABLE IF NOT EXISTS test_model(
    id INTEGER PRIMARY KEY,
    first_name TEXT NOT NULL,
    age INTEGER,
    last_name TEXT NOT NULL)
    `)
	require.NoError(t, err)
	res,err:=db.ExecContext(ctx,"INSERT INTO test_model(`id`,`first_name`,`age`,`last_name`) VALUES (?,?,?,?)",1,"Tom",18,"Jerry")
	require.NoError(t, err)
	affected,err:=res.RowsAffected()
	require.NoError(t, err)
	log.Println(affected)
	lastId,err:=res.LastInsertId()
	require.NoError(t, err)
	log.Println(lastId)

	rows:=db.QueryRowContext(ctx,"SELECT * FROM `test_model` WHERE `id`=?",1)
	tm:=TestModel{}
	err=rows.Scan(&tm.Id,&tm.FirstName,&tm.Age,&tm.LastName)
	require.NoError(t, err)

	rows=db.QueryRowContext(ctx,"SELECT * FROM `test_model` WHERE `id`=?",2)
	tm=TestModel{}
	err=rows.Scan(&tm.Id,&tm.FirstName,&tm.Age,&tm.LastName)
	require.Error(t, sql.ErrNoRows,err)
	tx,err:=db.BeginTx(ctx,&sql.TxOptions{})
	require.NoError(t, err)
	tx.Commit()
	tx.Rollback()

}

func TestPrepareStatement(t *testing.T){
	db,err:=sql.Open("sqlite3","file:test.db?cache=shared&mode=memory")
	require.NoError(t, err)
	defer db.Close()
	ctx,cancel:=context.WithTimeout(context.Background(),time.Second)
	defer cancel()
	stmt,err:=db.PrepareContext(ctx,"SELECT * FROM `test_model` WHERE `id`=?")
	require.NoError(t, err)
	rows,err:=stmt.QueryContext(ctx,1)
	require.NoError(t, err)
	for rows.Next(){
		tm:=&TestModel{}
		err:=rows.Scan(&tm.Id,&tm.FirstName,&tm.Age,&tm.LastName)
		require.NoError(t, err)
		log.Println(tm)
	}
	stmt.Close()
}
