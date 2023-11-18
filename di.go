package cloudcat

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
)

// di a simple dependencies injection
// Inspired by https://github.com/samber/do

var diServices = new(sync.Map)

type lazyService[T any] struct {
	load     atomic.Bool
	instance T
	initFunc func() (T, error)
}

func (s *lazyService[T]) initOrGet() (instance T, err error) {
	if s.load.Load() {
		return s.instance, nil
	}
	if !s.load.Swap(true) {
		s.instance, err = s.initFunc()
		if err != nil {
			s.load.Store(false)
		}
	}
	return s.instance, err
}

// Provide save the value and return is it saved
func Provide[T any](value T) bool {
	return ProvideNamed(getName[T](), value)
}

// ProvideLazy save the lazy init value and return is it saved
func ProvideLazy[T any](initFunc func() (T, error)) bool {
	return ProvideNamed(getName[T](), &lazyService[T]{initFunc: initFunc})
}

// ProvideNamed save the value for the name and return is it saved
func ProvideNamed(name string, value any) (ok bool) {
	_, ok = diServices.LoadOrStore(name, value)
	return !ok
}

// Override save the value and return is it override
func Override[T any](value T) bool {
	return OverrideNamed(getName[T](), value)
}

// OverrideLazy save the value for the name and return is it override
func OverrideLazy[T any](initFunc func() (T, error)) bool {
	return OverrideNamed(getName[T](), &lazyService[T]{initFunc: initFunc})
}

// OverrideNamed save the value for the name and return is it override
func OverrideNamed(name string, value any) (ok bool) {
	_, ok = diServices.Swap(name, value)
	return ok
}

// Resolve get the value, if not exist returns error
func Resolve[T any]() (T, error) {
	return ResolveNamed[T](getName[T]())
}

// ResolveLazy get the value lazy once, if not exist returns error
func ResolveLazy[T any]() func() (T, error) {
	var (
		once  sync.Once
		value T
		err   error
	)
	g := func() {
		value, err = Resolve[T]()
	}
	return func() (T, error) {
		once.Do(g)
		return value, err
	}
}

// ResolveNamed get the value for the name if not exist returns error
func ResolveNamed[T any](name string) (value T, err error) {
	if v, exists := diServices.Load(name); exists {
		switch target := v.(type) {
		case *lazyService[T]:
			return target.initOrGet()
		case T:
			return target, nil
		}
		return value, fmt.Errorf("%T type assertion to %T error", v, value)
	}

	return value, fmt.Errorf("%s not declared", name)
}

// MustResolve get the value, if not exist create panic
func MustResolve[T any]() T {
	value, err := Resolve[T]()
	if err != nil {
		panic(err)
	}
	return value
}

// MustResolveLazy get the value lazy once, if not exist create panic
func MustResolveLazy[T any]() func() T {
	g := ResolveLazy[T]()
	return func() T {
		value, err := g()
		if err != nil {
			panic(err)
		}
		return value
	}
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
