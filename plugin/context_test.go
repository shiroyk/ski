package plugin

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestContext(t *testing.T) {
	t.Parallel()
	url := "https://example.com/some/path?offset=1"
	baseURL := "https://example.com"
	logger := slog.Default().With(slog.String("source", "ctx"))
	ctx := NewContext(ContextOptions{
		URL:     url,
		Logger:  logger,
		Timeout: time.Minute,
		Values: map[any]any{
			"key1": "value1",
		},
	})
	defer ctx.Cancel()

	assert.NotNil(t, ctx.Logger())
	assert.Equal(t, ctx.Logger(), logger)
	assert.Equal(t, ctx.URL(), url)
	assert.Equal(t, ctx.BaseURL(), baseURL)
	assert.Equal(t, ctx.Value("key1"), "value1")
	assert.Nil(t, ctx.Value("notExists"))

	if _, ok := ctx.Deadline(); !ok {
		t.Error("deadline not set")
	}
	key := "test"
	value := "1"
	ctx.SetValue(key, value)
	if v, ok := ctx.GetValue(key); ok {
		assert.Equalf(t, v, value, "want %v, got %v", value, v)
	}
	if v := ctx.Value(key); v != value {
		t.Errorf("want %v, got %v", value, v)
	}
	ctx.ClearValue()
	assert.Nil(t, ctx.Value(key), "values should be nil")

	ctx.Cancel()
	assert.ErrorIs(t, ctx.Err(), context.Canceled)

	<-ctx.Done()

	ctx1 := NewContext(ContextOptions{Timeout: time.Nanosecond})
	<-ctx1.Done()
	assert.ErrorIs(t, ctx1.Err(), context.DeadlineExceeded)
}

func TestParentContext(t *testing.T) {
	t.Parallel()
	type k string
	key := k("parentKey")
	value := "foo"
	valueCtx := context.WithValue(context.Background(), key, value)
	parent, cancel := context.WithTimeout(valueCtx, time.Minute)

	ctx := NewContext(ContextOptions{Parent: parent})
	assert.Equal(t, value, ctx.Value(key))
	cancel()

	time.Sleep(time.Millisecond)

	assert.ErrorIs(t, ctx.Err(), context.Canceled)
}
