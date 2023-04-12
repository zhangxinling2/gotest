package obj_pool

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

type ReusableObj struct {

}

type ObjPool struct {
	bufChan chan *ReusableObj
}

func NewObjPool(numOfObj int) *ObjPool{
	objPool:=ObjPool{}
	objPool.bufChan = make(chan *ReusableObj,numOfObj)
	for i:=0;i<numOfObj;i++{
		objPool.bufChan<-&ReusableObj{}
	}
	return &objPool
}

func (p *ObjPool)GetObj(timeout time.Duration)(*ReusableObj,error)  {
	select {
	case ret:=<-p.bufChan:
		return ret,nil
	case <-time.After(timeout):
		return nil, errors.New("time out")
	}
}

func (p *ObjPool)ReleaseObj(obj *ReusableObj) error  {
	select {
	case p.bufChan<-obj:
		return nil
	default:
		return errors.New("overflow")
	}
}

func TestObjPool(t *testing.T) {
	pool:=NewObjPool(10)
	for i:=0;i<11;i++{
		if v,err:=pool.GetObj(time.Second*1);err!=nil{
			t.Error(err)
		}else{
			fmt.Printf("%T\n",v)
			if err:=pool.ReleaseObj(v);err!=nil{
				t.Error(err)
			}
		}
	}
	fmt.Println("Done")
}