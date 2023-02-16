package utils

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Pair a key and a value pair
type Pair[K comparable, V any] struct {
	Key   K
	Value V
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

// ReadYaml read the YAML file and convert it to T
func ReadYaml[T any](path string) (t *T, err error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return
	}

	t = new(T)
	err = yaml.Unmarshal(bytes, t)
	if err != nil {
		return
	}

	return
}
