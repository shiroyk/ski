// Package di a simple dependencies injection
package di

import (
	"fmt"
	"reflect"
	"sync"
)

var services = new(sync.Map)

// Provide save the value
func Provide[T any](value T) {
	ProvideNamed(getName[T](), value)
}

// ProvideNamed save the value for the name
func ProvideNamed(name string, value any) {
	if _, ok := services.LoadOrStore(name, value); ok {
		panic(fmt.Errorf("value already declared %s", name))
	}
}

// Override override the value
func Override[T any](value T) {
	services.Store(getName[T](), value)
}

// OverrideNamed override the value
func OverrideNamed(name string, value any) {
	services.Store(name, value)
}

// Resolve get the value, if not exist returns error
func Resolve[T any]() (T, error) {
	return ResolveNamed[T](getName[T]())
}

// ResolveNamed get the value for the name if not exist returns error
func ResolveNamed[T any](name string) (value T, err error) {
	if v, ok := services.Load(name); ok {
		if t, ok := v.(T); ok {
			return t, nil
		}
		return value, fmt.Errorf("value type asserted failed")
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

// getName returns the type name
func getName[T any]() string {
	var v T

	// struct
	if t := reflect.TypeOf(v); t != nil {
		return t.String()
	}

	// interface
	return reflect.TypeOf(new(T)).String()
}
