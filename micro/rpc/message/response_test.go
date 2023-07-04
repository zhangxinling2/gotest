package message

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEncodeResp(t *testing.T) {
	testCases := []struct {
		name string
		resp *Response
	}{
		{
			name: "no Error",
			resp: &Response{
				BodyLength: 0,
				RequestID:  123,
				Version:    12,
				Compresser: 1,
				Serializer: 1,
			},
		},
		{
			name: "with Error",
			resp: &Response{
				BodyLength: 0,
				RequestID:  123,
				Version:    15,
				Compresser: 1,
				Serializer: 1,
				Error:      []byte("this is error"),
				Data:       nil,
			},
		},
		{
			name: "With Data",
			resp: &Response{
				RequestID:  123,
				Version:    12,
				Compresser: 25,
				Serializer: 17,
				Data:       []byte("hello, world"),
			},
		},
		{
			name: "with All",
			resp: &Response{
				RequestID:  123,
				Version:    12,
				Compresser: 25,
				Serializer: 17,
				Error:      []byte("this is error"),
				Data:       []byte("hello, world"),
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.resp.SetHeadLength()
			tc.resp.SetBodyLength()
			data := EncodeResp(tc.resp)
			res := DecodeResp(data)
			assert.Equal(t, tc.resp, res)
		})
	}
}
