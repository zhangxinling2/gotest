package orm

import (
	"context"
	"gotest/orm/internal/errs"

)

type Selector[T any] struct {
	builder
	table TableReference

	where []Predicate

	//有了session，不需要db
	//db *DB
	columns []Selectable

	sess Session
}
type Selectable interface {
	selectable()
}
func NewSelector[T any](sess Session)*Selector[T]{
	c := sess.getCore()
	return &Selector[T]{
		builder:builder{
			core:c,//如何初始化core,无法初始化，只能让session返回

			quoter: c.dialect.quoter(),
		},
		sess: sess,
	}
}
func (s *Selector[T]) Build() (*Query, error) {
	var err error
	if s.model==nil{
		s.model,err=s.r.Get(new(T))
		if err!=nil{
			return nil,err
		}
	}

	s.sb.WriteString("SELECT ")
	if err=s.buildColumns();err!=nil{
		return nil, err
	}
	s.sb.WriteString(" FROM ")
/* 将table定义转换为TableReference就需改造
	if s.table.==""{
		//用反射拿到表名
		s.sb.WriteByte('`')
		s.sb.WriteString(s.model.TableName)
		s.sb.WriteByte('`')
	}else {
		//加了表名让用户自己加
		//sb.WriteByte('`')
		s.sb.WriteString(s.table)
		//sb.WriteByte('`')

	}
 */
	if err=s.buildTable(s.table);err!=nil{
		return nil,err
	}
	//args:=make([]any,0,4)
	if len(s.where)>0{
		s.sb.WriteString(" WHERE ")
		p:=s.where[0]
		for i := 1; i < len(s.where); i++ {
			p=p.And(s.where[i])
		}

		if err=s.buildExpression(p);err!=nil{
			return nil, err
		}
	}

	s.sb.WriteByte(';')
	return &Query{
		SQL: s.sb.String(),
		Args:s.args,
	},nil
}
// buildTable 因为Join查询和predicate相似，所以定义出的方法也差不多
func(s  *Selector[T])buildTable(table TableReference)error{
	switch t:=table.(type) {
	case nil:
		//这是代表完全没有调用from
		s.quote(s.model.TableName)
	case Table:
		//这个地方拿到指定的表的元数据
		m,err:=s.r.Get(t.entity)
		if err!=nil{
			return err
		}
		s.quote(m.TableName)
		if t.alias!=""{
			s.sb.WriteString(" AS ")
			s.quote(t.alias)
		}
	case Join:
		s.sb.WriteByte('(')
		//构造左侧
		if err:=s.buildTable(t.left);err!=nil{
			return err
		}
		s.sb.WriteByte(' ')
		s.sb.WriteString(t.typ)
		s.sb.WriteByte(' ')
		//构造右侧
		if err:=s.buildTable(t.right);err!=nil{
			return err
		}

		if len(t.using)>0{
			s.sb.WriteString(" USING (")
			//拼接 USING(xx,xx)
			for i,col:=range t.using{
				if i>0{
					s.sb.WriteString(",")
				}
				err:=s.buildColumn(Column{name: col})
				if err!=nil{
					return err
				}
			}
			s.sb.WriteByte(')')
		}
		if len(t.on)>0{
			s.sb.WriteString(" ON ")
			p:=t.on[0]
			for i := 1; i < len(s.where); i++ {
				p=p.And(t.on[i])
			}
			if err:=s.buildExpression(p);err!=nil{
				return err
			}
		}
		s.sb.WriteByte(')')
	default:
		return errs.NewUnsupportedTableReference(t)
	}
	return nil
}
func(s  *Selector[T])buildExpression(expr Expression)error{
	switch exp:=expr.(type) {
	case nil:
		
	case Predicate:
		//在这里处理p
		//p.left 构建好
		//p.op 构建好
		//p.right构建好
		_,ok:=exp.left.(Predicate)
		if ok{
			s.sb.WriteByte('(')
		}
		if err:=s.buildExpression(exp.left);err!=nil{
			return err
		}
		if ok{
			s.sb.WriteByte(')')
		}
		if exp.op!=""{
			s.sb.WriteByte(' ')
			s.sb.WriteString(exp.op.String())
			s.sb.WriteByte(' ')
		}

		_,ok=exp.right.(Predicate)
		if ok{
			s.sb.WriteByte('(')
		}
		if err:=s.buildExpression(exp.right);err!=nil{
			return err
		}
		if ok{
			s.sb.WriteByte(')')
		}
		//switch left:=expr.left.(type) {
		//case Column:
		//	sb.WriteByte('`')
		//	sb.WriteString(left.name)
		//	sb.WriteByte('`')
		//	//剩下的暂时不考虑
		//}
		//sb.WriteString(exp.op.String())
		//switch right:=p.right.(type) {
		//case value:
		//	sb.WriteByte('?')
		//	args= append(args, right.val)
		//	//剩下的暂时不考虑
		//}
	//为了处理列查询，把column抽取出去
	case Column:
		exp.alias=""
		return s.buildColumn(exp)
	//	fd,ok:=s.model.FieldMap[exp.name]
	//	if !ok{
	//		return errs.NewUnknownField(exp.name)
	//	}
	//	s.sb.WriteByte('`')
	//	s.sb.WriteString(fd.ColName)
	//	s.sb.WriteByte('`')
		//剩下的暂时不考虑
	case RawExpr:
		s.sb.WriteByte('(')
		s.sb.WriteString(exp.raw)
		s.addArg(exp.args...)
		s.sb.WriteByte(')')
	case value:
		s.sb.WriteByte('?')
		s.addArg(exp.val)
		//args= append(args, right.val)
		//剩下的暂时不考虑
	default:
		return errs.NewUnsupportedExpression(exp)

	}
	return nil
}

