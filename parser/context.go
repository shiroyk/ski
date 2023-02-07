package parser

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/exp/slog"
)

const (
	DefaultTimeout = time.Minute
)

type Context struct {
	mu                   sync.Mutex  // protects following fields
	timer                *time.Timer // Under Context.mu.
	deadline             time.Time
	done                 atomic.Value // of chan struct{}, created lazily, closed by first cancel call
	err                  error        // set to non-nil by the first cancel call
	cancelFunc           context.CancelFunc
	opt                  Options
	value                *sync.Map
	baseURL, redirectURL string
}

type Options struct {
	Config Config
	Logger *slog.Logger
	Url    string
}

func NewContext(opt Options) *Context {
	var d time.Time
	if opt.Config.Timeout > 0 {
		d = time.Now().Add(opt.Config.Timeout)
	} else {
		d = time.Now().Add(DefaultTimeout)
	}
	if opt.Logger == nil {
		opt.Logger = slog.Default()
	}
	c := &Context{
		deadline: d,
		value:    new(sync.Map),
		opt:      opt,
	}
	dur := time.Until(d)
	if dur <= 0 {
		c.cancel(context.DeadlineExceeded) // deadline has already passed
		c.cancelFunc = func() { c.cancel(context.Canceled) }
		return c
	}
	c.redirectURL = opt.Url
	if u, err := url.Parse(c.redirectURL); err == nil {
		c.baseURL = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
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

func (c *Context) Cancel() {
	c.mu.Lock()
	if c.cancelFunc != nil {
		c.cancelFunc()
	}
	c.mu.Unlock()
}

func (c *Context) Deadline() (time.Time, bool) {
	return c.deadline, true
}

func (c *Context) Err() error {
	c.mu.Lock()
	err := c.err
	c.mu.Unlock()
	return err
}

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

func (c *Context) ClearValue() {
	c.value = new(sync.Map)
}

func (c *Context) Value(key any) any {
	if v, ok := c.value.Load(key); ok {
		return v
	}
	return nil
}

func (c *Context) GetValue(key any) (any, bool) {
	return c.value.Load(key)
}

func (c *Context) SetValue(key any, value any) {
	c.value.Store(key, value)
}

func (c *Context) Config() Config {
	return c.opt.Config
}

func (c *Context) Logger() *slog.Logger {
	return c.opt.Logger
}

func (c *Context) BaseURL() string {
	return c.baseURL
}

func (c *Context) RedirectURL() string {
	return c.redirectURL
}
