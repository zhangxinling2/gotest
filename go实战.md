##### 2022/3/20**

###### reflect设置值

一般使用指针，先使用

```go

vals:=reflect.ValueOf(entity)//得到值信息
vals=vals.Elem()//传递指针时不论是type还是value都要使用elem得到指针指向的东西
val:=vals.FieldByName(field)//根据字段名得到字段值的信息
if !val.CanSet(){			//CanSet()判断能否赋值
		errors.New(fmt.Sprintf("%s不能被设置",field))
	}
val.Set(reflect.ValueOf(newVal))//赋值需要Value类型，所以取传入值的valueOf
```

goland中依赖爆红：

[goland 解决 cannot resolve directory 'xxxx'问题_Lucky小黄人的博客-CSDN博客](https://blog.csdn.net/qq_41767116/article/details/126863153)

****

###### reflect输出方法

方法的接收器有结构体和指针，定义在结构体上的方法使用指针也可以访问。

```go
func IterateFunc(entity any)(map[string]FuncInfo,error){
   typ:=reflect.TypeOf(entity)//得到类型信息
   if typ.Kind()!=reflect.Ptr&&typ.Kind()!=reflect.Struct{//判断是否为结构体或指针
      return nil, errors.New("非法类型")
   }
   numFunc := typ.NumMethod()//得到方法数量
   result:=make(map[string]FuncInfo,numFunc)
   for i := 0; i < numFunc; i++ {
      m:=typ.Method(i)//typ.Method(i)得到Method
      num:=m.Type.NumIn()//.Type得到方法信息 .NumIn()得到输入数量
      fn:=m.Func//.Func是方法的Value
      input:=make([]reflect.Type,0,num)//input是输入参数的类型
      inputValue:=make([]reflect.Value,0,num)//inputValue是输入参数的值
      inputValue=append(inputValue,reflect.ValueOf(entity))//输入的第一个永远是结构体本身，就如同java的this
      for j := 0; j < num; j++ {
         fnInType:=fn.Type().In(j)//In返回的是第j个参数的类型，与m.Type.In()等价
         input= append(input, fnInType)
         if j>0{
            inputValue=append(inputValue,reflect.Zero(fnInType))//输入都用0值即可，用来测试
         }
      }
      outNum:=m.Type.NumOut()
      output:=make([]reflect.Type,0,outNum)
      for j := 0; j < outNum; j++ {
         output= append(output, fn.Type().Out(j))
      }
      resValues:=fn.Call(inputValue)//执行方法，返回的是Value切片
      results:=make([]any,0,len(resValues))
      for _,v:=range resValues{
         results=append(results,v.Interface())
      }
      funcInfo:=FuncInfo{
         Name:m.Name,
         Input: input,
         Output: output,
         Result: results,
      }
      result[m.Name]=funcInfo
   }
   return result,nil
}
type FuncInfo struct {
   Name string
   Input []reflect.Type
   Output []reflect.Type
   Result []any
}
```

##### **2022/3/21**

###### 元数据解析

元数据很复杂，但是都是一点点加进去的，先从最简定义开始：

```go
type model struct {
   tableName string
   fields    map[string]*field
}
//field 保存字段信息
type field struct {
   colName string
}
```

有了定义，学了反射就可以开始使用反射来解析结构体来获得元数据。

通过反射获得结构体在数据库中的表名和字段在数据库中的列名。

```go
// parseModel 解析模型
func parseModel(entity any) (*model, error) {
   typ := reflect.TypeOf(entity)
   //限制输入
   if typ.Kind() != reflect.Ptr || typ.Elem().Kind() != reflect.Struct {
      return nil, errs.ErrPointerOnly
   }
   typ = typ.Elem()
   //获取字段数量
   numField := typ.NumField()
   fields := make(map[string]*field, numField)
   //解析字段名作为列名
   for i := 0; i < numField; i++ {
      fdType := typ.Field(i)
      fields[fdType.Name] = &field{
         colName: TransferName(fdType.Name),// TransferName是自己实现的字符串转换
      }
   }
   return &model{
      tableName: TransferName(typ.Name()),
      fields:    fields,
   }, nil
}
```

有了元数据就可以在selector中使用，在Column中校验列名是否在数据库中存在，用户若没有定义表名就可以使用元数据解析的表名。

```go
//处理expression为列的情况
case Column:
   //有了元数据后就可以校验列存不存在
   fd,ok:=s.model.fields[e.Name]
   if !ok{
      return errs.NewErrUnknownField(e.Name)
   }
   s.sb.WriteByte('`')
   s.sb.WriteString(fd.colName)
   s.sb.WriteByte('`')
```

###### 元数据注册中心

selector中每次都要解析一遍，所以我们可以把它缓存住。

DB在ORM中就相当于HTTPServer在Web框架中的地位，允许用户使用多个DB，DB实例可以单独配置，例如配置元数据中心，DB是天然的隔离和治理单位，所以使用DB来维护元数据。

先定义元数据注册中心registry,里面维护一个map[reflect.Type]*model，之所以要用reflect.Type是因为如果要用结构体名那么会有同结构体名不同表名无法处理，如果要使用表名，我们需要得到元数据但是我们现在在注册元数据，最后选择reflect.Type。把parseModel作为registry的方法把参数改为接受reflect.Type,因为我们希望用户使用get。

```go
//get 得到相应的model
func(r *registry)get(val any)(*model,error){
   typ:=reflect.TypeOf(val)
   //判断是否已经缓存了此类型的元数据
   m,ok:=r.models[typ]
   if !ok{
      var err error
      m,err=r.parseModel(typ)
      if err!=nil{
         return nil, err
      }
   }
   r.models[typ]=m
   return m,nil
}
```

### ORM：事务API

#### Session抽象

核心就是允许用户创建事务，在事物内部进行增删改查，核心有三个API：

·Begin:开启一个事务

·Commit：提交一个事务

·Rollback：回滚一个事务

事务由DB开启，方法定义在DB上，Commit和Rollback由Tx来决定。而将Begin定义在DB上就限制了在一个事务无法开启一个新事务。

![image-20230328073532423](C:\Users\123456\OneDrive\图片\go笔记\事务.png)

Tx的使用：原本Selector接收的是DB做参数，现在使它也可以接收Tx，因为可以在事务中运行(Tx)也可以无事务运行(DB)，那么就需要一个共同的抽象，让DB和Tx来实现。

共同的抽象：session，在ORM语境下，一般代表一个上下文；也可以理解为一个分组机制，在此分组内所有的查询会共享一些基本配置。

Session接口的定义：想要进行抽象，就要把已经被使用的方法提取出来在接口中，在之前代码中，db的方法使用了*sql.DB的QueryContext和ExecContext那么在接口中就定义queryContext和execContext替换掉DB的调用。

core定义:在把session放入NewSelector之后，之前的db.dialect之类的都无法找到，为了得到在DB中我们需要的东西，定义一个core,把增删改查所需要的共同的东西放入core中，重点是DB中持有的，builder中需要什么就放入什么，最后让builder来组合这个core.为了得到core的内容，让DB持有core，在session中新定义一个getCore方法，在Tx中持有创建自己的DB来获得core。builder来使用core所以也要组合core。

#### 事务闭包API

用户传入方法，框架创建事务，事务执行方法然后根据方法的执行情况来判断是提交还是回滚。回滚的条件：出现error或者panic。

在DB上定义DoTx来做事务闭包API,用户传入上下文，业务代码和opts。注意在出错时，需要把err都包装在一起。

```go
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
```

由于go没有try-catch机制，虽然DoTx能解决大部分问题,但有时还要自己控制事务，如果事务没有提交就回滚，直接Rollback,返回的错误可以判断。

```go
func(t *Tx)RollbackIfNotCommit()error{
   t.done=true
   err:=t.tx.Rollback()
   //尝试回滚如果事务已经被提交或者回滚那么会返回ErrTxDone
   if err==sql.ErrTxDone{
      return nil
   }
   return err
}
```

#### 事务扩散方案

就是在调用链中，上游方法开启了事务，那么下游方法可以开一个新事务或无事务运行或报错。一般在其他语言中是thread-local，在go中就使用context。核心就是在创建事务时判断context中有没有未完成的事务,tx中定义done判断事务是否完成。

```go
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
```

### AOP方案

基本上任何框架都要提供MiddleWare。设计基本照抄web框架的MiddleWare。

#### Beego

Beego设计为侵入式的设计，因为操作没有统一的接口，只有单独的Insert，Read等方法，而我们的ORM框架，对于Select出口只有Get和GetMuti。Insert,Update,Del只有Exec。

#### Gorm

Hook:跟时机强相关。

Create对应于插入，有四个分为两对，BeforeSave,BeforeCreate,AfterSave,AfterCreate。在自己的模型上定义这些方法就会自动执行。

Update也是四个，BeforeSave,BeforeUpdate,AfterSave,AfterUpdate。

Delete有两个，BeforeDelete，AfterDelete。

Query只有一个，AfterFind,没有Before就意味着没办法篡改语句。

#### Aop方案设计

##### 定义

而我们抄web的middleware,做一个函数式的。

```go
type Handler func(ctx context.Context,qc *QueryContext)*QueryResult
type Middleware func(next Handler)Handler
```

```go
//代表上下文
type QueryContext struct {
   // 查询类型，标记增删改查
   Type string

   //代表的是查询本身,大多数情况下需要转化到具体的类型才能篡改查询
   Builder QueryBuilder
   //一般都会暴露出来给用户做高级处理
   Model *model.Model
}
//代表查询结果
type QueryResult struct {
   //Result 在不同查询下类型不同
   //SELECT 可以是*T也可以是[]*T
   //其他就是类型Result
   Result any
   //查询本身出的问题
   Err error
}
```

Middleware用Builder模式

```go
type MiddlewareBuilder struct {
	logFunc func(query string,args []any)
}

func NewMiddlewareBuilder()*MiddlewareBuilder{
	return &MiddlewareBuilder{
		logFunc: func(query string, args []any) {
			log.Printf("sql: %s ,args: %v \n",query,args)
		},
	}
}
func (m *MiddlewareBuilder)LogFunc(fn func(query string,args []any))*MiddlewareBuilder  {
	m.logFunc=fn
	return m
}
func(m MiddlewareBuilder)Build()orm.Middleware{
	return func(next orm.Handler) orm.Handler {
		return func(ctx context.Context, qc *orm.QueryContext) *orm.QueryResult {
			q,err:=qc.Builder.Build()
			if err!=nil{
				//要考虑记录下来吗？
				//log.Println("构造 SQL 出错",err)
				return &orm.QueryResult{
					Err: err,
				}
			}
			//log.Printf("sql: %s ,args: %v \n",q.SQL,q.Args)
			//交给用户输出
			m.logFunc(q.SQL,q.Args)
			res:=next(ctx,qc)
			return res
		}
	}
}
```

如何把middleware接入到orm中？

放在db中，而middleware用于所有的增删改查所以放到core中。在DB中再暴露一个Option给middleware。

##### selector改造

有了middleware之后就可以在select中改造，把get的功能放进getHandle中，get用来给getHandle添加middleware,Inserter的改造与selector相同。

```go
func (s *Selector[T])Get(ctx context.Context)(*T,error){
    root:=s.getHandler
	for i:=len(s.mdls)-1;i>=0;i--{
		root=s.mdls[i](root)
	}
    res:= root(ctx,&QueryContext{
        Type:"SELECT",
        Builder:s,
    })
    if res.Result!=nil{
        return res.Result.(*T),res.Err
    }
    return nil,res.Err
}

func (s *Selector[T])getHandler[T any](ctx context.Context,qc *QueryContext) *QueryResult{
   q,err:=s.Build()
   if err!=nil{
      return &QueryResult{
         Err: err,
      }
   }
   //在这里发起查询并处理结果集
   rows,err:=s.sess.queryContext(ctx,q.SQL,q.Args...)
   //这是查询错误，数据库返回的
   if err!=nil{
      return &QueryResult{
         Err: err,
      }
   }
   //将row 转化成*T
   //在这里处理结果集
   if !rows.Next(){
      //要不要返回error
      //返回error,和sql包语义保持一致 sql.ErrNoRows
      //return nil, ErrNoRows
      return &QueryResult{
         Err: ErrNoRows,
      }
   }
   tp:=new(T)
   creator:=c.creator
   val:=creator(c.model,tp)
   err=val.SetColumns(rows)
   return tp,err
}
```

##### middleware增强

我们希望m.Trace.Start(ctx,"","")的span name是select-table_name即类型和表名的结合，所以需要增强一下QueryContext，向其中添加一个model字段以获取表名。

```go
type QueryContext struct {
   // 查询类型，标记增删改查
   Type string
   //代表的是查询本身,大多数情况下需要转化到具体的类型才能篡改查询
   Builder QueryBuilder
   //一般都会暴露出来给用户做高级处理
   Model *model.Model
}
```

那么显然的，在Get构造QueryContext时要加上model,但是s中的model直到build时才会被赋值，那么我们可以考虑：

提前给model赋值，在Get中加上

```go
    var err error
    s.model,err=s.r.Get(new(T))
    if err!=nil{
       return nil, err
    }
```

或者专门给一个middleware给添加model。

### 集成测试

orm框架：确保和数据库交互返回结果正确。

#### TestSuite

要使用不同的数据库，使用TestSuite:

1.它提供了一种分组机制效果

2.隔离：套件之间允许独立运行。

3.生命周期回调(钩子)：允许在套件前后执行一些动作。

4.参数控制：可用不同参数多次运行同一套件。

使用：

在一个结构体中集成suite.Suite，SetupSuite用来初始化db。

```go
type Suite struct {
   suite.Suite
   driver string
   dsn string
   db *orm.DB
}
// SetupSuite 所有suite执行前的钩子
func (s *Suite)SetupSuite(){
	db,err:=orm.Open(s.driver, s.dsn)
	require.NoError(s.T(), err)
	db.Wait()
	s.db=db
}
```

Wait是用来等待数据库启动并连接的，在DB上新增Wait方法：

```go
//Wait 主动等待数据库启动
func (d *DB) Wait()error{
   err:=d.db.Ping()
   //循环等待 
   for err==driver.ErrBadConn{
      log.Println("等待数据库启动...")
      err = d.db.Ping()
      time.Sleep(time.Second)
   }
   return err
}
```

想要进行什么测试，就用相应的结构体来集成Suite。

结构体上仍然可以再次定义SetupSuite，Suite中的用于在所有实例前运行，特定结构体的用于运行在特定实例上。

```go
type SelectSuite struct {
   Suite
}
//测试的进入方法
func TestMySQLTest(t *testing.T){
   suite.Run(t, &SelectSuite{
      Suite{
         driver: "mysql",
         dsn: "root:root@tcp(localhost:13306)/integration_test",
      },
   })
}
//TearDownSuite 所有都跑完清数据
func (s *InsertSuite)TearDownSuite(){
   orm.RawQuery[test.SimpleStruct](s.db,"TRUNCATE TABLE `simple_struct`").Exec(context.Background())
}
//Select的SetupSuite用来插入数据
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
       //t替换为s.T() 
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
```

#### 数据的准备

select时，把数据准备好，测试全部完成后删除。

insert时，数据单独准备，每个用例完成后删除。

#### 标签

在头部添加//go:build tag    那么在go test -tags=tag ./...,如果不加tag那么就不会测试有标签的测试。

### 原生查询

我们设计的orm select显然不能完全满足select的语法，那么就要给用户提供绕过orm框架写查询语句的机制，而结果集可通过orm框架也可通过sql.DB来封装。

#### 设计

显然的，我们需要原生的支持增删改查，那么就需要实现Querier(支持select)和Executor(支持增删改)和QueryBuilder(Build创建语句)。

```go
type RawQuerier[T any] struct {
   core
   sess Session
   //存储语句和参数
   sql string
   args []any
}
//需要一个构造函数来创建rawQuery，所以再实现QueryBuilder
func (r RawQuerier[T]) Build() (*Query, error) {
	return &Query{
		SQL: r.sql,
		Args: r.args,
	},nil
}
//需要一个构造函数来创建rawQuerier
func RawQuery[T any](sess Session,query string,args...any)*RawQuerier[T]{
	c:=sess.getCore()
	return &RawQuerier[T]{
		sql: query,
		args: args,
		sess: sess,
		core:c,
	}
}
func (i RawQuerier[T]) Exec(ctx context.Context) Result {
	if i.model==nil{
		var err error
		i.model,err=i.r.Get(new(T))
		if err!=nil{
			return Result{
				err: err,
			}
		}
	}

	res:=exec(ctx,i.sess,i.core,&QueryContext{
		Type: "RAW",
		Builder: i,
		Model: i.model,
	})

	var sqlRes sql.Result
	if res.Result!=nil{
		sqlRes = res.Result.(sql.Result)
	}
	return Result{
		err: res.Err,
		res:sqlRes,
	}
}
//实现的Querier的Get跟Selector中的差不多相同，但是getHandler定义在selector上，所以尝试把getHandler拆出来.
func (s RawQuerier[T]) Get(ctx context.Context) (*T, error) {
		var err error
    	//r从哪来？在RawQuerier中组合一个core
		s.model,err=s.r.Get(new(T))
		if err!=nil{
			return nil,err
		}
    //session从哪来？在RawQuerier中维护一个session
	res:=get[T](ctx,s.sess,s.core,&QueryContext{
			Type: "RAW",
			Builder: s,
        	//model从哪来？只能在get之前获取
			Model: s.model,
		})
	if res.Result!=nil{
		return res.Result.(*T),res.Err
	}
	return nil,res.Err
}

func (r RawQuerier[T]) GetMulti(ctx context.Context) ([]*T, error) {
	panic("implement me")
}

//由于get getHandler exec execHandler是通用的，所以将这些方法放入core中。
//由于在selector和RawQuerier中的Get逻辑非常相似，所以把get也提取出来
func get[T any](ctx context.Context,sess Session,c core,qc  *QueryContext)*QueryResult{
    //不符合方法签名
	//var root Handler = getHandler[T](ctx,s.sess,s.core,&QueryContext{
	//	Type: "RAW",
	//	Builder: s,
	//	Model: s.model,
	//})
    //为了使用getHandler，所以我们get也需要传入sess,c,qc
	var root Handler = func(ctx context.Context, qc *QueryContext) *QueryResult {
		return getHandler[T](ctx,sess,c,qc)
	}
	for i:=len(c.mdls)-1;i>=0;i--{
		root=c.mdls[i](root)
	}
	//return root(ctx,&QueryContext{
	//	Type: "RAW",
	//	Builder: builder,
	//	//问题在于s.model在Build时才会赋值，1.在Get初始化s.model 2.专门设置一个middleware来设置model
	//	Model: c.model,
	//})
	return root(ctx,qc)
}
//拆除来的getHandler缺少了selector中的sess和core那么我们就给它传入sess和core，build就是qc中的Builder
func getHandler[T any](ctx context.Context,sess Session,c core,qc *QueryContext) *QueryResult{
	q,err:=qc.Builder.Build()
	if err!=nil{
		return &QueryResult{
			Err: err,
		}
	}
	//在这里发起查询并处理结果集
	rows,err:=sess.queryContext(ctx,q.SQL,q.Args...)
	//这是查询错误，数据库返回的
	if err!=nil{
		return &QueryResult{
			Err: err,
		}
	}
	if !rows.Next(){
		return &QueryResult{
			Err: ErrNoRows,
		}
	}
	tp:=new(T)
	creator:=c.creator
	val:=creator(c.model,tp)
	err=val.SetColumns(rows)
	return &QueryResult{
		Err: err,
		Result: tp,
	}
}
```

### Join查询

### protobuf魔改

[protobuf-go](https://github.com/protocolbuffers/protobuf-go):下载了源码后，在proto-gen-go的main中找到了生成的函数GenerateFile，而我们要魔改的是生成的struct，在GenerateFile函数所在文件中找到了genMessageField，把

```go
tags := structTags{
   {"protobuf", fieldProtobufTagValue(field)},
   {"json", fieldJSONTagValue(field)},
}
```

改为：

```go
tags := structTags{
   {"protobuf", fieldProtobufTagValue(field)},
   {"json", fieldJSONTagValue(field)},
   {"orm", fieldORMTagValue(field)},
}
```

而我们自己定义的fieldORMTagValue的实现为：

```go
func fieldORMTagValue(field *protogen.Field) string {
   c:=field.Comments.Trailing.String()//Trailing就是跟在後面的，Leading是放在上面的
   c=strings.TrimSpace(c)
    //语法为 //@orm:column=xx 
   if strings.HasPrefix(c,"//@orm"){
      return c[7:]
   }
   return ""
}
```

总的流程为：

1.clone原来的protobuf-go代码库

2.修改protobuf-go代码

3.安装修改后的go插件  在protoc-gen-go文件夹下执行go install .

4.执行protoc命令

### 并发编程

#### context

#### sync.Mutex

#### sync.Once

#### sync.Pool

##### Put步骤

+ privite中要是没放数据就直接放在privite
+ 否则，准备放在poolChain
  + 如果poolChain的HEAD还没有创建，就创建一个Head，然后创建一个8容量的ring buffer,把数据丢过去
  + 如果poolChain的Head指向的ring buffer没满，直接放入
  + 如果已经满了，那么创建一个新的节点，在创建一个两倍容量的ring buffer，把数据放入

##### Pool和GC

正常情况下设计一个Pool要考虑容量和淘汰的问题：

+ 我们希望能控制住Pool的消耗量
+ 在这个前提下考虑淘汰的问题

Go的sync.Pool纯粹依赖于GC，用户完全没办法手工控制。

核心机制：

+ locals
+ victim：缓刑

一个P中的locals其实是有两个实例一个locals和一个bictim

GC过程：

+ local会挪入victim
+ victim会被直接回收

复活：如果victim的对象再次被使用则丢回locals，防止GC引起性能抖动。

##### 每个poolLocal都有一个pad字段

用于将poolLocal的内存补齐到128的整数倍

##### 为什么先偷窃再去找缓刑的？

因为Pool希望victim里的对象尽可能被回收。

##### 实例：bytebufferpool

+ 对sync进行了二次封装
+ defaultSize是每次创建的buffer的默认大小，超过maxSize的buffer就不会被放回去
+ 统计不同大小的buffer使用次数，例如0-64bytes的buffer被使用了多少次。这个称为分组统计使用次数。
+ 引入校准机制(calibrate)，就是动态计算defaultSize和maxSize

在Put中根据使用次数来决定defaultSize和maxSize。

```go
func (p *Pool) Put(b *ByteBuffer) {
   idx := index(len(b.B))//计算分组
   //分组对应的使用次数+1，大于阈值就开始校准 
   if atomic.AddUint64(&p.calls[idx], 1) > calibrateCallsThreshold {//判断有没有触发校准机制
      p.calibrate()
   }
   maxSize := int(atomic.LoadUint64(&p.maxSize))
   //没有限制或小于maxSize就放回去 
   if maxSize == 0 || cap(b.B) <= maxSize {
      b.Reset()
      p.pool.Put(b)
   }
}
```

```go
func (p *Pool) calibrate() {
    //确保只有一个人在校准，使用一个CAS操作
   if !atomic.CompareAndSwapUint64(&p.calibrating, 0, 1) {
      return
   }

   a := make(callSizes, 0, steps)
   var callsSum uint64
   for i := uint64(0); i < steps; i++ {
      //读取使用次数顺便重置一下 
      calls := atomic.SwapUint64(&p.calls[i], 0)
      //计算总次数 
      callsSum += calls
      a = append(a, callSize{
         calls: calls,
         size:  minSize << i,
      })
   }
   //按照使用次数从大到小排序
   sort.Sort(a)
	//得到使用次数最多的size设为default
   defaultSize := a[0].size
   maxSize := defaultSize
   //调用次数要超过maxPercentile比例，设遍历的最大的为maxSize
   maxSum := uint64(float64(callsSum) * maxPercentile)
   callsSum = 0
   for i := 0; i < steps; i++ {
      if callsSum > maxSum {
         break
      }
      callsSum += a[i].calls
      size := a[i].size
      if size > maxSize {
         maxSize = size
      }
   }

   atomic.StoreUint64(&p.defaultSize, defaultSize)
   atomic.StoreUint64(&p.maxSize, maxSize)
   //设计标记位代表校准完毕
   atomic.StoreUint64(&p.calibrating, 0)
}
```

#### sync.WaitGroup

用于同步多个goroutine

常见场景是把任务拆分个多个goroutine并行完成，在完成后需要合并这些结果。

开启goroutine之前要wg.Add(1),完成后要wg.Done,Wait用来等待所有子任务完成。

##### 设计

要实现waitGroup至少需要：

+ 记住当前有多少个任务还没有完成
+ 记住当前有多少goroutine调用了wai方法
+ 需要一个东西来协调goroutine的行为

所以按照道理，用三个字段来承载，搞个锁来维护这三个字段

##### 细节

```go
type WaitGroup struct {
   noCopy noCopy
   state1 uint64
   state2 uint32
}
type noCopy struct{}
//实现这两个方法，编译器就会认为noCopy就是一个锁结构，锁是没办法复制的。
func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}
```

nocopy：主要用于告诉编译器这个东西不能复制。

state1:在64位下，高32位记录还有多少任务在执行；低32位记录了有多少goroutine在等Wait()方法返回。

state2：信号量，用于挂起或唤醒goroutine,约等于Mutex中的Sema.

本质上，Wait Group是一个无锁实现，严重依赖CAS对state1的操作。

##### 实现细节

Add:state1的高32位自增一，原子操作一把梭

Done:state1的高32位自减一，原子操作一把梭，然后看是不是要唤醒等待goroutine。相当于Add(-1),因为Add(-1),所以唤醒操作也在Add中

wait：state1的低32位自增一，同时利用state2和runtine_Semacquire调用把当前goroutine挂起。里面使用CAS因为高32位可能也在操作。

#### channel

要点：

+ 带不带缓冲
+ 谁在发
+ 谁在收
+ 谁来关
+ 关了没

##### 缓冲

不带缓冲：收发两端必须有goutine，否则阻塞。

带缓冲：没满或没空前不会阻塞。

##### 利用思路

+ 看作是队列，主要用于传输数据
+ 利用阻塞特性，可以间接控制goroutine或者其他资源的消耗，有点像是令牌机制。

##### 发布订阅模式

利用channel实现发布订阅模式很简单，进程内的事件驱动可以依托与channel来实现。

缺点：

+ 没有消费组概念。不能说同一个事件被多个goroutine同时消费
+ 无法回退，也无法随即消费。并发队列可以解决此问题。

##### 利用channel实现基于内存的消息队列并有消费组的概念

思路：难在channel内的元素只能被一个goroutine取出来。

+ 方案一：每一个消费者订阅的时候，创建一个子channel
+ 方案二：轮询所有消费者

实现：

结构体中一个读写锁来保护一个消息的channel切片

```go
type Broker struct {
   mutex sync.RWMutex
   chans []chan Msg
}
type Msg struct {
	Content string
}
```

向消息队列发送消息

```go
// Send 向消息队列发数据,Msg不用指针是因为如果在接受时修改数据，其他消费者也会受到影响
func(b *Broker)Send(m Msg)error{
   b.mutex.RLock()
   defer b.mutex.RUnlock()
   for _,ch:=range b.chans{
      //ch <- m//这样写cap放满了这里会阻塞住，使用select
      select {
      case ch<-m:

      default:
         return errors.New("消息队列已满")
      }
   }
   return nil
}
```

订阅消息队列，创建一个子channel，向这个channel塞数据，这个消费者就可以消费

```go
// Subscribe 订阅    <-chan Msg  代表只读
func(b *Broker)Subscribe(cap int)(<-chan Msg,error){
   //该给多少缓冲?设置cap让用户管
   res:=make(chan Msg,cap)
   b.mutex.Lock()
   defer b.mutex.Unlock()
   b.chans=append(b.chans,res)
   return res,nil
}
```

关闭消息队列

```go
func (b *Broker)Close() error {
   b.mutex.Lock()
   chans:=b.chans
   //避开了b.chans被重复关闭的问题
   b.chans=nil
   b.mutex.Unlock()
   for _,ch:=range chans{
      close(ch)
   }
   return nil
}
```

##### 实现一个任务池

任务池允许开发者提交任务，并设定最多多少个goroutine同时运行。

难在决策：

+ 提交任务后，如果执行goroutine满了，任务池是缓存住这个任务还是直接阻塞提交者
+ 如果缓存，那么缓存需要多大？缓存满了该怎么办？

实现：

定义：task chan：task的缓存，close:用来控制channel

```go
type Task func()
type TaskPool struct {
	tasks chan Task
	//close *atomic.Bool
	//一般用这个
	close chan struct{}
	//closeOnce sync.Once
}
```

方法：

```go
// NewTaskPool nunG就是goroutine的数量，capacity是缓存的容量
func NewTaskPool(numG int,capacity int){
   res:=&TaskPool{
      tasks: make(chan Task,capacity),
      close: make(chan struct{}),
   }
   for i:=0;i<numG;i++{
      go func() {
         for{
            select {
            //Close运行之后，所有的goroutine都会return    
            case <-res.close:
               return
            case t:=<-res.tasks:
               t()
            }
         }
         //for t:=range res.tasks{
         // if res.close.Load(){
         //    return
         // }
         // t()
         //}
      }()
   }
}
//Submit 提交任务 task满了会被阻塞
func(p *TaskPool)Submit(ctx context.Context,t Task)error{
	select {
	case p.tasks<-t:
	case <- ctx.Done():
		//让用户自己判断是超时还是取消
		return ctx.Err()
	}
	return nil
}

