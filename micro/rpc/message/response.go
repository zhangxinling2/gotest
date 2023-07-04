package message

import (
	"encoding/binary"
)

func (resp *Response) SetHeadLength() {
	resp.HeadLength = 15 + uint32(len(resp.Error))
}
func (resp *Response) SetBodyLength() {
	resp.BodyLength = uint32(len(resp.Data))
}
func EncodeResp(resp *Response) []byte {
	data := make([]byte, resp.HeadLength+resp.BodyLength)
	binary.BigEndian.PutUint32(data[:4], resp.HeadLength)
	binary.BigEndian.PutUint32(data[4:8], resp.BodyLength)
	binary.BigEndian.PutUint32(data[8:12], resp.RequestID)
	data[12] = resp.Version
	data[13] = resp.Compresser
	data[14] = resp.Serializer
	cur := data[15:]
	copy(cur[:len(resp.Error)], resp.Error)
	cur = cur[len(resp.Error):]
	copy(cur, resp.Data)
	return data
}

func DecodeResp(bs []byte) *Response {
	resp := &Response{}
	resp.HeadLength = binary.BigEndian.Uint32(bs[:4])
	resp.BodyLength = binary.BigEndian.Uint32(bs[4:8])
	resp.RequestID = binary.BigEndian.Uint32(bs[8:12])
	resp.Version = bs[12]
	resp.Compresser = bs[13]
	resp.Serializer = bs[14]
	tmp := bs[15:resp.HeadLength]
	if resp.HeadLength > 15 {
		resp.Error = tmp
	}
	if resp.BodyLength != 0 {
		resp.Data = bs[resp.HeadLength:]
	}

	return resp
}
