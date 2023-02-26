package v1

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"text/template"

	"github.com/labstack/echo/v4"
	"github.com/shiroyk/cloudcat/analyzer"
	"github.com/shiroyk/cloudcat/di"
	"github.com/shiroyk/cloudcat/fetch"
	"github.com/shiroyk/cloudcat/parser"
	"github.com/shiroyk/cloudcat/schema"
	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog"
	"gopkg.in/yaml.v3"
)

const (
	requestHeader                    = "X-Request-Header"
	responseHeader                   = "X-Response-Header"
	debugHeader                      = "X-Debug"
	mimeApplicationNDJSONCharsetUTF8 = "application/x-ndjson; charset=UTF-8"
)

var skipOverrideHeader = []string{
	echo.HeaderContentType, echo.HeaderContentLength,
	echo.HeaderCookie, echo.HeaderSetCookie,
	echo.HeaderAccept, echo.HeaderAcceptEncoding,
	echo.HeaderContentEncoding,
}

// RouteAnalyze tha analyze routes
func RouteAnalyze(e *echo.Echo) {
	analyze := e.Group("/analyze")
	analyze.POST("", ModelAnalyze)
}

// ModelAnalyze analyzes the model
func ModelAnalyze(ctx echo.Context) error {
	overrideRequest := ctx.Request().Header.Get(requestHeader) != ""
	overrideResponse := ctx.Request().Header.Get(responseHeader) != ""
	debug := ctx.Request().Header.Get(debugHeader) != ""

	model := new(schema.Model)
	err := yaml.NewDecoder(ctx.Request().Body).Decode(model)
	if err != nil {
		return err
	}

	fetcher, err := di.Resolve[fetch.Fetch]()
	if err != nil {
		return err
	}
	tplFunc, _ := di.Resolve[template.FuncMap]()
	req, err := fetch.NewTemplateRequest(tplFunc, model.Source.HTTP, nil)
	req.Proxy = model.Source.Proxy
	if err != nil {
		return err
	}

	if overrideRequest {
		for k, v := range req.Header {
			if slices.Contains(skipOverrideHeader, k) {
				continue
			}
			for _, vv := range v {
				req.Header.Add(k, vv)
			}
		}
	}

	ctx.Response().Header().Set(echo.HeaderContentType, mimeApplicationNDJSONCharsetUTF8)
	var log slog.Handler = slog.NewTextHandler(os.Stdout)
	if debug {
		log = newResponseHandler(ctx.Response(), slog.LevelDebug)
	}
	parserCtx := parser.NewContext(parser.Options{
		Parent: ctx.Request().Context(),
		Logger: slog.New(log),
		URL:    model.Source.HTTP,
	})
	defer parserCtx.Cancel()

	res, err := fetcher.DoRequest(req.WithContext(parserCtx))
	if err != nil {
		return err
	}

	if overrideResponse {
		for k, v := range res.Header {
			if slices.Contains(skipOverrideHeader, k) {
				continue
			}
			for _, vv := range v {
				ctx.Response().Header().Add(k, vv)
			}
		}
	}

	result := analyzer.Analyze(parserCtx, model.Schema, res.String())

	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}

	_, err = ctx.Response().Write(bytes)
	if err != nil {
		return err
	}
	ctx.Response().Flush()

	return nil
}

// responseHandler is a Handler that writes Records to an echo.Response as
// line-delimited JSON objects.
type responseHandler struct {
	level        slog.Leveler
	w            *echo.Response
	attrs, group string
}

// newResponseHandler creates a responseHandler that writes to w,
// using the default options.
func newResponseHandler(res *echo.Response, l slog.Leveler) *responseHandler {
	return &responseHandler{
		level: l,
		w:     res,
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
	c.w.Flush()
	return
}
