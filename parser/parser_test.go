package parser

import "testing"

type testParser struct{}

func (t *testParser) GetString(*Context, any, string) (string, error) {
	return "", nil
}

func (t *testParser) GetStrings(*Context, any, string) ([]string, error) {
	return nil, nil
}

func (t *testParser) GetElement(*Context, any, string) (string, error) {
	return "", nil
}

func (t *testParser) GetElements(*Context, any, string) ([]string, error) {
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
