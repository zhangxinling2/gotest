package orm

import (
	"context"
)

// Querier 用于 SELECT 语句
type Querier[T any] interface {
	Get(ctx context.Context)(*T,error)
	GetMulti(ctx context.Context)([]*T,error)
}

// Executor 用于 INSERT , DELETE 和 UPDATE
type Executor interface {
	Exec(ctx context.Context)Result
}
type QueryBuilder interface {
	Build()(*Query,error)
}

type Query struct {
	SQL string
	Args []any
}


