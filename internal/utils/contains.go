package utils

import "slices"

// Contains 检查切片是否包含目标值
func Contains[T comparable](slice []T, target T) bool {
	return slices.Contains(slice, target)
}
