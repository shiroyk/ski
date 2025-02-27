package fetch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/promise"
	"github.com/shiroyk/ski/modules/buffer"
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
	_ = p.DefineAccessorProperty("redirect", rt.ToValue(r.redirect), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("referrer", rt.ToValue(r.referrer), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("integrity", rt.ToValue(r.integrity), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("signal", rt.ToValue(r.signal), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)

	_ = p.Set("clone", r.clone)
	_ = p.Set("text", r.text)
	_ = p.Set("json", r.json)
	_ = p.Set("arrayBuffer", r.arrayBuffer)
	_ = p.Set("formData", r.formData)
	_ = p.Set("blob", r.blob)
	_ = p.SetSymbol(sobek.SymToStringTag, func(sobek.FunctionCall) sobek.Value { return rt.ToValue("Request") })
	_ = p.SetSymbol(sobek.SymHasInstance, func(call sobek.FunctionCall) sobek.Value {
		return rt.ToValue(call.Argument(0).ExportType() == typeRequest)
	})
	return p
}

func (r *Request) constructor(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	instance := &request{
		method:      "GET",
		mode:        "cors",
		credentials: "same-origin",
		cache:       "default",
		redirect:    "follow",
		referrer:    "",
		integrity:   "",
		body:        io.NopCloser(http.NoBody),
	}

	if arg := call.Argument(0); !sobek.IsUndefined(arg) {
		if arg.ExportType() == typeRequest {
			req := arg.Export().(*request)
			instance.url = req.url
			if !req.bodyUsed {
				instance.body = req.body
			}
		} else {
			instance.url = arg.String()
		}
	}

	initRequest(rt, call.Argument(1), instance)
	obj := rt.ToValue(instance).(*sobek.Object)
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
	return rt.ToValue(toRequest(rt, call.This).method)
}

func (*Request) url(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toRequest(rt, call.This).url)
}

func (*Request) headers(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return toRequest(rt, call.This).headers
}

func (*Request) bodyUsed(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toRequest(rt, call.This).bodyUsed)
}

func (*Request) mode(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toRequest(rt, call.This).mode)
}

func (*Request) credentials(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toRequest(rt, call.This).credentials)
}

func (*Request) cache(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toRequest(rt, call.This).cache)
}

func (*Request) redirect(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toRequest(rt, call.This).redirect)
}

func (*Request) referrer(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toRequest(rt, call.This).referrer)
}

func (*Request) integrity(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toRequest(rt, call.This).integrity)
}

func (*Request) signal(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return toRequest(rt, call.This).signal
}

func (*Request) clone(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toRequest(rt, call.This)
	body := this.body
	if !this.bodyUsed {
		b1, b2 := new(bytes.Buffer), new(bytes.Buffer)
		defer body.Close()
		_, err := io.Copy(io.MultiWriter(b1, b2), body)
		if err != nil {
			js.Throw(rt, err)
		}
		this.body = io.NopCloser(b1)
		body = io.NopCloser(b2)
	}

	instance := &request{
		method:      this.method,
		url:         this.url,
		headers:     this.headers,
		body:        body,
		bodyUsed:    this.bodyUsed,
		mode:        this.mode,
		credentials: this.credentials,
		cache:       this.cache,
		redirect:    this.redirect,
		referrer:    this.referrer,
		integrity:   this.integrity,
	}

	obj := rt.ToValue(instance).(*sobek.Object)
	_ = obj.SetPrototype(call.This.ToObject(rt).Prototype())
	return obj
}

func (r *Request) text(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toRequest(rt, call.This)
	return rt.ToValue(promise.New(rt, this.read, func(data []byte, err error) (any, error) {
		if err != nil {
			return nil, err
		}
		return string(data), nil
	}))
}

func (r *Request) json(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toRequest(rt, call.This)
	return rt.ToValue(promise.New(rt, this.read, func(data []byte, err error) (any, error) {
		if err != nil {
			return nil, err
		}
		var ret any
		if err = json.Unmarshal(data, &ret); err != nil {
			return nil, err
		}
		return ret, nil
	}))
}

func (r *Request) arrayBuffer(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toRequest(rt, call.This)
	return rt.ToValue(promise.New(rt, this.read, func(data []byte, err error) (any, error) {
		if err != nil {
			return nil, err
		}
		return rt.NewArrayBuffer(data), nil
	}))
}

func (r *Request) formData(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toRequest(rt, call.This)
	return rt.ToValue(promise.New(rt, this.read, func(data []byte, err error) (any, error) {
		if err != nil {
			return nil, err
		}
		return js.New(rt, "FormData", rt.ToValue(string(data))), nil
	}))
}

