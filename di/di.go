package di

import (
	"fmt"
	"sync"
)

var (
	mu       sync.RWMutex
	services = make(map[string]any)
)

// Provide save the value
func Provide[T any](value T) {
	ProvideNamed(getName[T](), value)
}

// ProvideNamed save the value for the name
func ProvideNamed(name string, value any) {
	mu.Lock()
	defer mu.Unlock()

	if _, ok := services[name]; ok {
		panic(fmt.Errorf("value already declared %s", name))
	}

	services[name] = value
}

// Resolve get the value, if not exist returns error
func Resolve[T any]() (T, error) {
	return ResolveNamed[T](getName[T]())
}

// ResolveNamed get the value for the name if not exist returns error
func ResolveNamed[T any](name string) (value T, err error) {
	mu.RLock()
	defer mu.RUnlock()

	if v, ok := services[name]; ok {
		return v.(T), nil
	}

	return value, fmt.Errorf("value not declared %s", name)
}

// MustResolve get the value, if not exist create panic
func MustResolve[T any]() T {
	value, err := Resolve[T]()
	if err != nil {
		panic(err)
	}
	return value
}

// MustResolveNamed get the value for the name, if not exist create panic
func MustResolveNamed[T any](name string) T {
	value, err := ResolveNamed[T](name)
	if err != nil {
		panic(err)
	}
	return value
}

// getNamed returns the type name
func getName[T any]() string {
	var t T

	// struct
	name := fmt.Sprintf("%T", t)
	if name != "<nil>" {
		return name
	}

	// interface
	return fmt.Sprintf("%T", new(T))
}
