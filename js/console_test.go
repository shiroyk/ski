package js

import (
	"bytes"
	"context"
	"log/slog"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsole(t *testing.T) {
	t.Parallel()
	data := new(bytes.Buffer)
	vm := NewVM()
	ctx := WithLogger(context.Background(), slog.New(slog.NewTextHandler(data, nil)))

	for i, c := range []struct {
		str, want string
	}{
		{`console.info("hello %s", "ski");`, "hello ski"},
		{`console.warn("json %j", {'foo': 'bar'});`, `json {\"foo\":\"bar\"}`},
		{`console.log({'foo': 'bar'});`, `{\"foo\":\"bar\"}`},
		{`console.error({'foo': 123}, {'bar': 456});`, `{\"foo\":123} {\"bar\":456}`},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			data.Reset()
			_, err := vm.RunString(ctx, c.str)
			require.NoError(t, err)
			assert.Contains(t, data.String(), c.want)
		})
	}
}
