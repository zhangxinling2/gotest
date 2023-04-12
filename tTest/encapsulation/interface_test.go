package encapsulation

import (
	"fmt"
	"testing"
)

type Programmer interface {
	WriteHelloWorld() string
}
type GoProgrammer struct {

}
func (g *GoProgrammer) WriteHelloWorld() string{
	return "fmt.Println(\"hello\")"
}
func writeFirstProgram(p Programmer){
	fmt.Printf("%T %v\n",p,p.WriteHelloWorld())
}
func TestClient(t *testing.T){
	var p Programmer
	p=new(GoProgrammer)
	//接口必须是指针
	//p2:=GoProgrammer{}
	//writeFirstProgram(p2)
	t.Log(p.WriteHelloWorld())
}


type People interface {
	Speak(string) string
}

type Student struct{}

func (stu *Student) Speak(think string) (talk string) {
	if think == "sb" {
		talk = "你是个大帅比"
	} else {
		talk = "您好"
	}
	return
}

func TestStudent(t *testing.T) {
	var peo People = &Student{}
	think := "bitch"
	fmt.Println(peo.Speak(think))
}