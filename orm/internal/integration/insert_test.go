//go:build e2e
package integration

import (
	"context"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gotest/orm"
	"gotest/orm/internal/test"
	"testing"
	"time"
	_ "github.com/mattn/go-sqlite3"
)
type InsertSuite struct {
	Suite
}
//TearDownTest 单个跑完清数据
func (s *InsertSuite)TearDownTest(){
	orm.RawQuery[test.SimpleStruct](s.db,"TRUNCATE TABLE `simple_struct`").Exec(context.Background())
}
func TestMySQLInsert(t *testing.T){
	suite.Run(t,&InsertSuite{
		Suite{
			driver: "mysql",
			dsn: "root:root@tcp(localhost:13306)/integration_test",
		},
	})
}


func (i *InsertSuite)TestInsert(){
	db:=i.db
	t:=i.T()
	testCases:=[]struct{
		name string
		i *orm.Inserter[test.SimpleStruct]
		wantAffected int64//插入行数
	}{
		{
			name:"insert one",
			i: orm.NewInserter[test.SimpleStruct](db).Values(test.NewSimpleStruct(12)),
			wantAffected: 1,
		},
		{
			name:"insert multiple",
			i: orm.NewInserter[test.SimpleStruct](db).Values(
				test.NewSimpleStruct(13),
				test.NewSimpleStruct(14)),
			wantAffected: 2,
		},
		{
			name:"insert id",
			i: orm.NewInserter[test.SimpleStruct](db).Values(&test.SimpleStruct{
				Id: 15,
			}),
			wantAffected: 1,
		},
	}
	for _,tc:=range testCases{
		t.Run(tc.name, func(t *testing.T) {
			ctx,cancel:=context.WithTimeout(context.Background(),time.Second*10)
			defer cancel()
			res:=tc.i.Exec(ctx)
			affected,err:=res.RowsAffected()
			assert.NoError(t, err)
			assert.Equal(t, tc.wantAffected,affected)
		})
	}
}

//func TestMySQLInsert(t *testing.T){
//	testInsert(t,"mysql","root:root@tcp(localhost:13306)/integration_test")
//}
//func testInsert(t *testing.T,driver,dsn string) {
//	//db,err:=orm.Open("mysql","root:root@tcp(localhost:13306)/integration_test")
//	db,err:=orm.Open(driver,dsn)
//	require.NoError(t, err)
//
//	testCases:=[]struct{
//		name string
//		i *orm.Inserter[test.SimpleStruct]
//		wantAffected int64//插入行数
//	}{
//		{
//			name:"insert one",
//			i: orm.NewInserter[test.SimpleStruct](db).Values(test.NewSimpleStruct(12)),
//			wantAffected: 1,
//		},
//		{
//			name:"insert multiple",
//			i: orm.NewInserter[test.SimpleStruct](db).Values(
//				test.NewSimpleStruct(13),
//				test.NewSimpleStruct(14)),
//			wantAffected: 2,
//		},
//		{
//			name:"insert id",
//			i: orm.NewInserter[test.SimpleStruct](db).Values(&test.SimpleStruct{
//				Id: 15,
//			}),
//			wantAffected: 1,
//		},
//	}
//	for _,tc:=range testCases{
//		t.Run(tc.name, func(t *testing.T) {
//			ctx,cancel:=context.WithTimeout(context.Background(),time.Second*10)
//			defer cancel()
//			res:=tc.i.Exec(ctx)
//			affected,err:=res.RowsAffected()
//			assert.NoError(t, err)
//			assert.Equal(t, tc.wantAffected,affected)
//		})
//	}
//}
//SQLite3InsertSuite 接近于继承
type SQLite3InsertSuite struct {
	InsertSuite
}
func (i *SQLite3InsertSuite)SetupSuite(){
	db,err:=sql.Open(i.driver,i.dsn)
	//要建表，补上建表语句
	db.ExecContext(context.Background(),"")
	require.NoError(i.T(), err)
	i.db,err=orm.OpenDB(db)
	require.NoError(i.T(), err)
}
func TestSQLite3Insert(t *testing.T){
	suite.Run(t,&SQLite3InsertSuite{
		InsertSuite{
			Suite{
				driver: "sqlite3",
				dsn: "file:test.db?cache=shared&mode=memory",
			},
		},
	})
}