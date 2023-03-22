// Package v1 the version 1 api
package v1

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"text/template"

	"github.com/shiroyk/cloudcat/analyzer"
	"github.com/shiroyk/cloudcat/di"
	"github.com/shiroyk/cloudcat/fetch"
	"github.com/shiroyk/cloudcat/js"
	"github.com/shiroyk/cloudcat/parser"
	"github.com/shiroyk/cloudcat/schema"
	"golang.org/x/exp/slog"
	"gopkg.in/yaml.v3"
)

const (
	debugHeader                      = "X-Debug"
	mimeApplicationJavaScript        = "application/javascript"
	mimeApplicationJSONCharsetUTF8   = "application/json; charset=UTF-8"
	mimeApplicationNDJSONCharsetUTF8 = "application/x-ndjson; charset=UTF-8"
	contentType                      = "Content-Type"
)

func run(w http.ResponseWriter, req *http.Request) error {
	debug := req.Header.Get(debugHeader) != ""

	resContentType := mimeApplicationJSONCharsetUTF8
	var log slog.Handler = slog.NewTextHandler(os.Stdout)
	if debug {
		log = newResponseHandler(w, slog.LevelDebug)
		resContentType = mimeApplicationNDJSONCharsetUTF8
	}
	w.Header().Set(contentType, resContentType)

	isJs := strings.HasPrefix(req.Header.Get(contentType), mimeApplicationJavaScript)
	if isJs {
		parserCtx := parser.NewContext(parser.Options{
			Parent: req.Context(),
			Logger: slog.New(log),
		})
		defer parserCtx.Cancel()
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return err
		}
		result, err := js.RunString(parserCtx, string(body)) //nolint:contextcheck
		if err != nil {
			return err
		}

		return JSON(w, http.StatusOK, result)
	}

	model := new(schema.Model)
	err := yaml.NewDecoder(req.Body).Decode(model)
	if err != nil {
		return err
	}

	fetcher, err := di.Resolve[fetch.Fetch]()
	if err != nil {
		return err
	}
	tplFunc, _ := di.Resolve[template.FuncMap]()
	mReq, err := fetch.NewTemplateRequest(tplFunc, model.Source.HTTP, nil)
	mReq.Proxy = model.Source.Proxy
	if err != nil {
		return err
	}

	parserCtx := parser.NewContext(parser.Options{
		Parent: req.Context(),
		Logger: slog.New(log),
		URL:    mReq.URL.String(),
	})
	defer parserCtx.Cancel()
	mRes, err := fetcher.DoRequest(mReq.WithContext(parserCtx)) //nolint:contextcheck
	if err != nil {
		return err
	}

	result := analyzer.Analyze(parserCtx, model.Schema, mRes.String())

	return JSON(w, http.StatusOK, result)
}

// responseHandler is a Handler that writes Records to an echo.Response as
// line-delimited JSON objects.
type responseHandler struct {
	level        slog.Leveler
	w            http.ResponseWriter
	attrs, group string
}

// newResponseHandler creates a responseHandler that writes to w,
// using the default options.
func newResponseHandler(w http.ResponseWriter, l slog.Leveler) *responseHandler {
	return &responseHandler{
		level: l,
		w:     w,
	}
}

// Enabled reports whether the handler handles records at the given level.
// The handler ignores records whose level is lower.
func (c *responseHandler) Enabled(_ context.Context, l slog.Level) bool {
	minLevel := slog.LevelInfo
	if c.level != nil {
		minLevel = c.level.Level()
	}
	return l >= minLevel
}

// WithAttrs With returns a new responseHandler whose attributes consists
// of h's attributes followed by attrs.
func (c *responseHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	buf := new(strings.Builder)

	for _, attr := range attrs {
		buf.WriteString(attr.String())
	}

	return &responseHandler{
		level: c.level,
		w:     c.w,
		group: c.group,
		attrs: buf.String(),
	}
}

// WithGroup returns a new Handler with the given group appended to
// the receiver's existing groups.
func (c *responseHandler) WithGroup(name string) slog.Handler {
	return &responseHandler{
		level: c.level,
		w:     c.w,
		group: name,
		attrs: c.attrs,
	}
}

// Handle formats its argument Record as single line.
// Each call to Handle results in a single serialized call to io.Writer.Write.
func (c *responseHandler) Handle(_ context.Context, r slog.Record) (err error) {
	data := make(map[string]any, r.NumAttrs()+3)
	data["level"] = r.Level.String()
	data["msg"] = r.Message
	if !r.Time.IsZero() {
		data["time"] = r.Time.Format("15:04:05.000")
	}

	r.Attrs(func(a slog.Attr) {
		data[a.Key] = a.Value.String()
	})

	var bytes []byte
	bytes, err = json.Marshal(data)
	if err != nil {
		return
	}

	bytes = append(bytes, '\n')

	_, err = c.w.Write(bytes)
	if err != nil {
		return
	}
	c.w.(http.Flusher).Flush()
	return
}
