package ski

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewContext(t *testing.T) {
	timeout, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	ctx := NewContext(timeout, nil)
	assert.Nil(t, ctx.Value("key1"))

	ctx.SetValue("key1", "value1")
	assert.Equal(t, "value1", ctx.Value("key1"))

	var key2 byte
	ctx.SetValue(&key2, "value2")
	assert.Equal(t, "value2", ctx.Value(&key2))

	WithValue(context.WithValue(ctx, "key3", "value3"), "key4", "value4")

	assert.Equal(t, "value4", ctx.Value("key4"))
}
