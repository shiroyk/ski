package ski

import (
	"context"
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
