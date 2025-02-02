package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
)

var errBodyAlreadyRead = errors.New("body stream already read")

// NewResponse returns a new js Response
func NewResponse(rt *sobek.Runtime, res *http.Response, async bool) sobek.Value {
	instance := &response{
		status:     res.Status,
		statusCode: res.StatusCode,
		headers: sync.OnceValue(func() sobek.Value {
			h := make(headers, len(res.Header))
			for k := range res.Header {
				h[normalizeHeaderName(k)] = res.Header[k]
			}
			header, _ := js.New(rt, "Headers", rt.ToValue(h))
			return header
		}),
		body:  res.Body,
		type_: "basic",
		async: async,
	}
	if res.Request != nil {
		instance.url = res.Request.URL.String()
		if location := res.Header.Get("Location"); location != "" {
			instance.redirected = location != instance.url
		}
	}
	js.OnDone(rt, instance.close)
	obj := rt.ToValue(instance).(*sobek.Object)
	ctor := rt.Get("Response")
	if ctor == nil {
		panic(rt.NewTypeError("Response is not defined"))
	}
	_ = obj.SetPrototype(ctor.ToObject(rt).Prototype())
	return obj
}

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
	_ = p.SetSymbol(sobek.SymToStringTag, func(sobek.ConstructorCall) sobek.Value { return rt.ToValue("Response") })
	return p
}

func (r *Response) constructor(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	var body io.Reader = http.NoBody
	if arg := call.Argument(0); !sobek.IsUndefined(arg) {
		switch arg.ExportType() {
		case typeBlob, typeFile:
			body = toBlob(rt, arg).data
		case typeFormData:
			data := arg.Export().(*formData)
			reader, _, err := data.encode()
			if err != nil {
				js.Throw(rt, err)
			}
			body = reader
		case typeArrayBuffer:
			buffer := arg.Export().(sobek.ArrayBuffer)
			body = bytes.NewReader(buffer.Bytes())
		case typeBytes:
			buffer := arg.Export().([]byte)
			body = bytes.NewReader(buffer)
		default:
			body = strings.NewReader(arg.String())
		}
	}

	instance := &response{
		status:     "200 OK",
		statusCode: http.StatusOK,
		body:       io.NopCloser(body),
		type_:      "default",
	}

	if opt := call.Argument(1); !sobek.IsUndefined(opt) {
		init := opt.ToObject(rt)
		if status := init.Get("status"); status != nil {
			code := int(status.ToInteger())
			instance.statusCode = code
			instance.status = fmt.Sprintf("%d %s", code, http.StatusText(code))
		}
		if statusText := init.Get("statusText"); statusText != nil {
			instance.status = fmt.Sprintf("%d %s", instance.statusCode, statusText.String())
		}

		v := init.Get("headers")
		if v == nil {
			v = sobek.Undefined()
		}
		h, err := js.New(rt, "Headers", v)
		if err != nil {
			js.Throw(rt, err)
		}
		instance.headers = func() sobek.Value { return h }
	}

	obj := rt.ToValue(instance).(*sobek.Object)
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

func (*Response) Global() {}

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
		defer body.Close()
		_, err := io.Copy(io.MultiWriter(b1, b2), body)
		if err != nil {
			js.Throw(rt, err)
		}
		this.body = io.NopCloser(b1)
		body = io.NopCloser(b2)
	}
	obj := rt.ToValue(&response{
		status:     this.status,
		statusCode: this.statusCode,
		headers:    this.headers,
		body:       body,
		bodyUsed:   this.bodyUsed,
		url:        this.url,
		async:      this.async,
	}).(*sobek.Object)
	_ = obj.SetPrototype(call.This.ToObject(rt).Prototype())
	return obj
}

func (*Response) formData(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toResponse(rt, call.This)
	if this.async {
		return rt.ToValue(js.NewPromise(rt, this.text, func(b string, err error) (any, error) {
			if err != nil {
				return nil, err
			}
			return js.New(rt, "FormData", rt.ToValue(b))
		}))
	}
	data, err := this.text()
	if err != nil {
		js.Throw(rt, err)
	}
	f, err := js.New(rt, "FormData", rt.ToValue(data))
	if err != nil {
		js.Throw(rt, err)
	}
	return f
}