// Close 开了goroutine，channel一定要设置一个Close方法迎来控制
func(p *TaskPool)Close()error{
	//p.close.Store(true)
	//这种写法是不行的
	//p.close<-struct{}{}
	//直接关闭channel，这种实现又有一种缺陷，重复调用close会panic
	close(p.close)
	//不建议使用once控制，不需要考虑这么周全，可以在方法注释中直接告诉用户不要重复调用
	//p.closeOnce.Do(func() {
	//	close(p.close)
	//})
	return nil
}
```

##### channel原理

###### goroutine泄露：

+ 只发送不接收会导致发送者goroutine泄露
+ 只接受不发送，接收者会一直阻塞，会导致接收者goroutine泄露
+ 读写nil都会导致goroutine泄露，通常是因为忘记初始化

基本上，goroutine泄露都是因为goroutine被阻塞后没人唤醒它导致。

唯一的例外是业务层面上的goroutine长时间运行。

###### 如何判断泄露：

看runtime.numGoroutine的变化趋势,泄露时会出现可能有波动，但总体是上涨。寻找具体的使用pprof把状态dump下来。

###### 内存逃逸:

内存分配：

+ 分配到栈：分配很快，不需要考虑GC
+ 分配到堆：需要考虑GC

如果使用channel发送指针，那么必然逃逸。分配到栈上有一个前提是要直到是谁的栈，但是发送指针，编译器无法确定，发送的指针数据最终会被哪个goroutine接收，所以只能分配到堆。

###### 实现细节：

+ 要设计缓冲来存储数据，无缓冲=缓冲容量为0
+ 要能阻塞goroutine，也要能唤醒
+ 要维持住goroutine的等待队列，并且是收和发两个队列

buf是一个ring buffer结构用于存储数据，提高效率。因为channel的缓存是固定容量的，就可以复用一个ring buffer。

recvq,sendq都是一个waitq的实例，waitq是双向列表，就是一个等待队列。

chansend方法：

1.看是不是nil channel，是的话直接阻塞

2.看有没有被阻塞的接收者，有的话直接交付数据，返回

3.看缓冲有没有满，没有就缓冲，返回

4.阻塞，等待接收者来唤醒自己

5.被唤醒做一些清理工作

数据不被GC是靠keepalive来维持数据的。

chanrecv方法：

1.看是不是nil channel，是的话阻塞

2.看有没有被阻塞的发送者

​	2.1如果没有缓冲，直接拿数据，返回

​	2.2否则，从缓冲队首拿数据，并将发送者数据放到队尾，返回

3.看缓冲有没有数据，有就读缓冲，返回

4.阻塞，等待发送者唤醒

5.被唤醒做一些清理工作

#### 缓存模块

对性能有要求都会使用缓存

分为两类：

+ 本地缓存
+ 分布式缓存，如：redis，memcache

##### API设计

Beego:主要分成单个操作，批量操作，针对数字的自增自减。

go-cache:分成单个操作，针对数字的加减操作。

考虑接口，那么最简单的就是set,get,delete

```go
type Cache interface {
   // Set 方法会设置一个过期时间
   Set(ctx context.Context,key string,val any,expiration time.Duration)error
   Get(ctx context.Context,key string)(any,error)
   Delete(ctx context.Context,key string)(any,error)
}
```

不使用泛型是因为如果使用泛型那么存取数据被限制了类型

理想形态是接口不使用泛型而方法上使用泛型，但是go限制不能有泛型方法。

##### 本地缓存实现

数据以map形式存储，让它实现Cache结构体，使用sync.Map做不到精细的控制。

Get就是从map中取值。

delete就是删除map中的值

set就是设置值

那么设计为下

```go
type BuildInMapCache struct {
   data map[string]*item
   // 加锁保护数据
   mutex sync.RWMutex
   close chan struct{}
}
// item 为值加上超时控制
type item struct {
	val any
	//expiration 超时时间
	expiration time.Time
}

