package ski

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

var loggerKey byte

// Logger get slog.Logger from the context
func Logger(ctx context.Context) *slog.Logger {
	if logger := ctx.Value(&loggerKey); logger != nil {
		return logger.(*slog.Logger)
	}
	return slog.Default()
}

// WithLogger set the slog.Logger to context
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return WithValue(ctx, &loggerKey, logger)
}

// ExecToString convert Executor to string if it implements fmt.Stringer
func ExecToString(exec Executor) string {
	switch t := exec.(type) {
	case fmt.Stringer:
		return t.String()
	case _raw:
		if s, ok := t.any.(string); ok {
			return s
		}
	}
	return ""
}

// StringExecutor to create a new Executor with string argument
func StringExecutor(fn func(str string) (Executor, error)) NewExecutor {
	return func(args ...Executor) (Executor, error) {
		if len(args) == 0 {
			return nil, errors.New("needs 1 string argument")
		}
		return fn(ExecToString(args[0]))
	}
}

// MapKeys returns the keys of the map m.
// The keys will be in an indeterminate order.
func MapKeys[M ~map[K]V, K comparable, V any](m M) []K {
	r := make([]K, 0, len(m))
	for k := range m {
		r = append(r, k)
	}
	return r
}

// MapValues returns the values of the map m.
// The values will be in an indeterminate order.
func MapValues[M ~map[K]V, K comparable, V any](m M) []V {
	r := make([]V, 0, len(m))
	for _, v := range m {
		r = append(r, v)
	}
	return r
}
