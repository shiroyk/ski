package core

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

func TestMustResolve(t *testing.T) {
	t.Parallel()

	type test4 interface{}
	Provide(new(test4))
	MustResolve[test4]()
}

func TestMustResolveNamed(t *testing.T) {
	t.Parallel()

	type test5 struct{}
	ProvideNamed("t", test5{})
	MustResolveNamed[test5]("t")
}

func TestOverride(t *testing.T) {
	t.Parallel()

	type test6 int
	v1 := test6(1)
	assert.True(t, Provide(v1))
	assert.Equal(t, v1, MustResolve[test6]())
	v2 := test6(2)
	assert.True(t, Override(v2))
	assert.Equal(t, v2, MustResolve[test6]())
}