func(s *Selector[T])buildColumns()error{
	if len(s.columns)>0{
		for i,col:=range s.columns{
			if i>0{
				s.sb.WriteByte(',')
			}
			switch c:=col.(type) {
			case Column:

				err:=s.buildColumn(c)

				if err!=nil{
					return  err
				}
			case Aggregate:
				//聚合函数名
				s.sb.WriteString(c.fn)
				s.sb.WriteByte('(')
				err:=s.buildColumn(Column{name: c.arg})
				s.sb.WriteByte(')')
				if err!=nil{
					return err
				}
				//聚合函数本身的别名
				if c.alias!=""{
					s.sb.WriteString(" AS `")
					s.sb.WriteString(c.alias)
					s.sb.WriteByte('`')
				}
			case RawExpr:
				s.sb.WriteString(c.raw)

				s.addArg(c.args...)
			}

		}
	}else {
		//没有指定列
		s.sb.WriteByte('*')
	}
	return nil
}
//因为增删改查都用buildColumn所以放到builder里面去
//func(s *Selector[T])buildColumn(c Column)error{
//	//column添加TableReference后就需检测有没有指定table如果指定了就不能直接使用model
//	switch table:=c.table.(type) {
//	case nil:
//		fd,ok:=s.model.FieldMap[c.name]
//		if !ok{
//			return errs.NewUnknownField(c.name)
//		}
//		//s.sb.WriteByte('`')
//		//s.sb.WriteString(fd.ColName)
//		//s.sb.WriteByte('`')
//		s.quote(fd.ColName)
//		if c.alias!=""{
//			s.sb.WriteString(" AS ")
//			s.quote(c.alias)
//		}
//		return nil
//	case Table:
//		m,err:=s.r.Get(table.entity)
//		if err!=nil{
//			return err
//		}
//		fd,ok:=m.FieldMap[c.name]
//		if !ok{
//			return errs.NewUnknownField(c.name)
//		}
//		if table.alias!=""{
//			s.quote(table.alias)
//			s.sb.WriteByte('.')
//		}
//		s.quote(fd.ColName)
//		if c.alias!=""{
//			s.sb.WriteString(" AS ")
//			s.quote(c.alias)
//		}
//		return nil
//	default:
//		return errs.NewUnsupportedTableReference(table)
//	}
//	//fd,ok:=s.model.FieldMap[c.name]
//	//if !ok{
//	//	return errs.NewUnknownField(c.name)
//	//}
//	//s.sb.WriteByte('`')
//	//s.sb.WriteString(fd.ColName)
//	//s.sb.WriteByte('`')
//	//if c.alias!=""{
//	//	s.sb.WriteString(" AS `")
//	//	s.sb.WriteString(c.alias)
//	//	s.sb.WriteByte('`')
//	//}
//	return nil
//}
func(s *Selector[T])addArg(vals ...any){
	if len(vals)==0{
		return
	}
	if s.args==nil{
		s.args=make([]any,0,8)
	}
	s.args=append(s.args,vals...)
	return
}
//func(s *Selector[T])Where(query string,args ...any)*Selector[T]{
//
//}

