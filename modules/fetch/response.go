package fetch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
	"github.com/shiroyk/ski/modules/stream"
	"github.com/shiroyk/ski/modules/url"
)

var (
	errBodyAlreadyRead  = errors.New("body stream already read")
	errBodyStreamLocked = errors.New("body stream is locked")
	errBodyStreamRead   = errors.New("body stream already read")
)

// Response represents the response to a request.
// https://developer.mozilla.org/en-US/docs/Web/API/Response
type Response struct{}

func (r *Response) prototype(rt *sobek.Runtime) *sobek.Object {
	p := rt.NewObject()
	_ = p.DefineAccessorProperty("body", rt.ToValue(r.body), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("bodyUsed", rt.ToValue(r.bodyUsed), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("status", rt.ToValue(r.status), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("statusText", rt.ToValue(r.statusText), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("ok", rt.ToValue(r.ok), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("headers", rt.ToValue(r.headers), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("type", rt.ToValue(r.type_), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("url", rt.ToValue(r.url), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("redirected", rt.ToValue(r.redirected), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.Set("bytes", r.bytes)
	_ = p.Set("blob", r.blob)
	_ = p.Set("formData", r.formData)
	_ = p.Set("clone", r.clone)
	_ = p.Set("text", r.text)
	_ = p.Set("json", r.json)
	_ = p.Set("arrayBuffer", r.arrayBuffer)
	_ = p.Set("error", r.error)
	_ = p.Set("redirect", r.redirect)
	_ = p.SetSymbol(sobek.SymToStringTag, func(sobek.FunctionCall) sobek.Value { return rt.ToValue("Response") })
	return p
}

func unsafeReasonPhrase(r rune) bool {
	switch {
	case r == 0x09 || r == 0x20: // HTAB, SP
	case r >= 0x21 && r <= 0x7E: // VCHAR
	case r >= 0x80 && r <= 0xFF: // obs-text
	default:
		return true
	}
	return false
}

func (r *Response) constructor(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	res := &response{
		status:   http.StatusOK,
		bodyUsed: new(atomic.Bool),
		type_:    "default",
	}

	if opt := call.Argument(1); !sobek.IsUndefined(opt) {
		init := opt.ToObject(rt)
		if v := init.Get("status"); v != nil {
			status := int(v.ToInteger())
			if status < http.StatusOK || status > 599 {
				panic(types.New(rt, "RangeError", rt.ToValue("The status code is outside the range [200, 599]")))
			}
			res.status = status
		}
		if v := init.Get("statusText"); v != nil {
			res.statusText = v.String()
			if strings.ContainsFunc(res.statusText, unsafeReasonPhrase) {
				panic(rt.NewTypeError("Invalid statusText"))
			}
		}
		if v := init.Get("headers"); v != nil {
			res.headers = types.New(rt, "Headers", v)
		}
	}

	if res.headers == nil {
		res.headers = types.New(rt, "Headers")
	}

	if arg := call.Argument(0); !sobek.IsUndefined(arg) {
		switch arg.ExportType() {
		case typeFormData:
			data := arg.Export().(*formData)
			reader, t, err := data.encode()
			if err != nil {
				js.Throw(rt, err)
			}
			res.body = reader
			setContentType(res.headers, NormalizeHeaderValue(t))
		case url.TypeURLSearchParams:
			res.body = strings.NewReader(arg.String())
			setContentType(res.headers, "application/x-www-form-urlencoded;charset=UTF-8")
		default:
			if v, t, ok := buffer.GetReader(arg); ok {
				all, err := buffer.ReadAll(v)
				if err != nil {
					js.Throw(rt, err)
				}
				res.body = bytes.NewReader(all)
				if t != "" {
					setContentType(res.headers, t)
				}
			} else if v, ok := buffer.GetBuffer(rt, arg); ok {
				res.body = bytes.NewReader(slices.Clone(v))
			} else if !sobek.IsNull(arg) {
				res.body = strings.NewReader(arg.String())
				setContentType(res.headers, "text/plain;charset=UTF-8")
			}
		}
	}

	switch res.status {
	case http.StatusNoContent, http.StatusResetContent, http.StatusNotModified:
		if res.body != nil {
			panic(rt.NewTypeError("Response with no content status cannot have body"))
		}
	}

	obj := rt.NewObject()
	_ = obj.SetSymbol(symResponse, res)
	_ = obj.SetPrototype(call.This.Prototype())
	return obj
}

func (r *Response) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	proto := r.prototype(rt)
	ctor := rt.ToValue(r.constructor).(*sobek.Object)
	_ = proto.DefineDataProperty("constructor", ctor, sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_FALSE)
	_ = ctor.Set("prototype", proto)
	_ = ctor.SetPrototype(proto)
	return ctor, nil
}

func (*Response) bodyUsed(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toResponse(rt, call.This).bodyUsed.Load())
}

func (*Response) status(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toResponse(rt, call.This).status)
}

func (*Response) statusText(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toResponse(rt, call.This).statusText)
}

func (*Response) ok(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toResponse(rt, call.This)
	return rt.ToValue(this.status >= 200 && this.status < 300)
}

func (*Response) headers(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return toResponse(rt, call.This).headers
}

func (*Response) type_(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toResponse(rt, call.This).type_)
}

func (*Response) url(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toResponse(rt, call.This).url)
}

func (*Response) redirected(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toResponse(rt, call.This).redirected)
}

// redirect returns a Response resulting in a redirect to the specified URL.
func (*Response) redirect(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if call.This.ExportType() == types.TypeFunc {
		u := call.Argument(0).String()
		_, err := urlpkg.Parse(u)
		if err != nil {
			panic(rt.NewTypeError(err.Error()))
		}
		res := &response{
			url:      u,
			status:   http.StatusFound,
			bodyUsed: new(atomic.Bool),
			headers: types.New(rt, "Headers", rt.ToValue(Header{
				"location": []string{u},
			})),
			type_: "default",
		}

		if v := call.Argument(1); !sobek.IsUndefined(v) {
			res.status = int(v.ToInteger())
			switch res.status {
			case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther,
				http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
			default:
				panic(types.New(rt, "RangeError", rt.ToValue("Invalid status code")))
			}
		}

		obj := rt.NewObject()
		_ = obj.SetSymbol(symResponse, res)
		_ = obj.SetPrototype(call.This.ToObject(rt).Prototype())
		return obj
	}
	panic(rt.NewTypeError(`(intermediate value).redirect is not a function`))
}

// clone returns a copy of the Response object
func (*Response) clone(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toResponse(rt, call.This)
	if this.bodyUsed.Load() {
		panic(rt.NewTypeError("Response body already used"))
	}
	body := this.body
	if body != nil {
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
	obj := rt.NewObject()

	clone := *this
	res := &clone
	res.body = body
	res.bodyUsed = new(atomic.Bool)
	res.headers = types.New(rt, "Headers", this.headers)
	_ = obj.SetSymbol(symResponse, res)
	_ = obj.SetPrototype(call.This.ToObject(rt).Prototype())
	return obj
}

// formData returns a promise which resolves with the body as a FormData object.
func (*Response) formData(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toResponse(rt, call.This)
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

// text returns a promise which resolves with the body text as a string.
func (*Response) text(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toResponse(rt, call.This)
	return promise.New(rt, func(callback promise.Callback) {
		v, err := this.read()
		callback(func() (any, error) {
			if err != nil {
				panic(rt.NewTypeError(err.Error()))
			}
			return string(v), nil
		})
	})
}

// json static method of the Response interface returns a Response that contains the provided JSON data as body.
// returns a promise which resolves with the result of parsing the body text as JSON.
func (*Response) json(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if call.This.ExportType() == types.TypeFunc {
		data := call.Argument(0)
		if sobek.IsUndefined(data) {
			panic(rt.NewTypeError("Response.json requires at least 1 arguments"))
		}
		j, ok := rt.Get("JSON").(*sobek.Object)
		if !ok {
			panic(rt.NewTypeError("JSON is not defined"))
		}
		stringify, ok := sobek.AssertFunction(j.Get("stringify"))
		if !ok {
			panic(rt.NewTypeError("JSON.stringify is not defined"))
		}
		v, err := stringify(j, data)
		if err != nil {
			panic(err)
		}
		s := v.String()
		if s == "undefined" {
			panic(rt.NewTypeError("Response.json argument must be JSON serializable"))
		}
		ret := rt.NewObject()
		res := &response{
			statusText: "",
			status:     200,
			bodyUsed:   new(atomic.Bool),
			body:       io.NopCloser(strings.NewReader(s)),
			type_:      "default",
		}
		if arg := call.Argument(1); !sobek.IsUndefined(arg) {
			opts := arg.ToObject(rt)
			if v := opts.Get("headers"); v != nil {
				res.headers = types.New(rt, "Headers", v)
			}
			if v := opts.Get("status"); v != nil {
				res.status = int(v.ToInteger())
				switch res.status {
				case http.StatusNoContent, http.StatusResetContent, http.StatusNotModified:
					panic(rt.NewTypeError("Response with null body status cannot have body"))
				}
			}
			if v := opts.Get("statusText"); v != nil {
				res.statusText = v.String()
			}
		}
		if res.headers == nil {
			res.headers = types.New(rt, "Headers", rt.ToValue(Header{"content-type": {"application/json"}}))
		} else {
			h := res.headers.Export().(Header)
			if _, ok := h["content-type"]; !ok {
				h["content-type"] = []string{"application/json"}
			}
		}
		_ = ret.SetSymbol(symResponse, res)
		_ = ret.SetPrototype(call.This.ToObject(rt).Prototype())
		return ret
	}
	this := toResponse(rt, call.This)
	return promise.New(rt, func(callback promise.Callback) {
		v, err := this.read()
		callback(func() (any, error) {
			if err != nil {
				panic(rt.NewTypeError(err.Error()))
			}
			var ret any
			if err = json.Unmarshal(v, &ret); err != nil {
				panic(types.New(rt, "SyntaxError", rt.ToValue(err.Error())))
			}
			return ret, nil
		})
	})
}

// arrayBuffer returns a promise that resolves with an ArrayBuffer.
func (*Response) arrayBuffer(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toResponse(rt, call.This)
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

// error returns a new Response object associated with a network error.
func (*Response) error(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if call.This.ExportType() == types.TypeFunc {
		res := rt.NewObject()
		_ = res.SetSymbol(symResponse, &response{
			statusText: "",
			status:     0,
			bodyUsed:   new(atomic.Bool),
			headers:    types.New(rt, "Headers"),
			type_:      "error",
		})
		_ = res.SetPrototype(call.This.ToObject(rt).Prototype())
		return res
	}
	panic(rt.NewTypeError(`(intermediate value).error is not a function`))
}

// body returns a ReadableStream
func (*Response) body(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toResponse(rt, call.This)
	if this.body == nil {
		return sobek.Null()
	}
	if this.bodyStream == nil {
		this.bodyStream = stream.NewReadableStream(rt, this.body)
	}
	return this.bodyStream
}

// bytes returns a promise that resolves with a Uint8Array.
func (*Response) bytes(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toResponse(rt, call.This)
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

// blob returns a promise that resolves with a Blob.
func (*Response) blob(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toResponse(rt, call.This)
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

type response struct {
	status              int
	statusText          string
	headers, bodyStream sobek.Value
	body                io.Reader
	bodyUsed            *atomic.Bool
	redirected          bool
	url, type_          string
}

func (r *response) form() (*multipart.Form, error) {
	if stream.IsLocked(r.bodyStream) {
		return nil, errBodyStreamLocked
	}
	return parseFromData(r.body, r.bodyUsed, getContentType(r.headers))
}

func (r *response) read() ([]byte, error) {
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
		return nil, errBodyStreamRead
	}
	if c, ok := r.body.(io.Closer); ok {
		defer c.Close()
	}
	data, err := io.ReadAll(r.body)
	if err != nil {
		return nil, err
	}
	r.bodyUsed.Store(true)
	return data, nil
}

func (r *response) String() string {
	return fmt.Sprintf("[Response %d %s]", r.status, r.statusText)
}

var symResponse = sobek.NewSymbol("Symbol.Response")

// toResponse converts a js Response object to a response
func toResponse(rt *sobek.Runtime, value sobek.Value) *response {
	if o, ok := value.(*sobek.Object); ok {
		if v := o.GetSymbol(symResponse); v != nil {
			return v.Export().(*response)
		}
	}
	panic(rt.NewTypeError(`Value of "this" must be of type Response`))
}

// NewResponse returns a new js Response
func NewResponse(rt *sobek.Runtime, res *http.Response) sobek.Value {
	instance := &response{
		status:     res.StatusCode,
		statusText: res.Status,
		headers:    types.New(rt, "Headers", rt.ToValue(Header(res.Header))),
		body:       res.Body,
		bodyUsed:   new(atomic.Bool),
		type_:      "basic",
	}
	if res.Request != nil {
		instance.url = res.Request.URL.String()
		if location := res.Header.Get("Location"); location != "" {
			instance.redirected = location != instance.url
		}
	}
	obj := types.New(rt, "Response")
	_ = obj.SetSymbol(symResponse, instance)
	return obj
}

// ToResponse converts a js Response object to an http.Response
func ToResponse(value sobek.Value) (*http.Response, bool) {
	if o, ok := value.(*sobek.Object); ok {
		if v := o.GetSymbol(symResponse); v != nil {
			res := v.Export().(*response)
			var body io.ReadCloser
			if body, ok = res.body.(io.ReadCloser); !ok {
				body = io.NopCloser(res.body)
			}
			return &http.Response{
				StatusCode: res.status,
				Status:     res.statusText,
				Header:     http.Header(res.headers.Export().(Header)),
				Body:       body,
			}, true
		}
	}
	return nil, false
}
