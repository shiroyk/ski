package js

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsole(t *testing.T) {
	t.Parallel()
	data := new(bytes.Buffer)
	decoder := json.NewDecoder(data)
	vm := NewVM()
	ctx := WithLogger(context.Background(), slog.New(slog.NewJSONHandler(data, nil)))

	for i, c := range []struct {
		str, want string
	}{
		{`console.info(true);`, "true"},
		{`console.info(undefined, null, 114);`, "undefined null 114"},
		{`console.info("hello %s", "ski");`, "hello ski"},
		{`console.warn("json %j", {'foo': 'bar'});`, `json {"foo":"bar"}`},
		{`console.log({'foo': 'bar'});`, `{"foo":"bar"}`},
		{`console.error({'foo': 123}, {'bar': 456});`, `{"foo":123} {"bar":456}`},
		{`console.error('test:', new Error('ciallo'));`, "test: Error: ciallo\n\tat <eval>:1:24(5)\n"},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			data.Reset()
			_, err := vm.RunString(ctx, c.str)
			require.NoError(t, err)
			var output map[string]string
			err = decoder.Decode(&output)
			require.NoError(t, err)
			assert.Equal(t, c.want, output["msg"])
		})
	}
}
