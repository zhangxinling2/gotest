package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
)

type Context struct {
	Req *http.Request

	// Resp 如果用户直接使用这个。那么用户就绕开了RespData和RespStatusCode，那么部分middleware无法运作
	Resp http.ResponseWriter

	//这个主要是为了个 middleware读写用的
	RespData []byte
	RespStatusCode int

	PathParams map[string]string

	//query的缓存
	queryValues url.Values

	//cookieSameSite http.SameSite
	MatchRoute string

	tplEngine TemplateEngine

	UserValues map[string]any
}

func(c *Context)Render(tplName string,data any)error{
	RespData,err:=c.tplEngine.Render(c.Req.Context(),tplName,data)
	if err!=nil{
		c.RespStatusCode=http.StatusInternalServerError
		return err
	}
	c.RespData=RespData
	c.RespStatusCode=http.StatusOK
	return nil
}

func(c *Context)SetCookie(ck *http.Cookie){
	//ck.SameSite=c.cookieSameSite
	http.SetCookie(c.Resp,ck)
}

func(c *Context)RespJSONOK(val any)error{
	return c.RespJSON(200,val)
}
//如果val是string或者[]byte就不需要调用RespJSON
func(c *Context)RespJSON(status int,val any)error{
	data,err:=json.Marshal(val)
	if err!=nil{
		return err
	}
	c.Resp.WriteHeader(status)
	_, err=c.Resp.Write(data)
	c.RespData=data
	c.RespStatusCode=status
	return err
}

//解决大多数人需求，不为小众需求污染核心
func(c *Context)BindJSON(val any)error{
	if val==nil {
		return errors.New("web:输入为nil")
	}
	if c.Req.Body==nil{
		return errors.New("web:body为nil")
	}
	decoder:=json.NewDecoder(c.Req.Body)
	//userNumber 数字就用Number来表示
	//否则默认是float64
	//decoder.UseNumber()
	//如果有未知字段，就会报错
	// decoder.DisallowUnknownFields()
	return decoder.Decode(val)
}
//不会重复parse,PaserForm已经做过判断
func(c *Context)FormValue(key string)StringValue{
	err:=c.Req.ParseForm()
	if err!=nil{
		return StringValue{
			value : "",
			err: err,
		}
	}
	vals,ok:=c.Req.Form[key]
	if !ok{
		return StringValue{
			value: "",
			err:errors.New("web:key不存在"),
		}
	}
	return StringValue{vals[0],nil}
	//FormValue没有处理error
	//return c.Req.FormValue(key),nil
}
//如果要加此类方法，方法过多因为类型过多
//func(c *Context)FormValueAsInt64(key string)(int64,error){
//
//}

//Query调用的parseQuery没有缓存
func(c *Context)QueryValue(key string)StringValue{
	if c.queryValues==nil{
		c.queryValues=c.Req.URL.Query()
	}
	vals,ok:=c.queryValues[key]
	if !ok{
		return StringValue{"",errors.New("web:key不存在")}
	}
	return StringValue{vals[0],nil}
	//用户区别不出来是真的有值，但值是空字符串，还是没有值
	//在这里缓存住
	return StringValue{c.queryValues.Get(key),nil}
}

func(c *Context)PathValue(key string)StringValue{
	val,ok:=c.PathParams[key]
	if !ok{
		return StringValue{"",errors.New("web:key不存在")}
	}
	return StringValue{val,nil}
}

type StringValue struct {
	value string
	err error
}

func (s StringValue)AsInt64()(int64,error){
	if s.err!=nil{
		return 0,s.err
	}
	return strconv.ParseInt(s.value,10,64)
}