//go:build e2e
package integration

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gotest/orm"
	"gotest/orm/internal/test"
	"testing"
	"time"
)

type SelectSuite struct {
	Suite
}
func TestMySQLTest(t *testing.T){
	suite.Run(t, &SelectSuite{
		Suite{
			driver: "mysql",
			dsn: "root:root@tcp(localhost:13306)/integration_test",
		},
	})
}
//TearDownSuite 所有都跑完清数据
func (s *SelectSuite)TearDownSuite(){
	orm.RawQuery[test.SimpleStruct](s.db,"TRUNCATE TABLE `simple_struct`").Exec(context.Background())
}
func (s *SelectSuite)SetupSuite()  {
	s.Suite.SetupSuite()
	res:=orm.NewInserter[test.SimpleStruct](s.db).Values(test.NewSimpleStruct(100)).Exec(context.Background())
	require.NoError(s.T(),res.Err())
}

func(s *SelectSuite)TestSelect(){
	testCases:=[]struct{
		name string
		s *orm.Selector[test.SimpleStruct]

		wantRes *test.SimpleStruct
		wantErr error
	}{
		{
			name:"get data",
			s:orm.NewSelector[test.SimpleStruct](s.db).Where(orm.C("Id").Eq(100)),//数据从SetupSuite中插入
			wantRes: test.NewSimpleStruct(100),
		},
		{
			name:"no row",
			s:orm.NewSelector[test.SimpleStruct](s.db).Where(orm.C("Id").Eq(200)),//数据从SetupSuite中插入
			wantErr: orm.ErrNoRows,
		},
	}

	for _,tc:=range testCases{
		s.T().Run(tc.name, func(t *testing.T) {
			ctx,cancel:=context.WithTimeout(context.Background(),time.Second*10)
			defer cancel()
			res,err:=tc.s.Get(ctx)
			assert.Equal(t, tc.wantErr,err)
			if err!=nil{
				return
			}
			assert.Equal(t, tc.wantRes,res)
		})
	}
}
