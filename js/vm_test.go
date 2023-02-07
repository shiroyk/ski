package js

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestVM(t *testing.T) {
	t.Parallel()
	vm := newVM(false)

	testCases := []struct {
		script, want string
		isErr        bool
	}{
		{"return 1", "1", true},
		{"2", "2", false},
		{"a = 1; a + 2", "3", false},
		{"(() => 4)()", "4", false},
		{"[5]", "5", false},
		{"a = {'key':6}; a", "[object Object]", false},
		{"JSON.stringify({'key':7})", `{"key":7}`, false},
		{"JSON.stringify([8])", `[8]`, false},
		{"(async () => 9)()", `[object Promise]`, false},
	}

	for _, c := range testCases {
		t.Run(c.script, func(t *testing.T) {
			t.Parallel()
			v, err := vm.RunString(context.Background(), c.script)
			if err != nil {
				if c.isErr {
					return
				}
				t.Fatal(err)
			}
			if v.String() != c.want {
				t.Errorf("want %v, got %v", c.want, v.String())
			}
		})
	}
}

func TestTimeout(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := newVM(false).RunString(ctx, `while(true){}`)
	if !strings.Contains(err.Error(), context.DeadlineExceeded.Error()) {
		t.Fatal(err)
	}
}