func (i *item) deadlineBefore(t time.Time) bool {
	return !i.expiration.IsZero() && i.expiration.Before(t)
}
```

###### 如何做set的过期时间控制？

1.每个key使用一个goroutine盯着，过期就执行删除策略

time.AfterFunc(expiration,func(){delete(data,key)}}可以替代goroutine

这样写，假如第十秒设置了key1=value1,过期时间一分钟，第三十秒设置key1=value2，那么在一分钟时还是会被删掉。

那么就把value从any类型封装成一个item加一个deadline用于更新。

key多了goroutine就多了，大部分会阻塞，浪费资源

2.用一份goroutine定时轮询，找到所有过期的key，然后删除。

创建cache时，同时创建一个goroutine，这个goroutine会检查每个key的过期时间，过期则执行删除。

要点：

+ 控制检查的间隔：如果间隔过短，影响用户，资源消耗大
+ 控制遍历的资源开销：如果全部key遍历一遍，可能耗时极长
  + 可以控制遍历的时长，比如每次1ms
  + 可以控制遍历的长度，比如每次1000个



3.啥也不干，访问key时检查过期时间

在Get时添加一次检查。如果过期也不能直接删除，因为可能你在执行此代码时另一个goroutine更新了这个key，那么就不能删除，所以要在锁中再检查一次即，double-check。只使用这个也不行，因为如果出现一直set却没有get的情况，会出现问题。

Redis的过期处理，也是用的类似套路：Get、时检查是否过期，遍历key找出过期的删掉，同样的redis轮询时同样要控制资源开销。

sql.DB空闲连接：也不是空闲多少秒就关掉，也面临一样的性能问题，所以采用的是懒惰关闭，只在Get时检查，因为可能底层TCP已经超时关掉了，所以还是需要判断而不能直接拿。

###### 轮询实现

具体实现：NewBuildInMapCache

在NewBuildInMapCache时最好预估一个容量，要不就是传入一个size，要不就是自己预估,在里面开一个goroutine用来ticker(轮询)，使用for循环 for t:=range ticker.C在里面遍历map判断超时。这个goroutine如何退出？只能设计一个close方法，在struct中加一个close chan struct{}作为信号，使用for+select.

```go
// NewBuildInMapCache 新建cache,并且建立一个goroutine轮询
func NewBuildInMapCache(interval time.Duration)*BuildInMapCache{
	res:=&BuildInMapCache{
		data:  make(map[string]*item,100),
		close: make(chan struct{}),
	}
	//如何关闭这个goroutine？在结构体中维护一个channel来关闭
	go func() {
		//创建定时器
		ticker:=time.NewTicker(interval)
		//轮询
		//for t:=range ticker.C{
		//	i:=0
		//	for k,v:=range res.data{
		//		//要是过期时间不为0并且在t之前，那么就代表Key过期
		//		if v.deadlineBefore(t){
		//			delete(res.data,k)
		//		}
		//		i++
		//		//轮询一千个数后开始下一次轮询
		//		if i>1000{
		//			break
		//		}
		//	}
		//}
		//为了能够关闭goroutine，使用select
		for{
			select {
			case t:=<-ticker.C:
				res.mutex.Lock()
				i:=0
				for k,v:=range res.data{
					//要是过期时间不为0并且在t之前，那么就代表Key过期
					if v.deadlineBefore(t){
						delete(res.data,k)
					}
					i++
					//轮询一千个数后开始下一次轮询
					if i>1000{
						break
					}
				}
				res.mutex.Unlock()
			case <-res.close:
				return
			}
		}
	}()
	return res
}
```

单独使用定时轮询肯定不行，因为一个key可能已经过期了但是没有轮询到，所以要配合访问key时检查过期时间

###### Set实现

当前时间加上传入的过期时长就是过期时间

```go
func (b *BuildInMapCache) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
   var dl time.Time
   //如果expiration = 0那么它就是没有过期时间
   if expiration>0{
      dl=time.Now().Add(expiration)
   }
   b.data[key]=&item{
      val:        val,
      expiration: dl,
   }
   return nil
}
```

###### Get实现

先读数据，没有则返回error，如果有则判断超时，这里要使用double-check防止其它goroutine更新了此key但没有察觉

```go
// Get 在Get时判断key有没有超时
func (b *BuildInMapCache) Get(ctx context.Context, key string) (any, error) {
   //先进行读取数据 
   b.mutex.RLock()
   val,ok:=b.data[key]
   b.mutex.RUnlock()
   if !ok{
      return nil,errNoValue
   }
   b.mutex.Lock()
   defer b.mutex.Unlock()
   //使用double-check，防止在加写锁之前有goroutine更新
   val,ok=b.data[key]
   if !ok{
      return nil,errNoValue
   }
   if val.deadlineBefore(time.Now()){
      delete(b.data,key)
   }
   return val, nil
}
```

###### Delete实现

简单的加锁后delete

```go
// Delete 删除map值,同时也返回值
func (b *BuildInMapCache) Delete(ctx context.Context, key string) (any, error) {
   b.mutex.RLock()
   res,ok:=b.data[key]
   b.mutex.RUnlock()
   if !ok{
      return nil,errNoValue
   }
   delete(b.data,key)
   return res,nil
}
```

###### close实现

```go
func (b *BuildInMapCache)Close()error{
   select {
   case b.close<- struct{}{}:
      return nil
   default:
      return errors.New("cache 重复关闭")
   }
   return nil
}
```

###### 踩坑

在实际调用Close时，大概率会跑到default中。

###### evict回调与关闭

类似于redis的subscribe，数据发生变化要通知。

在本地缓存实现中，这种接口主要就是缓存过期被删除的回调。

有三个地方需要回调：

+ Delete方法
+ Get方法检查过期时间时，懒惰删除
+ 轮询删除过期key时

evict回调函数设计：

`onEvicted func(key string,val any)`

正常来说evict都是创建时，用户传进来，并且是可选的传入，所以需要使用option模式来创建cache，遍历opts应该放在新开的goroutine之前。

所以在结构体中添加回调，并且为NewBuildInMapCache，添加option。

```go
type BuildInMapCache struct {
   data map[string]*item
   // 加锁保护数据
   mutex sync.RWMutex
   close chan struct{}
   onEvicted func(key string,val any)
}
type BuildInMapCacheOption func(b *BuildInMapCache)
//添加回调的option
func BuildInMapCacheWithEvictCallBack(fn func(key string,val any))BuildInMapCacheOption{
	return func(b *BuildInMapCache) {
		b.onEvicted=fn
	}
}
```

###### 为delete添加回调

因为我们的回调主要用于删除，创造一个cache的delete方法，用于向里面添加回调，因为我们传入的是key，value，所以需要先读一下。

```go
func(b *BuildInMapCache)delete(key string){
// b.mutex.RLock() 在外部调用delete时都已经加了锁，所以在这里加锁会导致程序卡死
   val,ok:=b.data[key]
// b.mutex.RUnlock()
   if !ok{
      return
   }
   b.onEvicted(key,val.val)
   delete(b.data,key)
   return
}
```

###### 踩坑

在实现时在delete中加了锁，而调用delete之前也加了锁，导致程序卡死。

###### 测试轮询效果

使用一个整型数据作为探针

```go
//测试我们的轮询起效果
func TestNewBuildInMapCache(t *testing.T) {
   //探针
   cnt:=0
   c:=NewBuildInMapCache(time.Second,BuildInMapCacheWithEvictCallBack(func(key string, val any) {
      cnt++
   }))
   err:=c.Set(context.Background(),"key1",123,time.Millisecond)
   require.NoError(t, err)
   time.Sleep(3*time.Second)
   c.mutex.RLock()
   defer c.mutex.RUnlock()
   var _, ok = c.data["key1"]
   require.False(t, ok)
   require.Equal(t, cnt,1)
}
```

###### 控制本地缓存内存

大多数时候都要考虑控制住内存使用量。在考虑内存使用量时要考虑缓存快满了的时候怎么腾出空间来。腾出空间就引出了我们常用的LRU、LFU算法。

两种策略：

+ 控制键值对数量
+ 控制整体大小：需要计算每个对象的大小，然后累加。计算对象大小需要使用递归去计算对象中的对象。

可以尝试使用装饰器模式来无侵入地支持这种功能。

MaxCntCache直接组合BuildInMapCache的指针，为onEvict添加功能，还要重写Set方法，由于解锁后在return Set之前别的goroutine拿到锁还是可能设置值，导致计数多添加了，所以需要把Set中设置值的方法提取出来，这样就在锁住的时候进行set，把解锁用defer就不会出现并发问题了。

设计：

MaxCntCache直接组合BuildInMapCache的指针，并保存最大个数和当前个数：

```go
type MaxCntCache struct {
   *BuildInMapCache
   maxCnt int32
   cnt int32
}
```

在创建MaxCntCache时为onEvict新增计数减一功能：

```go
func NewMaxCntCache(b *BuildInMapCache,max int32)*MaxCntCache{
   res:=&MaxCntCache{
      BuildInMapCache: b,
      maxCnt:          max,
      cnt:             0,
   }
   evict:=b.onEvicted//原本的回调函数
   b.onEvicted= func(key string, val any) {
      atomic.AddInt32(&res.cnt,-1)
      if evict!=nil{
         evict(key,val)
      }
   }
   return res
}
```

重写Set

```go
func (c *MaxCntCache) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
   // 这种写法，如果 key 已经存在，你这计数就不准了
   //cnt := atomic.AddInt32(&c.cnt, 1)
   //if cnt > c.maxCnt {
   // atomic.AddInt32(&c.cnt, -1)
   // return errOverCapacity
   //}
   //return c.BuildInMapCache.Set(ctx, key, val, expiration)
	//这样写在unlock后如果有goroutine拿到锁后再次进行set而此goroutine中的Set还未结束的话仍然会导致计数错误
   //c.mutex.Lock()
   //_, ok := c.data[key]
   //if !ok {
   // c.cnt ++
   //}
   //if c.cnt > c.maxCnt {
   // c.mutex.Unlock()
   // return errOverCapacity
   //}
   //c.mutex.Unlock()
   //return c.BuildInMapCache.Set(ctx, key, val, expiration)

   c.mutex.Lock()
   defer c.mutex.Unlock()
   _, ok := c.data[key]
   if !ok {
      if c.cnt + 1 > c.maxCnt {
         // 后面，你可以在这里设计复杂的淘汰策略
         return errOverCapacity
      }
      c.cnt ++
   }
   return c.set(key, val, expiration)
}
```

##### Redis实现

使用go-redis/redis/v9

###### 设计

RedisCache包含redis的客户端  redis.Cmdable,NewRedisCache时传入 redis.Cmdable，如果不是传入redis.Cmdable而是传一些config，类似addr之类的会很麻烦，要我们进行Cmdable初始化，而Cmdable有很多实现。传入 redis.Cmdable是一种依赖注入，在应用程序启动时，肯定会初始化。依赖注入，我不自己创建，让用户自己传入，我不需要关心Cmdable是哪种实现。

```go
var (
   errFailedToSetCache = errors.New("cache: 写入 redis 失败")
)
type RedisCache struct {
   client redis.Cmdable
}
func NewRedisCache(client redis.Cmdable)*RedisCache{
   return &RedisCache{client: client}
}

