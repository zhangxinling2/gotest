package poker

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func shutter(list []int) []int{
	rand.Seed(time.Now().UnixNano())
	for i:=len(list)-1;i>0;i--{
		r:=rand.Intn(len(list))
		index :=list[i]
		list[i]=list[r]
		list[r]=index
	}
	return list
}
func shutter2(list []int) []int  {
	r:=rand.New(rand.NewSource(time.Now().UnixNano()))
	for _,i:= range r.Perm(len(list)){
		fmt.Println(i)
		val:=list[i]
		fmt.Println(val)
	}
	return list
}
func TestShutter(t *testing.T) {
	list:=[]int{1,2,11,4,5,6,7,12,9}
	t.Log(shutter(list))
	t.Log(shutter2(list))
}
