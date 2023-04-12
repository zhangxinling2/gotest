package err

import (
	"errors"
	"testing"
)
var LessThanTwoError = errors.New("error 1")
var MoreThanHubdredError = errors.New("error 2")
func GetFibonacci(n int) ([]int,error){
	if n<2{
		return nil,LessThanTwoError
	}
	if n>100{
		return nil,MoreThanHubdredError
	}
	fibList:=[]int{1,1}
	for i:=2;i<n;i++{
		fibList=append(fibList,fibList[i-2]+fibList[i-1])
	}
	return fibList,nil
}
func TestGetFibonacci(t *testing.T) {
	if v,err:=GetFibonacci(-10);err!=nil{
		t.Error(err)
	}else{
		t.Log(v)
	}

}
