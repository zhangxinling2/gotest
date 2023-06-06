package net

import (
	"encoding/binary"
	"net"
	"time"
)

type Client struct {
	network string
	addr    string
}

func (c *Client) Start() error {
	conn, err := net.DialTimeout(c.network, c.addr, time.Second*3)
	if err != nil {
		return err
	}
	defer func() {
		conn.Close()
	}()

	return nil
}
func (c *Client) Send(data []byte) (string, error) {
	conn, err := net.DialTimeout(c.network, c.addr, time.Second*3)
	if err != nil {
		return "", err
	}
	defer func() {
		conn.Close()
	}()
	reqData := make([]byte, len(data)+numOfLengthByte)
	binary.BigEndian.PutUint64(reqData[:numOfLengthByte], uint64(len(data)))
	copy(reqData[numOfLengthByte:], data)
	_, err = conn.Write(reqData)
	if err != nil {
		return "", err
	}
	//读长度
	lenData := make([]byte, numOfLengthByte)
	_, err = conn.Read(lenData)
	if err != nil {
		return "", err
	}
	length := binary.BigEndian.Uint64(lenData)
	//读数据
	resData := make([]byte, length)
	_, err = conn.Read(resData)
	if err != nil {
		return "", err
	}
	return string(resData), nil
}
