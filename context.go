package ski

import (
	"context"
	"maps"
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
	mu     sync.RWMutex
	values map[any]any
}

func (c *valuesCtx) Value(key any) any {
	c.mu.RLock()
	v, ok := c.values[key]
	c.mu.RUnlock()
	if ok {
		return v
	}
	return c.Context.Value(key)
}

func (c *valuesCtx) SetValue(key, value any) {
	c.mu.Lock()
	c.values[key] = value
	c.mu.Unlock()
}

var _ctxKey byte

// NewContext returns a new can store multiple values context with values
func NewContext(parent context.Context, values map[any]any) Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	var clone map[any]any
	if values == nil {
		clone = make(map[any]any)
	} else {
		clone = maps.Clone(values)
	}
	ctx := &valuesCtx{Context: parent, values: clone}
	clone[&_ctxKey] = ctx
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
