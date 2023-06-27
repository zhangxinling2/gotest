package orm

import (
	"context"
	"database/sql"
	"gotest/orm/internal/errs"
	"gotest/orm/model"
	"strings"
)

type Assignable interface {
	assigns()
}

type UpsertBuilder[T any] struct {
	i               *Inserter[T]
	conflictColumns []string
}

type Upsert struct {
	assigns         []Assignable
	conflictColumns []string
}

type Inserter[T any] struct {
	builder

	sess   Session
	values []*T

	columns []string
	//onDuplicate []Assignable
	onDuplicateKey *Upsert
}

func NewInserter[T any](sess Session) *Inserter[T] {
	c := sess.getCore()
	return &Inserter[T]{

		builder: builder{
			core:   c,
			quoter: c.dialect.quoter(),
		},
		sess: sess,
	}
}

// ConflictColumns 是一个中间方法
func (o *UpsertBuilder[T]) ConflictColumns(cols ...string) *UpsertBuilder[T] {
	o.conflictColumns = cols
	return o
}
func (o *UpsertBuilder[T]) Update(assigns ...Assignable) *Inserter[T] {
	o.i.onDuplicateKey = &Upsert{
		assigns:         assigns,
		conflictColumns: o.conflictColumns,
	}
	return o.i
}
func (i *Inserter[T]) OnDuplicateKey() *UpsertBuilder[T] {

	return &UpsertBuilder[T]{
		i: i,
	}
}

//	func(i *Inserter[T])Upsert(assigns...Assignable)*Inserter[T]{
//		i.onDuplicate=assigns
//		return i
//	}
func (i *Inserter[T]) Columns(cols ...string) *Inserter[T] {
	i.columns = cols
	return i
}

// Values 指定插入的数据
func (i *Inserter[T]) Values(vals ...*T) *Inserter[T] {
	i.values = vals
	return i
}

