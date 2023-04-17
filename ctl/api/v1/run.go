// Package v1 the version 1 api
package v1

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"text/template"

	"github.com/shiroyk/cloudcat/core"
	"github.com/shiroyk/cloudcat/core/js"
	"github.com/shiroyk/cloudcat/ctl/model"
	"github.com/shiroyk/cloudcat/fetch"
	"github.com/shiroyk/cloudcat/plugin"
	"golang.org/x/exp/slog"
	"gopkg.in/yaml.v3"
)

const (
	mimeApplicationJavaScript        = "application/javascript"
	mimeApplicationJSONCharsetUTF8   = "application/json; charset=UTF-8"
	mimeApplicationNDJSONCharsetUTF8 = "application/x-ndjson; charset=UTF-8"
	contentType                      = "Content-Type"
)

func modelRun() handleFunc {
	fetcher := core.MustResolve[core.Fetch]()
	tplFunc, _ := core.Resolve[template.FuncMap]()
	handler := slog.NewTextHandler(os.Stdout)
	return func(w http.ResponseWriter, req *http.Request) error {
		w.Header().Set(contentType, mimeApplicationJSONCharsetUTF8)

		isJs := strings.HasPrefix(req.Header.Get(contentType), mimeApplicationJavaScript)
		if isJs {
			parserCtx := plugin.NewContext(plugin.Options{
				Parent: req.Context(),
				Logger: slog.New(handler),
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

		model := new(model.Model)
		err := yaml.NewDecoder(req.Body).Decode(model)
		if err != nil {
			return err
		}

		mReq, err := fetch.NewTemplateRequest(tplFunc, model.Source.HTTP, nil)
		if err != nil {
			return err
		}

		parserCtx := plugin.NewContext(plugin.Options{
			Parent: req.Context(),
			Logger: slog.New(handler),
			URL:    mReq.URL.String(),
		})
		defer parserCtx.Cancel()
		mReq = fetch.WithRequestConfig(mReq, fetch.RequestConfig{Proxy: model.Source.Proxy}).
			WithContext(parserCtx)
		mRes, err := fetch.DoString(fetcher, mReq) //nolint:contextcheck
		if err != nil {
			return err
		}

		result := core.Analyze(parserCtx, model.Schema, mRes)

		return JSON(w, http.StatusOK, result)
	}
}

func modelDebug() handleFunc {
	reqAttr := slog.String("type", "request")
	resAttr := slog.String("type", "response")
	fetcher := core.MustResolve[core.Fetch]()
	tplFunc, _ := core.Resolve[template.FuncMap]()
	return func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set(contentType, mimeApplicationNDJSONCharsetUTF8)
		logger := slog.New(newResponseHandler(w, slog.LevelDebug))
		isJs := strings.HasPrefix(r.Header.Get(contentType), mimeApplicationJavaScript)

		if isJs {
			parserCtx := plugin.NewContext(plugin.Options{
				Parent: r.Context(),
				Logger: logger,
			})
			defer parserCtx.Cancel()
			body, err := io.ReadAll(r.Body)
			if err != nil {
				return err
			}
			result, err := js.RunString(parserCtx, string(body)) //nolint:contextcheck
			if err != nil {
				return err
			}

			return JSON(w, http.StatusOK, result)
		}

		model := new(model.Model)
		err := yaml.NewDecoder(r.Body).Decode(model)
		if err != nil {
			return err
		}

		req, err := fetch.NewTemplateRequest(tplFunc, model.Source.HTTP, nil)
		if err != nil {
			return err
		}

		reqs, _ := httputil.DumpRequest(req, true)
		logger.Debug("request", "result", string(reqs), reqAttr)

		parserCtx := plugin.NewContext(plugin.Options{
			Parent: r.Context(),
			Logger: logger,
			URL:    req.URL.String(),
		})
		defer parserCtx.Cancel()
		req = fetch.WithRequestConfig(req, fetch.RequestConfig{Proxy: model.Source.Proxy}).
			WithContext(parserCtx)
		res, err := fetcher.Do(req)
		if err != nil {
			return err
		}

		ress, _ := httputil.DumpResponse(res, true)
		logger.Debug("response", "result", string(ress), resAttr)

		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}

		return JSON(w, http.StatusOK, core.Analyze(parserCtx, model.Schema, string(body)))
	}
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
