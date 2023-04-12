package querylog

import (
	"context"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gotest/orm"
	"testing"
)

func TestMiddlewareBuilder_Build(t *testing.T) {
	var query string
	var args []any
	//m如何注入，在DB层面维护middleware切片
	m:=NewMiddlewareBuilder().LogFunc(func(q string, a []any) {
		query=q
		args=a
	})
	db,err:=orm.Open("sqlite3","file:test.db?cache=shared&mode=memory",orm.DBWithMiddleware(m.Build()))
	require.NoError(t, err)
	//不生效，因为selector还没有完成接入
	_,_=orm.NewSelector[TestModel](db).Where(orm.C("Id").Eq(10)).Get(context.Background())
	assert.Equal(t, "SELECT * FROM `test_model` WHERE `id` = ?;",query)
	assert.Equal(t, []any{10},args)
	orm.NewInserter[TestModel](db).Values(&TestModel{Id:18}).Exec(context.Background())
	assert.Equal(t, "INSERT INTO `test_model`(`id`,`first_name`,`age`,`last_name`) VALUES (?,?,?,?);",query)
	assert.Equal(t, []any{int64(18),"",int8(0),(*sql.NullString)(nil)},args)
}
type TestModel struct {
	Id int64 `eorm:"auto_increment,primary_key"`
	FirstName string
	Age int8
	LastName *sql.NullString
}