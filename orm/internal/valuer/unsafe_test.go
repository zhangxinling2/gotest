package valuer

import (
	"testing"
)


func Test_unsafe_Value_SetColumns(t *testing.T) {
	testSetColumns(t,NewUnsafeValue)
}


