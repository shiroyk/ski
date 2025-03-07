package fetch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/promise"
	"github.com/shiroyk/ski/js/types"
	"github.com/shiroyk/ski/modules/buffer"
	"github.com/shiroyk/ski/modules/stream"
)

var errBodyAlreadyRead = errors.New("body stream already read")

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
	_ = p.Set("bytes", r.bytes)
	_ = p.Set("blob", r.blob)
	_ = p.Set("formData", r.formData)
	_ = p.Set("clone", r.clone)
	_ = p.Set("text", r.text)
	_ = p.Set("json", r.json)
	_ = p.Set("arrayBuffer", r.arrayBuffer)
	_ = p.SetSymbol(sobek.SymToStringTag, func(sobek.FunctionCall) sobek.Value { return rt.ToValue("Response") })
	return p
}

func (r *Response) constructor(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	var body io.Reader = http.NoBody
	if arg := call.Argument(0); !sobek.IsUndefined(arg) {
		switch arg.ExportType() {
		case typeFormData:
			data := arg.Export().(*formData)
			reader, _, err := data.encode(rt)
			if err != nil {
				js.Throw(rt, err)
			}
			body = reader
		default:
			if v, ok := buffer.GetReader(arg); ok {
				body = v
			} else if v, ok := buffer.GetBuffer(rt, arg); ok {
				body = bytes.NewReader(v)
			} else {
				body = strings.NewReader(arg.String())
			}
		}
	}

	instance := &response{
		status:     "200 OK",
		statusCode: http.StatusOK,
		body:       body,
		type_:      "default",
	}

	if opt := call.Argument(1); !sobek.IsUndefined(opt) {
		init := opt.ToObject(rt)
		if v := init.Get("status"); v != nil {
			code := int(v.ToInteger())
			instance.statusCode = code
			instance.status = fmt.Sprintf("%d %s", code, http.StatusText(code))
		}
		if v := init.Get("statusText"); v != nil {
			instance.status = fmt.Sprintf("%d %s", instance.statusCode, v.String())
		}
		if v := init.Get("headers"); v != nil {
			instance.headers = func() sobek.Value { return js.New(rt, "Headers", v) }
		}
	}

	if instance.headers == nil {
		instance.headers = func() sobek.Value { return js.New(rt, "Headers") }
	}

	obj := rt.NewObject()
	_ = obj.SetSymbol(symResponse, instance)
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
	return toResponse(rt, call.This).headers()
}

func (*Response) type_(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toResponse(rt, call.This).type_)
}

func (*Response) url(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(toResponse(rt, call.This).url)
}

func (*Response) clone(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toResponse(rt, call.This)
	body := this.body
	if !this.bodyUsed {
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
		status:     this.status,
		statusCode: this.statusCode,
		headers:    this.headers,
		body:       body,
		bodyUsed:   this.bodyUsed,
		url:        this.url,
	})
	_ = obj.SetPrototype(call.This.ToObject(rt).Prototype())
	return obj
}

func (*Response) formData(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toResponse(rt, call.This)
	return promise.New(rt, func(callback promise.Callback) {
		b, err := this.text()
		callback(func() (any, error) {
			if err != nil {
				return nil, err
			}
			return js.New(rt, "FormData", rt.ToValue(b)), nil
		})
	})
}

func (*Response) text(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toResponse(rt, call.This)
	return promise.New(rt, func(callback promise.Callback) {
		v, err := this.text()
		callback(func() (any, error) { return v, err })
	})
}

func (*Response) json(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if call.This.ExportType() == types.TypeFunc {
		data := call.Argument(0)
		if sobek.IsUndefined(data) {
			panic(rt.NewTypeError("Response.json requires at least 1 arguments"))
		}
		b, err := data.ToObject(rt).MarshalJSON()
		if err != nil {
			js.Throw(rt, err)
		}
		res := rt.NewObject()
		_ = res.SetSymbol(symResponse, &response{
			status:     "200 OK",
			statusCode: http.StatusOK,
			body:       io.NopCloser(bytes.NewReader(b)),
			headers: sync.OnceValue(func() sobek.Value {
				return js.New(rt, "Headers", rt.ToValue(headers{"content-type": {"application/json"}}))
			}),
			type_:    "default",
			bodyUsed: false,
		})
		_ = res.SetPrototype(call.This.ToObject(rt).Prototype())
		return res
	}
	this := toResponse(rt, call.This)
	return promise.New(rt, func(callback promise.Callback) {
		v, err := this.json()
		callback(func() (any, error) { return v, err })
	})
}

func (*Response) arrayBuffer(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toResponse(rt, call.This)
	return promise.New(rt, func(callback promise.Callback) {
		data, err := this.read()
		callback(func() (any, error) {
			if err != nil {
				return nil, err
			}
			return rt.NewArrayBuffer(data), nil
		})
	})
}

func (*Response) body(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toResponse(rt, call.This)
	return stream.NewReadableStream(rt, this.body)
}

func (*Response) bytes(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toResponse(rt, call.This)
	return promise.New(rt, func(callback promise.Callback) {
		data, err := this.read()
		callback(func() (any, error) {
			if err != nil {
				return nil, err
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
				return nil, err
			}
			return js.New(rt, "Blob", rt.ToValue(rt.NewArrayBuffer(data))), nil
		})
	})
}

type response struct {
	status               string
	statusCode           int
	headers              func() sobek.Value
	body                 io.Reader
	bodyUsed, redirected bool
	url, type_           string
}

func (r *response) read() ([]byte, error) {
	if r.bodyUsed {
		return nil, errBodyAlreadyRead
	}
	r.bodyUsed = true
	if c, ok := r.body.(io.Closer); ok {
		defer c.Close()
	}
	data, err := io.ReadAll(r.body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (r *response) text() (string, error) {
	data, err := r.read()
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (r *response) json() (any, error) {
	data, err := r.read()
	if err != nil {
		return nil, err
	}
	var ret any
	if err = json.Unmarshal(data, &ret); err != nil {
		return nil, err
	}
	return ret, nil
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
		headers: sync.OnceValue(func() sobek.Value {
			h := make(headers, len(res.Header))
			for k := range res.Header {
				h[normalizeHeaderName(k)] = res.Header[k]
			}
			return js.New(rt, "Headers", rt.ToValue(h))
		}),
		body:  res.Body,
		type_: "basic",
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
				Header:     http.Header(res.headers().Export().(headers)),
				Body:       body,
			}, true
		}
	}
	return nil, false
}