func (*Response) text(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toResponse(rt, call.This)
	if this.async {
		return rt.ToValue(js.NewPromise(rt, this.text))
	}
	data, err := this.text()
	if err != nil {
		js.Throw(rt, err)
	}
	return rt.ToValue(data)
}

func (*Response) json(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	if call.This == rt.Get("Response") {
		data := call.Argument(0)
		if sobek.IsUndefined(data) {
			panic(rt.NewTypeError("Response.json requires at least 1 arguments"))
		}
		b, err := data.ToObject(rt).MarshalJSON()
		if err != nil {
			js.Throw(rt, err)
		}
		res := rt.ToValue(&response{
			status:     "200 OK",
			statusCode: http.StatusOK,
			body:       io.NopCloser(bytes.NewReader(b)),
			headers: sync.OnceValue(func() sobek.Value {
				ret, _ := js.New(rt, "Headers", rt.ToValue(headers{"content-type": {"application/json"}}))
				return ret
			}),
			type_:    "default",
			bodyUsed: false,
		}).(*sobek.Object)
		_ = res.SetPrototype(call.This.ToObject(rt).Prototype())
		return res
	}
	this := toResponse(rt, call.This)
	if this.async {
		return rt.ToValue(js.NewPromise(rt, this.json))
	}
	data, err := this.json()
	if err != nil {
		js.Throw(rt, err)
	}
	return rt.ToValue(data)
}

func (*Response) arrayBuffer(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toResponse(rt, call.This)
	if this.async {
		return rt.ToValue(js.NewPromise(rt, this.read, func(data []byte, err error) (any, error) {
			if err != nil {
				return nil, err
			}
			return rt.NewArrayBuffer(data), nil
		}))
	}
	data, err := this.read()
	if err != nil {
		js.Throw(rt, err)
	}
	return rt.ToValue(rt.NewArrayBuffer(data))
}

func (*Response) body(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toResponse(rt, call.This)
	rs := &readableStream{source: this.body}
	ctor := rt.Get("ReadableStream")
	if ctor == nil {
		panic(rt.NewTypeError("ReadableStream is not defined"))
	}
	obj := rt.ToValue(rs).(*sobek.Object)
	_ = obj.SetPrototype(ctor.ToObject(rt).Prototype())
	return obj
}

func (*Response) bytes(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toResponse(rt, call.This)
	if this.async {
		return rt.ToValue(js.NewPromise(rt, this.read, func(data []byte, err error) (any, error) {
			if err != nil {
				return nil, err
			}
			return js.New(rt, "Uint8Array", rt.ToValue(rt.NewArrayBuffer(data)))
		}))
	}
	data, err := this.read()
	if err != nil {
		js.Throw(rt, err)
	}
	r, err := js.New(rt, "Uint8Array", rt.ToValue(rt.NewArrayBuffer(data)))
	if err != nil {
		js.Throw(rt, err)
	}
	return r
}

func (*Response) blob(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toResponse(rt, call.This)
	if this.async {
		return rt.ToValue(js.NewPromise(rt, this.read, func(data []byte, err error) (any, error) {
			if err != nil {
				return nil, err
			}
			return js.New(rt, "Blob", rt.ToValue(rt.NewArrayBuffer(data)))
		}))
	}
	data, err := this.read()
	if err != nil {
		js.Throw(rt, err)
	}
	r, err := js.New(rt, "Blob", rt.ToValue(rt.NewArrayBuffer(data)))
	if err != nil {
		js.Throw(rt, err)
	}
	return r
}

type response struct {
	status               string
	statusCode           int
	headers              func() sobek.Value
	body                 io.ReadCloser
	bodyUsed, redirected bool
	url, type_           string
	async                bool
}

func (r *response) close() {
	if !r.bodyUsed {
		r.body.Close()
	}
}

func (r *response) read() ([]byte, error) {
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

var typeResponse = reflect.TypeOf((*response)(nil))

func toResponse(rt *sobek.Runtime, value sobek.Value) *response {
	if value.ExportType() == typeResponse {
		return value.Export().(*response)
	}
	panic(rt.NewTypeError(`Value of "this" must be of type Response`))
}
