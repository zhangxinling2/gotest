package cache

import (
	"context"
	"gotest/orm/reflect/types"
	"testing"
)

func TestReadThroughCache_Get(t *testing.T) {
	var c1 ReadThroughCache
	var c2 ReadThroughCacheV1[types.User]
	val, _ := c1.Get(context.Background(), "user_1")
	t.Log(val.(types.User))
	val, _ = c2.Get(context.Background(), "user_1")
	t.Log(val)
}
