package parser

import (
	"testing"

	"github.com/shiroyk/cloudcat/plugin"
)

type testParser struct{}

func (t *testParser) GetString(*plugin.Context, any, string) (string, error) {
	return "", nil
}

func (t *testParser) GetStrings(*plugin.Context, any, string) ([]string, error) {
	return nil, nil
}

func (t *testParser) GetElement(*plugin.Context, any, string) (string, error) {
	return "", nil
}

func (t *testParser) GetElements(*plugin.Context, any, string) ([]string, error) {
	return nil, nil
}

func TestRegister(t *testing.T) {
	t.Parallel()
	if _, ok := GetParser("test"); !ok {
		Register("test", new(testParser))
	}
	if _, ok := GetParser("test"); !ok {
		t.Fatal("unable get parser")
	}
}
