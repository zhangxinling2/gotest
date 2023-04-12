package orm

import (
	"gotest/orm/internal/errs"
	"strings"
)

type builder struct {
	sb strings.Builder
	args []any
	core
	quoter byte
}

func(b *builder)quote(name string){
	b.sb.WriteByte(b.quoter)
	b.sb.WriteString(name)
	b.sb.WriteByte(b.quoter)
}
//func(b *builder)buildColumn(name string)error{
//	fd,ok := b.model.FieldMap[name]
//	if !ok{
//		return errs.NewUnknownField(name)
//	}
//	b.quote(fd.ColName)
//	return nil
//}
func(b *builder)buildColumn(c Column)error{
	//column添加TableReference后就需检测有没有指定table如果指定了就不能直接使用model
	switch table:=c.table.(type) {
	case nil:
		fd,ok:= b.model.FieldMap[c.name]
		if !ok{
			return errs.NewUnknownField(c.name)
		}
		//b.sb.WriteByte('`')
		//b.sb.WriteString(fd.ColName)
		//b.sb.WriteByte('`')
		b.quote(fd.ColName)
		if c.alias!=""{
			b.sb.WriteString(" AS ")
			b.quote(c.alias)
		}
		return nil
	case Table:
		m,err:= b.r.Get(table.entity)
		if err!=nil{
			return err
		}
		fd,ok:=m.FieldMap[c.name]
		if !ok{
			return errs.NewUnknownField(c.name)
		}
		if table.alias!=""{
			b.quote(table.alias)
			b.sb.WriteByte('.')
		}
		b.quote(fd.ColName)
		if c.alias!=""{
			b.sb.WriteString(" AS ")
			b.quote(c.alias)
		}
		return nil
	default:
		return errs.NewUnsupportedTableReference(table)
	}
	//fd,ok:=b.model.FieldMap[c.name]
	//if !ok{
	//	return errs.NewUnknownField(c.name)
	//}
	//b.sb.WriteByte('`')
	//b.sb.WriteString(fd.ColName)
	//b.sb.WriteByte('`')
	//if c.alias!=""{
	//	b.sb.WriteString(" AS `")
	//	b.sb.WriteString(c.alias)
	//	b.sb.WriteByte('`')
	//}
	return nil
}
func(b *builder)addArgs(vals...any){
	if len(vals)==0{
		return
	}
	if b.args==nil{
		b.args=make([]any,0,8)
	}
	b.args=append(b.args,vals...)
	return
}