func (r *RedisCache) Set(ctx context.Context, key string, val any, expiration time.Duration) error {
   val,err:=r.client.Set(ctx,key,val,expiration).Result()
   if err!=nil{
      return err
   }
   if val!="OK"{
      return errors.New(fmt.Sprintf("%w ,res: %s",errFailedToSetCache,err))
   }
   return nil
}

func (r *RedisCache) Get(ctx context.Context, key string) (any, error) {
   return r.client.Get(ctx,key).Result()
}

func (r *RedisCache) Delete(ctx context.Context, key string) (any, error) {
   return r.client.GetDel(ctx,key).Result()
}
```

###### 单元测试

单元测试如何生成client？使用mockgen来测试redis，在根目录下执行mockgen,因为单元测试不希望连上redis。testCases中的mock设计为传入*gomock.Controller返回redis.Cmdable的函数，使用gomock.NewController来创建控制器。

在根目录下执行mockgen生成代码

mockgen -destination=cache/mocks/mock_redis_cache.gen.go -package=mocks github.com/go-redis/redis/v9 Cmdable

首先创建一个mock控制器：使用gomock.NewController(t)来创建controller，有了控制器就可以使用生成的代码来使用控制器。

在测试用例中用mock func(ctrl *gomock.Controller)redis.Cmdable来创建cmdable，之后使用此cmdable执行EXPECT执行Set，return的Status使用redis.NewStatusCmd来创建以来模拟return。

Set测正常设置，超时和返回不正常状态

```go
func TestRedisCache_Set(t *testing.T) {
   testCases:=[]struct{
      name string
      mock func(ctrl *gomock.Controller)redis.Cmdable
      key string
      val string
      expiration time.Duration
      wantErr error
   }{
      {
         name:       "set val",
         mock: func(ctrl *gomock.Controller) redis.Cmdable {
            cmd:=mocks.NewMockCmdable(ctrl)
            status:=redis.NewStatusCmd(context.Background())
            status.SetVal("OK")
            cmd.EXPECT().Set(context.Background(),"key1","value1",time.Second).Return(status)
            return cmd
         },
         key:        "key1",
         val:        "value1",
         expiration: time.Second,
      },
      {
         name:       "expiration",
         mock: func(ctrl *gomock.Controller) redis.Cmdable {
            cmd:=mocks.NewMockCmdable(ctrl)
            status:=redis.NewStatusCmd(context.Background())
            status.SetErr(context.DeadlineExceeded)
            cmd.EXPECT().Set(context.Background(),"key1","value1",time.Second).Return(status)
            return cmd
         },
         key:        "key1",
         val:        "value1",
         expiration: time.Second,
         wantErr: context.DeadlineExceeded,
      },
      {
         name:       "unexpected msg",
         mock: func(ctrl *gomock.Controller) redis.Cmdable {
            cmd:=mocks.NewMockCmdable(ctrl)
            status:=redis.NewStatusCmd(context.Background())
            status.SetVal("un ok")
            cmd.EXPECT().Set(context.Background(),"key1","value1",time.Second).Return(status)
            return cmd
         },
         key:        "key1",
         val:        "value1",
         expiration: time.Second,
         wantErr: errors.New(fmt.Sprintf("%v ,res: %s",errFailedToSetCache,"un ok")),
      },
   }
   for _,tc:=range testCases{
      t.Run(tc.name, func(t *testing.T) {
         ctrl:=gomock.NewController(t)
         defer ctrl.Finish()
         rdb:=NewRedisCache(tc.mock(ctrl))
         err:=rdb.Set(context.Background(),tc.key,tc.val,tc.expiration)
         assert.Equal(t, tc.wantErr,err)
      })
   }
}
```

而测试get需要把NewStatusCmd改为NewStringCmd，因为Get返回的是StringCmd。

集成测试则需要实际的redis环境，使用docker打开redis。

set在集成测试模拟不出来不"ok"和超时的情况。可以调用get来验证set。

table driver类型有一个before和after，before用来准备数据，after则用来删除数据。

```go
func TestRedisCacheGet(t *testing.T) {
   client:=redis.NewClient(&redis.Options{Addr: "localhost:6379"})
   c:=NewRedisCache(client)
   ctx, cancel := context.WithTimeout(context.Background(), time.Second * 10)
   defer cancel()
   err := c.Set(ctx, "key1", "value1", time.Minute)
   require.NoError(t, err)
   val, err := c.Get(ctx, "key1")
   require.NoError(t, err)
   assert.Equal(t, "value1", val)
}
func TestRedisCacheGetV1(t *testing.T) {
   rdb := redis.NewClient(&redis.Options{
      Addr: "localhost:6379",
   })

   testCases := []struct{
      name string
      //before func(t *testing.T)
      after func(t *testing.T)

      key string
      value string
      expiration time.Duration

      wantErr error
   } {
      {
         name:"set value",
         after: func(t *testing.T) {
            ctx, cancel := context.WithTimeout(context.Background(), time.Second * 3)
            defer cancel()
            res, err := rdb.Get(ctx, "key1").Result()
            require.NoError(t, err)
            assert.Equal(t, "value1", res)
            _, err = rdb.Del(ctx, "key1").Result()
            require.NoError(t, err)
         },
         key: "key1",
         value: "value1",
         expiration: time.Minute,
      },
   }

   for _, tc := range testCases {
      t.Run(tc.name, func(t *testing.T) {
         c := NewRedisCache(rdb)
         //tc.before()
         ctx, cancel := context.WithTimeout(context.Background(), time.Second * 3)
         defer cancel()
         err := c.Set(ctx, tc.key, tc.value, tc.expiration)
         require.NoError(t, err)
         tc.after(t)
      })
   }
}
```

##### 组合API

多个动作组合在一起，作为一个API提供出去：

+ LoadOrStore
+ LoadAndDelete
+ 自增自减API

要注意：线程安全，在本地加锁就可以，在redis中可能就要使用lua脚本。

#### 缓存模式

##### cache aside

什么缓存模式都不用cache aside,把cache当作一个普通的数据源，更新Cache和DB依赖于开发者自己写代码。

业务代码可以做决策：

+ 未命中时是否要从DB取数据，如果不从DB取可以考虑使用默认值进行业务处理
+ 同步or异步读取数据并写入：同步：缓存中没有则去DB中读取，更新缓存后继续执行业务代码。半异步：业务代码会同步的从数据库读取数据，而后用DB数据执行业务，同时异步刷新缓存。异步：缓存没有数据那么就返回没有数据，继续执行业务代码，再另开一个goroutine来更新缓存。
+ 采用singleflight：如果有10个goroutine加载key1，派一个goroutine去取，其他goroutine用此goroutine取回的数据。

**先写DB还是先写cache都会可能出现DB和cache不一致的问题，也就是不管怎么操作涉及缓存和数据库都会出现一致性问题。**

##### Read-Through

与cache aside的区别就是如果缓存没有数据，缓存去数据库中取数据，写数据的时候，业务代码需要自己写DB和写cache。

cache可以做决策：

+ 未命中时是否要从DB取数据，如果不从DB取可以考虑使用默认值进行业务处理
+ 同步or异步读取数据并写入
+ 采用singleflight

对用户来说，基本上只能是一个缓存类型一个实例

例如userCache:=%ReadThroughCache

这是因为Load Func在大多数时候没办法写成通用的。

###### 实现

装饰器模式：

ReadThroughCache组合Cache，维持一个LoadFunc func(ctx,key)(any,error),除Cache，LoadFunc，其余都为后续设计添加

```go
type ReadThroughCache struct {
   Cache
   LoadFunc func(ctx context.Context,key string)(any,error)
   expiration time.Duration
   g *singleflight.Group
}
```

只需要重新写Get方法，不管Cache是什么实现。

Get方法：

同步：先去缓存中取值，如果没有值，则使用LoadFunc来从数据库取值并set，那么超时时间从哪来？只能在ReadThroughCache放一个，要告诉用户ReadThroughCache一定要赋值LoadFunc和Expiration。如果Set Error了怎么办，返回一个哨兵错误，也可以不返回错误，也可以在log中输出。

```go
// Get 同步刷新缓存
func(r *ReadThroughCache)Get(ctx context.Context,key string)(any,error){
   //先从cache中取得值
   val,err:=r.Cache.Get(ctx,key)
   //没有值就可以进行LoadFunc
   if err==errNoValue{
      v,err:=r.LoadFunc(ctx,key)
      //LoadFunc成功
      if err==nil{
         //取得值后就刷新缓存
         er:=r.Set(ctx,key,v,r.expiration)
         if er!=nil{
            return nil,errors.New(fmt.Sprintf("%v,res: %s",errFailToRefreshCache,er))
         }
      }
   }
   return val, err
}
```

异步：如果没有值就开一个goroutine去取值并做后续操作，但由于是个goroutine，如果Set Error了就不能返回错误，只能log。

```go
// GetV2 异步刷新缓存，就是在缓存为空后异步的读取值和刷新缓存，在Get后开个goroutine即可
func(r *ReadThroughCache)GetV2(ctx context.Context,key string)(any,error){
   //先从cache中取得值
   val,err:=r.Cache.Get(ctx,key)
   //没有值就可以进行LoadFunc
   if err==errNoValue{
      go func() {
         v,err:=r.LoadFunc(ctx,key)
         //LoadFunc成功
         if err==nil{
            //取得值后就刷新缓存
            er:=r.Set(ctx,key,v,r.expiration)
            if er!=nil{
               //由于是goroutine，所以只能log记录一下
               log.Printf("%v,res: %s",errFailToRefreshCache,er)
            }
         }
      }()
   }
   return val, err
}
```

半异步：在LoadFunc后开一个goroutine并做后续操作。

```go
// GetV1 半异步刷新缓存，就是在取得值后异步的刷新缓存，在LoadFunc后开个goroutine即可
func(r *ReadThroughCache)GetV1(ctx context.Context,key string)(any,error){
   //先从cache中取得值
   val,err:=r.Cache.Get(ctx,key)
   //没有值就可以进行LoadFunc
   if err==errNoValue{
      v,err:=r.LoadFunc(ctx,key)
      //LoadFunc成功
      if err==nil{
         go func() {
            //取得值后就刷新缓存
            er:=r.Set(ctx,key,v,r.expiration)
            if er!=nil{
               //由于是goroutine，所以只能log记录一下
               log.Printf("%v,res: %s",errFailToRefreshCache,er)
            }
         }()
      }
   }
   return val, err
}
```

singleflight:可以在ReadThroughCache中再放一个g *singleflight.Group,那么在g.Do中做LoadFunc和后续操作。

```go
// GetV3 Singleflight 在ReadThroughCache中再放一个g *singleflight.Group
func(r *ReadThroughCache)GetV3(ctx context.Context,key string)(any,error){
   //先从cache中取得值
   val,err:=r.Cache.Get(ctx,key)
   //没有值就可以进行LoadFunc
   if err==errNoValue{
      val,err,_=r.g.Do(key, func() (interface{}, error) {
         v,er:=r.LoadFunc(ctx,key)
         //LoadFunc成功
         if er==nil{
            //取得值后就刷新缓存
            er=r.Set(ctx,key,v,r.expiration)
            if er!=nil{
               return nil,errors.New(fmt.Sprintf("%v,res: %s",errFailToRefreshCache,er))
            }
         }
         return v,er
      })
   }
   return val, err
}
```

可以考虑使用泛型，这样就可以强制用户指定这个ReadThroughCache是用于哪个结构的。

##### write-Through

开发者只需要写入cache,cache会更新数据库，在读未命中缓存的情况下，开发者需要自己去数据库捞数据，然后更新缓存(此时缓存不需要更新DB了)。

cache可以做决策：

+ 同步or异步写数据到DB，或者到cache
+ cache可以自由决定是先写DB还是先写cache。同步：cache会同步的将数据刷新到DB，而后返回相应，同时异步刷新缓存。异步：将请求给cache后就返回。

###### 设计

与readThrough相似，同样组合Cache，写一个StoreFunc func(ctx,key,val)error

重写Set，先写DB还是先写cache都会出现不一致的问题，所以两个顺序不是很重要

write_through与read_through设计类似，不过是一个写LoadFunc一个写StoreFunc

##### write-back

在写操作时写了缓存直接返回，不会直接更新数据库，读也是直接读缓存。在缓存过期时，将缓存写回去数据库。

优缺点：

+ 所有goroutine都是读写缓存，不存在一致性问题(如果是本地缓存依旧会有问题)
+ 数据可能丢失：如果在缓存过期刷新到数据库之前，缓存宕机，那么会丢失数据

如果不考虑丢失数据，那么它就是一致的。

主要时利用onEvicted回调，在里面将数据刷新到DB里。

用的不多，因为非常担忧数据丢失。

##### refresh-ahead

依赖于CDC(changed data capture)接口：

+ 数据库暴漏数据变更接口
+ cache或第四方监听到数据变更后自动更新数据
+ 如果读cache未命中，依旧要刷新缓存的话，依然会出现并发问题。

#### 缓存异常

##### 缓存穿透

+ 读请求对应的数据根本不存在，因此每次都会发起数据库请求，数据库返回NULL，所以下一次请求依旧会打到数据库。
+ 关键点就是这个数据根本没有，所以不会回写缓存。
+ 一般是黑客使用了一些非法的请求。

##### 缓存击穿

+ 缓存没有对应key的数据而DB有
+ 一般情况下，不会导致严重问题，但是如果该key的访问量非常大，都去数据库查询，可能压垮数据库。一般没有问题，因为数据库对读请求的支持非常大。
+ 击穿和穿透比起来，关键在于击穿本身数据在DB中是有的，只是缓存里没有，所以只要回写到缓存，此一次访问就是命中缓存。

##### 缓存雪崩：

+ 同一时刻，大量key过期，查询都要回查数据库
+ 常见场景是缓存预热，在启动时加载缓存，因为所有key的过期时间都一样，所以都在同一时间过期

共性都是大量请求落在数据库，所以解决思路就是让这些请求不会落到数据库。

##### singleflight

此设计模式能够有效的减轻对数据库的压力。

对数据库的压力本来是跟QPS相当，变为跟同一时刻不同key的数量和实例数量相当。热点越集中的应用效果越好。

普通的singleflight是和cache aside一起使用的。也可以和read through结合做成一个装饰器模式。

###### 实现一：

SingleflightCache组合ReadThroughCache,在newSingleflightCacheV1中传入cache,loadfunc,expiration复写掉他们。与在readThrough直接放入一个singflight不同，这样是非侵入式的设计。

这样只关注loadFunc，不需要重写Get方法。

```go
type SingleFlightCache struct {
   ReadThroughCache
}
//NewSingleFlightCache 中传入cache,loadfunc,expiration复写掉他们。与在readThrough直接放入一个singflight不同，这样是非侵入式的设计。
func NewSingleFlightCache(cache Cache,loadFunc   func(ctx context.Context, key string) (any, error),expiration time.Duration)*SingleFlightCache{
   return &SingleFlightCache{ReadThroughCache{
      Cache:      cache,
      //只关注loadfunc而不关注同步异步
      LoadFunc: func(ctx context.Context, key string) (any, error) {
         g:=&singleflight.Group{}
         val,err,_:=g.Do(key, func() (interface{}, error) {
            return loadFunc(ctx,key)
         })
         return val, err
      },
      expiration: expiration,
   }}
}
```

###### 实现二：

简单的装饰器模式，SingleflightCacheV1组合ReadThroughCache，持有一个singleflight.Group。Get的实现还是跟上述ReadThroughCache相同。

```go
type SingleFlightCacheV1 struct {
   ReadThroughCache
   g *singleflight.Group
}
func(r *SingleFlightCacheV1)Get(ctx context.Context, key string)(any, error){
   val, err := r.Cache.Get(ctx, key)
   if err == errNoValue {
      val, err, _ = r.g.Do(key, func() (interface{}, error) {
         v, er := r.LoadFunc(ctx, key)
         if er == nil {
            //_ = r.Cache.Set(ctx, key, val, r.Expiration)
            er = r.Cache.Set(ctx, key, val, r.expiration)
            if er != nil {
               return v, fmt.Errorf("%w, 原因：%s", errFailToRefreshCache, er.Error())
            }
         }
         return v, er
      })
   }
   return val, err
}
```

##### 缓存穿透解决方案

+ 使用singleflight能够缓解缓存问题，但如果攻击者时构造大量的不同的不存在的key，那么效果就不好了
+ 知道数据库里根本没有数据，缓存未命中就直接返回
  + 缓存里是全量数据，如果未命中就可以直接返回
  + 使用**布隆过滤器**，bit array等结构，未命中时就问一下这些结构

+ 缓存没有，直接使用默认值
+ 缓存未命中回表查询时，加上限流器

###### 综合BloomFilter

BloomFilter认为key存在，才会最终去DB查询。认为有不一定有，认为没有一定没有。

###### 实现

BoolmFilter接口有一个HasKey(ctx,key)bool方法

```go
type BloomFilter interface {
   HasKey(ctx context.Context, key string) bool
}
```

BoolmFilterCache组合ReadThroughCache，在NewBoolmFilterCache传入Cache和BoolmFilter和loadfunc，对LoadFunc方法进行装饰，跟singleflight类似。

```go
// BloomFilterCache 直接组合ReadThroughCache
type BloomFilterCache struct {
   ReadThroughCache
}

