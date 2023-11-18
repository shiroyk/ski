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
	context.Context                 // set to non-nil by the first cancel call
	parent          context.Context // the parent context
	cancelFunc      context.CancelFunc
	logger          *slog.Logger
	value           *sync.Map
	baseURL, url    string
}

// ContextOptions The Context options
type ContextOptions struct {
	Parent  context.Context // the parent context
	Timeout time.Duration   // the context timeout, default DefaultTimeout.
	Logger  *slog.Logger    // the context logger, default slog.Default if nil.
	Values  map[any]any     // the values
	URL     string          // the analyzer URL
}

// NewContext creates a new Context with ContextOptions
func NewContext(opt ContextOptions) *Context {
	ctx := &Context{
		value:  new(sync.Map),
		logger: opt.Logger,
		parent: opt.Parent,
	}
	if ctx.logger == nil {
		ctx.logger = slog.Default()
	}
	if ctx.parent == nil {
		ctx.parent = context.Background()
	}
	for k, v := range opt.Values {
		ctx.value.Store(k, v)
	}
	if opt.URL != "" {
		ctx.url = opt.URL
		if u, err := url.Parse(ctx.url); err == nil {
			ctx.baseURL = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
		}
	}

	timeout := DefaultTimeout
	if opt.Timeout > 0 {
		timeout = opt.Timeout
	}
	ctx.Context, ctx.cancelFunc = context.WithTimeout(ctx.parent, timeout)
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
	return c.parent.Value(key)
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
	return c.logger
}

// BaseURL returns the baseURL string
func (c *Context) BaseURL() string {
	return c.baseURL
}

// URL returns the absolute URL string
func (c *Context) URL() string {
	return c.url
}
