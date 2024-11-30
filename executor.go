package ski

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

type (
	// Executor accept the argument and output result.
	// when the parameter is a slice, it needs to be wrapped with Iterator.
	Executor interface {
		Exec(context.Context, any) (any, error)
	}
	// NewExecutor to create a new Executor
	NewExecutor func(Arguments) (Executor, error)
)

// Arguments slice of Executor
type Arguments []Executor

// Get index of Executor, if index out of range, return nil
func (a Arguments) Get(i int) Executor {
	if i < 0 || i >= len(a) {
		return nil
	}
	return a[i]
}

// GetString index of string, return empty string if index out of range or type is not string
func (a Arguments) GetString(i int) string {
	e := a.Get(i)
	switch t := e.(type) {
	case String:
		return string(t)
	case fmt.Stringer:
		return t.String()
	case _raw:
		s, ok := t.any.(string)
		if !ok {
			return ""
		}
		return s
	default:
		return ""
	}
}

var reName = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// Register registers the NewExecutor with the given name.
// Valid name characters: a-zA-Z0-9_
func Register(name string, fn NewExecutor) {
	if name == "" {
		panic("ski: invalid pattern")
	}
	if fn == nil {
		panic("ski: NewExecutor is nil")
	}
	before, method, _ := strings.Cut(name, ".")
	if !reName.MatchString(before) {
		panic(fmt.Sprintf("ski: invalid name %q", name))
	}
	if method != "" && !reName.MatchString(method) {
		panic(fmt.Sprintf("ski: invalid name %q", name))
	}

	executors.Lock()
	defer executors.Unlock()

	entries, ok := executors.registry[before]
	if !ok {
		entries = make(NewExecutors)
		executors.registry[before] = entries
	}
	entries[method] = fn
}

// NewExecutors map of NewExecutor
type NewExecutors map[string]NewExecutor

// Registers register the NewExecutors.
// Valid name characters: a-zA-Z0-9_
func Registers(e NewExecutors) {
	executors.Lock()
	defer executors.Unlock()

	for name, fn := range e {
		if name == "" {
			panic("ski: invalid pattern")
		}
		if fn == nil {
			panic("ski: NewExecutor is nil")
		}
		before, method, _ := strings.Cut(name, ".")
		if !reName.MatchString(before) {
			panic(fmt.Sprintf("ski: invalid name %q", name))
		}
		if method != "" && !reName.MatchString(method) {
			panic(fmt.Sprintf("ski: invalid name %q", name))
		}
		entries, ok := executors.registry[before]
		if !ok {
			entries = make(NewExecutors)
			executors.registry[before] = entries
		}
		entries[method] = fn
	}
}

// GetExecutor returns a NewExecutor with the given name
func GetExecutor(name string) (NewExecutor, bool) {
	executors.RLock()
	defer executors.RUnlock()

	name, method, _ := strings.Cut(name, ".")
	entries, ok := executors.registry[name]
	if !ok {
		return nil, false
	}
	e, ok := entries[method]
	return e, ok
}

// GetExecutors returns the all NewExecutor with the given name
func GetExecutors(name string) (map[string]NewExecutor, bool) {
	executors.RLock()
	defer executors.RUnlock()

	name, _, _ = strings.Cut(name, ".")
	entries, ok := executors.registry[name]
	if !ok {
		return nil, false
	}
	ret := make(map[string]NewExecutor, len(entries))
	for method, e := range entries {
		ret[method] = e
	}
	return ret, true
}

// RemoveExecutor removes Executor with the given names
func RemoveExecutor(names ...string) {
	if len(names) == 0 {
		return
	}

	executors.Lock()
	defer executors.Unlock()

	for _, name := range names {
		name, method, _ := strings.Cut(name, ".")
		entries, ok := executors.registry[name]
		if !ok {
			continue
		}

		if method == "" {
			delete(executors.registry, name)
			continue
		}

		delete(entries, method)

		if len(entries) == 0 {
			delete(executors.registry, name)
		}
	}
}

// AllExecutors returns the all NewExecutor
func AllExecutors() map[string]NewExecutor {
	executors.RLock()
	defer executors.RUnlock()

	ret := make(map[string]NewExecutor)
	for name, entries := range executors.registry {
		for method, e := range entries {
			if method == "" {
				ret[name] = e
			} else {
				ret[name+"."+method] = e
			}
		}
	}
	return ret
}

var executors = struct {
	sync.RWMutex
	registry map[string]NewExecutors
}{
	registry: make(map[string]NewExecutors),
}
