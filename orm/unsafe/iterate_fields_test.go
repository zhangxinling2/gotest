package unsafe

import "testing"

type User struct {
	//0
	Name string
	//16
	Age int32
	//24
	Alias []string
	//48
	Address string
}
type UserV1 struct {
	//0
	Name string
	//16
	Age int32
	//20
	AgeV1 int32
	//24
	Alias []string
	//48
	Address string
}
func TestPrintFieldOffset(t *testing.T) {

	tests := []struct {
		name string
		entity any
	}{
		{
			name: "user",
			entity: User{},
		},
		{
			name: "user",
			entity: UserV1{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			PrintFieldOffset(tc.entity)
		})
	}
}
