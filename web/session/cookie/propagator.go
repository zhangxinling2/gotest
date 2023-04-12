package cookie

import "net/http"

type Option func(p *Propagator)

type Propagator struct {
	cookieName string
	cookieOption func(cookie *http.Cookie)
}

func NewPropagator(opts ...Option)*Propagator{
	res:= &Propagator{
		cookieName: "sessId",
		cookieOption: func(cookie *http.Cookie) {
			
		},
	}
	for _,opt:=range opts{
		opt(res)
	}
	return res
}
func WithCookieName(name string)Option{
	return func(p *Propagator) {
		p.cookieName=name
	}
}

func (p *Propagator) Inject(id string, write http.ResponseWriter) error {
	c:=&http.Cookie{
		Name: p.cookieName,
		Value: id,
	}
	p.cookieOption(c)
	http.SetCookie(write,c)
	return nil
}

func (p *Propagator) Extract(req *http.Request) (string, error) {
	c,err:=req.Cookie(p.cookieName)
	if err!=nil{
		return "",err
	}
	return c.Value,nil
}

func (p *Propagator) Remove(writer http.ResponseWriter) error {
	c:=&http.Cookie{
		Name: p.cookieName,
		MaxAge: -1,
	}
	http.SetCookie(writer,c)
	return nil
}

