package errs

import (
	"errors"
	"fmt"
)

var(
	ErrPointOnly=errors.New("orm:只支持指向结构体的一级指针")
	errUnsupportedExpression=errors.New("orm:不支持的表达式类型")
	ErrNoRows = errors.New("orm:没有数据")
	ErrInsertZeroRow=errors.New("orm: 插入 0 行")
)
//func NewUnsupportedExpressionV1(expr any)error{
//	return fmt.Errorf("%w %v",errUnsupportedExpression,expr)
//}
func NewUnsupportedTableReference(table any)error{
	return fmt.Errorf("orm:不支持的TableReferebce类型 %v",table)
}
func NewUnsupportedExpression(expr any)error{
	return fmt.Errorf("orm:不支持的表达式类型 %v",expr)
}
func NewUnknownField(expr any)error{
	return fmt.Errorf("orm: 未知字段 %s",expr)
}
func NewUnknownColumn(expr any)error{
	return fmt.Errorf("orm: 未知列 %s",expr)
}
func NewErrInvalidTagContext(pair string)error{
	return fmt.Errorf("orm: 非法标签 %s",pair)
}
func NewErrUnSupportedAssignable(assign any)error{
	return fmt.Errorf("orm: 不支持的赋值表达式类型 %v",assign)
}
func NewErrFailedToRollbackTx(bizErr error,rbErr error,panicked bool)error{
	return fmt.Errorf("orm: 事务闭包回滚失败，业务错误：%w. 回滚错误%s. 是否panic: %t",bizErr,rbErr,panicked)
}