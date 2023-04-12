package orm

import (
	"context"
	"database/sql"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gotest/orm/internal/errs"
	"testing"
)


func TestSelector_Join(t *testing.T){
	db:=memoryDB(t)
	//测试Join的结构体
	type Order struct {
		Id int
		UsingCol1 string
		UsingCol2 string
	}

	type OrderDetail struct {
		OrderId int
		ItemId int

		UsingCol1 string
		UsingCol2 string
	}

	type Item struct {
		Id int
	}
	testCases:=[]struct{
		name string
		builder QueryBuilder
		wantQuery *Query
		wantErr error
	}{
		{
			name: "specify table",
			builder: NewSelector[Order](db).From(TableOf(&OrderDetail{})),
			wantQuery: &Query{
				SQL: "SELECT * FROM `order_detail`;",
			},
		},
		{
			name: "join-using",
			builder: func()QueryBuilder {
				t1:=TableOf(&Order{})
				t2:=TableOf(&OrderDetail{})
				t3:=t1.Join(t2).Using("UsingCol1","UsingCol2")
				return NewSelector[Order](db).From(t3)
			}(),
			wantQuery: &Query{
				SQL: "SELECT * FROM (`order` JOIN `order_detail` USING (`using_col1`,`using_col2`));",
			},
		},
		{
			name: "left join",
			builder: func()QueryBuilder {
				t1:=TableOf(&Order{})
				t2:=TableOf(&OrderDetail{})
				t3:=t1.LeftJoin(t2).Using("UsingCol1","UsingCol2")
				return NewSelector[Order](db).From(t3)
			}(),
			wantQuery: &Query{
				SQL: "SELECT * FROM (`order` LEFT JOIN `order_detail` USING (`using_col1`,`using_col2`));",
			},
		},
		{
			name: "right join",
			builder: func()QueryBuilder {
				t1:=TableOf(&Order{})
				t2:=TableOf(&OrderDetail{})
				t3:=t1.RightJoin(t2).Using("UsingCol1","UsingCol2")
				return NewSelector[Order](db).From(t3)
			}(),
			wantQuery: &Query{
				SQL: "SELECT * FROM (`order` RIGHT JOIN `order_detail` USING (`using_col1`,`using_col2`));",
			},
		},
		{
			name: "join-using",
			builder: func()QueryBuilder {
				t1:=TableOf(&Order{}).As("t1")
				t2:=TableOf(&OrderDetail{}).As("t2")
				//Eq(C("OrderId")要指定是哪个表的否则orm: 未知字段 OrderId,因为OrderId是另一个表中的
				//那么在table定义新的C方法，在column中维持table
				t3:=t1.Join(t2).On(t1.C("Id").Eq(t2.C("OrderId")))
				return NewSelector[Order](db).From(t3)
			}(),
			wantQuery: &Query{
				SQL: "SELECT * FROM (`order` AS `t1` JOIN `order_detail` AS `t2` ON `t1`.`id` = `t2`.`order_id`);",
			},
		},
		{
			name: "join-table",
			builder: func()QueryBuilder {
				t1:=TableOf(&Order{}).As("t1")
				t2:=TableOf(&OrderDetail{}).As("t2")
				//Eq(C("OrderId")要指定是哪个表的否则orm: 未知字段 OrderId,因为OrderId是另一个表中的
				//那么在table定义新的C方法，在column中维持table
				t3:=t1.Join(t2).On(t1.C("Id").Eq(t2.C("OrderId")))
				t4:=TableOf(&Item{}).As("t4")
				t5:=t3.Join(t4).On(t2.C("ItemId").Eq(t4.C("Id")))
				return NewSelector[Order](db).From(t5)
			}(),
			wantQuery: &Query{
				SQL: "SELECT * FROM ((`order` AS `t1` JOIN `order_detail` AS `t2` ON `t1`.`id` = `t2`.`order_id`) JOIN `item` AS `t4`" +
					" ON `t2`.`item_id` = `t4`.`id`);",
			},
		},
		{
			name: "table-join",
			builder: func()QueryBuilder {
				t1:=TableOf(&Order{}).As("t1")
				t2:=TableOf(&OrderDetail{}).As("t2")
				//Eq(C("OrderId")要指定是哪个表的否则orm: 未知字段 OrderId,因为OrderId是另一个表中的
				//那么在table定义新的C方法，在column中维持table
				t3:=t1.Join(t2).On(t1.C("Id").Eq(t2.C("OrderId")))
				t4:=TableOf(&Item{}).As("t4")
				t5:=t4.Join(t3).On(t2.C("ItemId").Eq(t4.C("Id")))
				return NewSelector[Order](db).From(t5)
			}(),
			wantQuery: &Query{
				SQL: "SELECT * FROM (`item` AS `t4` JOIN (`order` AS `t1` JOIN `order_detail` AS `t2` ON `t1`.`id` = `t2`.`order_id`) ON `t2`.`item_id` = `t4`.`id`);",
			},
		},
	}
	for _,tc:=range testCases{
		t.Run(tc.name, func(t *testing.T) {
			q,err:=tc.builder.Build()
			assert.Equal(t, tc.wantErr,err)
			if err!=nil{
				return
			}
			assert.Equal(t, tc.wantQuery,q)
		})
	}
}
func TestSelector_Build(t *testing.T) {
	db:=memoryDB(t)
	testCases:=[]struct{
		name string
		builder QueryBuilder
		wantQuery *Query
		wantErr error
	}{
		{
			name:"no from",
			builder: NewSelector[TestModel](db),
			wantQuery: &Query{
				SQL: "SELECT * FROM `test_model`;",
				Args: nil,
			},
		},
		//{
		//	name:"from",
		//	builder: NewSelector[TestModel](db).From("`test_model`"),
		//	wantQuery: &Query{
		//		SQL: "SELECT * FROM `test_model`;",
		//		Args: nil,
		//	},
		//},
		//{
		//	name:"empty from",
		//	builder: NewSelector[TestModel](db).From(""),
		//	wantQuery: &Query{
		//		SQL: "SELECT * FROM `test_model`;",
		//		Args: nil,
		//	},
		//},
		//{
		//	name:"from db",
		//	builder: NewSelector[TestModel](db).From("`test_db`.`test_model`"),
		//	wantQuery: &Query{
		//		SQL: "SELECT * FROM `test_db`.`test_model`;",
		//		Args: nil,
		//	},
		//},
		{
			name:"where",
			builder: NewSelector[TestModel](db).Where(C("Age").Eq(18)),
			wantQuery: &Query{
				SQL: "SELECT * FROM `test_model` WHERE `age` = ?;",
				Args: []any{18},
			},
		},
		{
			name:"not",
			builder: NewSelector[TestModel](db).Where(Not(C("Age").Eq(18))),
			wantQuery: &Query{
				SQL: "SELECT * FROM `test_model` WHERE  NOT (`age` = ?);",
				Args: []any{18},
			},
		},
		{
			name:"and",
			builder: NewSelector[TestModel](db).Where((C("Age").Eq(18).And(C("FirstName").Eq("Tom")))),
			wantQuery: &Query{
				SQL: "SELECT * FROM `test_model` WHERE (`age` = ?) AND (`first_name` = ?);",
				Args: []any{18,"Tom"},
			},
		},
		{
			name:"or",
			builder: NewSelector[TestModel](db).Where((C("Age").Eq(18).Or(C("FirstName").Eq("Tom")))),
			wantQuery: &Query{
				SQL: "SELECT * FROM `test_model` WHERE (`age` = ?) OR (`first_name` = ?);",
				Args: []any{18,"Tom"},
			},
		},
		{
			name:"empty where",
			builder: NewSelector[TestModel](db).Where(),
			wantQuery: &Query{
				SQL: "SELECT * FROM `test_model`;",
			},
		},
		{
			name:"invalid column",
			builder: NewSelector[TestModel](db).Where(C("Age").Eq(18).Or(C("XXXX").Eq(19))),
			wantErr: errs.NewUnknownField("XXXX"),
		},

	}
	for _,tc:=range testCases{
		t.Run(tc.name, func(t *testing.T) {
			q,err:=tc.builder.Build()
			assert.Equal(t, tc.wantErr,err)
			if err!=nil{
				return
			}
			assert.Equal(t, tc.wantQuery,q)
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
func TestGet(t *testing.T){
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
		s *Selector[TestModel]
		wantErr error
		wantRes *TestModel
	}{
		{
			name:"invalid query",
			s:NewSelector[TestModel](db).Where(C("XXX").Eq(1)),
			wantErr: errs.NewUnknownField("XXX"),
		},
		{
			name:"Query error",
			s:NewSelector[TestModel](db).Where(C("Id").Eq(1)),
			wantErr: errors.New("query error"),
		},
		{
			name:"no rows",
			s:NewSelector[TestModel](db).Where(C("Id").Lt(1)),
			wantErr: ErrNoRows,
		},
		{
			name:"data",
			s:NewSelector[TestModel](db).Where(C("Id").Lt(1)),
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
func TestSelector_Select(t *testing.T) {
	db:=memoryDB(t)
	testcases:=[]struct{
		name string
		s QueryBuilder
		wantQuery *Query
		wantErr error
	}{
		{
			name: "multiple columns",
			s:(NewSelector[TestModel](db)).Select(C("FirstName"),C("LastName")),
			wantQuery: &Query{SQL: "SELECT `first_name`,`last_name` FROM `test_model`;"},
		},
		{
			name: "column alias",
			s:(NewSelector[TestModel](db)).Select(C("FirstName").As("my_name")),
			wantQuery: &Query{SQL: "SELECT `first_name` AS `my_name` FROM `test_model`;"},
		},
		{
			name: "Avg",
			s:(NewSelector[TestModel](db)).Select(Avg("Age")),
			wantQuery: &Query{SQL: "SELECT AVG(`age`) FROM `test_model`;"},
		},
		{
			name: "Avg alias",
			s:(NewSelector[TestModel](db)).Select(Avg("Age").As("avg_age")),
			wantQuery: &Query{SQL: "SELECT AVG(`age`) AS `avg_age` FROM `test_model`;"},
		},
		{
			name: "Sum",
			s:(NewSelector[TestModel](db)).Select(Sum("Age")),
			wantQuery: &Query{SQL: "SELECT SUM(`age`) FROM `test_model`;"},
		},
		{
			name: "Count",
			s:(NewSelector[TestModel](db)).Select(Count("Age")),
			wantQuery: &Query{SQL: "SELECT COUNT(`age`) FROM `test_model`;"},
		},
		{
			name: "Max",
			s:(NewSelector[TestModel](db)).Select(Max("Age")),
			wantQuery: &Query{SQL: "SELECT MAX(`age`) FROM `test_model`;"},
		},
		{
			name: "Min",
			s:(NewSelector[TestModel](db)).Select(Min("Age")),
			wantQuery: &Query{SQL: "SELECT MIN(`age`) FROM `test_model`;"},
		},
		{
			name: "min invalid column",
			s:(NewSelector[TestModel](db)).Select(Min("invalid")),
			wantErr: errs.NewUnknownField("invalid"),
		},
		{
			name: "multiple Min Max",
			s:(NewSelector[TestModel](db)).Select(Min("Age"),Max("Age")),
			wantQuery: &Query{SQL: "SELECT MIN(`age`),MAX(`age`) FROM `test_model`;"},
		},
		{
			name: "Raw Expression",
			s:(NewSelector[TestModel](db)).Select(Raw("COUNT(DISTINCT `first_name`)")),
			wantQuery: &Query{SQL: "SELECT COUNT(DISTINCT `first_name`) FROM `test_model`;"},
		},
		{
			name: "Raw Expression as predicate",
			s:(NewSelector[TestModel](db)).Where(Raw("`id`<?",18).AsPredicate()),
			wantQuery: &Query{
				SQL: "SELECT * FROM `test_model` WHERE (`id`<?);",
				Args: []any{18},
			},
		},
		{
			name: "Raw Expression used in predicate",
			s:(NewSelector[TestModel](db)).Where(C("Id").Eq(Raw("`age`+?",1))),
			wantQuery: &Query{
				SQL: "SELECT * FROM `test_model` WHERE `id` = (`age`+?);",
				Args: []any{1},
			},
		},
		{
			name: "columns alias",
			s:(NewSelector[TestModel](db)).Where(C("Id").As("my_id").Eq(Raw("`age`+?",1))),
			wantQuery: &Query{
				SQL: "SELECT * FROM `test_model` WHERE `id` = (`age`+?);",
				Args: []any{1},
			},
		},
	}
	for _,tc:=range testcases{
		t.Run(tc.name, func(t *testing.T) {
			q,err:=tc.s.Build()
			assert.Equal(t, tc.wantErr,err)
			if err!=nil{
				return
			}
			assert.Equal(t, tc.wantQuery,q)
		})
	}
}
func memoryDB(t *testing.T)*DB{
	db,err:=Open("sqlite3","file:test.db?cache=shared&mode=memory",
		//仅用于单元测试
		DBWithDialect(DialectMySQL))
	require.NoError(t, err)
	return db
}

func memoryDBOpt(t *testing.T,opts...DBOption)*DB{

	db,err:=Open("sqlite3","file:test.db?cache=shared&mode=memory",
		//仅用于单元测试
		opts...)
	require.NoError(t, err)

	return db
}