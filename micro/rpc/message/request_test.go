package message

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEncodeReq(t *testing.T) {
	testCases := []struct {
		name string
		req  *Request
	}{
		{
			name: "no meta",
			req: &Request{
				BodyLength:  0,
				RequestID:   123,
				Version:     12,
				Compresser:  1,
				Serializer:  1,
				ServiceName: "user-service",
				MethodName:  "GetById",
				Meta:        nil,
				Data:        nil,
			},
		},
		{
			name: "with meta",
			req: &Request{
				BodyLength:  0,
				RequestID:   123,
				Version:     15,
				Compresser:  1,
				Serializer:  1,
				ServiceName: "user-service",
				MethodName:  "GetById",
				Meta: map[string]string{
					"trace-id": "123",
					"a/b":      "b",
					"shadow":   "true"},
				Data: nil,
			},
		},
		{
			name: "With Data",
			req: &Request{
				RequestID:   123,
				Version:     12,
				Compresser:  25,
				Serializer:  17,
				ServiceName: "user-service",
				MethodName:  "GetById",
				Data:        []byte("hello, world"),
			},
		},
		{
			name: "with All",
			req: &Request{
				RequestID:   123,
				Version:     12,
				Compresser:  25,
				Serializer:  17,
				ServiceName: "user-service",
				MethodName:  "GetById",
				Meta: map[string]string{
					"trace-id": "123",
					"a/b":      "",
					"shadow":   "true",
				},
				Data: []byte("hello, world"),
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.req.SetHeadLength()
			tc.req.BodyLength = uint32(len(tc.req.Data))
			data := EncodeReq(tc.req)
			res := DecodeReq(data)
			assert.Equal(t, tc.req, res)
		})
	}
}
