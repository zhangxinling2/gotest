package rpc

import (
	"encoding/binary"
	"net"
)

func ReadMsg(conn net.Conn) ([]byte, error) {
	//读长度
	lenData := make([]byte, numOfLengthByte)
	_, err := conn.Read(lenData)
	if err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint64(lenData)
	data := make([]byte, length)
	_, err = conn.Read(data)
	if err != nil {
		return nil, err
	}
	return data, nil
}
func EncodeMsg(data []byte) ([]byte, error) {
	respData := make([]byte, len(data)+numOfLengthByte)
	binary.BigEndian.PutUint64(respData[:numOfLengthByte], uint64(len(data)))
	copy(respData[numOfLengthByte:], data)
	return respData, nil
}
