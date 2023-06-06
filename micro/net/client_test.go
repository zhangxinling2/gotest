package net

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestClient_Send(t *testing.T) {
	go func() {
		s := &Server{}
		err := s.Start("tcp", ":8081")
		t.Log(err)
	}()
	time.Sleep(time.Second * 3)
	c := &Client{
		network: "tcp",
		addr:    ":8081",
	}
	res, err := c.Send([]byte("hello"))
	require.NoError(t, err)
	t.Log(res)
	assert.Equal(t, "hello", res)
}
