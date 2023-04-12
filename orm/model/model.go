package model

import (
	"gotest/orm/internal/errs"
	"reflect"
	"strings"
	"sync"
	"unicode"
)

const(
	tagKeyColumn="column"
)

type Registry interface {
	Get(val any)(*Model,error)
	Register(val any,opts...ModelOpt)(*Model,error)
}
type Model struct {
	TableName string
	Fields []*Field
	//字段名到字段的映射
	FieldMap map[string]*Field
	//列名到字段定义的映射
	ColumnMap map[string]*Field
}
//option变种
type ModelOpt func(m *Model)error

type Field struct {
	ColName string

	Type reflect.Type
	//字段名
	GoName string

	//字段相对于结构体本身的偏移量
	Offset uintptr
}

//var models = map[reflect.Type]*Model{}

type registry struct {
	//lock sync.RWMutex
	models sync.Map
}
//var models=&registry{
//	models: make(map[reflect.Type]*Model,16),
//}
func NewRegistry()Registry {
	return &registry{
		//models: make(map[reflect.Type]*Model,64),
	}
}
func(r *registry)Get(val any)(*Model,error){
	typ:=reflect.TypeOf(val)
	m,ok:=r.models.Load(typ)
	if ok{
		return m.(*Model),nil
	}
	m,err:=r.Register(val)
	if err!=nil{
		return nil, err
	}

	return m.(*Model), nil
}
//func(r *registry)get1(val any)(*Model,error){
//	typ:=reflect.TypeOf(val)
//	r.lock.RLock()
//	m,ok:=r.models[typ]
//	r.lock.Unlock()
//	if ok{
//		return m,nil
//	}
//	r.lock.Lock()
//	defer r.lock.Unlock()
//	m,ok=r.models[typ]
//	if ok{
//		return m,nil
//	}
//
//	m,err:=r.parseModel(val)
//	if err!=nil{
//		return nil, err
//	}
//	r.models[typ]=m
//
//	return m,nil
//}

//Register限制只能用一级指针
func (r *registry)Register(entity any,opts...ModelOpt)(*Model,error){
	typ :=reflect.TypeOf(entity)
	if typ.Kind()!=reflect.Pointer|| typ.Elem().Kind()!=reflect.Struct{
		return nil,errs.ErrPointOnly
	}
	elemTyp := typ.Elem()
	numField:= elemTyp.NumField()
	fieldMap:=make(map[string]*Field,numField)
	columnMap:=make(map[string]*Field,numField)
	fields :=make([]*Field,0,numField)
	for i := 0; i < numField; i++ {
		fd:= elemTyp.Field(i)
		pair,err:=r.parseTag(fd.Tag)
		if err!=nil{
			return nil, err
		}
		colName:=pair[tagKeyColumn]
		if colName==""{
			//用户没有设置
			colName= underscoreName(fd.Name)
		}
		fdMeta:=&Field{
			ColName: colName,
			Type:    fd.Type,
			GoName:  fd.Name,
			Offset:  fd.Offset,
		}
		fieldMap[fd.Name]=fdMeta
		columnMap[colName]=fdMeta
		fields= append(fields,fdMeta)
	}
	var tableName string
	if tbl,ok:= entity.(TableName);ok{
		tableName = tbl.TableName()
	}
	if tableName==""{
		tableName= underscoreName(elemTyp.Name())
	}


	res:= &Model{
		TableName: tableName,
		FieldMap:  fieldMap,
		ColumnMap: columnMap,
		Fields: fields,
	}
	for _,opt:=range opts{
		err:=opt(res)
		if err!=nil{
			return nil, err
		}
	}
	r.models.Store(typ,res)
	return res,nil
}
func WithColumnName(field string,columnName string) ModelOpt {
	return func(m *Model) error {
		fd,ok:=m.FieldMap[field]
		if !ok{
			return errs.NewUnknownField(field)
		}
		fd.ColName =columnName
		return nil
	}
}
func WithTableName(tableName string) ModelOpt {
	return func(m *Model) error {
		m.TableName =tableName
		//if m.tableName==""{
		//	return err
		//}
		return nil
	}
}
//type User struct {
//	ID uint64 `orm:"column=id,xxx=bbb"`
//}
func(r *registry) parseTag(tag reflect.StructTag)(map[string]string,error){
	ormTag,ok:=tag.Lookup("orm")
	if !ok{
		return map[string]string{},nil
	}
	pairs:=strings.Split(ormTag,",")
	res:=make(map[string]string,len(pairs))
	for _,pair:=range pairs{
		segs:=strings.Split(pair,"=")
		if len(segs)!=2{
			return nil,errs.NewErrInvalidTagContext(pair)
		}
		key:=segs[0]
		val:=segs[1]
		res[key]=val
	}
	return res,nil
}

func underscoreName(tableName string)string{
	var buf []byte
	for i,v:=range tableName{
		if unicode.IsUpper(v){
			if i!=0{
				buf = append(buf,'_')
			}
			buf=append(buf,byte(unicode.ToLower(v)))
		}else {
			buf=append(buf,byte(v))
		}
	}
	return string(buf)
}
type TableName interface {
	TableName() string
}