package logger

import (
	"fmt"

	"golang.org/x/exp/slog"
)

// Logf calls on the default logger.
func Logf(level slog.Level, format string, args ...any) {
	slog.Log(level, fmt.Sprintf(format, args...))
}

// Debugf calls LevelDebug on the default logger.
func Debugf(format string, args ...any) {
	Logf(slog.LevelDebug, format, args...)
}

// Infof calls LevelInfo on the default logger.
func Infof(format string, args ...any) {
	Logf(slog.LevelInfo, format, args...)
}

// Errorf calls LevelError on the default logger.
func Errorf(format string, args ...any) {
	Logf(slog.LevelError, format, args)
}

// Warnf calls LevelWarn on the default logger.
func Warnf(format string, args ...any) {
	Logf(slog.LevelWarn, format, args)
}

// Debug calls Logger.Debug on the default logger.
func Debug(msg string, args ...any) {
	slog.Debug(msg, args...)
}

// Info calls Logger.Info on the default logger.
func Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

// Warn calls Logger.Warn on the default logger.
func Warn(msg string, args ...any) {
	slog.Warn(msg, args...)
}

// Error calls Logger.Error on the default logger.
func Error(msg string, err error, args ...any) {
	slog.Error(msg, err, args...)
}
