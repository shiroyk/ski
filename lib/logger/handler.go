package logger

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"

	"golang.org/x/exp/slog"
)

const (
	red    = 31
	yellow = 33
	blue   = 36
	grey   = 38
)

var bufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

func freeBuffer(buf *bytes.Buffer) {
	buf.Reset()
	bufPool.Put(buf)
}

// ConsoleHandler is a Handler that writes Records to an io.Writer as
// line-delimited JSON objects.
type ConsoleHandler struct {
	level        slog.Leveler
	w            io.Writer
	attrs, group string
	noColor      bool
}

// NewConsoleHandler creates a ConsoleHandler that writes to w,
// using the default options.
func NewConsoleHandler(l slog.Leveler) *ConsoleHandler {
	return &ConsoleHandler{
		level:   l,
		w:       os.Stdout,
		noColor: os.Getenv("NO_COLOR") != "",
	}
}

// Enabled reports whether the handler handles records at the given level.
// The handler ignores records whose level is lower.
func (c *ConsoleHandler) Enabled(l slog.Level) bool {
	minLevel := slog.LevelInfo
	if c.level != nil {
		minLevel = c.level.Level()
	}
	return l >= minLevel
}

// WithAttrs With returns a new ConsoleHandler whose attributes consists
// of h's attributes followed by attrs.
func (c *ConsoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	buf := bufPool.Get().(*bytes.Buffer)
	defer freeBuffer(buf)

	for _, attr := range attrs {
		buf.WriteString(attr.String())
	}

	return &ConsoleHandler{
		level:   c.level,
		w:       c.w,
		group:   c.group,
		attrs:   buf.String(),
		noColor: c.noColor,
	}
}

// WithGroup returns a new Handler with the given group appended to
// the receiver's existing groups.
func (c *ConsoleHandler) WithGroup(name string) slog.Handler {
	return &ConsoleHandler{
		level:   c.level,
		w:       c.w,
		group:   name,
		attrs:   c.attrs,
		noColor: c.noColor,
	}
}

// Handle formats its argument Record as single line.
//
// If the Record's time is zero, the time is omitted.
//
// If the Record's level is zero, the level is omitted.
// Otherwise, the key is "level"
// and the value of [Level.String] is output.
//
// Each call to Handle results in a single serialized call to io.Writer.Write.
func (c *ConsoleHandler) Handle(r slog.Record) (err error) {
	time := ""
	if !r.Time.IsZero() {
		time = r.Time.Format("15:04:05.000")
	}

	buf := bufPool.Get().(*bytes.Buffer)
	defer freeBuffer(buf)

	r.Attrs(func(a slog.Attr) {
		buf.WriteString(a.Key)
		buf.WriteString(": ")
		buf.WriteString(a.Value.String())
		buf.WriteByte(' ')
	})

	var levelColor = grey
	switch r.Level {
	case slog.LevelDebug:
		levelColor = blue
	case slog.LevelWarn:
		levelColor = yellow
	case slog.LevelError:
		levelColor = red
	}

	if c.noColor {
		_, err = fmt.Fprintf(c.w, "[%s] %s %s %s\n", time, r.Level.String(), r.Message, buf.String())
		return
	}

	_, err = fmt.Fprintf(c.w, "[%s] \x1b[%dm%s \x1b[0m%s %s\n", time, levelColor, r.Level.String(), r.Message, buf.String())

	return
}
