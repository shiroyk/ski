package utils

import (
	"reflect"

	"golang.org/x/exp/constraints"
)

// Pair a key and a value pair
type Pair[K comparable, V any] struct {
	Key   K
	Value V
}

// FromPtr returns the value from pointer
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

// ZeroOr if value is zero value returns the defaultValue
func ZeroOr[T comparable](value, defaultValue T) T {
	var zero T
	if zero == value {
		return defaultValue
	}
	return value
}

// EmptyOr if slice is empty returns the defaultValue
func EmptyOr[T any](value, defaultValue []T) []T {
	if len(value) == 0 {
		return defaultValue
	}
	return value
}

// ToPtr returns the value pointer
func ToPtr[T constraints.Ordered](value T) *T {
	return &value
}
