package utils

import (
	"reflect"

	"golang.org/x/exp/constraints"
)

type Pair[K comparable, V any] struct {
	Key   K
	Value V
}

func FromPtr[T any](ptr T) (ret T) {
	v := reflect.ValueOf(ptr)
	if v.Kind() == reflect.Invalid {
		return
	}
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	return v.Interface().(T)
}

func ZeroOr[T comparable](value, defaultValue T) T {
	var zero T
	if zero == value {
		return defaultValue
	}
	return value
}

func EmptyOr[T any](value, defaultValue []T) []T {
	if len(value) == 0 {
		return defaultValue
	}
	return value
}

func ToPtr[T constraints.Ordered](value T) *T {
	return &value
}