func (*Request) blob(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toRequest(rt, call.This)
	return rt.ToValue(promise.New(rt, this.read, func(data []byte, err error) (any, error) {
		if err != nil {
			return nil, err
		}
		return js.New(rt, "Blob", rt.ToValue(rt.NewArrayBuffer(data))), nil
	}))
}

func (r *Request) body(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toRequest(rt, call.This)
	if this.body == nil {
		return sobek.Null()
	}
	return stream.NewReadableStream(rt, this.body)
}

type request struct {
	method          string
	url             string
	headers, signal sobek.Value
	body            io.ReadCloser
	bodyUsed        bool
	mode            string
	credentials     string
	cache           string
	redirect        string
	referrer        string
	integrity       string
}

func (r *request) read() ([]byte, error) {
	if r.body == nil {
		return []byte{}, nil
	}
	if r.bodyUsed {
		return nil, errBodyAlreadyRead
	}
	r.bodyUsed = true
	defer r.body.Close()
	data, err := io.ReadAll(r.body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (r *request) cancel() {
	if r.signal != nil {
		r.body.Close()
		r.signal.Export().(*abortSignal).abort(errAbort)
	}
}

func (r *request) toRequest(rt *sobek.Runtime) *http.Request {
	var ctx context.Context
	if r.signal == nil {
		ctx = js.Context(rt)
	} else {
		ctx = r.signal.Export().(*abortSignal).ctx
	}
	req, err := http.NewRequestWithContext(ctx, r.method, r.url, r.body)
	if err != nil {
		js.Throw(rt, err)
	}

	req.Header = http.Header(r.headers.Export().(headers))
	if r.cache != "" {
		req.Header.Set("Cache-Control", r.cache)
		req.Header.Set("Pragma", r.cache)
	}
	return req
}

var typeRequest = reflect.TypeOf((*request)(nil))

func toRequest(rt *sobek.Runtime, value sobek.Value) *request {
	if value.ExportType() == typeRequest {
		return value.Export().(*request)
	}
	panic(rt.NewTypeError(`Value of "this" must be of type Request`))
}

func initRequest(rt *sobek.Runtime, opt sobek.Value, req *request) {
	if sobek.IsUndefined(opt) {
		req.headers = js.New(rt, "Headers", rt.ToValue(headers{}))
		return
	}
	init := opt.ToObject(rt)
	if method := init.Get("method"); method != nil {
		req.method = strings.ToUpper(method.String())
	}
	if mode := init.Get("mode"); mode != nil {
		req.mode = mode.String()
	}
	if credentials := init.Get("credentials"); credentials != nil {
		req.credentials = credentials.String()
	}
	if cache := init.Get("cache"); cache != nil {
		req.cache = cache.String()
	}
	if redirect := init.Get("redirect"); redirect != nil {
		req.redirect = redirect.String()
	}
	if referrer := init.Get("referrer"); referrer != nil {
		req.referrer = referrer.String()
	}
	if integrity := init.Get("integrity"); integrity != nil {
		req.integrity = integrity.String()
	}
	if signal := init.Get("signal"); signal != nil {
		if signal.ExportType() != typeAbortSignal {
			js.Throw(rt, errors.New("options signal is not AbortSignal"))
		}
		req.signal = signal
	}
	if header := init.Get("headers"); header != nil {
		req.headers = js.New(rt, "Headers", header)
	} else {
		req.headers = js.New(rt, "Headers", rt.ToValue(headers{}))
	}
	if req.method == http.MethodGet || req.method == http.MethodHead {
		return
	}
	if b := init.Get("body"); b != nil {
		var body io.Reader = http.NoBody
		switch b.ExportType() {
		case typeFormData:
			data := b.Export().(*formData)
			reader, contentType, err := data.encode(rt)
			if err != nil {
				js.Throw(rt, err)
			}
			h := req.headers.Export().(headers)
			h["content-type"] = []string{contentType}
			body = reader
		case url.TypeURLSearchParams:
			h := req.headers.Export().(headers)
			h["content-type"] = []string{"application/x-www-form-url"}
			body = strings.NewReader(b.String())
		case stream.TypeReadableStream:
			body = stream.GetStreamSource(rt, b)
		default:
			if data, ok := buffer.GetBuffer(rt, b); ok {
				body = bytes.NewReader(data)
			} else {
				body = strings.NewReader(b.String())
				h := req.headers.Export().(headers)
				if _, ok := h["content-type"]; !ok {
					h["content-type"] = []string{"text/plain;charset=UTF-8"}
				}
			}
		}
		req.body = io.NopCloser(body)
	}
}
