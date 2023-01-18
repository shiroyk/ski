package di

import (
	"testing"

	"golang.org/x/exp/maps"
)

type test struct{}

func TestProvide(t *testing.T) {
	t.Parallel()
	defer maps.Clear(services)

	Provide(test{})
	if _, ok := services[getName[test]()]; !ok {
		t.Fatalf("Provide must declared value")
	}
}

func TestProvideNamed(t *testing.T) {
	t.Parallel()
	defer maps.Clear(services)

	ProvideNamed("t", test{})
	if _, ok := services["t"]; !ok {
		t.Fatalf("Provide must declared value")
	}
}

func TestResolve(t *testing.T) {
	t.Parallel()
	defer maps.Clear(services)

	Provide(test{})
	_, err := Resolve[test]()
	if err != nil {
		t.Fatal(err)
	}
}

func TestResolveNamed(t *testing.T) {
	t.Parallel()
	defer maps.Clear(services)

	ProvideNamed("t", test{})
	_, err := ResolveNamed[test]("t")
	if err != nil {
		t.Fatal(err)
	}
}

func TestMustResolve(t *testing.T) {
	t.Parallel()
	defer maps.Clear(services)

	Provide(test{})
	MustResolve[test]()
}

func TestMustResolveNamed(t *testing.T) {
	t.Parallel()
	defer maps.Clear(services)

	ProvideNamed("t", test{})
	MustResolveNamed[test]("t")
}
