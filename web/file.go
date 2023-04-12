package web

import (
	lru "github.com/hashicorp/golang-lru"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)
//下载上传可以考虑使用OSS，比我们手写代码稳定，安全，还会处理大文件，正常云厂商OSS和CDN配合
type FileUpLoader struct {
	FileField string
	//为什么用户传
	//要考虑重名的问题
	//很多时候目标名字

	//用于计算目标路径
	DstPathFunc func(header *multipart.FileHeader)string
}

func(u FileUpLoader)Handle()HandleFunc{
	return func(ctx *Context) {
		//上传文件的逻辑
		//第一步 读到文件内容
		//第二部 计算出目标路径
		//第三步 保存文件
		//第四步 返回响应
		file,fileHeader,err:=ctx.Req.FormFile(u.FileField)
		if err!=nil{
			ctx.RespStatusCode=500
			ctx.RespData=[]byte("上次失败"+err.Error())
			return
		}
		defer file.Close()
		// 怎么知道目标路径
		// 这种做法就是将目标路径计算的逻辑交给用户
		dst:=u.DstPathFunc(fileHeader)
		//可以尝试把dst上不存在的目录全部建立起来
		//os.Mkd// irAll()

		//os.O_WRONLY写入数据
		//os.O_TRUNC如果文件本身存在清空数据
		//os.O_CREATE如果不存在，创建
		dstFile,err:=os.OpenFile(dst,os.O_WRONLY|os.O_TRUNC|os.O_CREATE,0o666)
		if err!=nil{
			ctx.RespStatusCode=500
			ctx.RespData=[]byte("上次失败"+err.Error())
			return
		}
		defer dstFile.Close()
		//buf 影响性能
		// 要考虑复用
		_,err = io.CopyBuffer(dstFile,file,nil)
		if err!=nil{
			ctx.RespStatusCode=500
			ctx.RespData=[]byte("上次失败")
			return
		}
		ctx.RespStatusCode=200
		ctx.RespData=[]byte("上次成功")
	}
}


type FileDownLoader struct {
	Dir string//文件夹
}

func(d FileDownLoader)Handle()HandleFunc{
	return func(ctx *Context) {
		//用的是/xxx?file=xxx
		req:=ctx.QueryValue("file")
		if req.err!=nil{
			ctx.RespStatusCode=400
			ctx.RespData=[]byte("找不到目标文件")
			return
		}
		req.value=filepath.Clean(req.value)

		dst:=filepath.Join(d.Dir,req.value)
		//做一个校验，防止相对路径引起攻击者下载了你的系统文件
		//dst,err:=filepath.Abs(dst)
		//if strings.Contains(dst,d.Dir){
		//
		//}

		fn:=filepath.Base(dst)//last value
		header := ctx.Resp.Header()
		header.Set("Content-Disposition", "attachment;filename="+fn)
		header.Set("Content-Description", "File Transfer")
		header.Set("Content-Type", "application/octet-stream")
		header.Set("Content-Transfer-Encoding", "binary")
		header.Set("Expires", "0")//这两个是缓存选项
		header.Set("Cache-Control", "must-revalidate")//这两个是缓存选项
		header.Set("Pragma", "public")

		http.ServeFile(ctx.Resp,ctx.Req,dst)//没有缓存，每次会发起磁盘io
	}
}

type StaticResourceHandlerOption func(handler *StaticResourceHandler)


//大文件不缓存
//控制住了缓存的文件的数量
//最多消耗 size(cache)*maxsize 内存
type StaticResourceHandler struct {
	dir string
	cache *lru.Cache
	extContextTypeMap map[string]string
	maxSize int
}
func NewStaticResourceHandler(dir string,opts...StaticResourceHandlerOption)(*StaticResourceHandler,error){
	//总共缓存key value的数量
	c,err:=lru.New(100*1024*1024)
	if err!=nil{
		return nil, err
	}

	res:=&StaticResourceHandler{
		dir:dir,
		cache: c,
		maxSize: 1024*1024*10,
		extContextTypeMap: map[string]string{
			// 这里根据自己的需要不断添加
			"jpeg": "image/jpeg",
			"jpe":  "image/jpeg",
			"jpg":  "image/jpeg",
			"png":  "image/png",
			"pdf":  "image/pdf",
		},
	}
	for _,opt:=range opts{
		opt(res)
	}
	return res,nil
}
func StaticWithMaxFileSize(maxSize int)StaticResourceHandlerOption{
	return func(handler *StaticResourceHandler) {
		handler.maxSize=maxSize
	}
}

func WithFileCache(c *lru.Cache)StaticResourceHandlerOption{
	return func(handler *StaticResourceHandler) {
		handler.cache=c
	}
}

func StaticWithMoreExtions(extMap map[string]string)StaticResourceHandlerOption{
	return func(handler *StaticResourceHandler) {
		for ext, contentType := range extMap {
			handler.extContextTypeMap[ext] = contentType
		}
	}	
}

func (s *StaticResourceHandler)Handle(ctx *Context){
	//无缓存
	//1.拿到目标文件夹名
	//2.定位到目标文件，读出来
	//3.返回给前端

	//有缓存
	//
	req:=ctx.PathValue("file")
	if req.err!=nil{
		ctx.RespStatusCode=http.StatusBadRequest
		ctx.RespData=[]byte("请求路径不对")
		return
	}
	dst:=filepath.Join(s.dir,req.value)
	ext:=filepath.Ext(dst)[1:]
	header:=ctx.Resp.Header()

	if data,ok:=s.cache.Get(req.value);ok{
		contentType:=s.extContextTypeMap[ext]
		//可能的有文本文件，图片，多媒体
		header.Set("Content-Type",contentType)
		header.Set("Content-Length",strconv.Itoa(len(data.([]byte))))
		ctx.RespData=data.([]byte)
		ctx.RespStatusCode=200
		return
	}
	data,err:=os.ReadFile(dst)

	//大文件不缓存
	if len(data)<=s.maxSize{
		s.cache.Add(req.value,data)
	}




	if err!=nil{
		ctx.RespStatusCode=http.StatusInternalServerError
		ctx.RespData=[]byte("服务器错误")
		return
	}


	contentType:=s.extContextTypeMap[ext]
	//可能的有文本文件，图片，多媒体
	header.Set("Content-Type",contentType)
	header.Set("Content-Length",strconv.Itoa(len(data)))
	ctx.RespData=data
	ctx.RespStatusCode=200

}

