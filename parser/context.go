package parser

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shiroyk/cloudcat/meta"
)

const (
	DefaultTimeout = time.Minute
)

type Context struct {
	context.Context
	mu                   sync.Mutex  // protects following fields
	timer                *time.Timer // Under Context.mu.
	deadline             time.Time
	done                 atomic.Value // of chan struct{}, created lazily, closed by first cancel call
	err                  error        // set to non-nil by the first cancel call
	cancelFunc           context.CancelFunc
	opt                  *Options
	value                *sync.Map
	baseUrl, redirectUrl string
}

type Options struct {
	Config  meta.Config
	Url     string
	Timeout time.Duration
}

func NewContext(opt *Options) *Context {
	var d time.Time
	if opt.Timeout == 0 {
		d = time.Now().Add(DefaultTimeout)
	} else {
		d = time.Now().Add(opt.Timeout)
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
	c.redirectUrl = opt.Url
	if u, err := url.Parse(c.redirectUrl); err == nil {
		c.baseUrl = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
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

// closedchan is a reusable-closed channel.
var closedchan = make(chan struct{})

func init() {
	close(closedchan)
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
		c.done.Store(closedchan)
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

func (c *Context) Config() meta.Config {
	return c.opt.Config
}

func (c *Context) BaseUrl() string {
	return c.baseUrl
}

func (c *Context) RedirectUrl() string {
	return c.redirectUrl
}
