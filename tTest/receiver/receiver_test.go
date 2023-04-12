package receiver

import "testing"

type T struct {
	a int
}

func (t T) M1() {
	t.a = 10
}

func (t *T) M2() {
	t.a = 11
}
func TestReceiver(t *testing.T) {
	var t2 T
	println(t2.a) // 0

	t2.M1()
	t2.M2()//go提供的语法糖，自动转换为(&t2)
	println(t2.a) // 0

	p := &t2
	p.M2()
	p.M1()
	println(t2.a) // 11
}

