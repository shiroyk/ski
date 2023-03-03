package parser

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shiroyk/cloudcat/lib/utils"
	"golang.org/x/exp/slog"
)

const (
	// DefaultTimeout The Context default timeout
	DefaultTimeout = time.Minute
)

// Context The Parser context
type Context struct {
	mu           sync.Mutex  // protects following fields
	timer        *time.Timer // Under Context.mu.
	deadline     time.Time
	done         atomic.Value // of chan struct{}, created lazily, closed by first cancel call
	err          error        // set to non-nil by the first cancel call
	cancelFunc   context.CancelFunc
	opt          Options
	value        *sync.Map
	baseURL, url string
}

// Options The Context options
type Options struct {
	Parent  context.Context
	Timeout time.Duration
	Logger  *slog.Logger
	URL     string
}

// NewContext creates a new Context with Options
func NewContext(opt Options) *Context {
	d := time.Now().Add(utils.ZeroOr(opt.Timeout, DefaultTimeout))
	if opt.Logger == nil {
		opt.Logger = slog.Default()
	}
	c := &Context{
		deadline: d,
		value:    new(sync.Map),
		opt:      opt,
	}
	propagateCancel(opt.Parent, c)
	dur := time.Until(d)
	if dur <= 0 {
		c.cancel(context.DeadlineExceeded) // deadline has already passed
		c.cancelFunc = func() { c.cancel(context.Canceled) }
		return c
	}
	if opt.URL != "" {
		c.url = opt.URL
		if u, err := url.Parse(c.url); err == nil {
			c.baseURL = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err == nil {
		c.timer = time.AfterFunc(dur, func() {
			c.cancel(context.DeadlineExceeded)
		})
	}
	c.cancelFunc = func() { c.cancel(context.Canceled) }
	return c
}

// closedC is a reusable-closed channel.
var closedC = make(chan struct{})

func init() {
	close(closedC)
}

// cancel closes c.done
func (c *Context) cancel(err error) {
	if err == nil {
		panic("context: internal error: missing cancel error")
	}
	c.mu.Lock()
	if c.err != nil {
		c.mu.Unlock()
		return // already canceled
	}
	c.err = err
	d, _ := c.done.Load().(chan struct{})
	if d == nil {
		c.done.Store(closedC)
	} else {
		close(d)
	}
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
	c.mu.Unlock()
}

// Cancel this context releases resources associated with it, so code should
// call cancel as soon as the operations running in this Context complete.
func (c *Context) Cancel() {
	if c.cancelFunc != nil {
		c.cancelFunc()
	}
}

// Deadline returns the time when work done on behalf of this context
// should be canceled. Deadline returns ok==false when no deadline is
// set. Successive calls to Deadline return the same results.
func (c *Context) Deadline() (time.Time, bool) {
	return c.deadline, true
}

// Err If Done is not yet closed, Err returns nil.
// If Done is closed, Err returns a non-nil error explaining why:
// Canceled if the context was canceled
// or DeadlineExceeded if the context's deadline passed.
// After Err returns a non-nil error, successive calls to Err return the same error.
func (c *Context) Err() error {
	c.mu.Lock()
	err := c.err
	c.mu.Unlock()
	return err
}

// Done returns a channel that's closed when work done on behalf of this
// context should be canceled. Done may return nil if this context can
// never be canceled. Successive calls to Done return the same value.
// The close of the Done channel may happen asynchronously,
// after the cancel function returns.
func (c *Context) Done() <-chan struct{} {
	d := c.done.Load()
	if d != nil {
		return d.(chan struct{})
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	d = c.done.Load()
	if d == nil {
		d = make(chan struct{})
		c.done.Store(d)
	}
	return d.(chan struct{})
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

// propagateCancel arranges for child to be canceled when parent is.
func propagateCancel(parent context.Context, child *Context) {
	if parent == nil {
		return
	}
	done := parent.Done()
	if done == nil {
		return // parent is never canceled
	}

	go func() {
		select {
		case <-done:
			// parent is already canceled
			child.cancel(parent.Err())
		case <-child.Done():
		}
	}()
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

// Logger returns the logger, if Options.Logger is nil return slog.Default
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