func NewBloomFilterCache(cache Cache, filter BloomFilter, LoadFunc func(ctx context.Context, key string) (any, error)) *BloomFilterCache {
   return &BloomFilterCache{ReadThroughCache{
      Cache: cache,
      LoadFunc: func(ctx context.Context, key string) (any, error) {
         if filter.HasKey(ctx, key) {
            return LoadFunc(ctx, key)
         }
         return nil, errNoValue
      },
   }}
}
```

直接对Get方法进行更改，跟singleflight类似。

```go
//BloomFilterCacheV1 组合ReadThroughCache,持有一个BloomFilter，直接修改Get方法
type BloomFilterCacheV1 struct {
   ReadThroughCache
   bf BloomFilter
}

func (b *BloomFilterCacheV1) Get(ctx context.Context, key string) (any, error) {
   val, err := b.Cache.Get(ctx, key)
   if err != nil && b.bf.HasKey(ctx, key) {
      val, err = b.LoadFunc(ctx, key)
      if err == nil {
         er := b.Cache.Set(ctx, key, val, b.expiration)
         if er != nil {
            return val, fmt.Errorf("%w, 原因：%s", errFailToRefreshCache, er.Error())
         }
      }
   }
   return val, err
}
```

##### 缓存击穿解决方案

+ singleflight就足以解决问题，如果解决不了，那么大概率是因为DB需要扩容
+ 缓存未命中时，使用默认值
+ 在回查数据库时，加上限流器，不过这是保护系统，而不是解决问题

##### 缓存雪崩解决方案

设置key过期时间时，加一个随机偏移

###### 实现

RandomExpirationCache组合一个cache，重写Set方法



#### 服务器优雅退出-缓存实践

##### 概述

假设我们现在有一个 Web 服务。这个 Web 服务会监听两个端口：8080和8081。其中 8080 是用于监听正常的业务请求，它会被暴露在外部网络中；而 8081 用于监听我们开发者的内部管理请求，只在内部使用。

同时为了性能，我们在该服务中使用了本地缓存，并且采用了 write-back 的缓存模式。这个缓存模式要求，缓存在 key 过期的时候才将新值持久化到数据库中。这意味着在应用关闭的时候，我们必须将所有的 key 对应的数据都刷新到数据库中，否则会存在数据丢失的风险。

##### 要求

优雅退出，是指应用主动监听关闭信号，在收到关闭信号如 ctrl + C 的时候，能够主动停下来。优雅退出和一般的直接退出比起来，优雅退出需要确保正在处理的请求能够正常结束，同时能够释放掉应用所使用的资源。

为了给用户更好的体验，设计一个优雅退出的步骤，它需要完成：

​    ● 监听系统信号，当收到 ctrl + C 的时候，应用要立刻拒绝新的请求

​    ● 应用需要等待已经接收的请求被正常处理完成

​    ● 应用关闭 8080 和 8081 两个服务器

​    ● 我们能够注册一个退出的回调，在该回调内将缓存中的数据回写到数据库中

##### 基本结构

```go
//App 代表应用本身
type App struct {
   servers []*Server// 代表一个 HTTP 服务器，一个实例监听一个端口
}
// Server 本身可以是很多种 Server，例如 http server
// 或者 RPC server
// 理论上来说，如果你设计一个脚手架的框架，那么 Server 应该是一个接口
type Server struct {

}
```

注意，实际中拒绝新请求和等待已有请求完结，http 包已经帮我们处理好了，可以直接调用 shutdown。

##### 需求分析

###### 场景分析

**退出场景**

| **场景**                                                     | **详情**                                                     |
| ------------------------------------------------------------ | ------------------------------------------------------------ |
| 开发人员准备关机                                             | 应用监听关机信号，执行优雅退出步骤                           |
| 开发人员关机，在等待一段时间发现还未停下，需要强制关闭       | 应用需要提供强制退出的机制。在最开始关闭的时候，应用执行优雅退出的步骤。如果在这个时候再次收到关闭指令，那么应该能够立刻退出进程 |
| 开发人员关机，要求应用在十分钟后都还没优雅退出，就自动强制退出 | 应用需要提供优雅退出的超时机制。在收到关闭信号后，应用执行优雅退出的步骤。如果一段时间之后（如十分钟）优雅退出的步骤还没有全部执行完毕，那么应用能够自动强制退出。 |

**优雅退出场景**

| **场景**                 | **详情**                                                     |
| ------------------------ | ------------------------------------------------------------ |
| 应用正在处理请求         | 当应用收到关闭信号的时候，此时有些 server 正在处理请求，还没有完全执行完毕。例如这时候可能正在执行数据库事务，还未提交。此时应用需要等待这些正在处理的请求。 |
| 应用收到新的请求         | 当应用收到关闭信号的时候，又收到了新的请求。此时应用应该拒绝这些新请求。 |
| 应用需要释放资源         | 应用在运行期间持有了一些资源，在退出的时候要主动释放掉。例如关闭连接池，关闭数据库连接以及 Prepare Statement 等 |
| 开发者需要注册自定义回调 | 在应用退出的时候，开发者希望注册一些自定义的回调。例如开发者使用了 write-back 的缓存模式，那么在应用退出之前，开发者希望将缓存刷新到数据库中 |

###### 功能性需求

| **功能点**     | **描述**                                                     |
| -------------- | ------------------------------------------------------------ |
| 监听退出信号   | 应用能够监听系统信号，在收到信号的执行优雅退出。需要考虑的操作系统有：                ● Windows                ● Mac OS                ● Linux |
| 强制退出       | 应用监听系统信号，在第一次收到信号的时候执行优雅退出。如果退出期间再次收到信号，则直接退出，并且输出强制退出的日志 |
| 超时强制退出   | 当应用在优雅退出的过程中，如果超出了时间限制，那么直接强制退出。开发者可以配置该超时时间 |
| 拒绝新请求     | 应用在优雅退出的时候，拒绝掉新请求                           |
| 等待已接收请求 | 应用需要等待一段时间，等待已经接收、正在处理的请求执行完毕。等待时间应该可配置的。如果在等待期间，就触发了强制退出，那么应用会放弃等待，直接退出。 |
| 释放资源       | 应用释放自身持有的资源                                       |
| 注册回调接口   | 开发者可以注册自定义回调，目前只支持在应用等待当前请求执行完毕，释放应用持有资源之前，执行开发者注册的回调。回调应该是独立的，开发者需要自己确保回调之间没有任何依赖。 |

###### 非功能性

​    ● 系统信号兼容 Windows，Mac OS 和 Linux 

##### shutdown设计

![image-20230428141652999](C:\Users\Administrator\Desktop\translate\img\image-20230428141652999.png)

在这个流程里面，我们将执行回调的时机放在关闭 server 之后，应用释放资源之前，是因为考虑到开发者执行回调的时候，可能依旧需要应用的资源；与此同时，我们又不希望开发者继续利用 server，所以放在这两个步骤之间。

同时我们在回调设计上，也没有引入优先级，或者排序的特性。**我们会并发调用回调，开发者自己需要确保这些回调之间没有依赖。**

###### 拒绝新请求

理论上来说，如果要手动实现拒绝新请求，那么我们可以考虑在  http.Server 里面封装 ServeMux，在每一次处理请求之前，先检查一下当下需不需要拒绝新请求。

```go
// serverMux 既可以看做是装饰器模式，也可以看做委托模式
// serverMux 加上reject来拒绝新请求
type serverMux struct {
   reject bool
   *http.ServeMux
}
//封装http请求中原本的serveHTTP,在之前加上判断reject
func (s *serverMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
   // 只是在考虑到 CPU 高速缓存的时候，会存在短时间的不一致性
   if s.reject {
      w.WriteHeader(http.StatusServiceUnavailable)
      _, _ = w.Write([]byte("服务已关闭"))
      return
   }
   s.ServeMux.ServeHTTP(w, r)
}
```

而后，当开始优雅退出的时候，将 reject 标记位设置为  true。

###### 等待已有请求执行完毕

两个思路：

+ 等待一段固定时间
+ 实时维护当前正在执行的请求数量

这里先采用简单的方案一，开发者可以配置等待时间。这就需要APP中自己持有等待时间waitTime。

在serve上定义已有服务的关闭方法stop。

```go
func (s *Server) stop(ctx context.Context) error {
   //这就是HTTP包中的服务优雅退出，不过我们实现在app下的shutdown中
   return s.srv.Shutdown(ctx)
}
```

###### 自定义选项和注册回调

在一些步骤中，开发者希望自定义一些参数，比如说等待时间。此外，我们还需要允许开发者注册自己的退出回调，这些需求可以通过  Option 设计模式来解决，例如注册回调：

```go
// 典型的 Option 设计模式
type Option func(*App)
// ShutdownCallback 采用 context.Context 来控制超时，而不是用 time.After 是因为
// - 超时本质上是使用这个回调的人控制的
// - 我们还希望用户知道，他的回调必须要在一定时间内处理完毕，而且他必须显式处理超时错误
type ShutdownCallback func(ctx context.Context)

// 需要实现这个方法
func WithShutdownCallbacks(cbs ...ShutdownCallback) Option {
	panic("implement me")
}
```

回调本身是需要考虑超时问题，既我们不希望开发者的回调长时间运行。同时我们也希望开发者明确意识到超时这个问题，所以我们采用了 context 的方案，开发者需要自己处理超时问题。APP维护cbTimeout来控制回调超时。

注册的自定义回调将会被并发调用。

###### 监听系统信号

在 Go 里面监听系统信号是一件比较简单的事情。主要是依赖于：

```go
c := make(chan os.Signal, 1)
signal.Notify(c, signals)//signals是信号的类型，比如os.Interrupt
select {
  case <-c:
  // 监听到了关闭信号
}
```

在不同系统下，需要监听的信号 signals 是不一样的。因此要可以利用 Go 编译器的特性，为不同的平台定义 signals 的值

​                ● Mac OS 定义在以  _darwin.go 为结尾的文件中

​                ● Windows 定义在 _windows.go 为结尾的文件中

​                ● Linux 定义在以  _linux.go 为结尾的文件中

###### 强制退出

采用两次监听的策略。在第一次监听到退出信号的信号的时候，启动优雅退出。之后需要做两件事：

​                ● 再次监听系统退出信号，再次收到信号之后就强制退出

​                ● 启动超时计时器，超时则强制退出。超时的时间由APP维护timeout

```go
	// 从这里开始优雅退出监听系统信号，强制退出以及超时强制退出。
	// 优雅退出的具体步骤在 shutdown 里面实现
	// 所以你需要在这里恰当的位置，调用 shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
	go func() {
		select {
		case <-sig:
			log.Println("再次接收信号，强制退出")
			os.Exit(1)
		case <-time.After(app.shutdownTimeout):
			log.Println("超时，强制退出")
			os.Exit(1)
		}
	}()
	app.shutdown()
