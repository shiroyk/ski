package cloudcat

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProvide(t *testing.T) {
	t.Parallel()

	type test1 interface{}
	Provide(new(test1))
	_, err := Resolve[test1]()
	assert.NoError(t, err)
}

func TestProvideLazy(t *testing.T) {
	t.Parallel()

	times := 0
	type test2 interface{}
	ProvideLazy(func() (test2, error) {
		if times == 0 {
			times++
			return nil, errors.New("something")
		}
		return new(test2), nil //nolint:nilnil
	})
	v, _ := Resolve[test2]()
	assert.Nil(t, v)
	v, _ = Resolve[test2]()
	assert.NotNil(t, v)
}

func TestResolve(t *testing.T) {
	t.Parallel()

	type test3 struct{}
	Provide(test3{})
	_, err := Resolve[test3]()
	assert.NoError(t, err)
}

func TestResolveLazy(t *testing.T) {
	t.Parallel()

	type test7 struct{}
	Provide(test7{})
	f := ResolveLazy[test7]()
	_, err := f()
	assert.NoError(t, err)

	times := 0
	type test8 interface{}
	ProvideLazy(func() (test8, error) {
		if times == 0 {
			times++
			return nil, errors.New("something")
		}
		return new(test8), nil //nolint:nilnil
	})
	f2 := ResolveLazy[test8]()
	v, _ := f2()
	assert.Nil(t, v)
	v, _ = f2()
	assert.Nil(t, v)
}

func TestMustResolve(t *testing.T) {
	t.Parallel()

	type test4 interface{}
	Provide(new(test4))
	MustResolve[test4]()
}

func TestMustResolveLazy(t *testing.T) {
	t.Parallel()
	defer func() {
		r := recover()
		assert.NotNil(t, r)
		assert.ErrorContains(t, r.(error), "test10 not declared")
	}()

	type test9 interface{}
	Provide(new(test9))
	assert.NotNil(t, MustResolveLazy[test9]()())
	type test10 interface{}
	assert.NotNil(t, MustResolveLazy[test10]()())
}

func TestMustResolveNamed(t *testing.T) {
	t.Parallel()

	type test5 struct{}
	assert.True(t, ProvideNamed("named1", test5{}))
	assert.False(t, ProvideNamed("named1", test5{}))
	MustResolveNamed[test5]("named1")
}

func TestOverride(t *testing.T) {
	t.Parallel()

	type test6 struct{}
	assert.False(t, Override(test6{}))
	assert.True(t, Override(test6{}))
	MustResolve[test6]()
}