//cols 是用于Where的列，难以解决And Or和Not等问题
//func(s *Selector[T])Where(cols []string,args ...int)*Selector[T]{
//	return s
//}

func(s *Selector[T])Where(ps...Predicate)*Selector[T]{
	s.where=ps
	return s
}
func(s *Selector[T])From(table TableReference)*Selector[T]{
	s.table=table
	return s
}
func(s *Selector[T])Select(cols...Selectable)*Selector[T]{
	s.columns=cols
	return s
}

//func (s *Selector[T]) GetV1(ctx context.Context) (*T, error){
//	q,err:=s.Build()
//	if err!=nil{
//		return nil, err
//	}
//	//在这里发起查询并处理结果集
//	db:=s.db.db
//	rows,err:=db.QueryContext(ctx,q.SQL,q.Args...)
//	//这是查询错误，数据库返回的
//	if err!=nil{
//		return nil, err
//	}
//	//将row 转化成*T
//	//在这里处理结果集
//	if !rows.Next(){
//		//要不要返回error
//		//返回error,和sql包语义保持一致 sql.ErrNoRows
//		return nil, ErrNoRows
//	}
//
//	//拿到了select出来的列
//	cs,err:=rows.Columns()
//	if err!=nil{
//		return nil, err
//	}
//
//	var vals []any
//	tp:=new(T)
//	//起始地址
//	address:=reflect.ValueOf(tp).UnsafePointer()
//	for _,c:=range cs{
//		//c是列名
//		fd,ok:=s.model.ColumnMap[c]
//		if !ok{
//			return nil,errs.NewUnknownColumn(c)
//		}
//
//		//要计算字段的地址
//		fdAddress:=unsafe.Pointer(uintptr(address)+fd.Offset)
//		val:=reflect.NewAt(fd.Type,fdAddress)
//		vals=append(vals, val.Interface())
//	}
//	err=rows.Scan(vals...)
//	return tp,nil
//}
func (s *Selector[T]) Get(ctx context.Context) (*T, error) {
	var err error
	s.model,err=s.r.Get(new(T))
	if err!=nil {
		return nil, err
	}
	res:=get[T](ctx,s.sess,s.core,&QueryContext{
		Type: "SELECT",
		Builder: s,
		Model: s.model,
	})
	if res.Result!=nil{
		return res.Result.(*T),res.Err
	}
	return nil,res.Err
}
// Get 把Handler拆出来后重构
//func (s *Selector[T]) Get(ctx context.Context) (*T, error) {
//	if s.model==nil{
//		var err error
//		s.model,err=s.r.Get(new(T))
//		if err!=nil{
//			return nil, err
//		}
//	}
//
//	root:=s.getHandler
//	for i:=len(s.mdls)-1;i>=0;i--{
//		root=s.mdls[i](root)
//	}
//	res:=root(ctx,&QueryContext{
//		Type: "SELECT",
//		Builder: s,
//		//问题在于s.model在Build时才会赋值，1.在Get初始化s.model 2.专门设置一个middleware来设置model
//		Model: s.model,
//	})
//	//var t *T
//	//if val,ok:=res.Result.(*T);ok{
//	//	t=val
//	//}
//	//return t,res.Err
//	if res.Result!=nil{
//		return res.Result.(*T),res.Err
//	}
//	return nil,res.Err
//	//q,err:=s.Build()
//	//if err!=nil{
//	//	return nil, err
//	//}
//	////在这里发起查询并处理结果集
//	//
//	//rows,err:=s.sess.queryContext(ctx,q.SQL,q.Args...)
//	////这是查询错误，数据库返回的
//	//if err!=nil{
//	//	return nil, err
//	//}
//	////将row 转化成*T
//	////在这里处理结果集
//	//if !rows.Next(){
//	//	//要不要返回error
//	//	//返回error,和sql包语义保持一致 sql.ErrNoRows
//	//	return nil, ErrNoRows
//	//}
//	//
//	//
//	//
//	//tp:=new(T)
//	//creator:=s.creator
//	//val:=creator(s.model,tp)
//	//err=val.SetColumns(rows)
//	//
//	////接口定义好后，一个是用新接口的方法改造上层，一个就是提供不同实现
//	//return tp,err
//
//	////拿到了select出来的列
//	//cs,err:=rows.Columns()
//	//if err!=nil{
//	//	return nil, err
//	//}
//
//	//vals:=make([]any,0,len(cs))
//	//
//	//valElem:=make([]reflect.Value,0,len(cs))
//	//for _,c:=range cs{
//	//	//c是列名
//	//	fd,ok:=s.model.ColumnMap[c]
//	//	if !ok{
//	//		return nil,errs.NewUnknownColumn(c)
//	//	}
//	//	//反射创建新的实例
//	//	//这里创建的实例是原本类型的指针
//	//	//例如 fd.type=int 那么val是*int
//	//	val:=reflect.New(fd.Type)
//	//	//这样scan就不用取地址了
//	//	vals=append(vals, val.Interface())
//	//
//	//	valElem=append(valElem, val.Elem())
//	//
//	//	//for _,fd:=range s.model.FieldMap{
//	//	//	if fd.ColName==c{
//	//	//		//反射创建新的实例
//	//	//		//这里创建的实例是原本类型的指针
//	//	//		//例如 fd.type=int 那么val是*int
//	//	//		val:=reflect.New(fd.typ)
//	//	//		//这样scan就不用取地址了
//	//	//		vals=append(vals, val.Interface())
//	//	//	}
//	//	//}
//	//}
//	////1 顺序要匹配
//	////2 类型要匹配
//	//err=rows.Scan(vals...)
//	//if err!=nil{
//	//	return nil, err
//	//}
//	//tpValue:=reflect.ValueOf(tp)
//	//for i,c:=range cs{
//	//	fd,ok:=s.model.ColumnMap[c]
//	//	if !ok{
//	//		return nil,errs.NewUnknownColumn(c)
//	//	}
//	//	tpValue.Elem().FieldByName(fd.GoName).Set(valElem[i])
//	//	//for _,fd:=range s.model.FieldMap{
//	//	//	if fd.ColName==c{
//	//	//		tpValue.Elem().FieldByName(fd.GoName).Set(reflect.ValueOf(vals[i]).Elem())
//	//	//	}
//	//	//}
//	//
//	//}
//	//
//	//
//	//return tp, nil
//}

