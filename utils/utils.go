package utils

import (
	"reflect"

	"golang.org/x/exp/constraints"
)

type Pair[K comparable, V any] struct {
	Key   K
	Value V
}

func PtrToElem[T any](ptr T) (ret T) {
	v := reflect.ValueOf(ptr)
	if v.Kind() == reflect.Invalid {
		return
	}
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	return v.Interface().(T)
}

func ZeroOr[T any](value, defaultValue T) T {
	if v := reflect.ValueOf(value); v.IsZero() {
		return defaultValue
	}
	return value
}

func Ptr[T constraints.Ordered](value T) *T {
	return &value
}
