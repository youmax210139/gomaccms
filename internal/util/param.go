package util

import (
	"reflect"
)

/*
	简单参数处理
*/

// IsEmpty 判断各种基本类型是否为空
func IsEmpty(target any) bool {
	switch t := target.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return target == reflect.Zero(reflect.TypeOf(target)).Interface()
	case string:
		return t == ""
	case bool:
		return t
	default:
		return false
	}
}
