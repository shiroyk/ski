package di

import (
	"testing"
)

func TestProvide(t *testing.T) {
	t.Parallel()

	type test1 interface{}
	Provide(new(test1))
	_, err := Resolve[test1]()
	if err != nil {
		t.Fatal(err)
	}
}

func TestOverride(t *testing.T) {
	t.Parallel()

	type test2 interface{}
	Provide(new(test2))
	Override(new(test2))
	OverrideNamed("*di.test2", new(test2))
}

func TestResolve(t *testing.T) {
	t.Parallel()

	type test3 struct{}
	Provide(test3{})
	_, err := Resolve[test3]()
	if err != nil {
		t.Fatal(err)
	}
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