//func getHandler[T any](ctx context.Context,sess Session,c core,qc *QueryContext) *QueryResult{
//	q,err:=qc.Builder.Build()
//	if err!=nil{
//		return &QueryResult{
//			Err: err,
//		}
//	}
//	//在这里发起查询并处理结果集
//
//	rows,err:=sess.queryContext(ctx,q.SQL,q.Args...)
//	//这是查询错误，数据库返回的
//	if err!=nil{
//		return &QueryResult{
//			Err: err,
//		}
//	}
//	//将row 转化成*T
//	//在这里处理结果集
//	if !rows.Next(){
//		//要不要返回error
//		//返回error,和sql包语义保持一致 sql.ErrNoRows
//		//return nil, ErrNoRows
//		return &QueryResult{
//			Err: ErrNoRows,
//		}
//	}
//
//
//
//	tp:=new(T)
//	creator:=c.creator
//	val:=creator(c.model,tp)
//	err=val.SetColumns(rows)
//
//	//接口定义好后，一个是用新接口的方法改造上层，一个就是提供不同实现
//	return &QueryResult{
//		Err: err,
//		Result: tp,
//	}
//}
func (s *Selector[T]) GetMulti(ctx context.Context) ([]*T, error) {
	q,err:=s.Build()
	if err!=nil{
		return nil, err
	}
	//在这里发起查询并处理结果集

	_,err=s.sess.queryContext(ctx,q.SQL,q.Args...)
	return nil, nil
	//return rows.Next(){
	//	//在这里构造[]*T
	//},nil
}

