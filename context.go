package ski

import (
	"context"
	"sync"
)

// Context multiple values context
type Context interface {
	context.Context
	// SetValue store key with value
	SetValue(key, value any)
}

type valuesCtx struct {
	context.Context
	values *sync.Map
}

func (c *valuesCtx) Value(key any) any {
	value, ok := c.values.Load(key)
	if ok {
		return value
	}
	return c.Context.Value(key)
}

func (c *valuesCtx) SetValue(key, value any) { c.values.Store(key, value) }

var _ctxKey byte

// NewContext returns a new can store multiple values context with values
func NewContext(parent context.Context, values map[any]any) Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	m := new(sync.Map)
	if values != nil {
		for k, v := range values {
			m.Store(k, v)
		}
	}
	ctx := &valuesCtx{Context: parent, values: m}
	m.Store(&_ctxKey, ctx)
	return ctx
}

// WithValue if parent exists multiple values Context then set the key/value.
// or returns a copy of parent in which the value associated with key is val.
func WithValue(ctx context.Context, key, value any) context.Context {
	if v, ok := ctx.Value(&_ctxKey).(*valuesCtx); ok {
		v.SetValue(key, value)
		return ctx
	}
	return context.WithValue(ctx, key, value)
}
