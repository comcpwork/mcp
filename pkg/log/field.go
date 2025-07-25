package log

import (
	"github.com/cockroachdb/errors"
)

// Field 日志字段
type Field struct {
	Key   string
	Value interface{}
}

// String 创建字符串字段
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int 创建整数字段
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int64 创建64位整数字段
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Float64 创建浮点数字段
func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

// Bool 创建布尔字段
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Any 创建任意类型字段
func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Err 创建错误字段（避免与Error函数冲突）
func Err(err error) Field {
	if err == nil {
		return Field{Key: "error", Value: "<nil>"}
	}

	// 使用cockroachdb/errors来处理错误
	return Field{Key: "error", Value: errors.GetSafeDetails(err).SafeDetails}
}
