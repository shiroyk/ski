package ski

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"unicode"
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

// Register registers the NewExecutor with the given name.
// Valid name: [a-zA-Z_][a-zA-Z0-9_]* (leading and trailing underscores are allowed)
func Register(name string, fn NewExecutor) {
	if name == "" {
		panic("ski: invalid pattern")
	}
	if fn == nil {
		panic("ski: NewExecutor is nil")
	}
	if !isValidName(name) {
		panic(fmt.Sprintf("ski: invalid name %q", name))
	}

	executors.Lock()
	defer executors.Unlock()

	name, method, _ := strings.Cut(name, ".")
	entries, ok := executors.registry[name]
	if !ok {
		entries = make(NewExecutors)
		executors.registry[name] = entries
	}
	entries[method] = fn
}

// NewExecutors map of NewExecutor
type NewExecutors map[string]NewExecutor

// Registers register the NewExecutors.
// Valid name: [a-zA-Z_][a-zA-Z0-9_]* (leading and trailing underscores are allowed)
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
		if !isValidName(name) {
			panic(fmt.Sprintf("ski: invalid name %q", name))
		}
		name, method, _ := strings.Cut(name, ".")
		entries, ok := executors.registry[name]
		if !ok {
			entries = make(NewExecutors)
			executors.registry[name] = entries
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

// RemoveExecutor removes an Executor with the given name
func RemoveExecutor(name string) {
	executors.Lock()
	defer executors.Unlock()

	name, method, _ := strings.Cut(name, ".")
	entries, ok := executors.registry[name]
	if !ok {
		return
	}

	if method == "" {
		delete(executors.registry, name)
		return
	}

	delete(entries, method)

	if len(entries) == 0 {
		delete(executors.registry, name)
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

func isValidName(s string) bool {
	if s == "" {
		return false
	}
	if !unicode.IsLetter(rune(s[0])) && s[0] != '_' {
		return false
	}
	hasDot := false
	for i := 0; i < len(s); i++ {
		char := rune(s[i])
		if char == '.' {
			if hasDot {
				return false
			}
			hasDot = true
			if i == len(s)-1 {
				return false
			}
			next := s[i+1]
			if !unicode.IsLetter(rune(next)) && next != '_' {
				return false
			}
			i++
			continue
		}
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) && char != '_' {
			return false
		}
	}
	return true
}

var executors = struct {
	sync.RWMutex
	registry map[string]NewExecutors
}{
	registry: make(map[string]NewExecutors),
}
