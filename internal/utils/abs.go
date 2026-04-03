package utils

// SignedNumber 有符号数值类型约束
type SignedNumber interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~float32 | ~float64
}

// Abs 返回任何有符号数值类型的绝对值
func Abs[T SignedNumber](x T) T {
	if x < 0 {
		return -x
	}
	return x
}