```

##### 例子

```go
func main() {
   s1 := service.NewServer("business", "localhost:8080")
   s1.Handle("/", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
      _, _ = writer.Write([]byte("hello"))
   }))
   s2 := service.NewServer("admin", "localhost:8081")
   app := service.NewApp([]*service.Server{s1, s2}, service.WithShutdownCallbacks(StoreCacheToDBCallback))
   app.StartAndServe()
}
func StoreCacheToDBCallback(ctx context.Context) {
    // 需要处理 ctx 超时
}

```

##### 实现

###### APP

APP要负责启动服务并且监听退出信号(StartAndServe),优雅退出(shutdown),释放资源(close)

APP的最终定义为：

```go
// 这里我已经预先定义好了各种可配置字段
type App struct {
	servers []*Server

	// 优雅退出整个超时时间，默认30秒
	shutdownTimeout time.Duration

	// 优雅退出时候等待处理已有请求时间，默认10秒钟
	waitTime time.Duration
	// 自定义回调超时时间，默认三秒钟
	cbTimeout time.Duration

	cbs []ShutdownCallback
}
```

###### APP方法

新建一个APP，利用option来创建。没有option就使用默认值。

```go
// NewApp 创建 App 实例，注意设置默认值，同时使用这些选项
func NewApp(servers []*Server, opts ...Option) *App {
   app := &App{
      servers:         servers,
      shutdownTimeout: time.Second * 30,
      waitTime:        time.Second * 10,
      cbTimeout:       time.Second * 3,
      cbs:             nil,
   }
   for _, opt := range opts {
      opt(app)
   }
   return app
}
```

启动服务

```go
// StartAndServe 你主要要实现这个方法
func (app *App) StartAndServe() {
   for _, s := range app.servers {
      srv := s
      go func() {
         if err := srv.Start(); err != nil {
            if err == http.ErrServerClosed {
               log.Printf("服务器%s已关闭", srv.name)
            } else {
               log.Printf("服务器%s异常退出", srv.name)
            }
         }
      }()
   }
   // 从这里开始优雅退出监听系统信号，强制退出以及超时强制退出。
   // 优雅退出的具体步骤在 shutdown 里面实现
   // 所以你需要在这里恰当的位置，调用 shutdown
   sig := make(chan os.Signal, 1)
   signal.Notify(sig, os.Interrupt)
   <-sig
   go func() {
      select {
      case <-sig:
         log.Println("再次接收信号，强制退出")
         os.Exit(1)
      case <-time.After(app.shutdownTimeout):
         log.Println("超时，强制退出")
         os.Exit(1)
      }
   }()
   app.shutdown()
}
```

优雅退出

```go
// shutdown 你要设计这里面的执行步骤。
//http.Server#Shutdown的逻辑是：先关闭所有的 listeners， 然后等待所有的连接（connection）都处理完，再关闭服务。如果传入的 context 超时，则会直接返回。
//而这个shutdown就实现设计中的五个步骤
func (app *App) shutdown() {
   log.Println("开始关闭应用，停止接收新请求")
   // 你需要在这里让所有的 server 拒绝新请求
   //在serverMux中设置了reject标记位，所以直接设置每个server的标记位即可
   for _, ser := range app.servers {
      ser.rejectReq()
   }

   log.Println("等待正在执行请求完结")
   // 在这里等待一段时间
   //设置ctx来控制关闭服务超时时间
   serverCtx, cancel1 := context.WithTimeout(context.Background(), app.waitTime)
   defer cancel1()
   //使用 waitGroup来等待所有的goroutine关闭服务，或者超时才能继续向下执行
   var wg sync.WaitGroup
   for _, ser := range app.servers {
      wg.Add(1)
      go func(ser *Server, ctx context.Context) {
         //select {
         //case <-ctx.Done():
         // log.Println("关闭服务超时")
         // wg.Done()
         // return
         //default:
         // if err := ser.stop(ctx); err != nil {
         //    log.Println("关闭服务失败")
         // }
         // wg.Done()
         //}
         //使用了http包中的shutdown就直接使用ERROR,就不需要上面这个select
         if err := ser.stop(ctx); err != nil {
            log.Println("关闭服务失败,error:", err)
         }
         wg.Done()
      }(ser, serverCtx)
   }
   wg.Wait()

   log.Println("开始执行自定义回调")
   // 并发执行回调，要注意协调所有的回调都执行完才会步入下一个阶段
   //设置ctx来控制回调超时时间
   cbsCtx, cancel2 := context.WithTimeout(context.Background(), app.cbTimeout)
   defer cancel2()
   wg.Add(len(app.cbs))
   for _, cb := range app.cbs {
      callback := cb
      go func(ctx context.Context) {
         select {
         case <-ctx.Done():
            log.Println("执行回调超时")
            wg.Done()
            return
         default:
            callback(ctx)
            wg.Done()
         }
      }(cbsCtx)
   }
   wg.Wait()

   // 释放资源
   log.Println("开始释放资源")
   // 这一个步骤不需要你干什么，这是假装我们整个应用自己要释放一些资源
   app.close()
}
```

释放资源

```go
func (app *App) close() {
   // 在这里释放掉一些可能的资源
   time.Sleep(time.Second)
   log.Println("应用关闭")
}
```

###### 回调

```go
// 典型的 Option 设计模式
type Option func(*App)
// ShutdownCallback 采用 context.Context 来控制超时，而不是用 time.After 是因为
// - 超时本质上是使用这个回调的人控制的
// - 我们还希望用户知道，他的回调必须要在一定时间内处理完毕，而且他必须显式处理超时错误
type ShutdownCallback func(ctx context.Context)


func WithShutdownCallbacks(cbs ...ShutdownCallback) Option {
	return func(app *App) {
		app.cbs = cbs
	}
}
```

###### Server

serverMux装饰了一个reject

```go
// serverMux 既可以看做是装饰器模式，也可以看做委托模式
type serverMux struct {
   reject bool
   *http.ServeMux
}
//ServeHTTP在接收新请求时拒绝
func (s *serverMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 只是在考虑到 CPU 高速缓存的时候，会存在短时间的不一致性
	if s.reject {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("服务已关闭"))
		return
	}
	s.ServeMux.ServeHTTP(w, r)
}
```

serve保存server的名字和http.Server还保存serverMux

```go
// Server 本身可以是很多种 Server，例如 http server
// 或者 RPC server
// 理论上来说，如果你设计一个脚手架的框架，那么 Server 应该是一个接口
type Server struct {
   srv  *http.Server
   name string
   mux  *serverMux
}
```

###### Server 方法

要实现Handle,Start方法，让server能注册和启动

```go
func (s *Server) Handle(pattern string, handler http.Handler) {
   s.mux.Handle(pattern, handler)
}

func (s *Server) Start() error {
   return s.srv.ListenAndServe()
}
```

reject方法让server拒绝请求

```go
func (s *Server) rejectReq() {
   s.mux.reject = true
}
```

stop方法让server能够停止

```go
func (s *Server) stop(ctx context.Context) error {
   //这就是HTTP包中的服务优雅退出，不过我们实现在app下的shutdown中
   return s.srv.Shutdown(ctx)
}
```



#### redis实现分布式锁

##### 分布式锁

在分布式环境下不同实例之间抢一把锁。就是抢锁从进程变成了实例。难是因为基本都与网络有关。使用redis实现，加锁就是向redis中写值，解锁就是删除redis中的值。

##### setnx

起点就是setnx命令(要设置key，如果key已经存在那么返回失败，同时只有一个goroutine可以设置key)，确保可以排他地设置一个键值对。

##### 实现

###### Client

因为要与Redis通信，所以我们最好有一个抽象，封装redis.Cmdable,也就是对redis.Cmdable的二次封装，这个Client主要是用来加锁。

```go
//Client 用于加锁
type Client struct {
   client redis.Cmdable
}
```

Client有TryLock(ctx,key,expiration)(*Lock,error),setNX如果不ok代表的是别人抢到了锁，成功的话肯定会返回一个Lock之类的东西，所以也定义一个Lock。

```go
//TryLock 传入上下文，key和过期时间，返回一个Lock，即锁
func(c *Client)TryLock(ctx context.Context,key string,expiration time.Duration)(*Lock,error){
   val:=uuid.New().String()
   ok,err:=c.client.SetNX(ctx,key,val,expiration).Result()
   if err!=nil{
      return nil, err
   }
   if !ok{
      return nil, errFailToPreemptLock
   }
   return &Lock{
      client: c.client,
      key: key,
      value: val,
   }, nil
}
```

Unlock定义在Client不够优雅，应该定义在Lock上这样TryLock返回的Lock就可以直接调用Unlock，如果定义在client上在unlock参数中也不能传key，只能传Lock。

```go
//Lock 代表锁
type Lock struct {
   client redis.Cmdable
   key string
   value string//用来校对是否是自己的锁
}
```

Unlock放在Lock上那么Lock就需要持有redis.Cmdable和key来把键值对删掉。如果删除时，cnt!=1代表锁过期了或者有其他错误。

```go
//Unlock Lock删除key以释放锁
func(l *Lock)Unlock(ctx context.Context)error{
   //检查锁是否是自己的
   val,err:=l.client.Get(ctx,l.key).Result()
   if err!=nil{
      return err
   }
   if val!=l.value{
      return errors.New("不是自己的锁")
   }
   //上面check，下面do something,在中间这里的空缺，键值对就可能被删除了
   cnt,err:=l.client.Del(ctx,l.key).Result()
   if err!=nil{
      return err
   }
   if cnt!=1{
      return errLockNotExist
   }
   return nil
}
```

###### 为什么要有过期时间

如果没有过期时间，那么原本加锁的实例崩溃后，永远没有人去释放锁。如果业务执行超过了锁的过期时间又怎么办？那么就要判断是否是自己的锁，就要用uuid作为值

###### 为什么用uuid作为值

本质上我们需要一个唯一的值，用来比较这是某个实例加的锁，uuid是一个不错的选择，只要确保同一把锁的值不会冲突就可以。如果不确认锁那么可能会出现，实例1抢到锁执行业务，实例2等待锁，实例1锁过期，实例2抢到锁，实例1完成业务释放锁，实例2的锁被释放。那么Lock中就要继续加入value来存uuid。

那么Unlock的实现就是先判断锁是不是自己的，就是从redis中get key利用uuid来判断，然后把键值对删除掉。但是在判断之后，删除掉之前，此键值对可能就被别的键值对给删除了，紧接着另外一个实例加锁。也就是说中间这里有一小段时间空缺，所以需要进行原子操作，让get到del中不能有人插入，所以就需要lua脚本。用lua脚本来替代上述操作。使用redis的eval，其中的lua语句参数可以在外面写完lua脚本后，使用go embed来嵌入进一个变量。

```go
//go:embed lua/unlock.lua
luaUnlock string
func(l *Lock)Unlock(ctx context.Context)error{
	res,err:=l.client.Eval(ctx,luaUnlock,[]string{l.key},l.value).Int64()//结果直接转化成int64，因为del返回的就是删除的个数
	if err!=nil{
		return err
	}
	if res!=1{
		return errLockNotExist
	}
	return nil
}
```

###### lua脚本

Lua脚本运行期间，为了避免被其他操作污染数据，这期间将不能执行其它命令，一直等到执行完毕才可以继续执行其它请求。当Lua脚本执行时间超过了lua-time-limit时，其他请求将会收到Busy错误，除非这些请求是SCRIPT KILL（杀掉脚本）或者SHUTDOWN NOSAVE（不保存结果直接关闭Redis）。

```lua
--1. 检查是不是你的锁
--2. 删除
-- KEYS[1] 就是你的分布式锁的key
-- ARGV[1] 就是你预期的存在redis 里面的 value
if redis.call('get',KEYS[1])==ARGV[1] then
    --确实是你的锁
    return redis.call('del',KEYS[1])
else
    return 0
end
```

###### TryLock和Unlock的单元测试

```go
func TestClient_TryLock(t *testing.T) {
   testCases:=[]struct{
      name string
      mock func(ctrl *gomock.Controller)redis.Cmdable
      key string
      expiration time.Duration
      wantErr error
      wantVal *Lock
   }{
      {
         name:       "set nx error",
         mock: func(ctrl *gomock.Controller) redis.Cmdable {
            cmd:=mocks.NewMockCmdable(ctrl)
            //setNx返回的就是Bool
            res:=redis.NewBoolResult(false,context.DeadlineExceeded)
            cmd.EXPECT().SetNX(context.Background(),"key1",gomock.Any(),time.Second).Return(res)
            return cmd
         },
         key:        "key1",
         expiration: time.Second,
         wantErr:    context.DeadlineExceeded,
      },
      {
         name:       "fail to preempt lock",
         mock: func(ctrl *gomock.Controller) redis.Cmdable {
            cmd:=mocks.NewMockCmdable(ctrl)
            res:=redis.NewBoolResult(false,errFailToPreemptLock)
            cmd.EXPECT().SetNX(context.Background(),"key1",gomock.Any(),time.Second).Return(res)
            return cmd
         },
         key:        "key1",
         expiration: time.Second,
         wantErr:    errFailToPreemptLock,
      },
      {
         name:       "lock",
         mock: func(ctrl *gomock.Controller) redis.Cmdable {
            cmd:=mocks.NewMockCmdable(ctrl)
            res:=redis.NewBoolResult(true,nil)
            cmd.EXPECT().SetNX(context.Background(),"key1",gomock.Any(),time.Second).Return(res)
            return cmd
         },
         key:        "key1",
         expiration: time.Second,
         wantVal: &Lock{
            key:    "key1",
            expiration: time.Second,
         },
      },
   }

   for _,tc:=range testCases{
      t.Run(tc.name, func(t *testing.T) {
         ctrl:=gomock.NewController(t)
         defer ctrl.Finish()
         lock,err:=NewClient(tc.mock(ctrl)).TryLock(context.Background(),tc.key,tc.expiration)
         assert.Equal(t, tc.wantErr,err)
         if err!=nil{
            return
         }
         assert.Equal(t,tc.wantVal.key,lock.key)
         //无法得到准确的value只能通过判断是否有值做粗略的检查
         assert.NotEmpty(t, lock.value)
      })
   }
}

