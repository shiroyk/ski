package js

import (
	"context"
	"flag"
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	flag.Parse()
	code := m.Run()
	os.Exit(code)
}

func TestUseStrict(t *testing.T) {
	t.Parallel()
	vm := newVM(true)
	_, err := vm.RunString(context.Background(), `eval('a = 1');a`)
	if err != nil {
		if !strings.Contains(err.Error(), "ReferenceError: a is not defined") {
			t.Fatal(err)
		}
	}
}
