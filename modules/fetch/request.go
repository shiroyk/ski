package fetch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	urlpkg "net/url"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/promise"
	"github.com/shiroyk/ski/js/types"
	"github.com/shiroyk/ski/modules/buffer"
	"github.com/shiroyk/ski/modules/signal"
	"github.com/shiroyk/ski/modules/stream"
	"github.com/shiroyk/ski/modules/url"
)

// Request interface of the Fetch API represents a resource request.
// https://developer.mozilla.org/en-US/docs/Web/API/Request/Request
type Request struct{}

func (r *Request) prototype(rt *sobek.Runtime) *sobek.Object {
	p := rt.NewObject()
	_ = p.DefineAccessorProperty("method", rt.ToValue(r.method), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("url", rt.ToValue(r.url), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("headers", rt.ToValue(r.headers), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("body", rt.ToValue(r.body), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("bodyUsed", rt.ToValue(r.bodyUsed), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("mode", rt.ToValue(r.mode), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("credentials", rt.ToValue(r.credentials), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("cache", rt.ToValue(r.cache), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("destination", rt.ToValue(r.destination), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("redirect", rt.ToValue(r.redirect), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("referrer", rt.ToValue(r.referrer), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("referrerPolicy", rt.ToValue(r.referrerPolicy), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("integrity", rt.ToValue(r.integrity), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("isHistoryNavigation", rt.ToValue(r.isHistoryNavigation), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("signal", rt.ToValue(r.signal), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("keepalive", rt.ToValue(r.keepalive), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)

	_ = p.Set("clone", r.clone)
	_ = p.Set("bytes", r.bytes)
	_ = p.Set("text", r.text)
	_ = p.Set("json", r.json)
	_ = p.Set("arrayBuffer", r.arrayBuffer)
	_ = p.Set("formData", r.formData)
	_ = p.Set("blob", r.blob)
	_ = p.SetSymbol(sobek.SymToStringTag, func(sobek.FunctionCall) sobek.Value { return rt.ToValue("Request") })
	return p
}

func (r *Request) constructor(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	var req request

	if arg := call.Argument(0); !sobek.IsUndefined(arg) {
		if input, ok := toRequest(arg); ok {
			req = *input
			// copy headers
			req.headers = types.New(rt, "Headers", input.headers)
		} else {
			req = request{
				url:         arg.String(),
				method:      "GET",
				mode:        "cors",
				credentials: "same-origin",
				cache:       "default",
				redirect:    "follow",
				referrer:    "about:client",
				integrity:   "",
			}
		}
	}

	initRequest(rt, call.Argument(1), &req)

	switch {
	case req.bodyUsed.Load():
		panic(rt.NewTypeError(errBodyAlreadyRead.Error()))
	case stream.IsLocked(req.bodyStream):
		panic(rt.NewTypeError(errBodyStreamLocked.Error()))
	case stream.IsDisturbed(req.bodyStream):
		panic(rt.NewTypeError(errBodyStreamRead.Error()))
	}

	obj := rt.NewObject()
	_ = obj.SetSymbol(symRequest, &req)
	_ = obj.SetPrototype(call.This.Prototype())
	return obj
}

func (r *Request) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	proto := r.prototype(rt)
	ctor := rt.ToValue(r.constructor).(*sobek.Object)
	_ = proto.DefineDataProperty("constructor", ctor, sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_FALSE)
	_ = ctor.Set("prototype", proto)
	return ctor, nil
}

func (*Request) method(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toThisRequest(rt, call.This).method)
}

func (*Request) url(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toThisRequest(rt, call.This).url)
}

func (*Request) headers(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return toThisRequest(rt, call.This).headers
}

func (*Request) bodyUsed(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toThisRequest(rt, call.This).bodyUsed.Load())
}

func (*Request) mode(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toThisRequest(rt, call.This).mode)
}

func (*Request) credentials(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toThisRequest(rt, call.This).credentials)
}

func (*Request) cache(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toThisRequest(rt, call.This).cache)
}

func (*Request) redirect(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toThisRequest(rt, call.This).redirect)
}

func (*Request) referrer(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toThisRequest(rt, call.This).referrer)
}

func (*Request) referrerPolicy(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toThisRequest(rt, call.This).referrerPolicy)
}

func (*Request) destination(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toThisRequest(rt, call.This).destination)
}

func (*Request) integrity(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toThisRequest(rt, call.This).integrity)
}

func (*Request) isHistoryNavigation(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toThisRequest(rt, call.This).isHistoryNavigation)
}

func (*Request) signal(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return toThisRequest(rt, call.This).signal
}

func (*Request) keepalive(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toThisRequest(rt, call.This).keepalive)
}

// clone creates a clone of the request
func (*Request) clone(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toThisRequest(rt, call.This)
	body := this.body
	if body != nil && !this.bodyUsed.Load() {
		b1, b2 := new(bytes.Buffer), new(bytes.Buffer)
		if c, ok := body.(io.Closer); ok {
			defer c.Close()
		}
		_, err := io.Copy(io.MultiWriter(b1, b2), body)
		if err != nil {
			js.Throw(rt, err)
		}
		this.body = io.NopCloser(b1)
		body = io.NopCloser(b2)
	}

	clone := *this
	req := &clone
	req.body = body
	req.bodyUsed = new(atomic.Bool)
	req.headers = types.New(rt, "Headers", this.headers)

	obj := rt.NewObject()
	_ = obj.SetSymbol(symRequest, req)
	_ = obj.SetPrototype(call.This.ToObject(rt).Prototype())
	return obj
}

// bytes returns a promise which resolves with the body as a Uint8Array
func (*Request) bytes(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toThisRequest(rt, call.This)
	return promise.New(rt, func(callback promise.Callback) {
		data, err := this.read()
		callback(func() (any, error) {
			if err != nil {
				panic(rt.NewTypeError(err.Error()))
			}
			return types.New(rt, "Uint8Array", rt.ToValue(rt.NewArrayBuffer(data))), nil
		})
	})
}

// text returns a promise which resolves with the body text as a string
func (r *Request) text(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toThisRequest(rt, call.This)
	return promise.New(rt, func(callback promise.Callback) {
		data, err := this.read()
		callback(func() (any, error) {
			if err != nil {
				panic(rt.NewTypeError(err.Error()))
			}
			return string(data), nil
		})
	})
}

// json returns a promise that resolves with the result of parsing the request's body as JSON
func (r *Request) json(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toThisRequest(rt, call.This)
	return promise.New(rt, func(callback promise.Callback) {
		data, err := this.read()
		callback(func() (any, error) {
			if err != nil {
				panic(rt.NewTypeError(err.Error()))
			}
			var ret any
			if err = json.Unmarshal(data, &ret); err != nil {
				panic(types.New(rt, "SyntaxError", rt.ToValue(err.Error())))
			}
			return ret, nil
		})
	})
}

// arrayBuffer returns a promise that resolves with an ArrayBuffer.
func (r *Request) arrayBuffer(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toThisRequest(rt, call.This)
	return promise.New(rt, func(callback promise.Callback) {
		data, err := this.read()
		callback(func() (any, error) {
			if err != nil {
				panic(rt.NewTypeError(err.Error()))
			}
			return rt.NewArrayBuffer(data), nil
		})
	})
}

// formData returns a promise which resolves with the body as a FormData object
func (r *Request) formData(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toThisRequest(rt, call.This)
	return promise.New(rt, func(callback promise.Callback) {
		form, err := this.form()
		callback(func() (any, error) {
			if err != nil {
				panic(rt.NewTypeError(err.Error()))
			}
			return newFormData(rt, form), nil
		})
	})
}

// blob returns a promise that resolves with a Blob.
func (*Request) blob(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toThisRequest(rt, call.This)
	return promise.New(rt, func(callback promise.Callback) {
		data, err := this.read()
		callback(func() (any, error) {
			if err != nil {
				panic(rt.NewTypeError(err.Error()))
			}
			opt := sobek.Undefined()
			if v := getContentType(this.headers); v != "" {
				opt = rt.NewObject()
				_ = opt.(*sobek.Object).Set("type", v)
			}
			return types.New(rt, "Blob", rt.NewArray(rt.NewArrayBuffer(data)), opt), nil
		})
	})
}

// body returns a ReadableStream
func (r *Request) body(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toThisRequest(rt, call.This)
	if this.body == nil {
		return sobek.Null()
	}
	if this.bodyStream == nil {
		this.bodyStream = stream.NewReadableStream(rt, this.body)
	}
	return this.bodyStream
}

type request struct {
	method                             string
	url                                string
	headers, signal, bodyStream        sobek.Value
	body                               io.Reader
	bodyUsed                           *atomic.Bool
	credentials, cache, destination    string
	redirect, referrer, referrerPolicy string
	integrity                          string
	mode                               string
	keepalive, isHistoryNavigation     bool
}

func (r *request) form() (*multipart.Form, error) {
	if stream.IsLocked(r.bodyStream) {
		return nil, errBodyStreamLocked
	}
	return parseFromData(r.body, r.bodyUsed, getContentType(r.headers))
}

func (r *request) read() ([]byte, error) {
	if r.body == nil {
		return []byte{}, nil
	}
	if r.bodyUsed.Load() {
		return nil, errBodyAlreadyRead
	}
	if stream.IsLocked(r.bodyStream) {
		return nil, errBodyStreamLocked
	}
	if stream.IsDisturbed(r.bodyStream) {
		return nil, errBodyAlreadyRead
	}
	r.bodyUsed.Store(true)
	if c, ok := r.body.(io.Closer); ok {
		defer c.Close()
	}
	return io.ReadAll(r.body)
}

func (r *request) cancel() {
	if r.signal != nil {
		if c, ok := r.body.(io.Closer); ok {
			defer c.Close()
		}
		signal.Abort(r.signal, signal.ErrAbort)
	}
}

func (r *request) toRequest(rt *sobek.Runtime) *http.Request {
	var ctx context.Context
	if r.signal == nil {
		ctx = js.Context(rt)
	} else {
		ctx = signal.Context(rt, r.signal)
	}
	req, err := http.NewRequestWithContext(ctx, r.method, r.url, r.body)
	if err != nil {
		js.Throw(rt, err)
	}

	req.Header = http.Header(r.headers.Export().(Header))
	if r.cache != "" {
		req.Header.Set("Cache-Control", r.cache)
		req.Header.Set("Pragma", r.cache)
	}
	return req
}

var symRequest = sobek.NewSymbol("Symbol.Request")

// toThisRequest returns a request from the "this" value
func toThisRequest(rt *sobek.Runtime, value sobek.Value) *request {
	if o, ok := value.(*sobek.Object); ok {
		if v := o.GetSymbol(symRequest); v != nil {
			return v.Export().(*request)
		}
	}
	panic(rt.NewTypeError(`Value of "this" must be of type Request`))
}

// toRequest returns a request from the value
func toRequest(value sobek.Value) (*request, bool) {
	if o, ok := value.(*sobek.Object); ok {
		if v := o.GetSymbol(symRequest); v != nil {
			return v.Export().(*request), true
		}
	}
	return nil, false
}

// initRequest builds a request from the RequestInit options
func initRequest(rt *sobek.Runtime, opt sobek.Value, req *request) {
	if req.bodyUsed == nil {
		req.bodyUsed = new(atomic.Bool)
	}
	if sobek.IsUndefined(opt) {
		if req.headers == nil {
			req.headers = types.New(rt, "Headers")
		}
		return
	}
	init := opt.ToObject(rt)
	if v := init.GetSymbol(symRequest); v != nil {
		clone := *v.Export().(*request)
		*req = clone
		// copy headers
		req.headers = types.New(rt, "Headers", clone.headers)
		return
	}
	for _, key := range init.Keys() {
		v := init.Get(key)
		switch key {
		case "method":
			req.method = strings.ToUpper(v.String())
			switch req.method {
			case http.MethodConnect, http.MethodTrace, "TRACK":
				panic(rt.NewTypeError("Invalid request method"))
			}
		case "mode":
			req.mode = v.String()
		case "credentials":
			req.credentials = v.String()
		case "cache":
			req.cache = v.String()
		case "redirect":
			req.redirect = v.String()
		case "referrer":
			req.referrer = v.String()
		case "referrerPolicy":
			req.referrerPolicy = v.String()
		case "integrity":
			req.integrity = v.String()
		case "destination":
			req.destination = v.String()
		case "keepalive":
			req.keepalive = v.ToBoolean()
		case "signal":
			if v.ExportType() != signal.TypeAbortSignal {
				js.Throw(rt, errors.New("options signal is not AbortSignal"))
			}
			req.signal = v
		default:
			continue
		}
	}
	if header := init.Get("headers"); header != nil {
		req.headers = types.New(rt, "Headers", header)
	} else {
		req.headers = types.New(rt, "Headers")
	}
	if req.method == http.MethodGet || req.method == http.MethodHead {
		return
	}
	if b := init.Get("body"); b != nil {
		var body io.Reader
		switch b.ExportType() {
		case types.TypeNil:
		case typeFormData:
			data := b.Export().(*formData)
			reader, contentType, err := data.encode()
			if err != nil {
				js.Throw(rt, err)
			}
			setContentType(req.headers, contentType)
			body = reader
		case url.TypeURLSearchParams:
			body = strings.NewReader(b.String())
			setContentType(req.headers, "application/x-www-form-urlencoded;charset=UTF-8")
		case stream.TypeReadableStream:
			body = stream.GetStreamSource(rt, b)
		default:
			if v, t, ok := buffer.GetReader(b); ok {
				body = v
				if t != "" {
					setContentType(req.headers, t)
				}
			} else if v, ok := buffer.GetBuffer(rt, b); ok {
				body = bytes.NewReader(slices.Clone(v))
			} else {
				body = strings.NewReader(b.String())
				setContentType(req.headers, "text/plain;charset=UTF-8")
			}
		}
		req.bodyUsed.Store(false)
		req.body = body
		req.bodyStream = nil
	}
}

// NewRequest returns a new js Request
func NewRequest(rt *sobek.Runtime, req *http.Request) sobek.Value {
	instance := &request{
		method:      req.Method,
		url:         req.URL.String(),
		body:        req.Body,
		bodyUsed:    new(atomic.Bool),
		headers:     types.New(rt, "Headers", rt.ToValue(map[string][]string(req.Header))),
		referrer:    req.Referer(),
		signal:      signal.New(rt, req.Context()),
		mode:        "cors",
		credentials: "same-origin",
		cache:       "default",
		redirect:    "follow",
	}
	obj := types.New(rt, "Request")
	_ = obj.SetSymbol(symRequest, instance)
	return obj
}

// ToRequest converts a js Request object to an http.Request
func ToRequest(value sobek.Value) (*http.Request, bool) {
	if o, ok := value.(*sobek.Object); ok {
		if v := o.GetSymbol(symRequest); v != nil {
			req := v.Export().(*request)
			u, _ := urlpkg.Parse(req.url)
			return &http.Request{
				Method:     req.method,
				URL:        u,
				RequestURI: req.url,
				Header:     http.Header(req.headers.Export().(Header)),
				Body:       io.NopCloser(req.body),
			}, true
		}
	}
	return nil, false
}