func TestLock_Unlock(t *testing.T) {
   //在测试用例中使用Lock，那么里面的client就需要ctrl，只能定义一个总的ctrl来复用
   //可以使用下面的定义，这样就可以在每个测试用例中单独创建ctrl，再把lock创建起来即可
   //testCases := []struct{
   // name string
   // mock func(ctrl *gomock.Controller) redis.Cmdable
   // key string
   // value string
   // wantErr error
   //}
   ctrl:=gomock.NewController(t)
   testCases:=[]struct{
      name string
      lock *Lock
      wantErr error
   }{
      {
         name:    "unlock err",
         lock:    &Lock{
            client:     func(ctrl *gomock.Controller)redis.Cmdable{
               cmd:=mocks.NewMockCmdable(ctrl)
               res:=redis.NewCmd(context.Background())
               res.SetErr(context.DeadlineExceeded)
               cmd.EXPECT().Eval(context.Background(),luaUnlock,[]string{"key"},"value").Return(res)
               return cmd
            }(ctrl),
            key:        "key",
            value:      "value",
            expiration: time.Second,
         },
         wantErr: context.DeadlineExceeded,
      },
      {
         name:    "lock not hold",
         lock:    &Lock{
            client:     func(ctrl *gomock.Controller)redis.Cmdable{
               cmd:=mocks.NewMockCmdable(ctrl)
               res:=redis.NewCmd(context.Background())
               res.SetVal(int64(0))
               cmd.EXPECT().Eval(context.Background(),luaUnlock,[]string{"key"},"value").Return(res)
               return cmd
            }(ctrl),
            key:        "key",
            value:      "value",
            expiration: time.Second,
         },
         wantErr: errLockNotHold,
      },
      {
         name:    "unlock",
         lock:    &Lock{
            client:     func(ctrl *gomock.Controller)redis.Cmdable{
               cmd:=mocks.NewMockCmdable(ctrl)
               res:=redis.NewCmd(context.Background())
               res.SetVal(int64(1))
               cmd.EXPECT().Eval(context.Background(),luaUnlock,[]string{"key"},"value").Return(res)
               return cmd
            }(ctrl),
            key:        "key",
            value:      "value",
            expiration: time.Second,
         },
      },
   }
   for _,tc:=range testCases{
      t.Run(tc.name, func(t *testing.T) {
         err:=tc.lock.Unlock(context.Background())
         assert.Equal(t, tc.wantErr,err)
         if err!=nil{
            return
         }
      })
   }
}
```

###### 集成测试

集成测试需要连上实际redis，先创建一个redis客户端。

trylock的err是比较难测的，那么setNx不成功，和成功后返回的Lock才是要测的。

别人抢到了锁，说明已经有一个key了，在测试用例中最好有一个before的func(t *testing.T)来模拟别人有锁，手动的塞一个key进去。

那我怎么知道redis的值设对了，设置一个after使用getdel来得到值来判断。

unlock需要测没有锁，别人持有所，自己有锁的情况。

因为在每个before，after数据都不一样，所以不适合testsuite的beforetest和aftertest

```go
func TestClient_TryLock2(t *testing.T) {
   //before和after都要使用，所以放到外面
   rdb:=NewClient(redis.NewClient(&redis.Options{Addr: "localhost:6379"}))
   testCases:=[]struct{
      name string
      before func(t *testing.T)
      after func(t *testing.T)
      key string
      expiration time.Duration
      wantErr  error
      wantLock *Lock
   }{
      {
         name:       "locked",
         before: func(t *testing.T) {
            _,err:=rdb.client.SetNX(context.Background(),"key1","value1",time.Second*10).Result()
            if err!=nil{
               return
            }
         },
         after: func(t *testing.T) {
            res,err:=rdb.client.GetDel(context.Background(),"key1").Result()
            require.NoError(t, err)
            require.Equal(t, "value1",res)
         },
         key:        "key1",
         expiration: time.Second*10,
         wantErr:    errFailToPreemptLock,
      },
      {
         name:       "set lock",
         before: func(t *testing.T) {},
         after: func(t *testing.T) {
            _,err:=rdb.client.Del(context.Background(),"key2").Result()
            require.NoError(t, err)
         },
         key:        "key2",
         expiration: time.Second*10,
         wantLock:    &Lock{
            client:     rdb.client,
            key:        "key2",
            expiration: time.Second*10,
         },
      },
   }
   for _,tc:=range testCases{
      t.Run(tc.name, func(t *testing.T) {
         tc.before(t)
         ctx,cancel:=context.WithTimeout(context.Background(),time.Second*10)
         defer cancel()
         lock,err:=rdb.TryLock(ctx,tc.key,tc.expiration)
         assert.Equal(t, tc.wantErr,err)
         if err!=nil{
            return
         }
         assert.Equal(t, tc.wantLock.key,lock.key)
         assert.NotEmpty(t, lock.value)
         tc.after(t)
      })
   }
}

