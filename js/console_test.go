package js

import (
	"bytes"
	"context"
	"log/slog"
	"strconv"
	"testing"

	"github.com/shiroyk/ski"
	"github.com/stretchr/testify/assert"
)

func TestConsole(t *testing.T) {
	t.Parallel()
	data := new(bytes.Buffer)
	vm := NewVM()
	ctx := ski.WithLogger(context.Background(), slog.New(slog.NewTextHandler(data, nil)))

	for i, c := range []struct {
		str, want string
	}{
		{`console.log("hello %s", "ski");`, "hello ski"},
		{`console.log("json %j", {'foo': 'bar'});`, `json {\"foo\":\"bar\"}`},
		{`console.log({'foo': 'bar'});`, `{\"foo\":\"bar\"}`},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			data.Reset()
			vm.Run(ctx, func() {
				_, err := vm.Runtime().RunString(c.str)
				if assert.NoError(t, err) {
					assert.Contains(t, data.String(), c.want)
				}
			})
		})
	}
}
