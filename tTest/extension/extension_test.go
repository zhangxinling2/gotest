package extension

import (
	"fmt"
	"testing"
)

type Pet struct {

}

func (p *Pet) Speak(){
	fmt.Print("...")
}

func (p *Pet) SpeakTo(host string)  {
	p.Speak()
	fmt.Println(" ",host)
}

type Dog struct {
	Pet
}

func (d *Dog) Speak(){
	fmt.Println("waring")
}

func (d *Dog) SpeakTo(host string)  {
	d.Speak()
	d.Pet.SpeakTo(host)
}

func TestDog(t *testing.T) {
	dog2:=Dog{}
	dog2.Speak()
	dog:=new(Dog)
	dog.SpeakTo("chao")
}