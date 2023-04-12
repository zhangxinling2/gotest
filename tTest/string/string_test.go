package string

import "testing"

func TestString(t *testing.T) {
	var s string
	t.Log(s)
	s="hello"
	t.Log(len(s))
	s="\xE4\xB8\xA5"//可以存任意二进制数据
	t.Log(s)
	s="中"
	t.Log(len(s))
	c:=[]rune(s)
	t.Log(len(c))
	t.Logf("中 Unicode %x",c[0])
	t.Logf("中 Unicode %[1]c",c[0])
	t.Logf("中 Unicode %[1]d",c[0])
	t.Logf("中 utf-8 %x",s)
}
