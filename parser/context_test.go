package parser

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slog"
)

func TestContext(t *testing.T) {
	t.Parallel()
	ctx := NewContext(Options{
		URL:     "http://localhost",
		Logger:  slog.Default(),
		Timeout: time.Second,
	})
	defer ctx.Cancel()
	if _, ok := ctx.Deadline(); !ok {
		t.Error("deadline not set")
	}
	key := "test"
	value := "1"
	ctx.SetValue(key, value)
	if v, ok := ctx.GetValue(key); ok {
		if v != value {
			t.Errorf("want %v, got %v", value, v)
		}
	}
	if v := ctx.Value(key); v != value {
		t.Errorf("want %v, got %v", value, v)
	}
	ctx.ClearValue()
	if _, ok := ctx.GetValue(key); ok {
		t.Error("value not clear")
	}

	assert.NotNil(t, ctx.Logger())
	assert.NotEmpty(t, ctx.BaseURL())
	assert.NotEmpty(t, ctx.URL())

	ctx.Cancel()
	assert.ErrorIs(t, ctx.Err(), context.Canceled)

	<-ctx.Done()

	ctx1 := NewContext(Options{Timeout: time.Nanosecond})
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

	ctx := NewContext(Options{Parent: parent})
	assert.Equal(t, value, ctx.Value(key))
	cancel()

	time.Sleep(time.Millisecond)

	assert.ErrorIs(t, ctx.Err(), context.Canceled)
}