func TestLock_Unlock2(t *testing.T) {
   //before和after都要使用，所以放到外面
   rdb:=NewClient(redis.NewClient(&redis.Options{Addr: "localhost:6379"}))
   testCases:=[]struct{
      name string
      lock *Lock
      before func(t *testing.T)
      after func(t *testing.T)
      wantErr error

   }{
      {
         name:    "no locked",
         lock:    &Lock{
            client:     rdb.client,
            key:        "unlock_key1",
         },
         before:  func(t *testing.T){},
         after:   func(t *testing.T){},
         wantErr: errLockNotHold,
      },
      {
         name:    "other has locked",
         lock:    &Lock{
            client:     rdb.client,
            key:        "unlock_key2",
            value: "unlock_value",
         },
         before:  func(t *testing.T){
            _,err:=rdb.client.SetNX(context.Background(),"unlock_key2","unlock_value2",time.Second*10).Result()
            require.NoError(t, err)
            if err!=nil{
               return
            }
         },
         after:   func(t *testing.T){
            res,err:=rdb.client.GetDel(context.Background(),"unlock_key2").Result()
            require.NoError(t, err)
            require.Equal(t, "unlock_value2",res)
         },
         wantErr: errLockNotHold,
      },
      {
         name:    "unlocked",
         lock:    &Lock{
            client:     rdb.client,
            key:        "unlock_key3",
            value: "unlock_value3",
         },
         before:  func(t *testing.T){
            _,err:=rdb.client.SetNX(context.Background(),"unlock_key3","unlock_value3",time.Second*10).Result()
            require.NoError(t, err)
            if err!=nil{
               return
            }
         },
         after:   func(t *testing.T){
         },
      },
   }
   for _,tc:=range testCases{
      t.Run(tc.name, func(t *testing.T) {
         tc.before(t)
         ctx,cancel:=context.WithTimeout(context.Background(),time.Second*10)
         defer cancel()
         err:=tc.lock.Unlock(ctx)
         assert.Equal(t, tc.wantErr,err)
         if err!=nil{
            return
         }
         tc.after(t)
      })
   }
```

###### 踩坑

集成测试中的before,after即使不做动作也不能设置为nil，不然会出现空指针panic。

##### 续约与自动续约

###### 过期时间

对锁的用户来说，很难确定锁的过期时间应该设置多长：

+ 短了，业务还没完成，锁就过期
+ 长了，万一实例崩溃了，那么其它实例也长时间拿不到锁

更严重的，不管设置多长，极端情况下，都会出现业务执行时间超过过期时间。

###### 续约

在锁还没过期时，再次延长锁的过期时间。

+ 过期时间不必设置过长，因为有自动续约
+ 如果实例崩溃，则没有人再续约，过一段时间后就会过期，其它实例就能拿到锁

###### 实现

依旧要使用lua脚本，因为在refresh时也要判断一下是不是自己的锁，防止续约错了锁。redis的命令就是expire

expire key time [NX|XX...]

```lua
--1. 检查是不是你的锁
--2. 刷新
-- KEYS[1] 就是你的分布式锁的key
-- ARGV[1] 就是你预期的存在redis 里面的 value
if redis.call('get',KEYS[1])==ARGV[1] then
    --确实是你的锁
    return redis.call('expire',KEYS[1],ARGV[2])
else
    return 0
end
```

基本就是unlock，然后把lua变量和脚本更换,续约的时间就可以使用我们创建的时候使用的过期时间，所以Lock中就可以加上expiration来使用。

```go
//Lock 代表锁
type Lock struct {
   client     redis.Cmdable
   key        string
   value      string
   expiration time.Duration
}
	//go:embed lua/refresh.lua
	luaRefresh string
func (l *Lock) Refresh(ctx context.Context) error {
	res, err := l.client.Eval(ctx, luaRefresh, []string{l.key}, l.value,l.expiration.Seconds()).Int64()
	if err != nil {
		return err
	}
	if res != 1 {
		return errLockNotHold
	}
	return nil
}
```

###### 单元测试

跟unlock的单元测试是很像的，所以把unlock的单元测试改一改就可以了。测试的refresh新增的expiration要是float64(秒)。

```go
func TestLock_Refresh(t *testing.T) {
   ctrl := gomock.NewController(t)
   testCases := []struct {
      name    string
      lock    *Lock
      expiration time.Duration
      wantErr error
   }{
      {
         name: "refresh err",
         lock: &Lock{
            client: func(ctrl *gomock.Controller) redis.Cmdable {
               cmd := mocks.NewMockCmdable(ctrl)
               res := redis.NewCmd(context.Background())
               res.SetErr(context.DeadlineExceeded)
               cmd.EXPECT().Eval(context.Background(), luaRefresh, []string{"key"}, "value",float64(1)).Return(res)
               return cmd
            }(ctrl),
            key:        "key",
            value:      "value",
            expiration: time.Second,
         },
         wantErr: context.DeadlineExceeded,
         expiration: time.Second,
      },
      {
         name: "lock not hold",
         lock: &Lock{
            client: func(ctrl *gomock.Controller) redis.Cmdable {
               cmd := mocks.NewMockCmdable(ctrl)
               res := redis.NewCmd(context.Background())
               res.SetVal(int64(0))
               cmd.EXPECT().Eval(context.Background(), luaRefresh, []string{"key"}, "value",float64(1)).Return(res)
               return cmd
            }(ctrl),
            key:        "key",
            value:      "value",
            expiration: time.Second,
         },
         wantErr: errLockNotHold,
         expiration: time.Second,
      },
      {
         name: "Refresh",
         lock: &Lock{
            client: func(ctrl *gomock.Controller) redis.Cmdable {
               cmd := mocks.NewMockCmdable(ctrl)
               res := redis.NewCmd(context.Background())
               res.SetVal(int64(1))
               cmd.EXPECT().Eval(context.Background(), luaRefresh, []string{"key"}, "value",float64(1)).Return(res)
               return cmd
            }(ctrl),
            key:        "key",
            value:      "value",
            expiration: time.Second,
         },
         expiration: time.Second,
      },
   }
   for _, tc := range testCases {
      t.Run(tc.name, func(t *testing.T) {
         err := tc.lock.Refresh(context.Background())
         assert.Equal(t, tc.wantErr, err)
         if err != nil {
            return
         }
      })
   }
}
```

但是如何检测刷新成功？

在单元测试层面上是检测不出来的，只能在集成测试的实际环境检测。

###### 集成测试

在after中使用TTL,这会返回一个time.Duration,得到时间后使用require.True(t,timeout<=新设置的expiration)，这样测试就是把原本的expiration设置为1分钟，新set的设置为10s，即使考虑测试本身的运行时间，那么timeout肯定还是大于10s的。这样就可以验证刷新失败。

```go
func TestLock_Refresh2(t *testing.T) {
   //before和after都要使用，所以放到外面
   rdb := NewClient(redis.NewClient(&redis.Options{Addr: "localhost:6379"}))
   testCases := []struct {
      name    string
      lock    *Lock
      before  func(t *testing.T)
      after   func(t *testing.T)
      wantErr error
   }{
      {
         name: "no locked",
         lock: &Lock{
            client: rdb.client,
            key:    "unlock_key1",
            expiration: time.Second,
         },
         before:  func(t *testing.T) {},
         after:   func(t *testing.T) {},
         wantErr: errLockNotHold,
      },
      {
         name: "other has locked",
         lock: &Lock{
            client: rdb.client,
            key:    "unlock_key2",
            value:  "unlock_value",
            expiration: time.Second*10,
         },
         before: func(t *testing.T) {
            _, err := rdb.client.SetNX(context.Background(), "unlock_key2", "unlock_value2", time.Second*10).Result()
            require.NoError(t, err)
            if err != nil {
               return
            }
         },
         after: func(t *testing.T) {
            ctx,cancel:=context.WithTimeout(context.Background(),time.Second*3)
            defer cancel()
            timeout,err:=rdb.client.TTL(ctx,"unlock_key2").Result()
            require.NoError(t, err)
            require.True(t, timeout<time.Second*10)
            res, err := rdb.client.GetDel(context.Background(), "unlock_key2").Result()
            require.NoError(t, err)
            require.Equal(t, "unlock_value2", res)
         },
         wantErr: errLockNotHold,
      },
      {
         name: "Refresh",
         lock: &Lock{
            client: rdb.client,
            key:    "unlock_key3",
            value:  "unlock_value3",
            expiration: time.Minute,
         },
         before: func(t *testing.T) {
            _, err := rdb.client.SetNX(context.Background(), "unlock_key3", "unlock_value3", time.Minute).Result()
            require.NoError(t, err)
            if err != nil {
               return
            }
         },
         after: func(t *testing.T) {
            ctx,cancel:=context.WithTimeout(context.Background(),time.Second*3)
            defer cancel()
            timeout,err:=rdb.client.TTL(ctx,"unlock_key3").Result()
            require.NoError(t, err)
            // 如果要是刷新成功了，过期时间是一分钟，即便考虑测试本身的运行时间，timeout > 10s
            require.True(t, timeout>time.Second*50)
            _,err=rdb.client.Del(ctx,"unlock_key3").Result()
            require.NoError(t, err)
         },
      },
   }
   for _, tc := range testCases {
      t.Run(tc.name, func(t *testing.T) {
         tc.before(t)
         ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
         defer cancel()
         err := tc.lock.Refresh(ctx)
         assert.Equal(t, tc.wantErr, err)
         if err != nil {
            return
         }
         tc.after(t)
      })
   }
}
```

###### 手动续约：如何使用refresh方法

业务方使用起来很复杂。

使用Example来进行测试。基本上就是新开一个goroutine来使用ticker进行refresh，那么问题就是ctx怎么传，可以使用一个withTimeout设置一个一秒钟的ctx。

问题1：refresh error怎么办

error那么goroutine就需要向业务发信号，那只能设置一个error类型的errorChannel来进行通信，在业务执行的过程中要记得检测errChan有没有信号。如果是循环的业务代码，那么可以用select，但如果不是循环的，那只能在每个步骤都用select检测一下。

问题2：如何终止续约

只能是设置一个channel，使用select case来判断

问题3 ：有些错误是可以挽回的

比如说超时，那么就可以设置一个timeoutChan,在case <-ticker.C中就可以判断超时，超时就可以再次进行续约操作。

```go
func ExampleLock_Refresh() {
   //加锁成功
   var l *Lock
   //终止续约的channel
   stopChan:=make(chan struct{})
   //出现错误的channel
   errChan:=make(chan error)
   //可以挽回的error 比如超时的error chan,放入值后需要继续运行，所以设置缓冲为1
   timeoutChan:=make(chan struct{},1)
   //一个goroutine用来续约
   go func() {
      ticker:=time.NewTicker(time.Second*10)
      for{
         select {
         case <-ticker.C:
            ctx,cancel:=context.WithTimeout(context.Background(),time.Second)
            err:=l.Refresh(ctx)
            cancel()
            if err==context.DeadlineExceeded{
               timeoutChan<- struct{}{}
               continue
            }
            if err!=nil{
               errChan<-err
               //自己选择在哪close
               //close(stopChan)
               //close(errChan)
               return
            }
         case <-timeoutChan:
            ctx,cancel:=context.WithTimeout(context.Background(),time.Second)
            err:=l.Refresh(ctx)
            cancel()
            if err==context.DeadlineExceeded{
               timeoutChan<- struct{}{}
            }
            if err!=nil{
               return
            }
         case <-stopChan:
            return
         }
      }

   }()
   //执行业务
   //在业务执行过程中检测error
   //循环中的业务
   for i := 0; i < 100; i++ {
      select {
      //续约失败
      case <-errChan:
         break
      default:
         //正常业务逻辑
      }
   }
   //非循环的业务
   //只能每个步骤都要检测error
   select {
   case err:=<-errChan:
      log.Fatalln(err)
      return
   default:
      //业务步骤1
   }
   select {
   case err:=<-stopChan:
      log.Fatalln(err)
      return
   default:
      //业务步骤2
   }
   //执行完业务，终止续约
   stopChan<- struct{}{}
   // l.Unlock(context.Background())
}
```

###### 自动续约

因为用户手动续约很繁琐，处理分布式锁的各种异常是一个很棘手的事情，可以考虑提供自动续约的API。

问题基本和手动续约一样：

+ 隔多久续约，续多长？多久续约一次跟网络，redis服务器稳定性有关，每次续多长就直接使用原本的过期时间。
+ 如何处理超时，以及超时设置多长时间？由于超时一般是偶发性错误，可以立刻重新进行续约，超时时间让用户指定
+ 如何通知用户续约失败？只处理超时引起的失败，其他error通知用户
+ 要不要设置续约次数上限？不需要，用户自己决定

###### 自动续约实现

在lock上定义一个autorefresh(interval,timeout)error,就是使用手动续约的example的goroutine中的代码，还需要一些修改。

stopChan,由于是在goroutine外定义，goroutine内使用，现在不使用goroutine，那么可以在lock中加上unlockChan,并希望在unlock时把它close掉。

把ticker的时间改为interval，超时设置改为timeout

因为直接返回了error，所以不需要errorChan

```go
// AutoRefresh 自动续约 传入超时时间
func(l *Lock)AutoRefresh(interval time.Duration,timeout time.Duration)error{
   //可以挽回的error 比如超时的error chan,放入值后需要继续运行，所以设置缓冲为1
   timeoutChan:=make(chan struct{},1)
   ticker := time.NewTicker(interval)
   for {
      select {
      case <-ticker.C:
         ctx, cancel := context.WithTimeout(context.Background(), timeout)
         err := l.Refresh(ctx)
         cancel()
         if err == context.DeadlineExceeded {
            timeoutChan <- struct{}{}
            continue
         }
         if err != nil {
            return err
         }
      case <-timeoutChan:
         ctx, cancel := context.WithTimeout(context.Background(), timeout)
         err := l.Refresh(ctx)
         cancel()
         if err == context.DeadlineExceeded {
            timeoutChan <- struct{}{}
         }
         if err != nil {
            return err
         }
      case <-l.unlockCh:
         return nil
      }
   }
}
func (l *Lock) Unlock(ctx context.Context) error {
	res, err := l.client.Eval(ctx, luaUnlock, []string{l.key}, l.value).Int64()
	defer func() {
		select {
		case l.unlockCh<- struct{}{}:
		default:
			//说明没有人调用AutoRefresh
		}
	}()
	if err != nil {
		return err
	}
	if res != 1 {
		return errLockNotHold
	}
	return nil
}
```

自动续约可控性很差，所以即使提供了API也不提倡使用，如果用户想要 万无一失的使用这个分布式锁，那还是要手动调用refresh然后处理各种error。

此外，续约的间隔应该综合考虑服务可用性，如果将分布式锁的过期时间设置为10s，而且预期2s内大概率续约成功，那么就可以将续约间隔设为8s。

##### 加锁重试

###### 重试逻辑

加锁可能遇到偶发性的失败，在这种情况下可以尝试重试。

+ 如果超时，直接加锁
+ 检查一下key对应的值是不是我们刚才超时加锁请求的的值，如果是直接返回，前一次加锁成功了(这里可能要考虑重置一下过期时间)
+ 如果不是直接返回，加锁失败。

![image-20230510073210139](../../Users/123456/OneDrive/图片/go笔记/image-20230510073210139.png)

###### 策略

加锁重试类似自动续约，也要考虑：

+ 怎么重试？隔多久重试一次？总共重试几次？也就是重试策略的问题
+ 什么情况下应该重试，什么情况下不应该重试？超时就重试，其他error就不重试

根据自动续约的机制：

+ 超时了，这种情况下都不知道锁有没有拿到
+ 此时正有人持有锁，我们要等别人释放锁。

###### 实现

在client上定义Lock(ctx,key,expiration,timeout,retry)(*Lock,error)方法用来加锁，retry时自定义的RetryStrategy接口类型，内部有一个Next()(time.Duration,bool)，第一个返回值是重试的间隔，第二个是是否重试。

这是一种迭代器的形态，用户可以轻易通过扩展这个接口来实现自己的重试策略。缺点就是没有引入上下文的概念。

如果不同的key对应不同的策略那么retry就可以放在lock中，如果都是相同的策略那么retry可以集成在client中。

创建一个计时器 timer，在for循环中，使用trylock，先得到Next的返回值，之后判断是否重置，如果ok，那么就判断timer是初始化还是需要reset。后面就用一个select来控制超时。

```go
//Lock 与tryLock不同的是它是重试加锁,timeout时重试的超时时间
func (c *Client) Lock(ctx context.Context, key string, expiration,timeout time.Duration,retry RetryStrategy) (*Lock, error){
   val := uuid.New().String()
   //重试的计时器
   var timer *time.Timer
   for{
      tctx,cancel:=context.WithTimeout(ctx,timeout)
      res,err:=c.client.Eval(tctx,luaLock,[]string{key},val,expiration.Seconds()).Result()
      cancel()
      if err!=nil&&err!=context.DeadlineExceeded{
         return nil, err
      }
      //加锁ok了
      if res=="OK"{
         return &Lock{
            client:     c.client,
            key:        key,
            value:      val,
            expiration: expiration,
            unlockCh: make(chan struct{}, 1),
         },nil
      }
      //加锁失败，进行重试
      interval,ok:=retry.Next()
      //超出重试次数
      if !ok{
         return nil, fmt.Errorf("redis-lock: 超出重试限制, %w", errFailToPreemptLock)
      }
      //没有超出，则重置计时器
      if timer==nil{
         timer=time.NewTimer(interval)
      }else{
         timer.Reset(interval)
      }
      select {
      case <-timer.C:
         //什么都不用干，步入下一个循环
      case <-ctx.Done():
         return nil, ctx.Err()
      }
   }
}
```

直接使用trylock是不行的，因为我们有三种情况，就需要在lua脚本中照着三种情况写脚本。不需要使用SetNx因为已经判断不存在这个key了，而lua在redis执行中是单线程的。

```lua
local val = redis.call('get',KEYS[1])
if val==false then
    --可以进行加锁
    return redis.call('set',KEYS[1],ARGV[1])
else if val==ARGV[1] then
    redis.call('expire',KEYS[1],ARGV[2])
    --因为set的返回值也是ok
    return 'OK'
else
    --锁被别人拿着
    return ''
end
```

###### 踩坑

lua中使用redis的get得到不存在的值时返回的时false,涉及redis，go，lua三个值类型切换的问题。

由于set成功是返回ok,而expire是返回0，1，所以不return expire而是手动返回ok。

直接使用val = redis.call('get',KEYS[1])会报错试图更改readonly table script

###### 单元测试

由于重试时有withTimeOut，在mock中根本不可能与实际一模一样，所以会在这里报错，无法测试。

###### 集成测试

加锁重试重点还是在于跟redis的交互，所以重点还是测的lua脚本的情况。集成测试中ctx超时是比较难测的，在模拟环境中好测，因为正常环境下是比较稳定的，所以正常情况下是不测的。

实现一个retry用来测试.

测试情况：

1.成功拿到锁

2.别人持有锁，且重试次数超过限制

3.别人持有锁，但在重试时别人释放掉锁

```go
func TestClient_Lock2(t *testing.T) {
   rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
   testCases := []struct {
      name   string
      key    string
      before func(t *testing.T)
      after  func(t *testing.T)
      //key 过期时间
      expiration time.Duration
      //重试间隔
      timeout time.Duration
      //重试策略
      retry    RetryStrategy
      wantLock *Lock
      wantErr  error
   }{
      {
         name: "locked",
         before: func(t *testing.T) {
         },
         after: func(t *testing.T) {
            ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
            defer cancel()
            timeout, err := rdb.TTL(ctx, "lock_key1").Result()
            require.NoError(t, err)
            require.True(t, timeout >= time.Second*50)
            _, err = rdb.Del(ctx, "lock_key1").Result()
            require.NoError(t, err)
         },
         key:        "lock_key1",
         expiration: time.Minute,
         timeout:    time.Second * 2,
         retry: &FixedIntervalRetryStrategy{
            Interval: time.Second,
            MaxCnt:   10,
         },
         wantLock: &Lock{
            key:        "lock_key1",
            expiration: time.Minute,
         },
      },
      {
         name: "others hold lock",
         before: func(t *testing.T) {
            ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
            defer cancel()
            res, err := rdb.Set(ctx, "lock_key2", "123", time.Minute).Result()
            require.NoError(t, err)
            require.Equal(t, "OK", res)
         },
         after: func(t *testing.T) {
            ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
            defer cancel()
            res, err := rdb.GetDel(ctx, "lock_key2").Result()
            require.NoError(t, err)
            require.Equal(t, "123", res)
         },
         key:        "lock_key2",
         expiration: time.Minute,
         timeout:    time.Second * 3,
         retry: &FixedIntervalRetryStrategy{
            Interval: time.Second,
            MaxCnt:   3,
         },
         wantErr: fmt.Errorf("redis-lock: 超出重试限制, %w", errFailToPreemptLock),
      },
   }

   for _, tc := range testCases {
      t.Run(tc.name, func(t *testing.T) {
         tc.before(t)
         client := NewClient(rdb)
         lock, err := client.Lock(context.Background(), tc.key, tc.expiration, tc.timeout, tc.retry)
         assert.Equal(t, tc.wantErr, err)
         if err != nil {
            return
         }
         assert.Equal(t, tc.wantLock.key, lock.key)
         assert.Equal(t, tc.wantLock.expiration, lock.expiration)
         assert.NotEmpty(t, lock.value)
         assert.NotNil(t, lock.client)
         tc.after(t)
      })
   }
}
```

##### singleflight优化

在非常高并发并且热点集中的情况下，可以考虑结合singleflight来进行优化，也就是本地所有的goroutine自己先竞争，胜利者再去竞争全局的分布式锁。

###### 思路

内部会沿用lock的重试机制，在redislock中维持singleflighr.group,在一个for循环中使用DoChan,其中运行Lock。DoChan返回了，要判断是不是自己拿到了锁。设置一个flag，在DoChan中设为true，为true的那个就是拿到了锁。以防万一，加一个c.g.Forget(key)来确保其他goroutine能运行这段代码

###### 测试

要测试singleflight要开不同的进程才能测。

##### Redis主从切换

在主机加锁成功后，还没有复制到从机，主机宕机，从机提升为主机，那么另一个客户端也可以用这个key加锁成功，那么客户端1释放锁，释放了客户端2的锁。

##### 缓存一致性根源

+ 并发更新：分布式锁可以解决
+ 部分失败：不管先更新DB还是先更新缓存，又或者使用CDC方案，总是会有可能出现部分失败情况。分布式事务解决这个

###### write-back能不能解决缓存一致性

本地缓存不行，因为本地缓存不同的实例上都有一份所以不能解决。

如果是redis：

+ 如果缓存未命中不回查DB，那么站在调用者的调度，不会有缓存不一致的问题
+ 如果缓存未命中依旧回查DB，依旧有问题，大多数情况都会回查。在回查后写入缓存，如果用Set NX命令或版本号控制，也不会有一致性问题(除非在回查时，这个key对应的缓存来了又过期了)

write-back缺点就是如果key过期了还没有刷到DB里，那么就会数据丢失。

###### refresh-ahead能不能解决？

也不能解决，在Canal刷新缓存之前，都是不一致的。

###### 可能方案

结合哈希一致性负载均衡算法+singleflight,确保某个key对应的请求必然打到同一个机器上。