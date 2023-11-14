package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"sync"
	"time"
)

const (
	// DefaultTimeout The Context default timeout one minute.
	DefaultTimeout = time.Minute
)

// Context The Parser context
type Context struct {
	context.Context // set to non-nil by the first cancel call
	cancelFunc      context.CancelFunc
	opt             ContextOptions
	value           *sync.Map
	baseURL, url    string
}

// ContextOptions The Context options
type ContextOptions struct {
	Parent  context.Context
	Timeout time.Duration
	Logger  *slog.Logger
	URL     string
}

// NewContext creates a new Context with ContextOptions
func NewContext(opt ContextOptions) *Context {
	if opt.Logger == nil {
		opt.Logger = slog.Default()
	}
	ctx := &Context{
		value: new(sync.Map),
		opt:   opt,
	}
	if opt.URL != "" {
		ctx.url = opt.URL
		if u, err := url.Parse(ctx.url); err == nil {
			ctx.baseURL = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
		}
	}

	parent := opt.Parent
	if parent == nil {
		parent = context.Background()
	}
	timeout := DefaultTimeout
	if opt.Timeout > 0 {
		timeout = opt.Timeout
	}
	ctx.Context, ctx.cancelFunc = context.WithTimeout(parent, timeout)
	return ctx
}

// Cancel this context releases resources associated with it, so code should
// call cancel as soon as the operations running in this Context complete.
func (c *Context) Cancel() {
	if c.cancelFunc != nil {
		c.cancelFunc()
	}
}

// ClearValue clean all values
func (c *Context) ClearValue() {
	c.value = new(sync.Map)
}

// Value returns the value associated with this context for key, or nil
// if no value is associated with key. Successive calls to Value with
// the same key returns the same result.
func (c *Context) Value(key any) any {
	if v, ok := c.value.Load(key); ok {
		return v
	}
	if c.opt.Parent != nil {
		return c.opt.Parent.Value(key)
	}
	return nil
}

// GetValue returns the value associated with this context for key, or nil
// if no value is associated with key. Successive calls to Value with
// the same key returns the same result.
func (c *Context) GetValue(key any) (any, bool) {
	return c.value.Load(key)
}

// SetValue value associated with key is val.
func (c *Context) SetValue(key any, value any) {
	c.value.Store(key, value)
}

// Logger returns the logger, if ContextOptions.Logger is nil return slog.Default
func (c *Context) Logger() *slog.Logger {
	return c.opt.Logger
}

// BaseURL returns the baseURL string
func (c *Context) BaseURL() string {
	return c.baseURL
}

// URL returns the absolute URL string
func (c *Context) URL() string {
	return c.url
}
