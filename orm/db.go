package orm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"log"
	"time"

	"gotest/orm/internal/errs"
	"gotest/orm/internal/valuer"
	"gotest/orm/model"
)

type DBOption func(db *DB)
//DB 是一个sql.DB的装饰器
type DB struct {
	//为了得到core让DB持有core
	core
	//r model.Registry
	db *sql.DB
	//creator valuer.Creator
	//dialect Dialect

	//作用于增删改查，所以直接丢入core
	//mdls []Middleware


}


func Open(driver string,dataSourceName string,opts...DBOption)(*DB,error){
	db,err:=sql.Open(driver,dataSourceName)

	if err!=nil{
		return nil, err
	}
	return OpenDB(db,opts...)
	//res:= &DB{
	//	r: NewRegistry(),
	//	db: db,
	//}
	//for _,opt:=range opts{
	//	opt(res)
	//}
	//return res,nil
}

func OpenDB(db *sql.DB,opts...DBOption)(*DB,error){
	res:= &DB{
		core:core{
			r:  model.NewRegistry(),
			creator: valuer.NewUnsafeValue,
			dialect: DialectMySQL,
		},
		db: db,

	}
	for _,opt:=range opts{
		opt(res)
	}
	return res,nil
}
func DBWithMiddleware(mdls...Middleware)DBOption{
	return func(db *DB) {
		db.mdls=mdls
		//更倾向于一次性操作
		//db.mdls=append(db.mdls,mdls...)
	}
}
func DBWithDialect(dialect Dialect)DBOption{
	return func(db *DB) {
		db.dialect=dialect
	}
}

func DBWithReflect()DBOption{
	return func(db *DB) {
		db.creator=valuer.NewReflectValue
	}
}
func MustOpenDB(driver string,dataSourceName string,opts...DBOption)*DB{
	res,err:= Open(driver,dataSourceName,opts...)
	if err!=nil{
		panic(err)
	}
	return res
}

func(db *DB)BeginTx(ctx context.Context,opts *sql.TxOptions)(*Tx,error){
	tx,err:=db.db.BeginTx(ctx,opts)
	if err!=nil{
		return nil, err
	}
	return &Tx{
		tx: tx,
	},nil
}
type txKey struct {}
// ctx,tx,err:=db.BeginTxV2()
// doSomething(ctx,tx)
func(db *DB)BeginTxV2(ctx context.Context,opts *sql.TxOptions)(context.Context,*Tx,error){
	val:=ctx.Value(txKey{})
	tx,ok:=val.(*Tx)
	if ok&&!tx.done{
		return ctx,tx,nil
	}
	tx,err:=db.BeginTx(ctx,opts)
	if err!=nil{
		return nil,nil, err
	}
	ctx=context.WithValue(ctx,txKey{},tx)
	return ctx,tx,nil
}
// BeginTxV3 要求前面的人一定要开事务
//func(db *DB)BeginTxV3(ctx context.Context,opts *sql.TxOptions)(*Tx,error){
//	val:=ctx.Value(txKey{})
//	tx,ok:=val.(*Tx)
//	if ok{
//		return tx,nil
//	}
//	return nil,errors.New("没有开事务")
//}
func (db *DB) getCore()core{
	return db.core
}
func(db *DB)DoTx(ctx context.Context,
	fn func(ctx context.Context,tx *Tx)error,
	opts *sql.TxOptions)(err error){
	tx,err:=db.BeginTx(ctx,opts)
	if err!=nil{
		return err
	}
	panicked:=true
	defer func() {
		if panicked||err!=nil{
			e:=tx.Rollback()
			err=errs.NewErrFailedToRollbackTx(err,e,panicked)
		}else {
			err=tx.Commit()
		}
	}()
	fn(ctx,tx)
	panicked=false
	return err
}

func (d *DB) queryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return d.db.QueryContext(ctx,query,args...)
}

func (d *DB) execContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return d.db.ExecContext(ctx,query,args...)
}
//Wait 主动等待数据库启动
func (d *DB) Wait()error{
	err:=d.db.Ping()
	for err==driver.ErrBadConn{
		log.Println("等待数据库启动...")
		err = d.db.Ping()
		time.Sleep(time.Second)
	}
	return err
}