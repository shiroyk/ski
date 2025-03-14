package fetch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"slices"
	"strings"

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

func (r *Response) constructor(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	res := &response{
		status:     "200 OK",
		statusCode: http.StatusOK,
		type_:      "default",
	}

	if opt := call.Argument(1); !sobek.IsUndefined(opt) {
		init := opt.ToObject(rt)
		if v := init.Get("status"); v != nil {
			code := int(v.ToInteger())
			if code < http.StatusOK || code > http.StatusNetworkAuthenticationRequired {
				panic(js.New(rt, "RangeError", rt.ToValue("Invalid status code")))
			}
			res.statusCode = code
			res.status = fmt.Sprintf("%d %s", code, http.StatusText(code))
		}
		if v := init.Get("statusText"); v != nil {
			res.status = fmt.Sprintf("%d %s", res.statusCode, v.String())
		}
		if v := init.Get("headers"); v != nil {
			res.headers = js.New(rt, "Headers", v)
		}
	}

	if res.headers == nil {
		res.headers = js.New(rt, "Headers")
	}

	if arg := call.Argument(0); !sobek.IsUndefined(arg) {
		switch arg.ExportType() {
		case typeFormData:
			data := arg.Export().(*formData)
			reader, t, err := data.encode()
			if err != nil {
				js.Throw(rt, err)
			}
			if reader != nil {
				res.body = reader
				h := res.headers.Export().(headers)
				h["content-type"] = []string{strings.ToLower(t)}
			}
		case url.TypeURLSearchParams:
			res.body = strings.NewReader(arg.String())
			h := res.headers.Export().(headers)
			h["content-type"] = []string{"application/x-www-form-urlencoded;charset=UTF-8"}
		default:
			if v, t, ok := buffer.GetReader(arg); ok {
				all, err := buffer.ReadAll(v)
				if err != nil {
					js.Throw(rt, err)
				}
				res.body = bytes.NewReader(all)
				if t != "" {
					h := res.headers.Export().(headers)
					if _, ok := h["content-type"]; !ok {
						h["content-type"] = []string{strings.ToLower(t)}
					}
				}
			} else if v, ok := buffer.GetBuffer(rt, arg); ok {
				res.body = bytes.NewReader(slices.Clone(v))
			} else {
				res.body = strings.NewReader(arg.String())
			}
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
	return rt.ToValue(toResponse(rt, call.This).bodyUsed)
}

func (*Response) status(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toResponse(rt, call.This).statusCode)
}

func (*Response) statusText(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toResponse(rt, call.This).status)
}

func (*Response) ok(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toResponse(rt, call.This)
	return rt.ToValue(this.statusCode >= 200 && this.statusCode < 300)
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

func (*Response) redirect(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if call.This.ExportType() == types.TypeFunc {
		u := call.Argument(0).String()
		res := &response{
			url:        u,
			statusCode: 302,
			headers: js.New(rt, "Headers", rt.ToValue(headers{
				"location": []string{u},
			})),
			type_: "default",
		}

		if v := call.Argument(1); !sobek.IsUndefined(v) {
			res.statusCode = int(v.ToInteger())
		}

		obj := rt.NewObject()
		_ = obj.SetSymbol(symResponse, res)
		_ = obj.SetPrototype(call.This.ToObject(rt).Prototype())
		return obj
	}
	panic(rt.NewTypeError(`(intermediate value).redirect is not a function`))
}

func (*Response) clone(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toResponse(rt, call.This)
	body := this.body
	if body != nil && !this.bodyUsed {
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
	_ = obj.SetSymbol(symResponse, &response{
		statusCode: this.statusCode,
		headers:    this.headers,
		body:       body,
		bodyUsed:   this.bodyUsed,
		url:        this.url,
		type_:      this.type_,
	})
	_ = obj.SetPrototype(call.This.ToObject(rt).Prototype())
	return obj
}

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
			status:     "",
			statusCode: 200,
			body:       io.NopCloser(strings.NewReader(s)),
			type_:      "default",
			bodyUsed:   false,
		}
		if arg := call.Argument(1); !sobek.IsUndefined(arg) {
			opts := arg.ToObject(rt)
			if v := opts.Get("headers"); v != nil {
				res.headers = js.New(rt, "Headers", v)
			}
			if v := opts.Get("status"); v != nil {
				res.statusCode = int(v.ToInteger())
				switch res.statusCode {
				case http.StatusNoContent, http.StatusResetContent, http.StatusNotModified:
					panic(rt.NewTypeError("Response with null body status cannot have body"))
				}
			}
			if v := opts.Get("statusText"); v != nil {
				res.status = v.String()
			}
		}
		if res.headers == nil {
			res.headers = js.New(rt, "Headers", rt.ToValue(headers{"content-type": {"application/json"}}))
		} else {
			h := res.headers.Export().(headers)
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
				panic(js.New(rt, "SyntaxError", rt.ToValue(err.Error())))
			}
			return ret, nil
		})
	})
}

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

