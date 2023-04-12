package orm

import (
	"gotest/orm/internal/errs"

)

var (
	DialectMySQL Dialect = mysqlDialect{}
	DialectSQLite Dialect = sqliteDialect{}
	DialectPostgre Dialect= postgreDialect{}
)

type Dialect interface {
	//quoter 为了解决引号问题
	//MySQL `
	quoter()byte

	buildUpsert(b *builder, upsert *Upsert) error
}

type standardSQL struct {

}

func (s standardSQL) quoter() byte {
	panic("implement me")
}

func (s standardSQL) buildUpsert(b *builder,upsert *Upsert) error {
	panic("implement me")
}

type mysqlDialect struct {
	standardSQL
}
func (m mysqlDialect) quoter() byte {
	return '`'
}
func (m mysqlDialect) buildUpsert(b *builder,upsert *Upsert) error {
	b.sb.WriteString(" ON DUPLICATE KEY UPDATE ")
	for idx,assign:=range upsert.assigns{
		if idx>0{
			b.sb.WriteByte(',')
		}
		switch a:=assign.(type) {
		case Assignment:
			fd,ok:=b.model.FieldMap[a.col]
			if !ok{
				return errs.NewUnknownField(a.col)
			}
			b.quote(fd.ColName)
			//b.sb.WriteByte('`')
			//b.sb.WriteString(fd.ColName)
			//b.sb.WriteByte('`')
			b.sb.WriteString("=?")
			b.addArgs(a.val)
		case Column:
			fd,ok:=b.model.FieldMap[a.name]
			if !ok{
				return errs.NewUnknownField(a.name)
			}
			b.quote(fd.ColName)
			//b.sb.WriteByte('`')
			//b.sb.WriteString(fd.ColName)
			//b.sb.WriteByte('`')
			b.sb.WriteString("=VALUES(")
			b.quote(fd.ColName)
			//b.sb.WriteByte('`')
			//b.sb.WriteString(fd.ColName)
			//b.sb.WriteByte('`')
			b.sb.WriteByte(')')
		default:
			return errs.NewErrUnSupportedAssignable(assign)
		}
	}
	return nil
}

type sqliteDialect struct {
	standardSQL
}
func (s sqliteDialect) quoter() byte {
	return '`'
}
func (s sqliteDialect) buildUpsert(b *builder,upsert *Upsert) error {
	b.sb.WriteString("ON CONFLICT(")
	for i,col:=range upsert.conflictColumns{
		if i>0{
			b.sb.WriteByte(',')
		}
		err:=b.buildColumn(Column{
			name: col,
		})
		if err!=nil{
			return err
		}
	}
	b.sb.WriteString(") DO UPDATE SET ")
	for idx,assign:=range upsert.assigns{
		if idx>0{
			b.sb.WriteByte(',')
		}
		switch a:=assign.(type) {
		case Assignment:
			fd,ok:=b.model.FieldMap[a.col]
			if !ok{
				return errs.NewUnknownField(a.col)
			}
			b.quote(fd.ColName)
			//b.sb.WriteByte('`')
			//b.sb.WriteString(fd.ColName)
			//b.sb.WriteByte('`')
			b.sb.WriteString("=?")
			b.addArgs(a.val)
		case Column:
			fd,ok:=b.model.FieldMap[a.name]
			if !ok{
				return errs.NewUnknownField(a.name)
			}
			b.quote(fd.ColName)
			//b.sb.WriteByte('`')
			//b.sb.WriteString(fd.ColName)
			//b.sb.WriteByte('`')
			b.sb.WriteString("=excluded.")
			b.quote(fd.ColName)
			//b.sb.WriteByte('`')
			//b.sb.WriteString(fd.ColName)
			//b.sb.WriteByte('`')
			//b.sb.WriteByte(')')
		default:
			return errs.NewErrUnSupportedAssignable(assign)
		}
	}
	return nil
}
type postgreDialect struct {
	standardSQL
}

