package di

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProvide(t *testing.T) {
	t.Parallel()

	type test1 interface{}
	Provide(new(test1), false)
	_, err := Resolve[test1]()
	if err != nil {
		t.Fatal(err)
	}
}

func TestProvideLazy(t *testing.T) {
	t.Parallel()

	type test2 interface{}
	ProvideLazy(func() (test2, error) {
		return nil, nil
	}, false)
	assert.Nil(t, MustResolve[test2]())
	Provide(new(test2), true)
	assert.NotNil(t, MustResolve[test2]())
}

func TestResolve(t *testing.T) {
	t.Parallel()

	type test3 struct{}
	Provide(test3{}, false)
	_, err := Resolve[test3]()
	if err != nil {
		t.Fatal(err)
	}
}

func TestMustResolve(t *testing.T) {
	t.Parallel()

	type test4 interface{}
	Provide(new(test4), false)
	MustResolve[test4]()
}

func TestMustResolveNamed(t *testing.T) {
	t.Parallel()

	type test5 struct{}
	ProvideNamed("t", test5{}, false)
	MustResolveNamed[test5]("t")
}