func (*Response) error(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if call.This.ExportType() == types.TypeFunc {
		res := rt.NewObject()
		_ = res.SetSymbol(symResponse, &response{
			status:     "",
			statusCode: 0,
			headers:    js.New(rt, "Headers"),
			type_:      "error",
			bodyUsed:   false,
		})
		_ = res.SetPrototype(call.This.ToObject(rt).Prototype())
		return res
	}
	panic(rt.NewTypeError(`(intermediate value).error is not a function`))
}

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

func (*Response) bytes(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toResponse(rt, call.This)
	return promise.New(rt, func(callback promise.Callback) {
		data, err := this.read()
		callback(func() (any, error) {
			if err != nil {
				panic(rt.NewTypeError(err.Error()))
			}
			return js.New(rt, "Uint8Array", rt.ToValue(rt.NewArrayBuffer(data))), nil
		})
	})
}

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
			return js.New(rt, "Blob", rt.NewArray(rt.NewArrayBuffer(data)), opt), nil
		})
	})
}

type response struct {
	status               string
	statusCode           int
	headers, bodyStream  sobek.Value
	body                 io.Reader
	bodyUsed, redirected bool
	url, type_           string
}

func (r *response) form() (*multipart.Form, error) {
	if stream.IsLocked(r.bodyStream) {
		return nil, errBodyStreamLocked
	}
	return parseFromData(r.body, &r.bodyUsed, getContentType(r.headers))
}

func (r *response) read() ([]byte, error) {
	if r.body == nil {
		return []byte{}, nil
	}
	if r.bodyUsed {
		return nil, errBodyAlreadyRead
	}
	if stream.IsLocked(r.bodyStream) {
		return nil, errBodyStreamLocked
	}
	if c, ok := r.body.(io.Closer); ok {
		defer c.Close()
	}
	data, err := io.ReadAll(r.body)
	if err != nil {
		return nil, err
	}
	r.bodyUsed = true
	return data, nil
}

func (r *response) String() string {
	return fmt.Sprintf("[Response %d %s]", r.statusCode, r.status)
}

var symResponse = sobek.NewSymbol("Symbol.Response")

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
		status:     res.Status,
		statusCode: res.StatusCode,
		headers:    js.New(rt, "Headers", rt.ToValue(headers(res.Header))),
		body:       res.Body,
		type_:      "basic",
	}
	if res.Request != nil {
		instance.url = res.Request.URL.String()
		if location := res.Header.Get("Location"); location != "" {
			instance.redirected = location != instance.url
		}
	}
	obj := js.New(rt, "Response")
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
				Status:     res.status,
				StatusCode: res.statusCode,
				Header:     http.Header(res.headers.Export().(headers)),
				Body:       body,
			}, true
		}
	}
	return nil, false
}
