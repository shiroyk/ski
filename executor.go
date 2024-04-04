package ski

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"unicode"
)

type (
	// Executor accept the argument and output result
	Executor interface {
		Exec(context.Context, any) (any, error)
	}
	// NewExecutor to create a new Executor
	NewExecutor func(...Executor) (Executor, error)
)

// Register registers the NewExecutor with the given name.
// Valid name: [a-zA-Z_][a-zA-Z0-9_]* (leading and trailing underscores are allowed)
func Register(name string, fn NewExecutor) {
	if name == "" {
		panic("ski: invalid pattern")
	}
	if fn == nil {
		panic("ski: new function is nil")
	}
	if !isValidName(name) {
		panic(fmt.Sprintf("ski: invalid name %q", name))
	}

	executors.Lock()
	defer executors.Unlock()

	name, method, _ := strings.Cut(name, ".")
	entries := executors.registry[name]
	executors.registry[name] = append(entries, entry{fn, method})
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
	for _, entry := range entries {
		if entry.method == method {
			return entry.new, true
		}
	}
	return nil, false
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
	for _, entry := range entries {
		ret[entry.method] = entry.new
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

	newEntries := slices.DeleteFunc(entries, func(e entry) bool {
		return e.method == method
	})

	if len(newEntries) == 0 {
		delete(executors.registry, name)
	} else {
		executors.registry[name] = newEntries
	}
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

type entry struct {
	new    NewExecutor
	method string
}

var executors = struct {
	sync.RWMutex
	registry map[string][]entry
}{
	registry: make(map[string][]entry),
}
