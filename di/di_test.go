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

func TestResolve(t *testing.T) {
	t.Parallel()

	type test2 struct{}
	Provide(test2{})
	_, err := Resolve[test2]()
	if err != nil {
		t.Fatal(err)
	}
}

func TestMustResolve(t *testing.T) {
	t.Parallel()

	type test3 interface{}
	Provide(new(test3))
	MustResolve[test3]()
}

func TestMustResolveNamed(t *testing.T) {
	t.Parallel()

	type test4 struct{}
	ProvideNamed("t", test4{})
	MustResolveNamed[test4]("t")
}
