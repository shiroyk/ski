package parsers

import (
	"testing"
	"time"

	"golang.org/x/exp/slog"
)

func TestContext(t *testing.T) {
	t.Parallel()
	ctx := NewContext(Options{
		URL:     "http://localhost",
		Logger:  slog.Default(),
		Timeout: time.Second,
	})
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
	if ctx.Logger() == nil {
		t.Error("context logger not set")
	}
	if ctx.BaseURL() == "" {
		t.Error("context baseURL not set")
	}
	if ctx.RedirectURL() == "" {
		t.Error("context redirectURL not set")
	}
	ctx.Cancel()
	if ctx.Err() == nil {
		t.Error("context cancel error must not be none")
	}
	<-ctx.Done()

	ctx1 := NewContext(Options{Timeout: time.Nanosecond})
	<-ctx1.Done()

	ctx2 := NewContext(Options{Timeout: time.Millisecond})
	<-ctx2.Done()
}