func (i *Inserter[T]) Build() (*Query, error) {
	var err error
	if len(i.values) == 0 {
		return nil, errs.ErrInsertZeroRow
	}
	i.sb = strings.Builder{}
	//会引发复制问题
	//sb:=i.sb
	i.sb.WriteString("INSERT INTO ")
	if i.model == nil {
		m, err := i.r.Get(i.values[0])
		i.model = m
		if err != nil {
			return nil, err
		}
	}

	//拿到元数据之后，拼接表名
	//sb.WriteByte('`')
	//sb.WriteString(m.TableName)
	//sb.WriteByte('`')
	i.quote(i.model.TableName)
	//一定要显式的指定列的顺序，不然我们不知道数据库中默认的数据顺序
	//我们要构造 `test_model`(col,col2...)
	i.sb.WriteByte('(')

	fields := i.model.Fields
	//如果用户指定列，重构fields
	if len(i.columns) > 0 {
		fields = make([]*model.Field, 0, len(fields))
		for _, fd := range i.columns {
			fdMeta, ok := i.model.FieldMap[fd]
			if !ok {
				return nil, errs.NewUnknownColumn(fd)
			}
			fields = append(fields, fdMeta)
		}
	}

	//不能遍历FieldMap,ColMap，因为map的遍历顺序每一次顺序都不一样
	//所以额外引入一个[]Fields
	for idx, field := range fields {
		if idx > 0 {
			i.sb.WriteByte(',')
		}
		//sb.WriteByte('`')
		//sb.WriteString(field.ColName)
		//sb.WriteByte('`')
		i.quote(field.ColName)
	}
	i.sb.WriteByte(')')
	//拼接values
	i.sb.WriteString(" VALUES ")

	//预估的参数数量是，我有多少行乘有多少个字段
	i.args = make([]any, 0, len(i.values)*len(i.model.Fields))
	for j, v := range i.values {
		if j > 0 {
			i.sb.WriteByte(',')
		}
		i.sb.WriteByte('(')
		val := i.creator(i.model, v)
		for idx, field := range fields {
			if idx > 0 {
				i.sb.WriteByte(',')
			}
			i.sb.WriteByte('?')
			//把参数读出来
			//arg:=reflect.ValueOf(val).Elem().FieldByName(field.GoName).Interface()
			arg, err := val.Field(field.GoName)
			if err != nil {
				return nil, err
			}
			i.addArgs(arg)
		}
		i.sb.WriteByte(')')
	}

	//构造upsert
	if i.onDuplicateKey != nil {

		err = i.dialect.buildUpsert(&i.builder, i.onDuplicateKey)
		if err != nil {
			return nil, err
		}
		//for idx,assign:=range i.onDuplicateKey.assigns{
		//	if idx>0{
		//		sb.WriteByte(',')
		//	}
		//	switch a:=assign.(type) {
		//	case Assignment:
		//		fd,ok:=m.FieldMap[a.col]
		//		if !ok{
		//			return nil,errs.NewUnknownField(a.col)
		//		}
		//		sb.WriteByte('`')
		//		sb.WriteString(fd.ColName)
		//		sb.WriteByte('`')
		//		sb.WriteString("=?")
		//		args=append(args,a.val)
		//	case Column:
		//		fd,ok:=m.FieldMap[a.name]
		//		if !ok{
		//			return nil,errs.NewUnknownField(a.name)
		//		}
		//		sb.WriteByte('`')
		//		sb.WriteString(fd.ColName)
		//		sb.WriteByte('`')
		//		sb.WriteString("=VALUES(")
		//		sb.WriteByte('`')
		//		sb.WriteString(fd.ColName)
		//		sb.WriteByte('`')
		//		sb.WriteByte(')')
		//	default:
		//		return nil, errs.NewErrUnSupportedAssignable(assign)
		//	}
		//}
	}
	i.sb.WriteByte(';')
	return &Query{
		SQL:  i.sb.String(),
		Args: i.args,
	}, nil
}
func (i *Inserter[T]) Exec(ctx context.Context) Result {
	if i.model == nil {
		var err error
		i.model, err = i.r.Get(new(T))
		if err != nil {
			return Result{
				err: err,
			}
		}
	}
	//var root Handler= func(ctx context.Context, qc *QueryContext) *QueryResult {
	//	return execHandler(ctx,i.sess,i.core,qc)
	//}
	//for j:=len(i.mdls)-1;j>=0;j--{
	//	root=i.mdls[j](root)
	//}
	res := exec(ctx, i.sess, i.core, &QueryContext{
		Type:    "INSERT",
		Builder: i,
		Model:   i.model,
	})
	//res:=root(ctx,&QueryContext{
	//	Type: "INSERT",
	//	Builder: i,
	//	Model: i.model,
	//})
	//var t *T
	//if val,ok:=res.Result.(*T);ok{
	//	t=val
	//}
	//return t,res.Err
	var sqlRes sql.Result
	if res.Result != nil {
		sqlRes = res.Result.(sql.Result)
	}
	return Result{
		err: res.Err,
		res: sqlRes,
	}
	//q,err:=i.Build()
	//if err!=nil{
	//	return Result{
	//		err: err,
	//	}
	//}
	//res,err:=i.sess.execContext(ctx,q.SQL,q.Data...)
	//return Result{
	//	res: res,
	//	err: err,
	//}
}

//func (i *Inserter[T]) execHandler(ctx context.Context,qc *QueryContext) *QueryResult{
//	q,err:=i.Build()
//	if err!=nil{
//		return &QueryResult{
//			Err: err,
//			Result: Result{
//				err: err,
//			},
//		}
//	}
//	res,err:=i.sess.execContext(ctx,q.SQL,q.Data...)
//	return &QueryResult{
//		Err: err,
//		Result: Result{
//			res: res,
//			err: err,
//		},
//	}
//}
//func(s *Selector[T])addArg(vals ...any){
//	if len(vals)==0{
//		return
//	}
//	if s.args==nil{
//		s.args=make([]any,0,8)
//	}
//	s.args=append(s.args,vals...)
//	return
//}
//func(i *Inserter[T])buildColumn(c Column)error{
//	fd,ok:=s.model.FieldMap[c.name]
//	if !ok{
//		return errs.NewUnknownField(c.name)
//	}
//	s.sb.WriteByte('`')
//	s.sb.WriteString(fd.ColName)
//	s.sb.WriteByte('`')
//	if c.alias!=""{
//		s.sb.WriteString(" AS `")
//		s.sb.WriteString(c.alias)
//		s.sb.WriteByte('`')
//	}
//	return nil
//}
