package message

import (
	"bytes"
	"encoding/binary"
)

const (
	splitter     = '\n'
	pairSplitter = '\r'
)

func (req *Request) SetHeadLength() {
	// 固定部分
	res := 15
	res += len(req.ServiceName)
	// 分隔符
	res++
	res += len(req.MethodName)
	// 分隔符
	res++
	for key, value := range req.Meta {
		res += len(key)
		// 键值对分隔符
		res++
		res += len(value)
		// 分隔符
		res++
	}
	req.HeadLength = uint32(res)
}
func EncodeReq(req *Request) []byte {
	data := make([]byte, req.HeadLength+req.BodyLength)
	binary.BigEndian.PutUint32(data[:4], req.HeadLength)
	binary.BigEndian.PutUint32(data[4:8], req.BodyLength)
	binary.BigEndian.PutUint32(data[8:12], req.RequestID)
	data[12] = req.Version
	data[13] = req.Compresser
	data[14] = req.Serializer
	cur := data[15:]
	copy(cur, req.ServiceName)
	cur = cur[len(req.ServiceName):]
	cur[0] = splitter
	cur = cur[1:]
	copy(cur, req.MethodName)
	cur = cur[len(req.MethodName):]
	cur[0] = splitter
	cur = cur[1:]
	for key, val := range req.Meta {
		copy(cur, key)
		cur[len(key)] = pairSplitter
		cur = cur[len(key)+1:]
		copy(cur, val)
		cur[len(val)] = splitter
		cur = cur[len(val)+1:]
	}
	copy(cur, req.Data)
	return data
}

func DecodeReq(bs []byte) *Request {
	req := &Request{}
	req.HeadLength = binary.BigEndian.Uint32(bs[:4])
	req.BodyLength = binary.BigEndian.Uint32(bs[4:8])
	req.RequestID = binary.BigEndian.Uint32(bs[8:12])
	req.Version = bs[12]
	req.Compresser = bs[13]
	req.Serializer = bs[14]
	tmp := bs[15:req.HeadLength]
	index := bytes.IndexByte(tmp, splitter)
	req.ServiceName = string(tmp[:index])
	tmp = tmp[index+1:]
	index = bytes.IndexByte(tmp, splitter)
	req.MethodName = string(tmp[:index])
	tmp = tmp[index+1:]
	if len(tmp) > 0 {
		// 这个地方不好预估容量，但是大部分都很少，我们把现在能够想到的元数据都算法
		// 也就不超过四个
		metaMap := make(map[string]string, 4)
		index = bytes.IndexByte(tmp, splitter)
		for index != -1 {
			pairIndex := bytes.IndexByte(tmp, pairSplitter)
			metaMap[string(tmp[:pairIndex])] = string(tmp[pairIndex+1 : index])
			tmp = tmp[index+1:]
			index = bytes.IndexByte(tmp, splitter)
		}
		req.Meta = metaMap
	}
	req.Data = bs[req.HeadLength:]
	return req
}
