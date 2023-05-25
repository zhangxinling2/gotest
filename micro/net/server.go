package net

import (
	"encoding/binary"
	"net"
)

const (
	numOfLengthByte = 8
)

type Server struct {
}

func (s *Server) Start(network, addr string) error {
	listener, err := net.Listen(network, addr)
	if err != nil {
		return err
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go func() {
			if err = handleConn(conn); err != nil {
				conn.Close()
			}
		}()
	}
}

func handleConn(conn net.Conn) error {
	for {
		//读长度
		lenData := make([]byte, numOfLengthByte)
		_, err := conn.Read(lenData)
		if err != nil {
			return err
		}
		length := binary.BigEndian.Uint64(lenData)
		data := make([]byte, length)
		_, err = conn.Read(data)
		if err != nil {
			return err
		}
		res := handleMsg(data)
		//写数据
		respData := make([]byte, len(res)+numOfLengthByte)
		binary.BigEndian.PutUint64(respData[:numOfLengthByte], uint64(len(data)))
		copy(respData[numOfLengthByte:], res)
		_, err = conn.Write(respData)
		if err != nil {
			return err
		}
	}
	return nil
}

func handleMsg(data []byte) string {
	return "hello"
}
