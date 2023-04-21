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

![image-20230328073532423](C:\Users\123456\AppData\Roaming\Typora\typora-user-images\image-20230328073532423.png)

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
+ Get方法检查过期时间时
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

在delete中加了锁，而调用delete之前也加了锁，导致程序卡死。

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